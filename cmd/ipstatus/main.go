package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
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
	statusTable *walk.TableView
	statusModel *StatusModel
	lastUpdate  *walk.Label
)

// NICStatus はNICの状態を表します
type NICStatus struct {
	Name       string
	Status     string
	IPAddress  string
	SubnetMask string
	Gateway    string
	DNS        string
	DHCP       string
}

// StatusModel はテーブルモデルです
type StatusModel struct {
	walk.TableModelBase
	walk.SorterBase
	items     []NICStatus
	sortCol   int
	sortOrder walk.SortOrder
}

func (m *StatusModel) RowCount() int {
	return len(m.items)
}

func (m *StatusModel) Value(row, col int) interface{} {
	item := m.items[row]
	switch col {
	case 0:
		return item.Name
	case 1:
		return item.Status
	case 2:
		return item.IPAddress
	case 3:
		return item.SubnetMask
	case 4:
		return item.Gateway
	case 5:
		return item.DNS
	case 6:
		return item.DHCP
	default:
		return ""
	}
}

func (m *StatusModel) Sort(col int, order walk.SortOrder) error {
	m.sortCol = col
	m.sortOrder = order

	sort.SliceStable(m.items, func(i, j int) bool {
		var less bool
		switch col {
		case 0:
			less = m.items[i].Name < m.items[j].Name
		case 1:
			less = m.items[i].Status < m.items[j].Status
		case 2:
			less = m.items[i].IPAddress < m.items[j].IPAddress
		case 3:
			less = m.items[i].SubnetMask < m.items[j].SubnetMask
		case 4:
			less = m.items[i].Gateway < m.items[j].Gateway
		case 5:
			less = m.items[i].DNS < m.items[j].DNS
		case 6:
			less = m.items[i].DHCP < m.items[j].DHCP
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

func main() {
	statusModel = &StatusModel{
		items: []NICStatus{},
	}

	err := MainWindow{
		Title:    "Fast IP Change - 現在のネットワーク設定",
		Size:     Size{Width: 1000, Height: 400},
		MinSize:  Size{Width: 800, Height: 300},
		Layout:   VBox{Margins: Margins{Top: 10, Left: 10, Right: 10, Bottom: 10}},
		AssignTo: &mainWindow,
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{
						Text: "ネットワークアダプター一覧",
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
				AssignTo:         &statusTable,
				Model:            statusModel,
				AlternatingRowBG: true,
				ColumnsOrderable: true,
				Columns: []TableViewColumn{
					{Title: "アダプター名", Width: 150},
					{Title: "状態", Width: 80},
					{Title: "IPアドレス", Width: 130},
					{Title: "サブネットマスク", Width: 130},
					{Title: "ゲートウェイ", Width: 130},
					{Title: "DNS", Width: 150},
					{Title: "DHCP", Width: 80},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						Text: "更新",
						OnClicked: func() {
							refreshStatus()
						},
					},
					PushButton{
						Text: "コマンドプロンプトで確認",
						OnClicked: func() {
							openIPConfig()
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
	refreshStatus()

	mainWindow.Run()
}

func refreshStatus() {
	statusModel.items = getNICStatusList()
	statusModel.PublishRowsReset()
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

	// Shift-JISからUTF-8に変換
	return decodeShiftJIS(output), nil
}

func getNICStatusList() []NICStatus {
	var statuses []NICStatus

	// ipconfig /all を使用して情報を取得（より信頼性が高い）
	output, err := runCommand("ipconfig", "/all")
	if err != nil {
		return statuses
	}

	// アダプターごとにパース
	adapters := parseIPConfigOutput(output)

	for _, adapter := range adapters {
		statuses = append(statuses, adapter)
	}

	return statuses
}

func parseIPConfigOutput(output string) []NICStatus {
	var statuses []NICStatus

	// アダプターセクションを分割
	// "イーサネット アダプター" または "Wireless LAN adapter" などで始まるセクション
	adapterPattern := regexp.MustCompile(`(?m)^[^\s].*(?:アダプター|adapter|Adapter).*:`)

	sections := adapterPattern.Split(output, -1)
	matches := adapterPattern.FindAllString(output, -1)

	for i, match := range matches {
		if i+1 >= len(sections) {
			continue
		}

		section := sections[i+1]

		// アダプター名を取得
		name := strings.TrimSuffix(strings.TrimSpace(match), ":")
		// "イーサネット アダプター " などのプレフィックスを除去
		name = regexp.MustCompile(`^.*(?:アダプター|adapter|Adapter)\s*`).ReplaceAllString(name, "")
		name = strings.TrimSpace(name)

		status := NICStatus{
			Name:       name,
			Status:     "不明",
			IPAddress:  "-",
			SubnetMask: "-",
			Gateway:    "-",
			DNS:        "-",
			DHCP:       "-",
		}

		// メディアの状態を確認
		if strings.Contains(section, "メディアは接続されていません") ||
		   strings.Contains(section, "Media disconnected") ||
		   strings.Contains(section, "Media State") && strings.Contains(section, "disconnected") {
			status.Status = "切断"
			statuses = append(statuses, status)
			continue
		}

		lines := strings.Split(section, "\n")
		var dnsServers []string

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			// キーと値を分離（": " または " : " で区切る）
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				// DNSサーバーの追加行（インデントされたIPアドレスのみ）
				if isIPAddress(line) && len(dnsServers) > 0 {
					dnsServers = append(dnsServers, strings.TrimSpace(line))
				}
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// 各項目をパース
			keyLower := strings.ToLower(key)

			// 接続固有の DNS サフィックス があれば接続済み
			if strings.Contains(key, "接続固有") || strings.Contains(keyLower, "connection-specific") {
				status.Status = "接続済み"
			}

			// IPv4 アドレス
			if strings.Contains(key, "IPv4") || strings.Contains(key, "IP Address") || key == "IP アドレス" {
				// "(優先)" などを除去
				value = regexp.MustCompile(`\(.*\)`).ReplaceAllString(value, "")
				status.IPAddress = strings.TrimSpace(value)
				status.Status = "接続済み"
			}

			// サブネットマスク
			if strings.Contains(key, "サブネット") || strings.Contains(keyLower, "subnet") {
				status.SubnetMask = value
			}

			// デフォルトゲートウェイ
			if strings.Contains(key, "ゲートウェイ") || strings.Contains(keyLower, "gateway") {
				if value != "" && status.Gateway == "-" {
					status.Gateway = value
				}
			}

			// DNSサーバー
			if strings.Contains(key, "DNS") && strings.Contains(key, "サーバー") ||
			   strings.Contains(keyLower, "dns") && strings.Contains(keyLower, "server") {
				if value != "" {
					dnsServers = append(dnsServers, value)
				}
			}

			// DHCP有効
			if strings.Contains(key, "DHCP") && strings.Contains(key, "有効") ||
			   strings.Contains(keyLower, "dhcp") && strings.Contains(keyLower, "enabled") {
				if value == "はい" || strings.ToLower(value) == "yes" {
					status.DHCP = "有効"
				} else {
					status.DHCP = "無効"
				}
			}
		}

		if len(dnsServers) > 0 {
			status.DNS = strings.Join(dnsServers, ", ")
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// isIPAddress は文字列がIPアドレス形式かどうかを判定します
func isIPAddress(s string) bool {
	s = strings.TrimSpace(s)
	// シンプルなIPv4チェック
	pattern := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	return pattern.MatchString(s)
}

func openIPConfig() {
	// コマンドプロンプトでipconfig /allを実行
	cmd := exec.Command("cmd", "/c", "start", "cmd", "/k", "ipconfig /all")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	cmd.Start()
}
