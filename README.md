# bitcoin-system-trade-backend
# フォルダ構成
- API通信：bitflyer（他の取引所を使う際はapi/bitflyerにする）
- アプリケーション層：application（DB操作とかSerializerとかを呼んでレスポンスをまとめる）
- ドメイン層：domain（ModelとかControllerから呼ぶserializerとか）
### フロー
- controller→service→database→repositoryな感じ
  - controllerからは外部API通信かservice（DB）を呼ぶだけ（serviceは名前だけで何をするかわかるようにできるだけ具体的な名前がいいかも）
  - repositoryが実際にDB通信できる箇所
- 現状完全なレイヤードアーキではないけどリリースを最優先で徐々にきれいにしてく
### システムトレードアプリのAPI
- 現在所持している現金やビットコインの情報を取得する：`GetBalance`
- ビットコインの情報（現在の価格等）を取得する：`GetTicker`
- リアルタイムなビットコインの情報を取得する：`GetRealTimeTicker`
- 手数料を取得する：`GetTradingCommission`
- 売買する：`SendOrder`
- 売買履歴を確認する：`ListOrder`
- 指定したプロダクトコード・時間足のキャンドル情報を取得する：`GetAllCandle`
  - 確認方法：`http://localhost:8080/api/chart?product_code=FX_BTC_JPY&duration=1h`

# SETUP
- アプリ起動
  - `docker-compose up`
  - 下図のようにデバッグ設定を追加
![スクリーンショット 2020-06-14 10 15 39](https://user-images.githubusercontent.com/39196956/84582665-f70df280-ae29-11ea-9531-4580cdef853f.jpg)
- godoc
  - コンテナに入る必要有り
  - `godoc -http=:6060`
  
# データベース
- Mysqlを使用する
- ORマッパーは使用しない
- マイグレーションは[sql-migrate](https://github.com/rubenv/sql-migrate)を使用する
  - `sql-migrate new テーブル名`でマイグレーションファイル作成
  - `sql-migrate up`でマイグレーション（アップグレード）
  - `sql-migrate down`でダウンダウングレード
  
# github運用
- issueベースのPR開発
  - issueを登録する
  - `feature/Issues#○○`でブランチを作る
  - `git commit -m "close #○○" --allow-empty`で空コミットしてissueと紐付ける
  
# デプロイ
- とりあえずec2
  - 言語設定
    - `sudo vim /etc/sysconfig/i18n`
```i18n
LANG=ja_JP.UTF-8
```
  - 時間設定
    - `sudo cp /usr/share/zoneinfo/Japan /etc/localtime`
    - `sudo vim /etc/sysconfig/clock`
```click
ZONE="Asia/Tokyo"
UTC=true
```

# 検証
- 11/21
  - 条件
    - 対象足：1分足
    - オープンインディケータ数：1以上
    - クローズインディケータ数：1以上
  - 結果
    - 収益：-313円
    - 取引数：16（SIGNAL_EVENTSのレコード的に言うと32）
    - 所感：急激な値段の変化に対応できない感じ（18時頃まで+1000円程度出てたが急激な変化2取引で一気にマイナス）
- 11/22
  - 条件
    - 対象足：1分足
    - オープンインディケータ数：2以上
    - クローズインディケータ数：1以上
  - 結果
    - 収益：+70円
    - 取引数：9（SIGNAL_EVENTSのレコード的に言うと18）
    - 所感：取引条件を厳しくした割にマイナス取引も多かったため半日で終了。やるとしたらもっと厳しくしないと意味ないっぽい。
- 11/23
  - 対象足：1分足
  - オープンインディケータ数：1以上（EMAを固定(7,14,50))
  - クローズインディケータ数：1以上
  