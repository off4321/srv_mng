package service

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLiteドライバ
)

// 型 START===========================================================START

// MonitorTarget は monitor_targets テーブルから読み込まれる設定の構造体です。
// power_control.sh の実行に必要な全情報を含みます。
type MonitorTarget struct {
	Name        string `json:"name"`         // DB column: name
	Type        string `json:"type"`         // DB column: type ("host" or "container")
	HostIP      string `json:"host_ip"`      // DB column: host_ip
	Port        string `json:"port"`         // DB column: port (死活確認用、SSHポートやHTTPポートなど)
	MacAddress  string `json:"mac_address"`  // DB column: mac_address (WOL用, hostのみ使用)
	SSHUser     string `json:"ssh_user"`     // DB column: ssh_user (SSH Shutdown用, hostのみ使用)
	SSHPass     string `json:"ssh_pass"`     // DB column: ssh_pass (SSH Shutdown用, hostのみ使用)
	BroadcastIP string `json:"broadcast_ip"` // DB column: broadcast_ip (WOL用, hostのみ使用)
}

// TargetStatus は、APIエンドポイントで返す監視対象のステータス構造体です。
type TargetStatus struct {
	Type     string `json:"type"`      // "host", "container"
	Name     string `json:"name"`      // ターゲット名
	HostPort string `json:"host_port"` // IP:Port
	Status   string `json:"status"`    // "Running", "Stopped/Unreachable", "Unknown"
}

// 型 END===========================================================END

// DB系 START===========================================================START

// DB接続
var db *sql.DB

// InitDB はデータベース接続を初期化します。
// DSN (Data Source Name) は SQLite ファイルのパスを想定しています。
func InitDB(dsn string) error {
	var err error
	// sqlite3 ドライバを使用
	db, err = sql.Open("sqlite3", dsn)
	if err != nil {
		log.Printf("[ERROR] Failed to open database: %v", err)
		return fmt.Errorf("error opening database (SQLite): %w", err)
	}

	// 接続確認 (SQLiteではファイルが存在すれば成功する)
	if err = db.Ping(); err != nil {
		log.Printf("[ERROR] Database ping failed: %v", err)
		return fmt.Errorf("error pinging database (SQLite): %w", err)
	}
	log.Printf("[INFO] Database connection initialized successfully (DSN: %s)", dsn)
	return nil
}

// CreateInitialTables は monitor_targets テーブルが存在しない場合に作成します。
func CreateInitialTables() error {
	if db == nil {
		return fmt.Errorf("database connection not initialized")
	}

	// テーブル作成クエリ
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS monitor_targets (
		name TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		host_ip TEXT NOT NULL,
		port TEXT NOT NULL,
		mac_address TEXT,
		ssh_user TEXT,
		ssh_pass TEXT,
		broadcast_ip TEXT
	);
	`
	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create monitor_targets table: %w", err)
	}

	// 初期データの挿入 (ダミーデータ。パスワードは安全のため空欄にしています)
	insertDataSQL := `
	INSERT OR IGNORE INTO monitor_targets 
	(name, type, host_ip, port, mac_address, ssh_user, ssh_pass, broadcast_ip) 
	VALUES 
	('sample_name', 'sample_host', 'sample_ip', 'sample_port', 'sample_mac', 'sample_user', 'sample_pass', 'sample_broadcast_ip'),
	`
	_, err = db.Exec(insertDataSQL)
	if err != nil {
		return fmt.Errorf("failed to insert initial data: %w", err)
	}

	return nil
}

// SaveMonitorTarget は、ターゲット設定をDBに保存（または既存のものを更新）します。
func SaveMonitorTarget(config *MonitorTarget) error {
	if db == nil {
		log.Printf("[ERROR] database connection not initialized")
		return fmt.Errorf("database connection not initialized")
	}
	if config.Name == "" || config.HostIP == "" || config.Port == "" || config.Type == "" {
		log.Printf("[ERROR] name, host_ip, port, and type are required fields Name:'%s', HostIP:'%s', Port:'%s', Type:'%s'", config.Name, config.HostIP, config.Port, config.Type)
		return fmt.Errorf("name, host_ip, port, and type are required fields")
	}

	// INSERT OR REPLACE は、PRIMARY KEY(name)が衝突した場合に既存の行を削除し、新しい行を挿入します。
	query := `
	INSERT OR REPLACE INTO monitor_targets 
	(name, type, host_ip, port, mac_address, ssh_user, ssh_pass, broadcast_ip) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(query,
		config.Name,
		config.Type,
		config.HostIP,
		config.Port,
		config.MacAddress,
		config.SSHUser,
		config.SSHPass,
		config.BroadcastIP,
	)

	if err != nil {
		log.Printf("[ERROR] Failed to save target '%s': %v", config.Name, err)
		return fmt.Errorf("failed to save target config to database: %w", err)
	}

	log.Printf("[INFO] Target saved/updated: %s (%s)", config.Name, config.HostIP)
	return nil
}

// GetTargetConfig は指定されたターゲットの設定をDBから取得
func GetTargetConfig(targetName string) (*MonitorTarget, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	// DBから全フィールドを選択
	query := "SELECT name, type, host_ip, port, mac_address, ssh_user, ssh_pass, broadcast_ip FROM monitor_targets WHERE name = ?"

	config := &MonitorTarget{}
	err := db.QueryRow(query, targetName).Scan(
		&config.Name,
		&config.Type,
		&config.HostIP,
		&config.Port,
		&config.MacAddress,
		&config.SSHUser,
		&config.SSHPass,
		&config.BroadcastIP,
	)

	if err == sql.ErrNoRows {
		log.Printf("[ERROR] target '%s' not found in database", targetName)
		return nil, fmt.Errorf("target '%s' not found in database", targetName)
	}
	if err != nil {
		log.Printf("[ERROR] database query error: %w", err)
		return nil, fmt.Errorf("database query error: %w", err)
	}
	log.Printf("[SUCCESS] GetTarget query succeed")
	return config, nil
}

// GetAllTargetsFromDB はすべてのターゲットの設定をDBから取得します。
func GetAllTargetsFromDB() ([]MonitorTarget, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	query := "SELECT name, type, host_ip, port, mac_address, ssh_user, ssh_pass, broadcast_ip FROM monitor_targets"
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("[ERROR] database query error: %w", err)
		return nil, fmt.Errorf("database query error: %w", err)
	}
	defer rows.Close()

	var targets []MonitorTarget
	for rows.Next() {
		config := MonitorTarget{}
		err := rows.Scan(
			&config.Name,
			&config.Type,
			&config.HostIP,
			&config.Port,
			&config.MacAddress,
			&config.SSHUser,
			&config.SSHPass,
			&config.BroadcastIP,
		)
		if err != nil {
			// DBスキーマと構造体が一致しない、またはデータエラー
			log.Printf("[ERROR] error scanning row from database: %w", err)
			return nil, fmt.Errorf("error scanning row from database: %w", err)
		}
		targets = append(targets, config)
	}

	if err = rows.Err(); err != nil {
		log.Printf("[ERROR] error iterating over database rows: %w", err)
		return nil, fmt.Errorf("error iterating over database rows: %w", err)
	}

	if len(targets) == 0 {
		// データがない場合もエラーとして扱う
		log.Printf("[ERROR] no monitoring targets found in database")
		return nil, fmt.Errorf("no monitoring targets found in database")
	}
	log.Printf("[INFO] GetALLTarget query succeed")
	return targets, nil
}

// DB系 END===========================================================END

// 電源操作 START===========================================================START

// ExecutePowerScript は、すべての電源ON/OFF操作を行うサービスロジックです。
// Goコードで直接WOLパケットを送信し、エージェント経由でシャットダウンを実行します。
func ExecutePowerScript(action string, config *MonitorTarget) (string, error) {
	log.Printf("[INFO] Executing power action '%s' for target '%s'...", action, config.Name)

	switch action {
	case "start":
		// WOLパケットを直接送信
		return sendWOLPacket(config.MacAddress, config.BroadcastIP, config.Name)
		
	case "stop":
		// エージェント経由でシャットダウン
		return shutdownViaAgent(config)
	default:
		return "", fmt.Errorf("unsupported action: %s", action)
	}
}

// sendWOLPacket は、指定されたMACアドレスにWOLパケットを送信します。
func sendWOLPacket(macAddress string, broadcastIP string, Name string) (string, error) {
	// MACアドレスをバイト配列に変換
	macBytes, err := parseMAC(macAddress)
	if err != nil {
		log.Printf("[ERROR] invalid MAC address: %v", err)
		return "", fmt.Errorf("invalid MAC address: %v", err)
	}

	// WOLパケットを構築
	pkt := buildWOLPacket(macBytes)

	// ブロードキャストアドレスに送信
	broadcastIP = broadcastIP + ":9"
	conn, err := net.Dial("udp", broadcastIP)
	if err != nil {
		log.Printf("[ERROR] failed to create UDP connection: %v", err)
		return "", fmt.Errorf("failed to create UDP connection: %v", err)
	}
	defer conn.Close()

	// WOLパケットを送信
	_, err = conn.Write(pkt)
	if err != nil {
		log.Printf("[ERROR] failed to send WOL packet: %v", err)
		return "", fmt.Errorf("failed to send WOL packet: %v", err)
	}
	log.Printf("[INFO] send WOL packet succeessfuly: %s", Name)
	return "WOL packet sent successfully", nil
}

// parseMAC はMACアドレス文字列をバイト配列に変換します。
func parseMAC(mac string) ([]byte, error) {
	// MACアドレスの形式を正規化
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")

	// 12文字の16進数文字列に変換
	if len(mac) != 12 {
		return nil, fmt.Errorf("invalid MAC address length")
	}

	// バイト配列に変換
	macBytes, err := hex.DecodeString(mac)
	if err != nil {
		return nil, fmt.Errorf("invalid MAC address format: %v", err)
	}

	return macBytes, nil
}

// buildWOLPacket は、WOLパケットを構築します。
func buildWOLPacket(macBytes []byte) []byte {
	// WOLパケットは、6バイトのパケットヘッダ + 16回繰り返されたMACアドレス
	pkt := make([]byte, 0, 102)

	// 6バイトのパケットヘッダ（すべて0xFF）
	for i := 0; i < 6; i++ {
		pkt = append(pkt, 0xFF)
	}

	// MACアドレスを16回繰り返す
	for i := 0; i < 16; i++ {
		pkt = append(pkt, macBytes...)
	}

	return pkt
}

// shutdownViaAgent は、エージェント経由でシャットダウンを実行します。
func shutdownViaAgent(config *MonitorTarget) (string, error) {
	// エージェントのAPIエンドポイントを構築
	url := fmt.Sprintf("http://%s:%s/shutdown", config.HostIP, config.Port)

	// パスワードを含むJSONボディを構築
	body := map[string]string{
		"password": config.SSHPass, // ここでパスワードを設定
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// HTTPリクエストを送信
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to shutdown via agent: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("agent shutdown returned status code: %d", resp.StatusCode)
	}

	return "Shutdown command sent via agent successfully", nil
}

// 電源操作 END===========================================================END

// 死活確認 START===========================================================START

// CheckServiceStatus は、指定されたホストとポートへのTCP接続を試み、死活確認を行います。
// エージェント経由で死活確認を実行します。
func CheckServiceStatus(host, port string) string {
	// エージェントのAPIエンドポイントを構築
	// 例: http://<host_ip>:<agent_port>/status
	url := fmt.Sprintf("http://%s:%s/status", host, port)

	// HTTPリクエストを送信
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("[INFO] Health check: %s is Down", url)
		return "Stopped/Unreachable"
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Printf("[INFO] Health check: %s is Up", url)
		return "Running"
	}

	log.Printf("[INFO] Health check: %s is Down (status code: %d)", url, resp.StatusCode)
	return "Stopped/Unreachable"
}

// GetAllTargetsStatus は、DBからターゲットリストを読み込み、それぞれの死活確認結果を返します。
func GetAllTargetsStatus() ([]TargetStatus, error) {
	// ターゲットリストをDBから取得
	targets, err := GetAllTargetsFromDB()
	if err != nil {
		return nil, err
	}

	var results []TargetStatus
	for _, target := range targets {
		// ホストIPとポートを使って死活確認
		status := CheckServiceStatus(target.HostIP, target.Port)

		results = append(results, TargetStatus{
			Type:     target.Type,
			Name:     target.Name,
			HostPort: fmt.Sprintf("%s:%s", target.HostIP, target.Port),
			Status:   status,
		})
	}

	return results, nil
}

// 死活確認 END===========================================================END
