package main

import (
	"log"
	"net/http"
	"os"
	"time"

	// routersパッケージをインポート
	"srv_mng/routers" 
)


func main() {
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