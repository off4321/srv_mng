FROM golang:1.25-bookworm

# 開発・デバッグに必要なツールのインストール
RUN apt update
RUN apt install -y curl git make ethtool wakeonlan iproute2 openssh-server iputils-ping netcat-openbsd expect sqlite3

# Goソースコードの作業ディレクトリを設定
ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH
WORKDIR $GOPATH/src/srvmng_api

# アプリケーションのソースコード、Goモジュール定義、設定ファイルを全てコンテナ内にコピー
COPY go.mod .
COPY main.go .
COPY api/ api/        
COPY routers/ routers/
COPY service/ service/
COPY utils/ utils/
COPY start.sh .


# 依存関係をダウンロード
RUN go mod tidy

RUN go build -o srvmng_api .

# start.shに実行権限を付与
RUN chmod +x ./start.sh

CMD ["./start.sh"]