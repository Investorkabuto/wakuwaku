# wakuwaku（仮：わくわく農園）

どうぶつの森風の農業スローライフゲーム × 高齢者の服薬管理・見守り。
現実の服薬・運動が、ゲーム内の畑を育てる「水」や「肥料」になる。家族はゲームの中で遊びに行く形で自然に見守れる。

- クライアント: Flutter
- 3D: flutter_scene（第2候補 Godot）※Phase 0で検証
- サーバー: Go（モジュラーモノリス）+ Connect RPC + PostgreSQL（Cloud Run / Cloud SQL 東京）
- Firebase: Auth / FCM / Crashlytics / Remote Config / App Check / Storage

## ドキュメント

- [01. 構想（コンセプト）](docs/01_concept.md)
- [02. 要件定義](docs/02_requirements.md)
- [03. システム設計（初期案）](docs/03_architecture.md)
- [04. 技術選定の検討（ADR）](docs/04_tech_decisions.md)
- [05. テスト運用環境（10人規模）](docs/05_test_environment.md)
