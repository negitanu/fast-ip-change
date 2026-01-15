package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fast-ip-change/fast-ip-change/pkg/models"
)

const (
	configDirName  = "FastIPChange"
	configFileName = "settings.json"
)

// GetConfigPath は設定ファイルのパスを返します
func GetConfigPath() (string, error) {
	appData, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("設定ディレクトリの取得に失敗: %w", err)
	}

	configDir := filepath.Join(appData, configDirName)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("設定ディレクトリの作成に失敗: %w", err)
	}

	return filepath.Join(configDir, configFileName), nil
}

// LoadConfig は設定ファイルを読み込みます
func LoadConfig() (*models.Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// ファイルが存在しない場合はデフォルト設定を返す
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return GetDefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("設定ファイルの読み込みに失敗: %w", err)
	}

	var config models.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("設定ファイルの解析に失敗: %w", err)
	}

	// バージョンが設定されていない場合はデフォルト値を設定
	if config.Version == "" {
		config.Version = "1.0"
	}

	return &config, nil
}

// SaveConfig は設定ファイルを保存します
func SaveConfig(config *models.Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("設定のシリアライズに失敗: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("設定ファイルの書き込みに失敗: %w", err)
	}

	return nil
}

// GetDefaultConfig はデフォルト設定を返します
func GetDefaultConfig() *models.Config {
	return &models.Config{
		Version:   "1.0",
		AutoStart: false,
		Profiles:  []models.Profile{},
		Settings: models.Settings{
			LogLevel:            "INFO",
			EnableNotifications: true,
		},
	}
}
