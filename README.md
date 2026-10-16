# wakuwaku（仮：わくわく農園）

どうぶつの森風の農業スローライフゲーム × 高齢者の服薬管理・見守り。
現実の服薬・運動が、ゲーム内の畑を育てる「水」や「肥料」になる。家族はゲームの中で遊びに行く形で自然に見守れる。

- クライアント（最初）: Web版PWA — SvelteKit + three.js + Webプッシュ（ADR-005）
- クライアント（将来）: ネイティブアプリ（Capacitor または Flutter。テスト後に判断）
- サーバー: Go（モジュラーモノリス）+ Connect RPC + PostgreSQL（Cloud Run / Cloud SQL 東京）
- Firebase: Auth / FCM / Crashlytics / Remote Config / App Check / Storage

## ドキュメント

- [01. 構想（コンセプト）](docs/01_concept.md)
- [02. 要件定義](docs/02_requirements.md)
- [03. システム設計（初期案）](docs/03_architecture.md)
- [04. 技術選定の検討（ADR）](docs/04_tech_decisions.md)
- [05. テスト運用環境（10人規模）](docs/05_test_environment.md)
- [06. Webアプリ先行の課題](docs/06_web_first.md)

## 開発

```
server/   Go API（Connect RPC, PostgreSQL）
web/      Web版（SvelteKit PWA）
proto/    API定義（ここから Go / TypeScript のコードを生成）
```

必要なもの: Go 1.26+, Node.js 22+, PostgreSQL 16（`docker compose up -d db` でも可）

```sh
make tools      # コード生成ツールと web の依存をインストール
make gen        # proto を変更したら実行
make run-api    # API（http://localhost:8080、開発用ログイン）
make run-web    # Web（http://localhost:5173）
make test check
```

ローカルでは Firebase を使わず、「開発用ログイン」（名前を入れるだけ）で動作する。
