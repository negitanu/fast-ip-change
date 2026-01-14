# Fast IP Change

Windowsのタスクバー（システムトレイ）に常駐し、指定したIPアドレスに自動でネットワークインターフェースカード（NIC）の設定を変更するアプリケーション。

## 機能

- システムトレイ常駐
- 複数のIPアドレス設定プロファイルの管理
- ワンクリックでIPアドレス設定を切り替え
- DHCP（自動取得）への切り替え
- 現在のIP設定の表示
- 設定の保存・読み込み

## 要件

- Windows 10以降
- Go 1.21以上（開発時）
- 管理者権限での実行が必要

## インストール

### ビルド方法

```bash
# 依存関係の取得
go mod download

# ビルド
go build -ldflags="-H windowsgui -s -w" -o fast-ip-change.exe ./cmd/fast-ip-change
```

### リリースビルド

```bash
go build -ldflags="-H windowsgui -s -w" -trimpath -o fast-ip-change.exe ./cmd/fast-ip-change
```

## 使用方法

1. **管理者として実行**: アプリケーションを管理者権限で起動します
2. **システムトレイ**: タスクバーの通知領域にアイコンが表示されます
3. **プロファイル選択**: アイコンを右クリックして、プロファイルを選択します
4. **IPアドレス変更**: 選択したプロファイルの設定が自動的に適用されます

## 設定ファイル

設定ファイルは以下の場所に保存されます：

```
%APPDATA%\FastIPChange\settings.json
```

### 設定ファイルの構造

```json
{
  "version": "1.0",
  "autoStart": false,
  "profiles": [
    {
      "id": "uuid",
      "name": "プロファイル名",
      "ipAddress": "192.168.1.100",
      "subnetMask": "255.255.255.0",
      "gateway": "192.168.1.1",
      "dnsPrimary": "8.8.8.8",
      "dnsSecondary": "8.8.4.4",
      "nicName": "イーサネット"
    }
  ],
  "settings": {
    "logLevel": "INFO",
    "enableNotifications": true
  }
}
```

## ログ

ログファイルは以下の場所に保存されます：

```
%APPDATA%\FastIPChange\logs\fast-ip-change-YYYY-MM-DD.log
```

## 開発

### プロジェクト構造

```
fast-ip-change/
├── cmd/
│   └── fast-ip-change/
│       └── main.go              # エントリーポイント
├── internal/
│   ├── config/
│   │   └── config.go            # 設定ファイル管理
│   ├── network/
│   │   └── network.go           # ネットワーク設定変更
│   ├── systray/
│   │   └── systray.go           # システムトレイ管理
│   ├── logger/
│   │   └── logger.go             # ログ管理
│   └── utils/
│       └── admin.go              # ユーティリティ
├── pkg/
│   └── models/
│       ├── profile.go           # データモデル
│       └── errors.go            # エラー定義
├── go.mod
├── go.sum
└── README.md
```

### 依存関係

主要な依存関係：

- `github.com/getlantern/systray` - システムトレイ
- `github.com/go-ole/go-ole` - COMインターフェース
- `github.com/go-toast/toast` - Windows通知
- `github.com/lxn/walk` - Windows GUI
- `golang.org/x/sys/windows` - Windows API

### テスト

```bash
# すべてのテストを実行
go test ./...

# カバレッジを取得
go test -cover ./...
```

## 注意事項

- IPアドレス変更により、現在のネットワーク接続が切断される可能性があります
- 管理者権限が必要なため、UACプロンプトが表示されます
- 設定ミスによりネットワーク接続が失われる可能性があるため、バックアップ機能を活用してください

## ライセンス

このプロジェクトは Apache License 2.0 のもとで公開されています。詳細は [LICENSE](LICENSE) ファイルを参照してください。

## バージョン履歴

### 1.0.0
- 初回リリース
- 基本的なIPアドレス切り替え機能
- システムトレイ常駐
- プロファイル管理
