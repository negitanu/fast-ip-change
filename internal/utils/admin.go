package utils

import (
	"os"
)

// IsAdmin は現在のプロセスが管理者権限で実行されているかどうかを確認します
func IsAdmin() bool {
	// Windowsでは、管理者権限がある場合、特定のファイルにアクセスできる
	// 物理ドライブへのアクセスを試みる
	f, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		return false
	}
	f.Close()
	return true
}
