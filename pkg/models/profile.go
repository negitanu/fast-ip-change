package models

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// Config はアプリケーション全体の設定を表します
type Config struct {
	Version   string    `json:"version"`
	AutoStart bool      `json:"autoStart"`
	Profiles  []Profile `json:"profiles"`
	Settings  Settings  `json:"settings"`
}

// Profile はIPアドレス設定のプロファイルを表します
type Profile struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	IPAddress    string `json:"ipAddress"`
	SubnetMask   string `json:"subnetMask"`
	Gateway      string `json:"gateway,omitempty"`
	DNSPrimary   string `json:"dnsPrimary,omitempty"`
	DNSSecondary string `json:"dnsSecondary,omitempty"`
	NICName      string `json:"nicName"`
}

// Settings はアプリケーションの設定を表します
type Settings struct {
	LogLevel            string   `json:"logLevel"`
	EnableNotifications bool     `json:"enableNotifications"`
	EnabledDHCPNICs     []string `json:"enabledDHCPNICs,omitempty"`
}

// IsNICEnabledForDHCP は指定されたNICがDHCPメニューで有効かどうかを判定します
// EnabledDHCPNICs が nil または空の場合は全てのNICが有効（後方互換性）
func (s *Settings) IsNICEnabledForDHCP(nicName string) bool {
	if len(s.EnabledDHCPNICs) == 0 {
		return true
	}
	for _, nic := range s.EnabledDHCPNICs {
		if nic == nicName {
			return true
		}
	}
	return false
}

// NewProfile は新しいプロファイルを作成し、UUIDを生成します
func NewProfile() *Profile {
	return &Profile{
		ID: uuid.New().String(),
	}
}

// Validate はプロファイルの設定が有効かどうかを検証します
func (p *Profile) Validate() error {
	if p.Name == "" {
		return ErrInvalidProfileName
	}
	if p.IPAddress == "" {
		return ErrInvalidIPAddress
	}
	if p.SubnetMask == "" {
		return ErrInvalidSubnetMask
	}
	if p.NICName == "" {
		return ErrInvalidNICName
	}

	// NIC名に危険な文字が含まれていないかチェック
	if !isValidNICName(p.NICName) {
		return fmt.Errorf("%w: 不正な文字が含まれています", ErrInvalidNICName)
	}

	// IPアドレスの形式検証
	if !isValidIPv4(p.IPAddress) {
		return fmt.Errorf("%w: %s", ErrInvalidIPAddress, p.IPAddress)
	}

	// サブネットマスクの形式検証
	if !isValidSubnetMask(p.SubnetMask) {
		return fmt.Errorf("%w: %s", ErrInvalidSubnetMask, p.SubnetMask)
	}

	// ゲートウェイの形式検証（設定されている場合）
	if p.Gateway != "" && !isValidIPv4(p.Gateway) {
		return fmt.Errorf("無効なゲートウェイアドレス: %s", p.Gateway)
	}

	// DNSサーバーの形式検証（設定されている場合）
	if p.DNSPrimary != "" && !isValidIPv4(p.DNSPrimary) {
		return fmt.Errorf("無効な優先DNSサーバー: %s", p.DNSPrimary)
	}
	if p.DNSSecondary != "" && !isValidIPv4(p.DNSSecondary) {
		return fmt.Errorf("無効な代替DNSサーバー: %s", p.DNSSecondary)
	}

	return nil
}

// isValidIPv4 はIPv4アドレスが有効かどうかを検証します
func isValidIPv4(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	// IPv4であることを確認
	return parsed.To4() != nil
}

// isValidSubnetMask はサブネットマスクが有効かどうかを検証します
func isValidSubnetMask(mask string) bool {
	parts := strings.Split(mask, ".")
	if len(parts) != 4 {
		return false
	}

	var maskBits uint32
	for i, part := range parts {
		val, err := strconv.Atoi(part)
		if err != nil || val < 0 || val > 255 {
			return false
		}
		maskBits |= uint32(val) << (24 - 8*i)
	}

	// 有効なサブネットマスクは連続した1ビットの後に連続した0ビットが続く
	// 例: 255.255.255.0 = 11111111.11111111.11111111.00000000
	if maskBits == 0 {
		return false
	}

	// ビット反転して1を加算すると2の累乗になるはず
	inverted := ^maskBits
	return (inverted & (inverted + 1)) == 0
}

// isValidNICName はNIC名が安全かどうかを検証します
// コマンドインジェクション対策として危険な文字を禁止
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
