package network

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"github.com/fast-ip-change/fast-ip-change/internal/logger"
	"github.com/fast-ip-change/fast-ip-change/pkg/models"
)

// createHiddenCmd はコンソールウィンドウを表示しないコマンドを作成します
func createHiddenCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	return cmd
}

// NetworkError はネットワーク関連のエラーを表します
type NetworkError struct {
	Code    string
	Message string
	Err     error
}

func (e *NetworkError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// isValidNICName はNIC名が安全かどうかを検証します
func isValidNICName(name string) bool {
	if len(name) == 0 || len(name) > 256 {
		return false
	}
	// 危険な文字のチェック
	dangerousChars := []string{
		"&", "|", ";", "$", "`", "!", "<", ">", "(", ")", "{", "}", "[", "]",
		"\"", "'", "\\", "\n", "\r", "\t", "\x00",
	}
	for _, char := range dangerousChars {
		if strings.Contains(name, char) {
			return false
		}
	}
	return true
}

// GetNICList は利用可能なNICのリストを取得します
func GetNICList() ([]string, error) {
	cmd := createHiddenCmd("netsh", "interface", "show", "interface")
	output, err := cmd.Output()
	if err != nil {
		return nil, &NetworkError{
			Code:    "GET_NIC_LIST_FAILED",
			Message: "NICリストの取得に失敗しました",
			Err:     err,
		}
	}

	lines := strings.Split(string(output), "\n")
	var nics []string
	// ヘッダー行をスキップするためのパターン（日本語/英語両対応）
	headerPatterns := []string{
		"Admin State", "State", // 英語
		"管理状態", "状態", // 日本語
		"---", // セパレーター
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// ヘッダー行かどうかチェック
		isHeader := false
		for _, pattern := range headerPatterns {
			if strings.HasPrefix(line, pattern) || strings.Contains(line, "-----") {
				isHeader = true
				break
			}
		}
		if isHeader {
			continue
		}

		// NIC名を抽出（フォーマット: "状態 タイプ NIC名"）
		// 例: "接続済み    専用    イーサネット" または "Connected Dedicated Ethernet"
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			nicName := strings.Join(parts[3:], " ")
			// 3列目以降がNIC名（4列フォーマット: 管理状態、状態、タイプ、NIC名）
			if len(parts) >= 4 {
				nicName = strings.Join(parts[3:], " ")
			} else {
				nicName = strings.Join(parts[2:], " ")
			}
			if nicName != "" {
				nics = append(nics, nicName)
			}
		}
	}

	return nics, nil
}

// GetCurrentIPConfig は指定されたNICの現在のIP設定を取得します
func GetCurrentIPConfig(nicName string) (*models.Profile, error) {
	cmd := createHiddenCmd("netsh", "interface", "ipv4", "show", "config", "name="+nicName)
	output, err := cmd.Output()
	if err != nil {
		return nil, &NetworkError{
			Code:    "GET_IP_CONFIG_FAILED",
			Message: fmt.Sprintf("NIC '%s' の設定取得に失敗しました", nicName),
			Err:     err,
		}
	}

	// 出力を解析してIP設定を抽出
	profile := &models.Profile{
		NICName: nicName,
	}

	// 日本語/英語両対応のパターン
	ipPrefixes := []string{"IP アドレス", "IP Address"}
	subnetPrefixes := []string{"サブネット マスク", "サブネット プレフィックス", "Subnet Mask", "Subnet Prefix"}
	gatewayPrefixes := []string{"デフォルト ゲートウェイ", "Default Gateway"}
	dnsPrefixes := []string{"DNS サーバー", "DNS Servers"}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// IPアドレスを抽出
		for _, prefix := range ipPrefixes {
			if strings.HasPrefix(line, prefix) {
				profile.IPAddress = extractLastValue(line)
				break
			}
		}

		// サブネットマスクを抽出
		for _, prefix := range subnetPrefixes {
			if strings.HasPrefix(line, prefix) {
				value := extractLastValue(line)
				// サブネットプレフィックス（例: /24）をマスク形式に変換
				if strings.HasPrefix(value, "/") || isNumeric(value) {
					profile.SubnetMask = prefixToSubnetMask(value)
				} else {
					profile.SubnetMask = value
				}
				break
			}
		}

		// デフォルトゲートウェイを抽出
		for _, prefix := range gatewayPrefixes {
			if strings.HasPrefix(line, prefix) {
				profile.Gateway = extractLastValue(line)
				break
			}
		}

		// DNSサーバーを抽出（最初のものをプライマリDNSとして）
		for _, prefix := range dnsPrefixes {
			if strings.HasPrefix(line, prefix) && profile.DNSPrimary == "" {
				profile.DNSPrimary = extractLastValue(line)
				break
			}
		}
	}

	return profile, nil
}

// extractLastValue は行から最後の値を抽出します
func extractLastValue(line string) string {
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return ""
}

// isNumeric は文字列が数値かどうかを判定します
func isNumeric(s string) bool {
	s = strings.TrimPrefix(s, "/")
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// prefixToSubnetMask はCIDRプレフィックスをサブネットマスクに変換します
func prefixToSubnetMask(prefix string) string {
	prefix = strings.TrimPrefix(prefix, "/")
	prefixLen := 0
	fmt.Sscanf(prefix, "%d", &prefixLen)

	if prefixLen <= 0 || prefixLen > 32 {
		return "255.255.255.0" // デフォルト
	}

	// プレフィックス長からマスクを計算
	mask := uint32(0xFFFFFFFF) << (32 - prefixLen)
	return fmt.Sprintf("%d.%d.%d.%d",
		(mask>>24)&0xFF,
		(mask>>16)&0xFF,
		(mask>>8)&0xFF,
		mask&0xFF)
}

// ApplyProfile はプロファイルの設定をNICに適用します
func ApplyProfile(profile *models.Profile) error {
	logger.Info("IPアドレス設定を適用中", "profile", profile.Name, "nic", profile.NICName)

	// 現在の設定をバックアップ（将来のロールバック用）
	currentConfig, err := GetCurrentIPConfig(profile.NICName)
	if err != nil {
		logger.Warn("現在の設定の取得に失敗（バックアップスキップ）", "error", err)
	}

	// IPアドレスとサブネットマスクを設定
	args := []string{"interface", "ipv4", "set", "address",
		"name=" + profile.NICName,
		"static",
		profile.IPAddress,
		profile.SubnetMask}
	if profile.Gateway != "" {
		args = append(args, profile.Gateway)
	}
	cmd := createHiddenCmd("netsh", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("IPアドレス設定の適用に失敗", err, "output", string(output))
		return &NetworkError{
			Code:    "APPLY_IP_FAILED",
			Message: fmt.Sprintf("IPアドレス設定の適用に失敗しました: %s", string(output)),
			Err:     err,
		}
	}

	// DNS設定を適用
	if profile.DNSPrimary != "" {
		dnsCmd := createHiddenCmd("netsh", "interface", "ipv4", "set", "dns",
			"name="+profile.NICName,
			"static",
			profile.DNSPrimary)
		if output, err := dnsCmd.CombinedOutput(); err != nil {
			logger.Error("DNS設定の適用に失敗", err, "output", string(output))
			// DNS設定の失敗は警告として記録するが、処理は続行
		}
	}

	if profile.DNSSecondary != "" {
		dnsCmd2 := createHiddenCmd("netsh", "interface", "ipv4", "add", "dns",
			"name="+profile.NICName,
			profile.DNSSecondary,
			"index=2")
		if output, err := dnsCmd2.CombinedOutput(); err != nil {
			logger.Error("代替DNS設定の適用に失敗", err, "output", string(output))
		}
	}

	// 設定が正しく適用されたか確認
	appliedConfig, err := GetCurrentIPConfig(profile.NICName)
	if err != nil {
		logger.Warn("適用後の設定確認に失敗", "error", err)
	} else if appliedConfig.IPAddress != profile.IPAddress {
		return &NetworkError{
			Code:    "VERIFY_FAILED",
			Message: "設定の適用が確認できませんでした",
			Err:     fmt.Errorf("期待: %s, 実際: %s", profile.IPAddress, appliedConfig.IPAddress),
		}
	}

	logger.Info("IPアドレス設定の適用が完了", "profile", profile.Name, "ip", profile.IPAddress)
	
	// バックアップ情報をログに記録（将来のロールバック機能用）
	if currentConfig != nil {
		logger.Info("バックアップ設定", "previous_ip", currentConfig.IPAddress)
	}

	return nil
}

// ApplyDHCP は指定されたNICをDHCPに切り替えます
func ApplyDHCP(nicName string) error {
	// NIC名の検証（コマンドインジェクション対策）
	if !isValidNICName(nicName) {
		return &NetworkError{
			Code:    "INVALID_NIC_NAME",
			Message: "NIC名に不正な文字が含まれています",
		}
	}

	logger.Info("DHCPに切り替え中", "nic", nicName)

	// IPアドレスをDHCPに設定
	cmd := createHiddenCmd("netsh", "interface", "ipv4", "set", "address",
		"name="+nicName,
		"source=dhcp")
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("DHCP設定の適用に失敗", err, "output", string(output))
		return &NetworkError{
			Code:    "APPLY_DHCP_FAILED",
			Message: fmt.Sprintf("DHCP設定の適用に失敗しました: %s", string(output)),
			Err:     err,
		}
	}

	// DNSをDHCPに設定
	dnsCmd := createHiddenCmd("netsh", "interface", "ipv4", "set", "dns",
		"name="+nicName,
		"source=dhcp")
	if output, err := dnsCmd.CombinedOutput(); err != nil {
		logger.Error("DNS DHCP設定の適用に失敗", err, "output", string(output))
		// 警告として記録するが、処理は続行
	}

	logger.Info("DHCP設定の適用が完了", "nic", nicName)
	return nil
}
