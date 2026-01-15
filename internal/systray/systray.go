package systray

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/fast-ip-change/fast-ip-change/assets"
	"github.com/fast-ip-change/fast-ip-change/internal/config"
	"github.com/fast-ip-change/fast-ip-change/internal/logger"
	"github.com/fast-ip-change/fast-ip-change/internal/network"
	"github.com/fast-ip-change/fast-ip-change/pkg/models"
	"github.com/getlantern/systray"
	"github.com/go-toast/toast"
)

// Note: internal/ui パッケージは settings.exe で使用されるため、このファイルでは使用しない

// Windows プロセス作成フラグ
const (
	createNoWindow = 0x08000000 // CREATE_NO_WINDOW: コンソールウィンドウを表示しない
)

var (
	appConfig      *models.Config
	appConfigMu    sync.RWMutex                  // appConfig の排他制御用
	menuItems      map[string]*systray.MenuItem
	profileStopChs map[string]chan struct{}      // goroutine停止用のチャネル
	profileStopMu  sync.Mutex                    // profileStopChs の排他制御用
	dhcpMenuItems  map[string]*systray.MenuItem  // DHCPメニュー項目（NIC名 -> メニュー項目）
)

// Run はシステムトレイアプリケーションを起動します
func Run() error {
	// 設定を読み込み
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("設定の読み込みに失敗: %w", err)
	}

	appConfigMu.Lock()
	appConfig = cfg
	appConfigMu.Unlock()

	// システムトレイを起動
	systray.Run(onReady, onExit)
	return nil
}

func onReady() {
	// アイコンを設定
	iconData := getIcon()
	if len(iconData) == 0 {
		logger.Warn("アイコンデータが空です。デフォルトアイコンを使用します。")
	} else {
		logger.Info("アイコンを設定します", "size", len(iconData))
		// Windowsでは、アイコンを設定する前にタイトルを設定する必要がある場合がある
		systray.SetTitle("Fast IP Change")
		systray.SetTooltip("Fast IP Change - IPアドレスを簡単に切り替え")
		// アイコンを設定
		systray.SetIcon(iconData)
		logger.Info("アイコンを正常に設定しました")
	}

	menuItems = make(map[string]*systray.MenuItem)
	profileStopChs = make(map[string]chan struct{})

	// 現在のNIC設定を表示
	mNICStatus := systray.AddMenuItem("現在のNIC設定を表示", "現在のIP設定を表示")
	// 現在のルーティングテーブルを表示
	mRouteTable := systray.AddMenuItem("現在のルーティングテーブルを表示", "ルーティングテーブルを表示")
	systray.AddSeparator()

	// プロファイルメニューを動的に生成
	updateProfileMenu()

	systray.AddSeparator()

	// DHCPメニュー（NICごとにサブメニューを作成）
	mDHCP := systray.AddMenuItem("DHCP（自動取得）", "DHCPに切り替え")
	setupDHCPSubMenu(mDHCP)

	systray.AddSeparator()

	// 設定メニュー
	mSettings := systray.AddMenuItem("設定...", "設定を開く")
	mLogs := systray.AddMenuItem("ログを表示...", "ログを表示")
	systray.AddSeparator()

	// 終了メニュー
	mQuit := systray.AddMenuItem("終了", "アプリケーションを終了")

	// イベントハンドラー
	go func() {
		for {
			select {
			case <-mNICStatus.ClickedCh:
				showNICStatus()
			case <-mRouteTable.ClickedCh:
				showRouteTable()
			case <-mSettings.ClickedCh:
				openSettings()
			case <-mLogs.ClickedCh:
				showLogs()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func setupDHCPSubMenu(parent *systray.MenuItem) {
	nics, err := network.GetNICList()
	if err != nil || len(nics) == 0 {
		logger.Warn("NICリストの取得に失敗、DHCPサブメニューを作成できません", "error", err)
		return
	}

	dhcpMenuItems = make(map[string]*systray.MenuItem)

	// 有効なNICのみサブメニュー項目を作成
	appConfigMu.RLock()
	settings := appConfig.Settings
	appConfigMu.RUnlock()

	for _, nic := range nics {
		if !settings.IsNICEnabledForDHCP(nic) {
			continue
		}

		subItem := parent.AddSubMenuItem(nic, fmt.Sprintf("%s をDHCPに切り替え", nic))
		dhcpMenuItems[nic] = subItem

		// クリックイベントを監視
		go func(nicName string, item *systray.MenuItem) {
			for range item.ClickedCh {
				applyDHCPToNIC(nicName)
			}
		}(nic, subItem)
	}
}

// updateDHCPMenu はDHCPサブメニューの表示/非表示を更新します
// 注意: systrayライブラリの制限により、動的な更新はできません
// 設定変更後はアプリケーションの再起動が必要です
func updateDHCPMenu() {
	// 設定変更は再起動後に反映される
}

func onExit() {
	logger.Info("アプリケーションを終了します")
	os.Exit(0)
}

func updateProfileMenu() {
	profileStopMu.Lock()
	defer profileStopMu.Unlock()

	// 既存のgoroutineを停止
	for _, stopCh := range profileStopChs {
		close(stopCh)
	}
	profileStopChs = make(map[string]chan struct{})

	// 既存のプロファイルメニューを非表示
	for _, item := range menuItems {
		item.Hide()
	}
	menuItems = make(map[string]*systray.MenuItem)

	// 設定を読み取り
	appConfigMu.RLock()
	profiles := appConfig.Profiles
	appConfigMu.RUnlock()

	// プロファイルが存在しない場合
	if len(profiles) == 0 {
		mNoProfile := systray.AddMenuItem("プロファイルがありません", "")
		mNoProfile.Disable()
		return
	}

	// プロファイルメニューを追加
	for _, profile := range profiles {
		menuTitle := fmt.Sprintf("%s [%s]", profile.Name, profile.NICName)
		menuItem := systray.AddMenuItem(menuTitle, fmt.Sprintf("IP: %s", profile.IPAddress))
		menuItems[profile.ID] = menuItem

		// 停止用チャネルを作成
		stopCh := make(chan struct{})
		profileStopChs[profile.ID] = stopCh

		// 各メニューアイテムのクリックイベントを監視（停止可能なgoroutine）
		go func(id string, item *systray.MenuItem, stop chan struct{}) {
			for {
				select {
				case <-stop:
					return
				case <-item.ClickedCh:
					applyProfile(id)
				}
			}
		}(profile.ID, menuItem, stopCh)
	}
}

func applyProfile(profileID string) {
	// プロファイルを検索
	appConfigMu.RLock()
	var profile *models.Profile
	for i := range appConfig.Profiles {
		if appConfig.Profiles[i].ID == profileID {
			p := appConfig.Profiles[i] // コピーを作成
			profile = &p
			break
		}
	}
	appConfigMu.RUnlock()

	if profile == nil {
		showNotification("エラー", "プロファイルが見つかりませんでした", false)
		return
	}

	// プロファイルの検証（設定ファイル改ざん対策）
	if err := profile.Validate(); err != nil {
		logger.Error("プロファイルの検証に失敗", err, "profile", profile.Name)
		showNotification("エラー", fmt.Sprintf("プロファイル設定が不正です: %v", err), false)
		return
	}

	// プロファイルを適用
	if err := network.ApplyProfile(profile); err != nil {
		logger.Error("プロファイルの適用に失敗", err, "profile", profile.Name)
		showNotification("エラー", fmt.Sprintf("IPアドレス設定の適用に失敗しました: %v", err), false)
	} else {
		logger.Info("プロファイルを適用しました", "profile", profile.Name)
		showNotification("成功", fmt.Sprintf("IPアドレスを %s に変更しました", profile.IPAddress), true)
	}
}

func applyDHCPToNIC(nicName string) {
	if err := network.ApplyDHCP(nicName); err != nil {
		logger.Error("DHCP設定の適用に失敗", err)
		showNotification("エラー", fmt.Sprintf("DHCP設定の適用に失敗しました: %v", err), false)
	} else {
		logger.Info("DHCP設定を適用しました", "nic", nicName)
		showNotification("成功", fmt.Sprintf("%s をDHCP（自動取得）に切り替えました", nicName), true)
	}
}

func showNICStatus() {
	logger.Info("NIC状態画面を起動します")

	// 実行ファイルと同じディレクトリにあるipstatus.exeを起動
	exePath, err := os.Executable()
	if err != nil {
		logger.Error("実行ファイルのパス取得に失敗", err)
		showNotification("エラー", "NIC状態画面を起動できませんでした", false)
		return
	}

	ipstatusPath := filepath.Join(filepath.Dir(exePath), "ipstatus.exe")

	// ipstatus.exeが存在するか確認
	if _, err := os.Stat(ipstatusPath); os.IsNotExist(err) {
		logger.Error("NIC状態画面が見つかりません", err, "path", ipstatusPath)
		showNotification("エラー", "ipstatus.exe が見つかりません", false)
		return
	}

	// NIC状態画面を起動（非同期）
	cmd := exec.Command(ipstatusPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}

	if err := cmd.Start(); err != nil {
		logger.Error("NIC状態画面の起動に失敗", err)
		showNotification("エラー", fmt.Sprintf("NIC状態画面を起動できませんでした: %v", err), false)
		return
	}

	// プロセス終了を待ってハンドルを解放
	go func() {
		cmd.Wait()
	}()
}

func showRouteTable() {
	logger.Info("ルーティングテーブル画面を起動します")

	// 実行ファイルと同じディレクトリにあるroutetable.exeを起動
	exePath, err := os.Executable()
	if err != nil {
		logger.Error("実行ファイルのパス取得に失敗", err)
		showNotification("エラー", "ルーティングテーブル画面を起動できませんでした", false)
		return
	}

	routetablePath := filepath.Join(filepath.Dir(exePath), "routetable.exe")

	// routetable.exeが存在するか確認
	if _, err := os.Stat(routetablePath); os.IsNotExist(err) {
		logger.Error("ルーティングテーブル画面が見つかりません", err, "path", routetablePath)
		showNotification("エラー", "routetable.exe が見つかりません", false)
		return
	}

	// ルーティングテーブル画面を起動（非同期）
	cmd := exec.Command(routetablePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}

	if err := cmd.Start(); err != nil {
		logger.Error("ルーティングテーブル画面の起動に失敗", err)
		showNotification("エラー", fmt.Sprintf("ルーティングテーブル画面を起動できませんでした: %v", err), false)
		return
	}

	// プロセス終了を待ってハンドルを解放
	go func() {
		cmd.Wait()
	}()
}

func openSettings() {
	logger.Info("設定アプリを起動します")

	// 実行ファイルと同じディレクトリにあるsettings.exeを起動
	exePath, err := os.Executable()
	if err != nil {
		logger.Error("実行ファイルのパス取得に失敗", err)
		showNotification("エラー", "設定アプリを起動できませんでした", false)
		return
	}

	settingsPath := filepath.Join(filepath.Dir(exePath), "settings.exe")

	// 設定アプリが存在するか確認
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		logger.Error("設定アプリが見つかりません", err, "path", settingsPath)
		showNotification("エラー", "settings.exe が見つかりません", false)
		return
	}

	// 設定アプリを起動（非同期）
	cmd := exec.Command(settingsPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}

	if err := cmd.Start(); err != nil {
		logger.Error("設定アプリの起動に失敗", err)
		showNotification("エラー", fmt.Sprintf("設定アプリを起動できませんでした: %v", err), false)
		return
	}

	// 設定アプリの終了を待って、設定を再読み込み
	go func() {
		cmd.Wait()
		logger.Info("設定アプリが終了しました。設定を再読み込みします。")

		cfg, err := config.LoadConfig()
		if err != nil {
			logger.Error("設定の再読み込みに失敗", err)
			return
		}

		appConfigMu.Lock()
		appConfig = cfg
		appConfigMu.Unlock()

		updateProfileMenu()
		updateDHCPMenu()
	}()
}

func showLogs() {
	logger.Info("ログビューアを起動します")

	// 実行ファイルと同じディレクトリにあるlogviewer.exeを起動
	exePath, err := os.Executable()
	if err != nil {
		logger.Error("実行ファイルのパス取得に失敗", err)
		showNotification("エラー", "ログビューアを起動できませんでした", false)
		return
	}

	logviewerPath := filepath.Join(filepath.Dir(exePath), "logviewer.exe")

	// ログビューアが存在するか確認
	if _, err := os.Stat(logviewerPath); os.IsNotExist(err) {
		logger.Error("ログビューアが見つかりません", err, "path", logviewerPath)
		showNotification("エラー", "logviewer.exe が見つかりません", false)
		return
	}

	// ログビューアを起動（非同期）
	cmd := exec.Command(logviewerPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}

	if err := cmd.Start(); err != nil {
		logger.Error("ログビューアの起動に失敗", err)
		showNotification("エラー", fmt.Sprintf("ログビューアを起動できませんでした: %v", err), false)
		return
	}

	// プロセス終了を待ってハンドルを解放
	go func() {
		cmd.Wait()
	}()
}

func showNotification(title, message string, success bool) {
	appConfigMu.RLock()
	enableNotifications := appConfig.Settings.EnableNotifications
	appConfigMu.RUnlock()

	if !enableNotifications {
		return
	}

	notification := toast.Notification{
		AppID:   "Fast IP Change",
		Title:   title,
		Message: message,
	}

	if err := notification.Push(); err != nil {
		logger.Warn("通知の送信に失敗", "error", err)
	}
}

func getIcon() []byte {
	// 埋め込まれたアイコンデータを返す
	return assets.IconData
}
