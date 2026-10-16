DATABASE_URL ?= postgres://wakuwaku:wakuwaku@localhost:5432/wakuwaku?sslmode=disable
export PATH := $(HOME)/go/bin:$(PATH)

.PHONY: tools gen lint test run-api run-web check

tools: ## コード生成ツールのインストール
	go install github.com/bufbuild/buf/cmd/buf@v1.47.2
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.5
	go install connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.18.1
	cd web && npm ci

gen: ## proto から Go / TypeScript のコードを生成
	buf lint
	buf generate

test: ## サーバーとWebのテスト（サーバーはDBが必要）
	cd server && TEST_DATABASE_URL="$(DATABASE_URL)" go test ./...
	cd web && npm test

check: ## 静的チェック
	cd server && go vet ./...
	cd web && npm run check

run-api: ## APIサーバーを起動（開発用ログイン）
	cd server && DATABASE_URL="$(DATABASE_URL)" go run ./cmd/api

run-web: ## Webアプリを起動
	cd web && npm run dev
