FROM golang:1.25-bookworm

# 開発・デバッグに必要なツールのインストール
RUN apt update
RUN apt install -y curl git make ethtool wakeonlan iproute2 openssh-server iputils-ping netcat-openbsd expect

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


# 依存関係をダウンロード
RUN go mod tidy

RUN go build -o srvmng_api .

CMD ["./srvmng_api"]