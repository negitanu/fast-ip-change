package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var (
	mainWindow *walk.MainWindow
	logText    *walk.TextEdit
	fileCombo  *walk.ComboBox
	logDir     string
	logFiles   []string
)

func main() {
	// ログディレクトリを取得
	appData, err := os.UserConfigDir()
	if err != nil {
		walk.MsgBox(nil, "エラー", fmt.Sprintf("設定ディレクトリの取得に失敗: %v", err), walk.MsgBoxIconError)
		return
	}
	logDir = filepath.Join(appData, "FastIPChange", "logs")

	// ログファイル一覧を取得
	logFiles = getLogFiles()

	err = MainWindow{
		Title:    "Fast IP Change - ログ",
		Size:     Size{Width: 900, Height: 600},
		MinSize:  Size{Width: 600, Height: 400},
		Layout:   VBox{Margins: Margins{Top: 10, Left: 10, Right: 10, Bottom: 10}},
		AssignTo: &mainWindow,
		Children: []Widget{
			Composite{
				Layout: HBox{},
				Children: []Widget{
					Label{Text: "ログファイル:"},
					ComboBox{
						AssignTo:     &fileCombo,
						Model:        logFiles,
						CurrentIndex: 0,
						OnCurrentIndexChanged: func() {
							loadLogFile()
						},
					},
					HSpacer{},
					PushButton{
						Text: "更新",
						OnClicked: func() {
							loadLogFile()
						},
					},
					PushButton{
						Text: "フォルダを開く",
						OnClicked: func() {
							openLogFolder()
						},
					},
				},
			},
			TextEdit{
				AssignTo: &logText,
				ReadOnly: true,
				VScroll:  true,
				HScroll:  true,
				Font:     Font{Family: "Consolas", PointSize: 9},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
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

	// 初期ログを読み込み
	if len(logFiles) > 0 {
		loadLogFile()
	} else {
		logText.SetText("ログファイルが見つかりません。\n\nログディレクトリ: " + logDir)
	}

	mainWindow.Run()
}

func getLogFiles() []string {
	var files []string

	// ログディレクトリが存在しない場合
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		return files
	}

	entries, err := os.ReadDir(logDir)
	if err != nil {
		return files
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
			files = append(files, entry.Name())
		}
	}

	// 新しい順にソート
	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	return files
}

func loadLogFile() {
	if fileCombo.CurrentIndex() < 0 || fileCombo.CurrentIndex() >= len(logFiles) {
		return
	}

	fileName := logFiles[fileCombo.CurrentIndex()]
	filePath := filepath.Join(logDir, fileName)

	file, err := os.Open(filePath)
	if err != nil {
		logText.SetText(fmt.Sprintf("ログファイルを開けませんでした: %v", err))
		return
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		logText.SetText(fmt.Sprintf("ログファイルの読み込みエラー: %v", err))
		return
	}

	// ログを表示（新しい行を上に）
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== %s ===\n", fileName))
	sb.WriteString(fmt.Sprintf("更新時刻: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("行数: %d\n\n", len(lines)))

	for _, line := range lines {
		sb.WriteString(line)
		sb.WriteString("\r\n")
	}

	logText.SetText(sb.String())

	// 最下部にスクロール
	logText.SetTextSelection(len(sb.String()), len(sb.String()))
}

func openLogFolder() {
	// エクスプローラーでログフォルダを開く
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		walk.MsgBox(mainWindow, "情報", "ログフォルダが存在しません。\n\n"+logDir, walk.MsgBoxIconInformation)
		return
	}

	cmd := exec.Command("explorer", logDir)
	cmd.Start()
}
