package service

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ホストOSの電源制御スクリプトのパス
const powerControlScript = "/usr/local/bin/power_control.sh"

// 監視リストファイルのパス (Go APIのコンテキストルートからの相対パス)
const MonitorListFile = "./srv_container_list/list.txt"

// MonitorTarget は list.txt から読み込まれる各監視対象の構造体です。
type MonitorTarget struct {
	Name string
	Host string // IPアドレスまたはホスト名
	Port string // 死活確認に使用するポート番号
	Type string // 対象のタイプ（"host" or "container"）
}

// TargetStatus は、APIエンドポイントで返す監視対象のステータス構造体です。
// JSONエンコーディングのためにフィールド名をエクスポート（大文字始まり）します。
type TargetStatus struct {
	Type string `json:"type"`
	Name string `json:"name"`
	HostPort string `json:"host_port"`
	Status string `json:"status"`
}

// ExecutePowerScript は、すべての電源ON/OFF操作を外部スクリプトに委譲するサービスロジックです。
func ExecutePowerScript(action, targetName string) (string, error) {
	if _, err := os.Stat(powerControlScript); os.IsNotExist(err) {
		return "", fmt.Errorf("error: Power control script not found at %s", powerControlScript)
	}

	// スクリプトを実行 (例: /usr/local/bin/power_control.sh start gpu_server)
	cmd := exec.Command(powerControlScript, action, targetName)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errorMsg := fmt.Sprintf("Script failed: %v. Stderr: %s", err, stderr.String())
		return stderr.String(), fmt.Errorf(errorMsg)
	}

	return stdout.String(), nil
}

// LoadMonitorTargets は list.txt ファイルを読み込み、監視対象のリストを返します。
// ファイル形式: セクションヘッダー (# hosts, # containers) と Name:Value
func LoadMonitorTargets() ([]MonitorTarget, error) {
	file, err := os.Open(MonitorListFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open monitor list file: %w", err)
	}
	defer file.Close()

	var targets []MonitorTarget
	scanner := bufio.NewScanner(file)
	currentType := "" // "host" or "container"

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// セクションヘッダーを先にチェックし、処理を継続
		if strings.HasPrefix(line, "# hosts") {
			currentType = "host"
			continue // 設定したら次の行へ
		} else if strings.HasPrefix(line, "# containers") {
			currentType = "container"
			continue // 設定したら次の行へ
		}

		// 通常のコメント（# で始まり、セクションヘッダーではないもの）や空行をスキップ
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 値の解析 (Name:Value 形式を想定)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue // 無効な行はスキップ
		}

		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if name == "" || value == "" || currentType == "" {
			continue
		}

		newTarget := MonitorTarget{
			Name: name,
			Type: currentType,
		}

		if currentType == "host" {
			// ホストの場合: Name:IP (ポートはSSHの22をデフォルトとする)
			newTarget.Host = value
			newTarget.Port = "22"
		} else if currentType == "container" {
			// コンテナの場合: Name:IP:Port の形式を解析
			ipPortParts := strings.SplitN(value, ":", 2)
			if len(ipPortParts) != 2 {
				// IP:Port 形式でない場合はスキップ
				continue
			}
			newTarget.Host = strings.TrimSpace(ipPortParts[0])
			newTarget.Port = strings.TrimSpace(ipPortParts[1])

			// ポート番号が数値であることを確認
			if _, err := strconv.Atoi(newTarget.Port); err != nil {
				continue
			}
		} else {
			continue
		}

		targets = append(targets, newTarget)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading monitor list file: %w", err)
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no valid monitoring targets found in %s", MonitorListFile)
	}

	return targets, nil
}

// CheckServiceStatus は、指定されたホストとポートへのTCP接続を試み、死活確認を行います。
func CheckServiceStatus(host, port string) string {
	address := fmt.Sprintf("%s:%s", host, port)

	// TCPポートチェック (タイムアウト: 2秒)
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return "Stopped/Unreachable"
	}
	conn.Close()
	return "Running"
}

// GetAllTargetsStatus は、監視対象リストを読み込み、それぞれの死活確認結果を返します。
func GetAllTargetsStatus() ([]TargetStatus, error) {
	targets, err := LoadMonitorTargets()
	if err != nil {
		return nil, err
	}

	var results []TargetStatus
	for _, target := range targets {
		status := CheckServiceStatus(target.Host, target.Port)

		results = append(results, TargetStatus{
			Type:  target.Type,
			Name: target.Name,
			HostPort: fmt.Sprintf("%s:%s", target.Host, target.Port),
			Status: status,
		})
	}

	return results, nil
}