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
- 5分足・オープン（EMA）・クローズ（


# ec2のユーザデータで初期化
```
#!/bin/bash

# ホスト名
sed -i 's/^HOSTNAME=[a-zA-Z0-9\.\-]*$/HOSTNAME={ホスト名}/g' /etc/sysconfig/network
hostname '{ホスト名}'

# タイムゾーン
cp /usr/share/zoneinfo/Japan /etc/localtime
sed -i 's|^ZONE=[a-zA-Z0-9\.\-\"]*$|ZONE="Asia/Tokyo”|g' /etc/sysconfig/clock

# 言語設定
echo "LANG=ja_JP.UTF-8" > /etc/sysconfig/i18n

```

# ec2にGoをインストールする
- インストール可能なバージョンを確認する
  - amazon-linux-extras list | grep golang
- インストールする
  - sudo amazon-linux-extras install golang1.11
- gopathの設定

```.bashrc

export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin

```
- 保存してから読み込み
  - source ~/.bashrc
  
- モジュールのインストール
  - go get
  
- sql-migrateのインストール
  - プロジェクトの階層にて`go get -u github.com/rubenv/sql-migrate/sql-migrate`
  
- pathの設定
```.bash_profile
PATH=$PATH:$HOME/.local/bin:$HOME/bin
export PATH="$HOME/go/bin:$PATH"       ←これ追加
export PATH
```
# rds
- サブネットグループの作成
  - プライベートサブネットをマルチA-Zで選択
  
- パラメータグループの作成
  - max_prepared_stmt_count：1048576
  
# バックグラウンドでの実行と停止
- go run main.go &
- 以下でもいけると思ったがバックグランドで実行できなかった。
  - go build -o bitcoin-system-trade-backend ./main.go
  - ./bitcoin-system-trade-backend &
  
- 参考
  - [Go製のツールを使うときはパスを通す必要がある](https://kdnakt.hatenablog.com/entry/2019/11/03/080000)
  - [Amazon Linux2にGolangの1.11をインストールする](https://public-constructor.com/amazon-linux2-golang-installation/)