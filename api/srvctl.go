package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"srv_mng/service" // サービス層 (ビジネスロジック)
	"srv_mng/utils" // utilsパッケージを使用
)

// PowerActionRequest は /power/start, /power/stop リクエストのペイロード
type PowerActionRequest struct {
	Target string `json:"target"`
}

// PowerHandler は /power/start と /power/stop を処理するハンドラです。
func PowerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.WriteJSON(w, http.StatusMethodNotAllowed, utils.JSONResponse{Status: "error", Message: "Only POST method is supported"})
		return
	}

	// URLからアクション (start/stop) を取得
	pathParts := strings.Split(r.URL.Path, "/")
	action := pathParts[len(pathParts)-1]
	
	// アクション名が不正でないかチェック
	if action != "start" && action != "stop" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.JSONResponse{Status: "error", Message: fmt.Sprintf("Invalid action '%s'. Must be 'start' or 'stop'.", action)})
		return
	}

	var req PowerActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteJSON(w, http.StatusBadRequest, utils.JSONResponse{Status: "error", Message: "Invalid JSON format or missing required fields in request body"})
		return
	}

	targetName := req.Target
	if targetName == "" {
		utils.WriteJSON(w, http.StatusBadRequest, utils.JSONResponse{Status: "error", Message: "Missing 'target' parameter in request."})
		return
	}

	// ターゲットの設定情報をDBから取得
	// service.GetTargetConfig() は DB から targetName に一致するレコードを検索します
	config, err := service.GetTargetConfig(targetName)
	if err != nil {
		// DB接続エラー、またはターゲットが見つからない場合
		utils.WriteJSON(w, http.StatusBadRequest, utils.JSONResponse{
			Status: "error", 
			Message: fmt.Sprintf("Target configuration fetch failed: %s", err.Error()),
		})
		return
	}
	
	// ホストターゲット（IP/MAC/User/Passが必要なもの）か確認
	if config.Type != "host" {
		// host 以外のタイプは電源制御の対象外とする
		utils.WriteJSON(w, http.StatusBadRequest, utils.JSONResponse{
			Status: "error", 
			Message: fmt.Sprintf("Power control only supported for 'host' type targets. Target '%s' is type '%s'.", config.Name, config.Type),
		})
		return
	}

	// サービス層の実行
	output, err := service.ExecutePowerScript(action, config)

	if err != nil {
		utils.WriteJSON(w, http.StatusInternalServerError, utils.JSONResponse{
			Status: "failure",
			Action: action,
			Target: config.Name,
			Message: err.Error(),
			ScriptOutput: output,
		})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.JSONResponse{
		Status:  "success",
		Action:  action,
		Target:  config.Name,
		Message: fmt.Sprintf("Power '%s' command executed for %s.", action, config.Name),
		ScriptOutput: output,
	})
}

// StatusHandler は /status を処理するハンドラです。JSONまたはプレーンテキストを返します。
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.WriteJSON(w, http.StatusMethodNotAllowed, utils.JSONResponse{Status: "error", Message: "Only GET method is supported"})
		return
	}

	// サービス層からすべてのターゲットのステータスを取得
	statuses, err := service.GetAllTargetsStatus()
	if err != nil {
		// 設定ファイルロードエラー時は、JSON/Plain Textどちらの場合でもエラーを返す
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("ERROR: Configuration loading failed: %s\n", err.Error())))
		return
	}

	accept := r.Header.Get("Accept")

	// 1. Acceptヘッダーがない、または明示的にJSONが含まれている場合、JSONを返す
	// JSONをデフォルトにします。
	// CLI/curl向けに、text/plainが明示的に要求された場合のみプレーンテキストを返すようにロジックを反転させます。
	isPlainTextRequested := strings.Contains(accept, "text/plain") && !strings.Contains(accept, "application/json")
	
	if isPlainTextRequested {
		// プレーンテキストを返す（CLIのデフォルト）
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		
		// プレーンテキスト形式でデータを整形
		output := formatStatusAsPlainText(statuses)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(output))
		return
	}
	
	// 2. それ以外（Acceptヘッダーがない、またはJSONが優先される）の場合、JSONを返す
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK) 

	// JSONとしてエンコード
	if err := json.NewEncoder(w).Encode(statuses); err != nil {
		http.Error(w, "Error encoding JSON response", http.StatusInternalServerError)
	}
}

// formatStatusAsPlainText は service.TargetStatus スライスを ASCII 表形式に整形します。
func formatStatusAsPlainText(statuses []service.TargetStatus) string {
	// ヘッダー
	header := "\nSHOW SERVERS AND CONTAINERS STATUS\n" +
		"TYPE     TARGET         HOST:PORT          STATUS\n" +
		"------------------------------------------------------------------------\n"

	var sb strings.Builder
	sb.WriteString(header)

	// 各ターゲットに対して結果を整形
	for _, s := range statuses {
		// プレーンテキストとして固定幅で整形
		line := fmt.Sprintf(
			"%-12s%-25s%-25s%s\n",
			s.Type,
			s.Name,
			s.HostPort,
			strings.ToUpper(s.Status),
		)
		sb.WriteString(line)
	}

	return sb.String()
}