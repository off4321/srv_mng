package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"srv_mng/service" 
)

// RegResponse はターゲット登録APIの応答構造体です。
type RegResponse struct {
	Status  string `json:"status"`
	Target  string `json:"target,omitempty"`
	Message string `json:"message"`
}

// RegisterTargetHandler は /targets/register を処理するハンドラです。
// POSTリクエストを受け付け、DBに新しいターゲット設定を保存または既存のものを更新します。
func RegisterTargetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"status":"error", "message":"Only POST method is supported"}`, http.StatusMethodNotAllowed)
		return
	}

	// リクエストボディを service.MonitorTarget 構造体としてデコード
	var config service.MonitorTarget 
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		resp := RegResponse{
			Status: "error",
			Message: fmt.Sprintf("Invalid JSON format or missing required fields: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// 1. service層にDB保存を依頼
	if err := service.SaveMonitorTarget(&config); err != nil {
		resp := RegResponse{
			Status: "failure",
			Target: config.Name,
			Message: fmt.Sprintf("Failed to save target configuration: %v", err),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(resp)
		return
	}

	// 成功応答
	resp := RegResponse{
		Status: "success",
		Target: config.Name,
		Message: fmt.Sprintf("Target '%s' configuration successfully saved or updated.", config.Name),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}