package routers

import (
	"net/http"
	
	// APIハンドラ層をインポート
	"srv_mng/api" 
)

// NewRouter はルーティングを設定した ServeMux を返します。
// すべてのAPIエンドポイントをここで定義し、apiパッケージのハンドラを関連付けます。
func NewRouter() *http.ServeMux {
	// ルーターの作成
	mux := http.NewServeMux()

	// [電源制御エンドポイント] POSTリクエストでターゲットの電源操作を実行
	mux.HandleFunc("/power/start", api.PowerHandler)
	mux.HandleFunc("/power/stop", api.PowerHandler)

	// [ステータス確認エンドポイント] GETリクエストで全ターゲットの死活確認結果を取得
	mux.HandleFunc("/status", api.StatusHandler)
    
	// [ターゲット登録/更新エンドポイント] POSTリクエストで新しいターゲットをDBに登録または更新
	mux.HandleFunc("/targets/register", api.RegisterTargetHandler)

	return mux
}