package routers

import (
	"net/http"
	
	// APIハンドラ層をインポート
	"srv_mng/api" 
)

// NewRouter はルーティングを設定した ServeMux を返します。
// すべてのAPIエンドポイントをここで定義し、apiパッケージのハンドラを関連付けます。
func NewRouter() *http.ServeMux {
	mux := http.NewServeMux()

	// /status ルート
	mux.HandleFunc("/status", api.StatusHandler)

	// /power/start や /power/stop ルート
	// /power/ の後をパスとして処理するため、末尾にスラッシュを付ける
	mux.HandleFunc("/power/", api.PowerHandler)

	return mux
}