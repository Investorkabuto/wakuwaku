# 03. システム設計（初期案）

> ステータス: ドラフト v0.2 / 2026-10-16（技術選定は [04_tech_decisions.md](04_tech_decisions.md) を正とする）

## 1. 全体構成

```
┌──────────────── Flutter アプリ (iOS / Android) ────────────────┐
│  UI層 (Flutter Widgets)   3Dゲーム層 (flutter_scene ※検証後確定) │
│                 └── GameBridge（エンジン差し替え用の境界）       │
│  状態管理 (Riverpod)                                            │
│  ローカルDB (drift / SQLite, 暗号化)  ← オフラインファースト     │
│  ローカル通知 (flutter_local_notifications)                     │
│  ヘルス連携 (health パッケージ: HealthKit / Health Connect)      │
│  同期エンジン (送信キュー + 差分取得)                             │
└──────────────┬───────────────────────────┬─────────────────────┘
               │ HTTPS (Connect RPC/protobuf) │ WebSocket (Phase 3 リアルタイム訪問)
┌──────────────▼───────────────────────────▼─────────────────────┐
│ API サーバー (Go モジュラーモノリス) ── Cloud Run / コンテナ     │
│  ├ 認証 (Firebase Auth のIDトークン検証)                        │
│  ├ タスク・服薬API / 同期API                                    │
│  ├ ゲームAPI (農園状態、インベントリ、ショップ、訪問)              │
│  ├ 家族・交流API                                                │
│  └ 管理API                                                      │
│ ワーカー (見守り判定ジョブ / 週次サマリー / プッシュ送信)          │
└──────┬─────────────┬───────────────┬───────────────┬───────────┘
       │             │               │               │
  PostgreSQL       Redis        オブジェクト      FCM / APNs
  (主データ)   (キャッシュ/      ストレージ       (プッシュ通知)
               ジョブキュー)    (薬写真・アセット)
```

## 2. 技術選定

| 領域 | 候補（推奨を太字） | 理由 |
|---|---|---|
| クライアント | **Flutter** | 要件指定。iOS/Android単一コード |
| 3D描画 | **flutter_scene**（第2候補: Godot+LibGodot） | 3D要件。Phase 0の技術検証で確定（ADR-001） |
| 3Dアセット | **Blender Python スクリプト → glTF** | 自社制作をコードで行う（ADR-002） |
| 状態管理 | **Riverpod** | テスト容易・非同期処理に強い |
| ローカルDB | **drift (SQLite)** + SQLCipher | 型安全、マイグレーション、暗号化対応 |
| ヘルス連携 | `health` パッケージ | HealthKit/Health Connect 両対応 |
| サーバー言語 | **Go**（モジュラーモノリス） | 性能・運用・拡張性。3Dエンジン選定に左右されない（ADR-003） |
| API定義 | **Connect RPC (protobuf)** | Go/Dartのコード生成でスキーマを一元管理 |
| 認証 | **Firebase Authentication** | SMS・Apple・Googleログインを低工数で実現 |
| DB | **PostgreSQL**（Cloud SQL 等） | リレーショナルな家族・権限モデルに適する |
| キャッシュ/キュー | Redis | セッション、レート制限、ジョブキュー |
| プッシュ | FCM（iOSはAPNs経由） | |
| インフラ | **GCP (Cloud Run + Cloud SQL + Cloud Storage)** | Firebaseとの親和性、国内リージョン（東京）でデータ保管 |
| 監視 | Crashlytics, Cloud Logging/Monitoring, Sentry | |
| CI/CD | GitHub Actions + Fastlane | |

> 代替案: 初期速度を最優先する場合は **Supabase / Firebase(Firestore+Functions)** だけで MVP を作る方法もある。ただし見守り権限（項目単位の共有制御）とゲーム経済の不正防止をサーバーで厳密に扱う必要があるため、本案では専用APIサーバーを推奨。

## 3. クライアント構成

```
lib/
├ app/            # ルーティング、テーマ（やさしい/ふつう）、DI
├ core/           # 共通（ネットワーク、DB、同期、通知、アクセシビリティ）
├ features/
│  ├ auth/
│  ├ onboarding/   # QR代行セットアップ
│  ├ tasks/        # 服薬・運動・汎用タスク
│  ├ watch/        # 見守りパネル
│  ├ family/       # 家族グループ・交流
│  ├ farm/         # 農園（3D / GameBridge経由）
│  ├ village/      # NPC・ショップ・イベント
│  └ settings/
└ gen/             # protobuf から生成したAPIクライアント
```

- **モードはテーマとして実装**: `EasyMode` / `NormalMode` で文字サイズ・タップ領域・画面遷移パターンを切替。機能ロジックは共通。
- **ゲームロジックの決定性**: 作物の成長は「植えた時刻＋累積した水/肥料」から計算する純関数とし、サーバー(Go)を権威、クライアント(Dart)は同等実装で予測表示。共通ゴールデンテストで一致を保証（不正防止とオフライン対応の両立）。

## 4. オフライン同期方針

| データ | 方式 | 競合解決 |
|---|---|---|
| タスク記録（服薬済み等） | 追記専用イベント（UUIDをクライアント生成） | 競合なし。同一UUIDは冪等 |
| 薬・タスク定義 | バージョン付きレコード | 最終更新優先＋本人編集を家族編集より優先 |
| 農園状態 | イベントソーシング風（植える/水やり/収穫のアクションログ） | サーバーでアクションを検証・再計算、不正値は破棄 |
| インベントリ・通貨 | サーバー権威 | オフライン中は仮反映、同期時にサーバー値で確定 |

- 服薬リマインドは**サーバーに依存せず端末で予約**。薬定義が家族によって更新された場合はサイレントプッシュで再同期・再予約。

## 5. 見守りアラートの仕組み

```
[端末] 服薬記録・アプリ操作・歩数 ──同期──▶ [API] イベント保存
                                                │
[ワーカー] 5分毎: 「予定時刻+閾値を過ぎても記録がない服薬」を検索
           1時間毎: 「最終アクティビティから N 時間経過」を検索
                                                │
           本人の共有同意 & 見守る人の通知設定を確認
                                                │
                    重複抑止（同一アラートは1回）→ FCMで見守る人へ通知
```

- 端末がオフラインで記録が届かない可能性があるため、アラート文言は「記録がまだ届いていません」とし断定しない。
- 本人側にも先にもう一度リマインド（家族通知の前に本人に再確認の機会）。

## 6. データモデル（主要エンティティ）

```
User(id, auth_uid, display_name, mode[easy|normal], birth_year?, timezone, created_at)
FamilyGroup(id, name, owner_user_id)
FamilyMember(group_id, user_id, role[watched|watcher|member])
ShareConsent(watched_user_id, watcher_user_id, scope[medication|steps|mood|activity], granted_at, revoked_at)

Medication(id, user_id, name?, photo_url?, dose_text, active, version)
MedicationSchedule(id, medication_id, timing[morning|noon|evening|bedtime|custom], time, days_of_week, interval_days)
Task(id, user_id, type[exercise|general], title, schedule, reward_type)
TaskLog(id[uuid], user_id, task_ref, kind[med|exercise|general], scheduled_at, done_at, status[done|skipped], source[app|notification|family], client_created_at)
StepDaily(user_id, date, steps, source)
MoodLog(id, user_id, level 1-5, logged_at)

Farm(id, user_id, layout_json, updated_at)
Plot(id, farm_id, x, y, crop_id?, planted_at?, water, fertilizer)
FarmAction(id[uuid], farm_id, actor_user_id, type[plant|water|fertilize|harvest|gift], payload, at)
Inventory(user_id, item_id, qty)
Wallet(user_id, coins)
ItemMaster / CropMaster / ShopMaster / RewardRule   # サーバー管理のマスタ

Visit(id, host_user_id, guest_user_id, at)
Gift(id, from_user_id, to_user_id, item_id, qty, message_template_id, opened_at)
Message(id, group_id, sender_id, kind[stamp|template|text|voice], body, at)

AlertRule(id, watched_user_id, watcher_user_id, type[missed_med|inactivity], threshold_min)
AlertEvent(id, rule_id, triggered_at, notified_at, resolved_at)
AuditLog(id, actor, action, target, at)
```

## 7. API（主要エンドポイント案）

| 分類 | エンドポイント例 |
|---|---|
| 同期 | `POST /sync/push`（イベント一括送信） / `GET /sync/pull?since=` |
| 服薬・タスク | `CRUD /medications`, `CRUD /tasks`, `POST /task-logs` |
| 家族 | `POST /families`, `POST /families/{id}/invites`, `POST /invites/{code}/accept`, `PUT /consents` |
| 見守り | `GET /watch/{userId}/today`, `GET /watch/{userId}/history`, `PUT /alert-rules` |
| ゲーム | `GET /farm/{userId}`, `POST /farm/actions`, `GET /shop`, `POST /shop/purchase` |
| 交流 | `POST /visits`, `POST /gifts`, `POST /messages` |

権限チェックは全てサーバー側で、`ShareConsent` に基づき項目単位でフィルタする。

## 8. セキュリティ設計の要点

- Firebase IDトークン検証 → 内部ユーザーIDへマッピング
- 行レベルの認可（自分 / 同意済み見守り / 家族グループ）をAPI層の共通ミドルウェアで強制
- 薬の写真は署名付きURLでのみ配信、EXIF位置情報を除去
- レート制限（訪問・ギフト・メッセージのスパム対策）
- ゲーム経済の不正対策: 恵みの付与はサーバー側の `RewardRule` で計算し、1日の上限を設ける（タスクを大量登録して稼ぐ行為を防ぐ）

## 9. 決定事項・未決事項

決定:
- 表現は3D／アートは自社（Blenderスクリプト）／開発はAIが担当、期間は問わない
- サーバーは Go＋Connect RPC（[ADR-003](04_tech_decisions.md)）

未決:
1. 3Dエンジンの最終決定（Phase 0の技術検証で flutter_scene か Godot かを決める）
2. インフラ: GCP 前提で良いか
3. 対応言語: 日本語のみで開始か
4. B2B展開の時期（構造上は `care` モジュールを切り出せるようにしておく）
