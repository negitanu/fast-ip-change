package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

var (
	mainWindow  *walk.MainWindow
	routeTable  *walk.TableView
	routeModel  *RouteModel
	lastUpdate  *walk.Label
)

// RouteEntry はルーティングエントリを表します
type RouteEntry struct {
	Destination string
	Netmask     string
	Gateway     string
	Interface   string
	Metric      int
	MetricStr   string
}

// RouteModel はテーブルモデルです
type RouteModel struct {
	walk.TableModelBase
	walk.SorterBase
	items     []RouteEntry
	sortCol   int
	sortOrder walk.SortOrder
}

func (m *RouteModel) RowCount() int {
	return len(m.items)
}

func (m *RouteModel) Value(row, col int) interface{} {
	item := m.items[row]
	switch col {
	case 0:
		return item.Destination
	case 1:
		return item.Netmask
	case 2:
		return item.Gateway
	case 3:
		return item.Interface
	case 4:
		return item.MetricStr
	default:
		return ""
	}
}

func (m *RouteModel) Sort(col int, order walk.SortOrder) error {
	m.sortCol = col
	m.sortOrder = order

	sort.SliceStable(m.items, func(i, j int) bool {
		var less bool
		switch col {
		case 0:
			less = compareIP(m.items[i].Destination, m.items[j].Destination)
		case 1:
			less = compareIP(m.items[i].Netmask, m.items[j].Netmask)
		case 2:
			less = compareIP(m.items[i].Gateway, m.items[j].Gateway)
		case 3:
			less = compareIP(m.items[i].Interface, m.items[j].Interface)
		case 4:
			less = m.items[i].Metric < m.items[j].Metric
		default:
			return false
		}

		if order == walk.SortDescending {
			return !less
		}
		return less
	})

	return m.SorterBase.Sort(col, order)
}

// compareIP はIPアドレスを数値的に比較します
func compareIP(a, b string) bool {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	for i := 0; i < 4 && i < len(partsA) && i < len(partsB); i++ {
		numA, _ := strconv.Atoi(partsA[i])
		numB, _ := strconv.Atoi(partsB[i])
		if numA != numB {
			return numA < numB
		}
	}
	return a < b
}

func main() {
	routeModel = &RouteModel{
		items: []RouteEntry{},
	}

	err := MainWindow{
		Title:    "Fast IP Change - ルーティングテーブル",
		Size:     Size{Width: 900, Height: 500},
		MinSize:  Size{Width: 700, Height: 400},
		Layout:   VBox{Margins: Margins{Top: 10, Left: 10, Right: 10, Bottom: 10}},
		AssignTo: &mainWindow,
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{
						Text: "IPv4 ルーティングテーブル",
						Font: Font{Bold: true, PointSize: 10},
					},
					HSpacer{},
					Label{
						AssignTo: &lastUpdate,
						Text:     "",
					},
				},
			},
			TableView{
				AssignTo:         &routeTable,
				Model:            routeModel,
				AlternatingRowBG: true,
				ColumnsOrderable: true,
				Columns: []TableViewColumn{
					{Title: "宛先ネットワーク", Width: 150},
					{Title: "ネットマスク", Width: 150},
					{Title: "ゲートウェイ", Width: 150},
					{Title: "インターフェース", Width: 150},
					{Title: "メトリック", Width: 80},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						Text: "更新",
						OnClicked: func() {
							refreshRouteTable()
						},
					},
					PushButton{
						Text: "コマンドプロンプトで確認",
						OnClicked: func() {
							openRouteCmd()
						},
					},
					HSpacer{},
					PushButton{
						Text: "閉じる",
						OnClicked: func() {
							mainWindow.Close()
						},
					},
				},
			},
		},
	}.Create()
	if err != nil {
		walk.MsgBox(nil, "エラー", fmt.Sprintf("ウィンドウの作成に失敗: %v", err), walk.MsgBoxIconError)
		return
	}

	// 初期データを読み込み
	refreshRouteTable()

	mainWindow.Run()
}

func refreshRouteTable() {
	routeModel.items = getRouteEntries()
	routeModel.PublishRowsReset()
	lastUpdate.SetText(fmt.Sprintf("最終更新: %s", time.Now().Format("15:04:05")))
}

// decodeShiftJIS はShift-JISエンコードされたバイト列をUTF-8に変換します
func decodeShiftJIS(b []byte) string {
	decoder := japanese.ShiftJIS.NewDecoder()
	result, _, err := transform.Bytes(decoder, b)
	if err != nil {
		return string(b)
	}
	return string(result)
}

// runCommand はコマンドを実行してUTF-8文字列として結果を返します
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return decodeShiftJIS(output), nil
}

func getRouteEntries() []RouteEntry {
	var entries []RouteEntry

	// route print コマンドでルーティングテーブルを取得
	output, err := runCommand("route", "print", "-4")
	if err != nil {
		return entries
	}

	entries = parseRouteOutput(output)
	return entries
}

func parseRouteOutput(output string) []RouteEntry {
	var entries []RouteEntry

	lines := strings.Split(output, "\n")

	// IPv4ルートテーブルセクションを探す
	inRouteSection := false
	headerFound := false

	// IPアドレスパターン
	ipPattern := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// ルートテーブルセクションの開始を検出
		if strings.Contains(line, "アクティブ ルート") || strings.Contains(line, "Active Routes") {
			inRouteSection = true
			continue
		}

		// 固定ルートセクションや永続ルートセクションで終了
		if strings.Contains(line, "固定ルート") || strings.Contains(line, "Persistent Routes") {
			break
		}

		if !inRouteSection {
			continue
		}

		// ヘッダー行をスキップ
		if strings.Contains(line, "ネットワーク") || strings.Contains(line, "Network") {
			headerFound = true
			continue
		}

		if !headerFound {
			continue
		}

		// 空行やセパレータをスキップ
		if line == "" || strings.HasPrefix(line, "==") || strings.HasPrefix(line, "--") {
			continue
		}

		// ルートエントリをパース
		fields := strings.Fields(line)
		if len(fields) >= 5 {
			// 最初の4つがIPアドレス形式かチェック
			if ipPattern.MatchString(fields[0]) {
				metric, _ := strconv.Atoi(fields[4])
				entry := RouteEntry{
					Destination: fields[0],
					Netmask:     fields[1],
					Gateway:     fields[2],
					Interface:   fields[3],
					Metric:      metric,
					MetricStr:   fields[4],
				}
				entries = append(entries, entry)
			}
		}
	}

	return entries
}

func openRouteCmd() {
	cmd := exec.Command("cmd", "/c", "start", "cmd", "/k", "route print -4")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	cmd.Start()
}
