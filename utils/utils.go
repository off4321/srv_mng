package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// JSONResponse は API レスポンスの共通構造体です。
type JSONResponse struct {
	Status string `json:"status"`
	Action string `json:"action,omitempty"`
	Target string `json:"target,omitempty"`
	Message string `json:"message"`
	ScriptOutput string `json:"script_output,omitempty"`
}

// WriteJSON は、HTTPレスポンスライターにJSONデータを書き込むヘルパー関数です。
// すべてのAPIハンドラはこの関数を使って応答を返します。
func WriteJSON(w http.ResponseWriter, status int, data JSONResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// エンコードエラーが発生した場合、サーバーログに出力
		fmt.Printf("Error writing JSON response: %v\n", err)
	}
}