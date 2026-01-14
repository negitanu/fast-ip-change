package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/fast-ip-change/fast-ip-change/internal/logger"
	"github.com/fast-ip-change/fast-ip-change/internal/systray"
	"github.com/fast-ip-change/fast-ip-change/internal/utils"
)

var (
	version = "1.0.0"
)

func main() {
	// コマンドライン引数の解析
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "バージョン情報を表示")
	flag.BoolVar(&showVersion, "v", false, "バージョン情報を表示（短縮形）")
	flag.Parse()

	if showVersion {
		fmt.Printf("Fast IP Change version %s\n", version)
		os.Exit(0)
	}

	// 管理者権限の確認
	if !utils.IsAdmin() {
		fmt.Fprintf(os.Stderr, "エラー: このアプリケーションは管理者権限で実行する必要があります。\n")
		fmt.Fprintf(os.Stderr, "管理者として実行してください。\n")
		os.Exit(1)
	}

	// ロガーの初期化
	if err := logger.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "ロガーの初期化に失敗: %v\n", err)
		// ロガーの初期化失敗は致命的ではないので続行
	}
	defer logger.Close()

	logger.Info("Fast IP Change を起動しました", "version", version)

	// システムトレイアプリケーションを起動
	if err := systray.Run(); err != nil {
		logger.Error("アプリケーションの起動に失敗", err)
		fmt.Fprintf(os.Stderr, "エラー: %v\n", err)
		os.Exit(1)
	}
}
