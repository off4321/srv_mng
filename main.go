package main

import (
	"log"
	"net/http"
	"os"
	"time"
	"fmt"

	"srv_mng/routers" 
	"srv_mng/service" 
)

const (
	// SQLiteデータベースファイルへのパスを定義
	SQLiteDSN = "./monitor.db" 
	
	// APIサーバーがリッスンするポート
	ServerPort = "8080"
)


func main() {
	// ────────────────────────────────
	// 1. データベースの初期化
	// ────────────────────────────────
	fmt.Printf("Initializing database connection (DSN: %s)...\n", SQLiteDSN)
	if err := service.InitDB(SQLiteDSN); err != nil {
		fmt.Printf("FATAL: Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Database connection successful.")

	// ────────────────────────────────
	// 2. データベースファイルが存在しない場合の初期テーブル作成
	// ────────────────────────────────
	// 簡単な初期化関数を呼び出し
	if err := service.CreateInitialTables(); err != nil {
		fmt.Printf("WARNING: Failed to create initial tables (may already exist): %v\n", err)
	}

	// ログフォーマットを設定
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) 
	
	// routersパッケージからルーターを取得し、すべてのハンドラを設定
	r := routers.NewRouter()

	// 正常性チェック用ルート
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Go Power API is running."))
	})


	port := os.Getenv("PORT")
	if port == "" {
		port = "5001"
	}

	srv := &http.Server{
		Handler: r,
		Addr: ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout: 15 * time.Second,
	}

	log.Printf("Server listening on port %s", port)
	
	// サーバー起動
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
