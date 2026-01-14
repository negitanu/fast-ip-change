.PHONY: build build-all build-cross build-debug clean run test rsrc

# リソースファイルの生成（rsrc.syso）
rsrc:
	cd cmd/fast-ip-change && rsrc -manifest fast-ip-change.manifest -ico ../../assets/systray.ico -o rsrc.syso
	cd cmd/settings && rsrc -manifest settings.manifest -ico ../../assets/systray.ico -o rsrc.syso
	cd cmd/ipstatus && rsrc -manifest ipstatus.manifest -ico ../../assets/systray.ico -o rsrc.syso
	cd cmd/routetable && rsrc -manifest routetable.manifest -ico ../../assets/systray.ico -o rsrc.syso
	cd cmd/logviewer && rsrc -manifest logviewer.manifest -ico ../../assets/systray.ico -o rsrc.syso

# ビルド（Windows環境用）- メインアプリケーションのみ
build: rsrc
	go build -ldflags="-H windowsgui -s -w" -trimpath -o fast-ip-change.exe ./cmd/fast-ip-change

# すべてのアプリケーションをビルド
build-all: rsrc
	go build -ldflags="-H windowsgui -s -w" -trimpath -o fast-ip-change.exe ./cmd/fast-ip-change
	go build -ldflags="-H windowsgui -s -w" -trimpath -o settings.exe ./cmd/settings
	go build -ldflags="-H windowsgui -s -w" -trimpath -o ipstatus.exe ./cmd/ipstatus
	go build -ldflags="-H windowsgui -s -w" -trimpath -o routetable.exe ./cmd/routetable
	go build -ldflags="-H windowsgui -s -w" -trimpath -o logviewer.exe ./cmd/logviewer

# クロスコンパイル（WSL/Linux環境からWindows向けビルド）
build-cross: rsrc
	GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -trimpath -o fast-ip-change.exe ./cmd/fast-ip-change
	GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -trimpath -o settings.exe ./cmd/settings
	GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -trimpath -o ipstatus.exe ./cmd/ipstatus
	GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -trimpath -o routetable.exe ./cmd/routetable
	GOOS=windows GOARCH=amd64 go build -ldflags="-H windowsgui -s -w" -trimpath -o logviewer.exe ./cmd/logviewer

# デバッグビルド（コンソールウィンドウを表示）
build-debug:
	go build -o fast-ip-change.exe ./cmd/fast-ip-change
	go build -o settings.exe ./cmd/settings
	go build -o ipstatus.exe ./cmd/ipstatus
	go build -o routetable.exe ./cmd/routetable
	go build -o logviewer.exe ./cmd/logviewer

# クリーン
clean:
	rm -f *.exe
	rm -f cmd/*/rsrc.syso

# 実行（デバッグモード）
run:
	go run ./cmd/fast-ip-change

# テスト
test:
	go test ./...

# テスト（カバレッジ）
test-coverage:
	go test -cover ./...

# 依存関係の更新
deps:
	go mod download
	go mod tidy

# リント（golangci-lintがインストールされている場合）
lint:
	golangci-lint run
