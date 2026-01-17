// agent.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// エージェントの設定
type AgentConfig struct {
	Port string
}

// エージェントの初期化
func initAgent(config *AgentConfig) {
	// HTTPサーバーのルートハンドラを設定
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/shutdown", shutdownHandler)
	http.HandleFunc("/cpucheck", cpuHandler)

	// サーバーを起動
	log.Printf("Agent starting on port %s", config.Port)
	log.Fatal(http.ListenAndServe(":"+config.Port, nil))
}

// ステータス確認エンドポイント
func statusHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Status Confirm Endpoint start")
	// GETリクエストのみを許可
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		log.Printf("Method not allowed")
		return
	}

	// サービスの状態を確認
	status := checkServiceStatus()

	// 状態をレスポンスとして返す
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "%s"}`, status)
	log.Printf(`{"status": "%s"}`, status)
	log.Printf("Status Confirm Endpoint Successfully finished")
}

// shutdownHandler は、シャットダウンリクエストを処理します。
func shutdownHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Shutdown Request Endpoint start")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	password, exists := body["password"]
	if !exists {
		http.Error(w, "Password required", http.StatusBadRequest)
		return
	}

	// sudo shutdownコマンドを実行
	cmd := exec.Command("sudo", "-S", "shutdown", "-h", "now")
	cmd.Stdin = strings.NewReader(password + "\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Shutdown failed: %v, Output: %s", err, output)
		http.Error(w, "Shutdown failed", http.StatusInternalServerError)
		return
	}

	log.Printf("Shutdown successful: %s", output)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Shutdown initiated successfully"))
	log.Printf("Status Confirm Endpoint Successfully finished")
}

// ただrunningを返す
func checkServiceStatus() string {
	log.Printf("Running return")
	return "running"
}

// CPU使用率を取得
func getCPUUsage() (float64, error) {
	log.Printf("CPU Check Start")
	// CPU使用率を取得するためのコマンドを実行
	cmd := exec.Command("top", "-bn1", "-d0.5")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to execute top command: %v", err)
	}

	// 出力を解析してCPU使用率を抽出
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Cpu(s)") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				// "us" (user) + "sy" (system) = CPU使用率
				us, err := strconv.ParseFloat(strings.Trim(parts[1], "%id"), 64)
				if err != nil {
					return 0, fmt.Errorf("failed to parse user CPU usage: %v", err)
				}
				sy, err := strconv.ParseFloat(strings.Trim(parts[3], "%id"), 64)
				if err != nil {
					return 0, fmt.Errorf("failed to parse system CPU usage: %v", err)
				}
				return us + sy, nil
			}
		}
	}
	return 0, fmt.Errorf("failed to find CPU usage in output")
}

// CPU使用率確認エンドポイント
func cpuHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("CPU Check Endpoint Start")
	// GETリクエストのみを許可
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// CPU使用率を取得
	usage, err := getCPUUsage()
	if err != nil {
		log.Printf("Failed to get CPU usage: %v", err)
		http.Error(w, "Failed to get CPU usage", http.StatusInternalServerError)
		return
	}

	// 使用率をレスポンスとして返す
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"cpu_usage": %.2f}`, usage)
	log.Printf(`{"cpu_usage": %.2f}`, usage)
	log.Printf("CPU Check Endpoint successfully finished")
}

func main() {
	// 環境変数からポートを取得（デフォルトは8080）
	port := os.Getenv("AGENT_PORT")
	if port == "" {
		// コマンドライン引数からポートを取得
		if len(os.Args) > 1 {
			port = os.Args[1]
		} else {
			port = "8080"
		}
	}

	config := &AgentConfig{
		Port: port,
	}

	// エージェントを初期化
	initAgent(config)
}
