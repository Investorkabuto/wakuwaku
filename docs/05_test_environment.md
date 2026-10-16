# 05. テスト運用環境（10人程度のクローズドテスト）

> ステータス: ドラフト v0.1 / 2026-10-16
> 想定: 高齢者2〜3人＋その家族の合計10人程度。期間は数か月
> 金額は2026年時点の公開価格をもとにした**目安**。確定前に各サービスの料金ページで確認する。

## 1. 全体像

```
[テスター端末 10台程度]
   iOS: TestFlight で配布 ／ Android: Google Play 内部テスト（または Firebase App Distribution）で配布
        │
        ▼
[Firebase]   Auth / FCM / Crashlytics / Remote Config / App Check / Storage
        │
        ▼
[GCP 東京リージョン（asia-northeast1）]
   Cloud Run（Go API、アクセスがないときは0台まで縮小）
   Cloud SQL for PostgreSQL（最小構成）
   Cloud Scheduler（見守り判定ジョブ）
   Secret Manager / Artifact Registry / Cloud Logging
        ▲
        │ GitHub Actions でテスト・ビルド・デプロイ
[GitHub リポジトリ]
```

## 2. 用意するアカウント・契約（オーナーが手続きする）

| 項目 | 費用（目安） | 備考 |
|---|---|---|
| Google Cloud／Firebase プロジェクト（課金アカウント付きの従量課金プラン） | 下の月額費用を参照 | Cloud Run・Cloud SQLを使うには課金アカウントが必須。**予算アラートを必ず設定する** |
| Apple Developer Program | 年99 USD | TestFlight での配布と HealthKit の利用に必要。**登録審査に数日〜数週間**かかるので早めに申請する |
| Google Play Console | 25 USD（1回のみ） | 内部テストトラックで最大100人に配布できる。新規の個人アカウントは、本番公開の前にクローズドテストの要件がある点に注意 |
| GitHub | 無料〜 | プライベートリポジトリで macOS ランナーを使うと、使用時間が多く消費される（下記参照） |

## 3. GCP / Firebase 構成と月額費用の目安

| リソース | 設定 | 月額の目安 |
|---|---|---|
| Cloud Run（Go API） | 最小0台・最大2台、CPU 1／メモリ 512MiB | 無料枠内の見込み（ほぼ0円） |
| Cloud SQL for PostgreSQL | 共有コアの最小構成、SSD 10GB、自動バックアップあり、高可用性構成なし | **約1,500〜2,500円**（費用の大半） |
| Cloud Scheduler | ジョブ3本（服薬未記録の判定、無操作の判定、日次の集計） | 無料枠内 |
| Firebase Auth | Apple・Googleログインは無料。SMSは1通ごとに課金 | SMSを使わなければ0円 |
| FCM / Crashlytics / Remote Config / App Check | — | 無料 |
| Cloud Storage | 薬の写真＋3Dアセット 数GB | 数十円 |
| Artifact Registry / Secret Manager / Logging | — | 数十〜百円程度 |
| **合計** | | **月2,000〜3,000円程度＋Apple年会費** |

> さらに安くする方法として、DBを外部の無料枠（Neonなど）にすることもできる。ただし健康情報を国内の管理下に置く方針（ADR-004）と、本番への移行のしやすさを考え、テストでも Cloud SQL（東京）を推奨する。

### 環境の分け方
- **dev**（開発・自動テスト用）と **test**（10人のテスターが使う環境）の2プロジェクトに分ける。本番（prod）は公開前に追加する。
- dev では Cloud SQL を常時起動せず、Docker 上の PostgreSQL で代用して費用を抑える。

## 4. ビルド・配布

| 項目 | 方法 |
|---|---|
| Android ビルド | GitHub Actions の Linux ランナー |
| **iOS ビルド** | **macOS が必須**。GitHub Actions の macOS ランナー、または Codemagic などの外部サービス。開発用のクラウド環境（Linux）では iOS アプリをビルドできない |
| 署名 | iOS：App Store Connect のAPIキー＋fastlane match。Android：アップロード鍵を GitHub Secrets に保管 |
| 配布 | iOS：TestFlight の内部テスター（最大100人、審査なし）。ただし内部テスターは App Store Connect への登録が必要なため、家族テスターには**外部テスター（初回のみ簡易審査あり）**のほうが現実的。Android：Play の内部テスト |
| サーバーのデプロイ | GitHub Actions → Cloud Run。認証は鍵ファイルではなく **Workload Identity 連携**で行う |

## 5. テスト用の端末

| 種別 | 推奨 |
|---|---|
| iPhone | iOS 16 以上。古めの機種（iPhone SE 第2世代など）を1台は含める |
| Android | **中位〜下位の機種を1〜2台**（3D描画の性能検証、ADR-001の判定に使う）。Android 9以上 |
| 高齢テスターの端末 | 本人が普段使っている端末をそのまま使う（文字サイズ設定などの実際の環境で確かめるため） |

## 6. テスト運用で最低限やること

- **同意と説明**：テストでも本物の服薬情報を扱うため、簡易版のプライバシーポリシーとテスト参加の同意書を用意する。「緊急通報サービスではない」ことも明記する。
- **監視**：Cloud Monitoring で APIの外形監視（数分おきに応答を確認）、Error Reporting、見守り通知の送信失敗アラートを設定する。通知先はオーナーのメール。
- **予算アラート**：月5,000円などのしきい値を超えたらメールで知らせる。
- **バックアップ**：Cloud SQL の自動バックアップ（7日分）を有効にする。
- **フィードバック窓口**：アプリ内の「ご意見」ボタン（Firestoreではなく API 経由で保存）と、家族向けの連絡手段（LINEグループなど）。

## 7. オーナーに対応してもらうこと（AIでは実施できない作業）

1. GCP の課金アカウント作成と、Firebase プロジェクトの作成（dev / test）
2. Apple Developer Program と Google Play Console の登録・支払い
3. GitHub Actions から GCP へデプロイする権限の付与（Workload Identity 連携の設定。手順書はこちらで用意する）
4. iOS 署名用のAPIキー発行と、GitHub Secrets への登録
5. テスターの募集と同意書の受け取り

それ以外（インフラ設定のコード化（Terraform）、CI/CD、サーバー・アプリの実装、手順書）はすべて開発側（AI）で用意する。
