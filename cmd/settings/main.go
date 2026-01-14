package main

import (
	"fmt"
	"strings"

	"github.com/fast-ip-change/fast-ip-change/internal/config"
	"github.com/fast-ip-change/fast-ip-change/internal/network"
	"github.com/fast-ip-change/fast-ip-change/pkg/models"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

var (
	settingsWindow    *walk.MainWindow
	profileModel      *ProfileModel
	nicListBox        *walk.ListBox
	allNICs           []string
	enabledDHCPNICMap map[string]bool
)

// ProfileModel はプロファイルのテーブルモデルです
type ProfileModel struct {
	walk.TableModelBase
	items []models.Profile
}

func (m *ProfileModel) RowCount() int {
	return len(m.items)
}

func (m *ProfileModel) Value(row, col int) interface{} {
	profile := m.items[row]
	switch col {
	case 0:
		return profile.Name
	case 1:
		return profile.IPAddress
	case 2:
		return profile.SubnetMask
	case 3:
		return profile.NICName
	default:
		return ""
	}
}

func main() {
	var (
		tableView *walk.TableView
		addBtn    *walk.PushButton
		editBtn   *walk.PushButton
		deleteBtn *walk.PushButton
	)

	// 設定を読み込み
	cfg, err := config.LoadConfig()
	if err != nil {
		walk.MsgBox(nil, "エラー", fmt.Sprintf("設定の読み込みに失敗: %v", err), walk.MsgBoxIconError)
		return
	}

	// プロファイルモデルを作成
	profileModel = &ProfileModel{
		items: cfg.Profiles,
	}

	// NICリストを取得
	allNICs, err = network.GetNICList()
	if err != nil {
		allNICs = []string{}
	}

	// DHCP有効NICのマップを初期化
	enabledDHCPNICMap = make(map[string]bool)
	if len(cfg.Settings.EnabledDHCPNICs) == 0 {
		// 未設定の場合は全て有効
		for _, nic := range allNICs {
			enabledDHCPNICMap[nic] = true
		}
	} else {
		for _, nic := range cfg.Settings.EnabledDHCPNICs {
			enabledDHCPNICMap[nic] = true
		}
	}

	err = MainWindow{
		Title:    "Fast IP Change - 設定",
		Size:     Size{Width: 800, Height: 600},
		MinSize:  Size{Width: 600, Height: 500},
		Layout:   VBox{Margins: Margins{Top: 10, Left: 10, Right: 10, Bottom: 10}},
		AssignTo: &settingsWindow,
		Children: []Widget{
			Label{
				Text: "プロファイル一覧",
				Font: Font{Bold: true, PointSize: 10},
			},
			TableView{
				AssignTo:         &tableView,
				Model:            profileModel,
				AlternatingRowBG: true,
				ColumnsOrderable: true,
				Columns: []TableViewColumn{
					{Title: "名前", Width: 150},
					{Title: "IPアドレス", Width: 150},
					{Title: "サブネットマスク", Width: 150},
					{Title: "NIC名", Width: 200},
				},
				OnItemActivated: func() {
					editProfile(tableView)
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					PushButton{
						Text:      "追加",
						AssignTo:  &addBtn,
						OnClicked: func() { addProfile() },
					},
					PushButton{
						Text:      "編集",
						AssignTo:  &editBtn,
						OnClicked: func() { editProfile(tableView) },
					},
					PushButton{
						Text:      "削除",
						AssignTo:  &deleteBtn,
						OnClicked: func() { deleteProfile(tableView) },
					},
					HSpacer{},
				},
			},
			VSpacer{Size: 10},
			Label{
				Text: "DHCP表示設定",
				Font: Font{Bold: true, PointSize: 10},
			},
			Label{
				Text: "DHCPメニューに表示するNICを選択してください（複数選択可）:",
			},
			ListBox{
				AssignTo:       &nicListBox,
				Model:          allNICs,
				MultiSelection: true,
				MinSize:        Size{Height: 100},
			},
			Label{
				Text: "※ 選択されているNICのみがDHCPメニューに表示されます",
				Font: Font{PointSize: 8},
			},
			VSpacer{Size: 10},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						Text: "保存して閉じる",
						OnClicked: func() {
							if err := saveConfig(); err != nil {
								walk.MsgBox(settingsWindow, "エラー", fmt.Sprintf("設定の保存に失敗しました: %v", err), walk.MsgBoxIconError)
								return
							}
							walk.MsgBox(settingsWindow, "情報", "設定を保存しました。\nDHCP表示設定の変更を反映するには、アプリケーションを再起動してください。", walk.MsgBoxIconInformation)
							settingsWindow.Close()
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

	// テーブルの選択が変更されたときにボタンの有効/無効を切り替え
	tableView.CurrentIndexChanged().Attach(func() {
		hasSelection := tableView.CurrentIndex() >= 0
		editBtn.SetEnabled(hasSelection)
		deleteBtn.SetEnabled(hasSelection)
	})

	// 初期状態でボタンを無効化
	editBtn.SetEnabled(false)
	deleteBtn.SetEnabled(false)

	// NICリストの初期選択状態を設定（イベントハンドラ登録前に行う）
	var initialSelectedIndexes []int
	for i, nic := range allNICs {
		if enabledDHCPNICMap[nic] {
			initialSelectedIndexes = append(initialSelectedIndexes, i)
		}
	}
	if len(initialSelectedIndexes) > 0 {
		nicListBox.SetSelectedIndexes(initialSelectedIndexes)
	}

	// NICリストの選択が変更されたときに設定を保存
	nicListBox.SelectedIndexesChanged().Attach(func() {
		// 現在の選択状態を取得
		selectedIndexes := nicListBox.SelectedIndexes()
		enabledDHCPNICMap = make(map[string]bool)
		for _, idx := range selectedIndexes {
			if idx >= 0 && idx < len(allNICs) {
				enabledDHCPNICMap[allNICs[idx]] = true
			}
		}
		// 設定を保存
		if err := saveConfig(); err != nil {
			walk.MsgBox(settingsWindow, "エラー", fmt.Sprintf("設定の保存に失敗しました: %v", err), walk.MsgBoxIconError)
		}
	})

	settingsWindow.Run()
}

func addProfile() {
	editProfileDialog(nil)
}

func editProfile(tableView *walk.TableView) {
	idx := tableView.CurrentIndex()
	if idx < 0 {
		return
	}
	profile := &profileModel.items[idx]
	editProfileDialog(profile)
}

func deleteProfile(tableView *walk.TableView) {
	idx := tableView.CurrentIndex()
	if idx < 0 {
		return
	}

	result := walk.MsgBox(settingsWindow, "確認", "このプロファイルを削除しますか？", walk.MsgBoxYesNo|walk.MsgBoxIconQuestion)
	if result != walk.DlgCmdYes {
		return
	}

	// プロファイルを削除
	profileModel.items = append(profileModel.items[:idx], profileModel.items[idx+1:]...)
	profileModel.PublishRowsRemoved(idx, idx)

	// 設定を保存
	if err := saveProfiles(); err != nil {
		walk.MsgBox(settingsWindow, "エラー", fmt.Sprintf("設定の保存に失敗しました: %v", err), walk.MsgBoxIconError)
		return
	}
}

func editProfileDialog(profile *models.Profile) {
	var (
		dlg            *walk.Dialog
		nameEdit       *walk.LineEdit
		ipEdit         *walk.LineEdit
		subnetEdit     *walk.LineEdit
		gatewayEdit    *walk.LineEdit
		dnsPrimaryEdit *walk.LineEdit
		dnsSecEdit     *walk.LineEdit
		nicCombo       *walk.ComboBox
		saveBtn        *walk.PushButton
	)

	isNew := profile == nil
	if isNew {
		profile = models.NewProfile()
	}

	// NICリストを取得
	nics, err := network.GetNICList()
	if err != nil {
		nics = []string{"イーサネット", "Wi-Fi"}
	}

	// NICのインデックスを取得
	nicIndex := 0
	for i, nic := range nics {
		if nic == profile.NICName {
			nicIndex = i
			break
		}
	}

	dialogTitle := "プロファイル追加"
	if !isNew {
		dialogTitle = "プロファイル編集"
	}

	err = Dialog{
		AssignTo: &dlg,
		Title:    dialogTitle,
		Size:     Size{Width: 400, Height: 450},
		MinSize:  Size{Width: 350, Height: 400},
		Layout:   VBox{Margins: Margins{Top: 10, Left: 10, Right: 10, Bottom: 10}},
		Children: []Widget{
			Label{Text: "プロファイル名:"},
			LineEdit{
				AssignTo: &nameEdit,
				Text:     profile.Name,
			},
			VSpacer{Size: 5},
			Label{Text: "IPアドレス:"},
			LineEdit{
				AssignTo: &ipEdit,
				Text:     profile.IPAddress,
			},
			VSpacer{Size: 5},
			Label{Text: "サブネットマスク:"},
			LineEdit{
				AssignTo: &subnetEdit,
				Text:     profile.SubnetMask,
			},
			VSpacer{Size: 5},
			Label{Text: "デフォルトゲートウェイ (オプション):"},
			LineEdit{
				AssignTo: &gatewayEdit,
				Text:     profile.Gateway,
			},
			VSpacer{Size: 5},
			Label{Text: "優先DNSサーバー (オプション):"},
			LineEdit{
				AssignTo: &dnsPrimaryEdit,
				Text:     profile.DNSPrimary,
			},
			VSpacer{Size: 5},
			Label{Text: "代替DNSサーバー (オプション):"},
			LineEdit{
				AssignTo: &dnsSecEdit,
				Text:     profile.DNSSecondary,
			},
			VSpacer{Size: 5},
			Label{Text: "NIC名:"},
			ComboBox{
				AssignTo:     &nicCombo,
				Editable:     true,
				Model:        nics,
				CurrentIndex: nicIndex,
			},
			VSpacer{Size: 10},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &saveBtn,
						Text:     "保存",
						OnClicked: func() {
							// 入力値を取得
							profile.Name = strings.TrimSpace(nameEdit.Text())
							profile.IPAddress = strings.TrimSpace(ipEdit.Text())
							profile.SubnetMask = strings.TrimSpace(subnetEdit.Text())
							profile.Gateway = strings.TrimSpace(gatewayEdit.Text())
							profile.DNSPrimary = strings.TrimSpace(dnsPrimaryEdit.Text())
							profile.DNSSecondary = strings.TrimSpace(dnsSecEdit.Text())
							profile.NICName = strings.TrimSpace(nicCombo.Text())

							// バリデーション
							if err := profile.Validate(); err != nil {
								walk.MsgBox(dlg, "エラー", fmt.Sprintf("入力値が不正です: %v", err), walk.MsgBoxIconError)
								return
							}

							// 新規の場合は追加、既存の場合は更新
							if isNew {
								profileModel.items = append(profileModel.items, *profile)
								profileModel.PublishRowsInserted(len(profileModel.items)-1, len(profileModel.items)-1)
							} else {
								for i, p := range profileModel.items {
									if p.ID == profile.ID {
										profileModel.items[i] = *profile
										profileModel.PublishRowsChanged(i, i)
										break
									}
								}
							}

							// 設定を保存
							if err := saveProfiles(); err != nil {
								walk.MsgBox(dlg, "エラー", fmt.Sprintf("設定の保存に失敗しました: %v", err), walk.MsgBoxIconError)
								return
							}

							walk.MsgBox(dlg, "成功", "プロファイルを保存しました。", walk.MsgBoxIconInformation)
							dlg.Accept()
						},
					},
					PushButton{
						Text: "キャンセル",
						OnClicked: func() {
							dlg.Cancel()
						},
					},
				},
			},
		},
	}.Create(settingsWindow)
	if err != nil {
		walk.MsgBox(settingsWindow, "エラー", fmt.Sprintf("ダイアログの作成に失敗: %v", err), walk.MsgBoxIconError)
		return
	}

	dlg.Run()
}

func saveProfiles() error {
	return saveConfig()
}

func saveConfig() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// プロファイルを保存
	cfg.Profiles = profileModel.items

	// DHCP有効NICを保存
	var enabledNICs []string
	for nic, enabled := range enabledDHCPNICMap {
		if enabled {
			enabledNICs = append(enabledNICs, nic)
		}
	}
	cfg.Settings.EnabledDHCPNICs = enabledNICs

	return config.SaveConfig(cfg)
}
