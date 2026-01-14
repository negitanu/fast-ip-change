# Fast IP Change アプリケーション仕様書

## 1. プロジェクト概要

### 1.1 目的

Windows のタスクバー（システムトレイ）に常駐し、指定した IP アドレスに自動でネットワークインターフェースカード（NIC）の設定を変更するアプリケーション。複数の実行ファイルで構成されるモジュラーアーキテクチャを採用し、各機能を独立したアプリケーションとして実装しています。

### 1.2 背景

ネットワーク環境の切り替えを頻繁に行う必要がある場合、手動での IP アドレス設定変更は時間がかかり、ミスが発生しやすい。本アプリケーションにより、ワンクリックで IP アドレス設定を切り替えることを可能にします。

### 1.3 アーキテクチャの特徴

- **モジュラー設計**: 各機能を独立した実行ファイルとして分離
- **リソース効率**: 必要な機能のみを起動可能
- **保守性**: 個別の機能更新が容易
- **拡張性**: 新機能の追加が容易

## 2. 機能要件

### 2.1 コア機能

#### 2.1.1 システムトレイ常駐

- Windows 起動時に自動起動可能（オプション）
- タスクバーの通知領域（システムトレイ）にアイコンを表示
- 最小化時もバックグラウンドで動作

#### 2.1.2 IP アドレス設定の管理

- 複数の IP アドレス設定プロファイルを保存・管理
- 各プロファイルには以下を設定可能：
  - プロファイル名（識別用）
  - IP アドレス
  - サブネットマスク
  - デフォルトゲートウェイ（オプション）
  - 優先 DNS サーバー（オプション）
  - 代替 DNS サーバー（オプション）
  - 対象 NIC（複数 NIC 環境での選択）

#### 2.1.3 IP アドレスの自動変更

- システムトレイアイコンを右クリックして表示されるメニューから、保存済みプロファイルを選択
- 選択したプロファイルの設定を指定 NIC に適用
- プロファイル名と IP アドレス、NIC 名がメニューに表示される
- **DHCP への切り替え**: NIC ごとのサブメニューから選択可能（自動取得）
- 設定変更の成功/失敗を Windows 通知で通知

#### 2.1.4 現在の設定表示

- **NIC 状態表示**: すべての NIC の現在の設定を確認可能（`ipstatus.exe`）
- **ルーティングテーブル表示**: 現在のルーティングテーブルを確認可能（`routetable.exe`）
- 自動更新機能により、リアルタイムで情報を確認可能

### 2.2 補助機能

#### 2.2.1 設定管理

- **プロファイルの追加・編集・削除**: `settings.exe`で GUI 操作
- プロファイルのバリデーション（`Profile.Validate()`メソッド）
- NIC リストの自動取得とドロップダウン選択
- 設定ファイルの自動保存
- プロファイルのインポート・エクスポート（JSON 形式、将来実装予定）

#### 2.2.2 ログ機能

- IP アドレス変更の履歴を記録
- エラー発生時のログ記録
- **ログビューア**: `logviewer.exe`でログファイルを表示
- 日次ログファイル（`fast-ip-change-YYYY-MM-DD.log`）
- ログファイルの選択と表示機能

#### 2.2.3 通知機能

- IP アドレス変更成功時の通知
- エラー発生時の通知
- Windows 通知センターへの通知

## 3. 非機能要件

### 3.1 パフォーマンス

- 起動時間：3 秒以内
- IP アドレス変更処理：5 秒以内
- メモリ使用量：50MB 以下（常駐時）

### 3.2 セキュリティ

- 管理者権限での実行が必要（IP アドレス変更のため）
- 設定ファイルの暗号化（オプション）
- ログファイルへの機密情報の記録を避ける

### 3.3 ユーザビリティ

- 直感的な UI/UX
- 最小限のクリック数で操作可能
- エラーメッセージは分かりやすく日本語で表示

### 3.4 互換性

- Windows 10 以降をサポート
- 複数 NIC 環境に対応
- IPv4 をサポート（IPv6 は将来対応）

### 3.5 信頼性

- エラー発生時の適切な処理
- 設定変更前の状態をログに記録（将来のロールバック機能用）
- アプリケーションクラッシュ時の自動復旧（将来実装予定）

## 4. 技術スタック

### 4.1 採用技術: Go

#### 4.1.1 選択理由

- **シングルバイナリ配布**: 依存関係を含む単一の実行ファイルで配布可能
- **優れたパフォーマンス**: 低メモリ使用量、高速起動
- **クロスコンパイル**: 将来のクロスプラットフォーム対応が容易
- **静的型付け**: コンパイル時エラー検出により信頼性が高い
- **豊富な標準ライブラリ**: JSON、ファイル操作、ログなどが標準装備

#### 4.1.2 使用ライブラリ

**コアライブラリ**

- `github.com/getlantern/systray` - システムトレイ（通知領域）アイコンとメニュー
- `golang.org/x/sys/windows` - Windows API へのアクセス（間接的に使用）

**補助ライブラリ**

- `encoding/json` - 設定ファイルの読み書き（標準ライブラリ）
- `log` / `log/slog` - ログ機能（標準ライブラリ）
- `github.com/google/uuid` - プロファイル ID 生成
- `github.com/go-toast/toast` - Windows 通知センターへの通知

**UI（設定ウィンドウ）**

- `github.com/lxn/walk` - Windows GUI ライブラリ（設定ウィンドウ用）
- または `github.com/webview/webview` - 軽量 WebView ベースの UI

#### 4.1.3 アーキテクチャ

本アプリケーションは、**複数の実行ファイルで構成されるモジュラーアーキテクチャ**を採用しています。各機能を独立した実行ファイルとして分離することで、以下の利点があります：

- **保守性の向上**: 各機能が独立しているため、個別に更新・修正が可能
- **リソース効率**: 必要な機能のみを起動できるため、メモリ使用量を削減
- **拡張性**: 新機能を追加する際に、既存コードへの影響を最小化

#### 4.1.4 プロジェクト構造

```
fast-ip-change/
├── cmd/
│   ├── fast-ip-change/          # メインアプリケーション（システムトレイ常駐）
│   │   ├── main.go
│   │   ├── fast-ip-change.manifest
│   │   └── rsrc.syso            # 生成されたリソースファイル（アイコン、マニフェスト）
│   ├── settings/                # 設定管理アプリケーション
│   │   ├── main.go
│   │   ├── settings.manifest
│   │   └── rsrc.syso
│   ├── ipstatus/                # NIC状態表示アプリケーション
│   │   ├── main.go
│   │   ├── ipstatus.manifest
│   │   └── rsrc.syso
│   ├── routetable/              # ルーティングテーブル表示アプリケーション
│   │   ├── main.go
│   │   ├── routetable.manifest
│   │   └── rsrc.syso
│   └── logviewer/               # ログビューアアプリケーション
│       ├── main.go
│       ├── logviewer.manifest
│       └── rsrc.syso
├── internal/
│   ├── config/
│   │   └── config.go            # 設定ファイル管理
│   ├── network/
│   │   └── network.go           # ネットワーク設定変更
│   ├── systray/
│   │   └── systray.go           # システムトレイ管理
│   ├── logger/
│   │   └── logger.go            # ログ管理
│   └── utils/
│       └── admin.go             # ユーティリティ
├── pkg/
│   └── models/
│       ├── profile.go           # データモデル
│       └── errors.go            # エラー定義
├── assets/
│   ├── assets.go                # 埋め込みリソース
│   └── systray.ico              # システムトレイアイコン（ICO形式）
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

#### 4.1.5 実行ファイル構成

| 実行ファイル         | 説明                                         | 管理者権限 |
| -------------------- | -------------------------------------------- | ---------- |
| `fast-ip-change.exe` | メインアプリケーション（システムトレイ常駐） | 必要       |
| `settings.exe`       | プロファイル設定管理                         | 不要       |
| `ipstatus.exe`       | NIC 状態表示                                 | 不要       |
| `routetable.exe`     | ルーティングテーブル表示                     | 不要       |
| `logviewer.exe`      | ログビューア                                 | 不要       |

**注意**: すべての実行ファイルは、メインアプリケーション（`fast-ip-change.exe`）と同じディレクトリに配置する必要があります。

#### 4.1.6 ビルド要件

- Go 1.21 以上（推奨: Go 1.24 以上）
- Windows SDK（CGO を使用する場合）
- 管理者権限での実行が必要（メインアプリケーションのみ）

#### 4.1.7 ビルドコマンド

**前提条件**: `rsrc`ツールをインストール（アイコンとマニフェストを埋め込むため）

```bash
# rsrcツールのインストール
go install github.com/akavel/rsrc@latest
```

**ビルド手順**:

```bash
# 方法1: Makefileを使用（推奨）
make build-all

# 方法2: 手動でビルド
# リソースファイルの生成
cd cmd/fast-ip-change && rsrc -manifest fast-ip-change.manifest -ico ../../assets/systray.ico -o rsrc.syso
cd cmd/settings && rsrc -manifest settings.manifest -ico ../../assets/systray.ico -o rsrc.syso
cd cmd/ipstatus && rsrc -manifest ipstatus.manifest -ico ../../assets/systray.ico -o rsrc.syso
cd cmd/routetable && rsrc -manifest routetable.manifest -ico ../../assets/systray.ico -o rsrc.syso
cd cmd/logviewer && rsrc -manifest logviewer.manifest -ico ../../assets/systray.ico -o rsrc.syso

# 各アプリケーションのビルド
go build -ldflags="-H windowsgui -s -w" -trimpath -o fast-ip-change.exe ./cmd/fast-ip-change
go build -ldflags="-H windowsgui -s -w" -trimpath -o settings.exe ./cmd/settings
go build -ldflags="-H windowsgui -s -w" -trimpath -o ipstatus.exe ./cmd/ipstatus
go build -ldflags="-H windowsgui -s -w" -trimpath -o routetable.exe ./cmd/routetable
go build -ldflags="-H windowsgui -s -w" -trimpath -o logviewer.exe ./cmd/logviewer
```

**注意**: 
- `rsrc.syso`ファイルは各`cmd`ディレクトリに生成されます（`.gitignore`に含まれています）
- アイコンは`assets/systray.ico`から各実行ファイルに埋め込まれます
- マニフェストファイルも同時に埋め込まれます

### 4.2 データ保存

- 設定ファイル：JSON 形式（`%APPDATA%\FastIPChange\settings.json`）
- ログファイル：テキスト形式（`%APPDATA%\FastIPChange\logs\`）
- バックアップ：設定変更前の状態をログに記録（将来のロールバック機能用）

## 5. UI/UX 仕様

### 5.1 システムトレイメニュー

```
[アイコン] Fast IP Change
├─ 現在のNIC設定を表示
├─ 現在のルーティングテーブルを表示
├─ ────────────────
├─ プロファイル1 (IP: 192.168.1.100 (イーサネット))
├─ プロファイル2 (IP: 192.168.0.50 (Wi-Fi))
├─ プロファイル3
├─ ────────────────
├─ DHCP（自動取得）
│   ├─ イーサネット
│   ├─ Wi-Fi
│   └─ ...
├─ ────────────────
├─ 設定...
├─ ログを表示...
└─ 終了
```

**注意**: プロファイルが存在しない場合は、「プロファイルがありません」という無効化されたメニュー項目が表示されます。

**メニュー項目の説明**:

- **現在の NIC 設定を表示**: `ipstatus.exe`を起動し、すべての NIC の現在の設定を表示
- **現在のルーティングテーブルを表示**: `routetable.exe`を起動し、ルーティングテーブルを表示
- **プロファイル**: 保存済みプロファイルを選択すると、その設定を適用
- **DHCP（自動取得）**: サブメニューから NIC を選択して DHCP に切り替え
- **設定...**: `settings.exe`を起動し、プロファイルの管理を行う
- **ログを表示...**: `logviewer.exe`を起動し、ログファイルを表示

### 5.2 設定ウィンドウ（settings.exe）

#### 5.2.1 プロファイル一覧

- テーブル形式でプロファイルを表示（名前、IP アドレス、サブネットマスク、NIC 名）
- テーブル行をダブルクリックで編集
- 追加・編集・削除ボタン
- 選択されたプロファイルのみ編集・削除ボタンが有効

#### 5.2.2 プロファイル編集ダイアログ

- プロファイル名（テキスト入力、必須）
- IP アドレス（IP アドレス入力フィールド、必須）
- サブネットマスク（IP アドレス入力フィールド、必須）
- デフォルトゲートウェイ（IP アドレス入力フィールド、オプション）
- 優先 DNS サーバー（IP アドレス入力フィールド、オプション）
- 代替 DNS サーバー（IP アドレス入力フィールド、オプション）
- 対象 NIC（ドロップダウン選択、編集可能、必須）
- 保存・キャンセルボタン
- 入力値のバリデーション（`Profile.Validate()`メソッドを使用）

#### 5.2.3 動作

- 設定ファイル（`%APPDATA%\FastIPChange\settings.json`）を直接読み書き
- プロファイルの追加・編集・削除時に自動保存
- メインアプリケーション終了時に設定を再読み込み

### 5.3 NIC 状態表示ウィンドウ（ipstatus.exe）

- すべての NIC の現在の設定をテーブル形式で表示
- 表示項目：
  - NIC 名
  - 状態（有効/無効）
  - IP アドレス
  - サブネットマスク
  - デフォルトゲートウェイ
  - DNS サーバー
  - DHCP 設定（有効/無効）
- 自動更新機能（定期的に情報を更新）
- 最終更新時刻の表示
- テーブルのソート機能

### 5.4 ルーティングテーブル表示ウィンドウ（routetable.exe）

- ルーティングテーブルをテーブル形式で表示
- 表示項目：
  - 宛先ネットワーク
  - ネットマスク
  - ゲートウェイ
  - インターフェース
  - メトリック
- 自動更新機能（定期的に情報を更新）
- 最終更新時刻の表示
- テーブルのソート機能

### 5.5 ログビューア（logviewer.exe）

- ログファイル一覧の表示（ドロップダウン）
- 選択されたログファイルの内容を表示
- ログファイルの自動読み込み
- テキストエリアでのスクロール表示

## 6. 実装詳細

### 6.1 IP アドレス変更処理フロー

1. ユーザーがプロファイルを選択（システムトレイメニューから）
2. 現在の設定をログに記録（将来のロールバック機能用）
3. 管理者権限の確認（アプリケーション起動時に確認済み）
4. 指定 NIC の現在の設定を取得（オプション、ログ記録用）
5. 新しい設定を適用（`netsh`コマンドを使用）
6. DNS 設定を適用（設定されている場合）
7. 設定の確認（変更が正しく適用されたか検証）
8. 成功/失敗の通知（Windows 通知センター）
9. ログの記録（成功/失敗の詳細）

### 6.2 Go 実装の詳細

#### 6.2.1 ネットワーク設定変更の実装方法

**方法 1: WMI（Windows Management Instrumentation）を使用**

- `go-ole`と`oleutil`を使用して COM インターフェース経由で WMI にアクセス
- `Win32_NetworkAdapterConfiguration`クラスを使用して IP 設定を変更
- メリット: 標準的な Windows API を使用
- デメリット: COM インターフェースの扱いが複雑

**方法 2: netsh コマンドを実行**

- `os/exec`パッケージを使用して`netsh`コマンドを実行
- メリット: 実装が簡単、確実に動作
- デメリット: 外部プロセスに依存

**推奨実装**: 方法 2（netsh）を初期実装とし、将来的に方法 1（WMI）への移行を検討

#### 6.2.2 システムトレイ実装

```go
// systrayパッケージの使用例
systray.Run(onReady, onExit)

func onReady() {
    systray.SetIcon(iconData)
    systray.SetTitle("Fast IP Change")

    // メニュー項目の追加
    mCurrent := systray.AddMenuItem("現在の設定を表示", "")
    systray.AddSeparator()

    // プロファイルメニューの動的生成
    // ...

    mQuit := systray.AddMenuItem("終了", "")
    go func() {
        <-mQuit.ClickedCh
        systray.Quit()
    }()
}
```

#### 6.2.3 設定ウィンドウ実装

- `github.com/lxn/walk`を使用してネイティブ Windows GUI を構築
- または、`webview`を使用して HTML/CSS/JavaScript で UI を構築（より柔軟）

#### 6.2.4 管理者権限の確認

```go
import (
    "golang.org/x/sys/windows"
)

func isAdmin() bool {
    _, err := os.Open("\\\\.\\PHYSICALDRIVE0")
    return err == nil
}
```

または、マニフェストファイルで管理者権限を要求：

- `app.manifest`ファイルを作成
- `<requestedExecutionLevel level="requireAdministrator" />`を設定

### 6.3 エラーハンドリング

#### 6.3.1 想定されるエラー

- 管理者権限不足
- 無効な IP アドレス
- NIC が見つからない
- 設定変更の失敗
- ネットワーク接続の切断
- netsh コマンドの実行失敗

#### 6.3.2 エラー処理

- Go の標準的なエラーハンドリングパターンを使用
- カスタムエラータイプを定義してエラーの種類を区別
- 各エラーに対して適切なエラーメッセージを表示
- 可能な場合はロールバックを実行
- エラーログを記録（`log/slog`を使用）

```go
type NetworkError struct {
    Code    string
    Message string
    Err     error
}

func (e *NetworkError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
```

### 6.4 並行処理

- システムトレイのメニュー操作と IP アドレス変更処理は別の goroutine で実行
- プロファイルメニューのクリックイベントは、各プロファイルごとに独立した goroutine で監視
- プロファイルメニュー更新時に、既存の goroutine を適切に停止（チャネルによる停止制御）
- 設定変更中は UI を無効化して重複実行を防止
- チャネルを使用して goroutine 間の通信を実現

### 6.5 外部アプリケーションの起動

メインアプリケーション（`fast-ip-change.exe`）から、以下の外部アプリケーションを起動します：

- **settings.exe**: 設定管理アプリケーション
- **ipstatus.exe**: NIC 状態表示アプリケーション
- **routetable.exe**: ルーティングテーブル表示アプリケーション
- **logviewer.exe**: ログビューア

起動方法：

- `os.Executable()`でメインアプリケーションのパスを取得
- 同じディレクトリにある外部アプリケーションを`exec.Command()`で起動
- `syscall.SysProcAttr`でコンソールウィンドウを非表示に設定
- 非同期で起動（`cmd.Start()`）

設定アプリケーション終了時の処理：

- `cmd.Wait()`で終了を待機
- 設定ファイルを再読み込み
- プロファイルメニューを更新

### 6.6 プロファイルバリデーション

`Profile.Validate()`メソッドにより、以下の検証を実行します：

- **必須項目の確認**:

  - プロファイル名が空でないこと
  - IP アドレスが空でないこと
  - サブネットマスクが空でないこと
  - NIC 名が空でないこと

- **IP アドレスの形式検証**:

  - IPv4 形式であること（`net.ParseIP()`を使用）
  - IPv6 は現在サポートしていない

- **サブネットマスクの形式検証**:

  - 4 オクテット形式（例: 255.255.255.0）
  - 各オクテットが 0-255 の範囲内
  - 連続した 1 ビットの後に連続した 0 ビットが続く形式（有効なサブネットマスク形式）

- **オプション項目の検証**:
  - ゲートウェイが設定されている場合、IPv4 形式であること
  - 優先 DNS サーバーが設定されている場合、IPv4 形式であること
  - 代替 DNS サーバーが設定されている場合、IPv4 形式であること

### 6.7 設定ファイル構造

#### 6.7.1 JSON 構造

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

#### 6.7.2 Go 構造体定義

```go
package models

import "github.com/google/uuid"

type Config struct {
    Version   string    `json:"version"`
    AutoStart bool      `json:"autoStart"`
    Profiles  []Profile `json:"profiles"`
    Settings  Settings  `json:"settings"`
}

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

type Settings struct {
    LogLevel          string `json:"logLevel"`
    EnableNotifications bool `json:"enableNotifications"`
}

// NewProfile creates a new profile with a generated UUID
func NewProfile() *Profile {
    return &Profile{
        ID: uuid.New().String(),
    }
}
```

## 7. セキュリティ考慮事項

### 7.1 権限管理

- 管理者権限での実行が必要
- UAC（User Account Control）の適切な処理

### 7.2 データ保護

- 設定ファイルへの機密情報の保存を最小限に
- 必要に応じて設定ファイルの暗号化

### 7.3 入力検証

- IP アドレスの形式検証
- サブネットマスクの妥当性チェック

## 8. テスト要件

### 8.1 単体テスト

- IP アドレス設定の変更処理（モックを使用）
- 設定ファイルの読み書き（`testing`パッケージ）
- 入力検証（IP アドレス形式、サブネットマスクなど）
- エラーハンドリングのテスト

**テストファイルの命名規則**: `*_test.go`

**実行コマンド**:

```bash
# すべてのテストを実行
go test ./...

# カバレッジを取得
go test -cover ./...

# 詳細な出力
go test -v ./...
```

### 8.2 統合テスト

- システムトレイからの操作
- 複数 NIC 環境での動作
- エラー発生時の動作
- netsh コマンドの実行結果の検証

### 8.3 ユーザーテスト

- 実際のネットワーク環境での動作確認
- 異なる Windows バージョンでの動作確認（Windows 10, 11）
- 管理者権限での実行確認
- メモリリークの確認

### 8.4 ベンチマークテスト

```go
func BenchmarkIPChange(b *testing.B) {
    // パフォーマンステスト
}
```

**実行コマンド**:

```bash
go test -bench=. -benchmem
```

## 9. 将来の拡張機能

### 9.1 短期拡張

- プロファイルのショートカットキー割り当て
- プロファイル切り替えのスケジュール機能
- 設定のクラウド同期

### 9.2 長期拡張

- IPv6 サポート
- ネットワーク設定の詳細表示
- 複数 NIC の同時設定変更
- コマンドラインインターフェース

## 10. 開発フェーズ

### フェーズ 1: 基本機能（実装済み）

- Go プロジェクトの初期化（`go mod init`）
- システムトレイ常駐機能（`systray`ライブラリ）
- 単一プロファイルでの IP アドレス変更（`netsh`コマンド実行）
- 基本的な設定 UI（`walk`ライブラリ）
- 管理者権限の確認と要求

### フェーズ 2: 機能拡張（実装済み）

- 複数プロファイル管理
- 設定の保存・読み込み（JSON 形式）
- ログ機能（`log/slog`）
- 現在の設定表示機能（`ipstatus.exe`）
- ルーティングテーブル表示機能（`routetable.exe`）
- ログビューア（`logviewer.exe`）
- DHCP サブメニュー（NIC ごとの選択）
- プロファイルメニューの動的更新
- Windows 通知機能（`toast`）

### フェーズ 3: 仕上げ（実装済み）

- エラーハンドリングの強化
- UI/UX の改善
- プロファイルバリデーション
- 複数実行ファイル構成の実装
- ドキュメント作成
- リリースビルドの最適化

### フェーズ 4: 将来の拡張

- 自動起動機能（レジストリ設定）
- プロファイルのインポート・エクスポート
- プロファイルのショートカットキー割り当て
- プロファイル切り替えのスケジュール機能

### 10.1 依存関係管理

#### 10.1.1 初期セットアップ

```bash
# プロジェクトの初期化
go mod init github.com/yourusername/fast-ip-change

# 依存関係の追加
go get github.com/getlantern/systray
go get github.com/lxn/walk
go get github.com/go-toast/toast
go get github.com/google/uuid

# 依存関係の自動解決
go mod tidy
```

#### 10.1.2 依存関係の更新

```bash
# すべての依存関係を更新
go get -u ./...

# 依存関係の整理
go mod tidy
```

### 10.2 開発環境

- **IDE**: Visual Studio Code with Go extension または GoLand
- **デバッグ**: Delve（`go install github.com/go-delve/delve/cmd/dlv@latest`）
- **テスト**: 標準の`testing`パッケージ
- **リント**: `golangci-lint`（オプション）

## 11. 注意事項

### 11.1 一般的な注意事項

- IP アドレス変更により、現在のネットワーク接続が切断される可能性がある
- 管理者権限が必要なため、インストール時に適切な説明が必要
- 設定ミスによりネットワーク接続が失われる可能性があるため、バックアップ機能が重要

### 11.2 Go 実装に関する注意事項

#### 11.2.1 CGO の使用

- `systray`や一部のライブラリは CGO を必要とする場合がある
- CGO を使用する場合は、C コンパイラ（MinGW 等）が必要
- クロスコンパイルが複雑になる可能性がある

#### 11.2.2 Windows API の呼び出し

- Windows API を直接呼び出す場合は、`golang.org/x/sys/windows`を使用
- 適切なエラーハンドリングが必要（Windows API はエラーコードを返す）

#### 11.2.3 バイナリサイズ

- デフォルトではバイナリサイズが大きくなる可能性がある
- `-ldflags="-s -w"`を使用してシンボル情報を削除
- UPX などの圧縮ツールを使用する場合は注意（ウイルス対策ソフトに検出される可能性）

#### 11.2.4 実行時の権限

- 管理者権限が必要なため、UAC プロンプトが表示される
- マニフェストファイルで管理者権限を要求する場合は、常に UAC プロンプトが表示される

#### 11.2.5 デバッグ

- システムトレイアプリケーションは通常のコンソール出力が表示されない
- デバッグ時はログファイルに出力するか、デバッガーを使用
- `-ldflags`の`-H windowsgui`を外すとコンソールウィンドウが表示される（開発時のみ）

### 11.3 配布に関する注意事項

- シングルバイナリで配布可能だが、ウイルス対策ソフトに誤検出される可能性がある
- コード署名証明書を使用することで信頼性を向上可能
- インストーラーを作成する場合は、NSIS や Inno Setup などを使用
