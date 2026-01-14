package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const (
	logDirName = "logs"
)

var (
	logger  *slog.Logger
	logFile *os.File
)

// Init はロガーを初期化します
func Init() error {
	appData, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("設定ディレクトリの取得に失敗: %w", err)
	}

	logDir := filepath.Join(appData, "FastIPChange", logDirName)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("ログディレクトリの作成に失敗: %w", err)
	}

	// ログファイル名は日付ベース
	logFileName := fmt.Sprintf("fast-ip-change-%s.log", time.Now().Format("2006-01-02"))
	logPath := filepath.Join(logDir, logFileName)

	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("ログファイルの作成に失敗: %w", err)
	}

	// 標準出力とファイルの両方に出力するMultiWriterを作成
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// slogハンドラーを作成（標準出力とファイル両方に出力）
	logger = slog.New(slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return nil
}

// GetLogger はロガーインスタンスを返します
func GetLogger() *slog.Logger {
	if logger == nil {
		// フォールバック: 標準ロガーを使用
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}
	return logger
}

// Info は情報ログを記録します
func Info(msg string, args ...any) {
	GetLogger().Info(msg, args...)
}

// Error はエラーログを記録します
func Error(msg string, err error, args ...any) {
	allArgs := append([]any{"error", err}, args...)
	GetLogger().Error(msg, allArgs...)
}

// Warn は警告ログを記録します
func Warn(msg string, args ...any) {
	GetLogger().Warn(msg, args...)
}

// Close はロガーを閉じます
func Close() {
	if logFile != nil {
		logFile.Close()
	}
}
