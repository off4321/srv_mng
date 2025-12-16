package service

import (
	"bytes"
	"database/sql" 
	"fmt"
	"net"
	"os" 
	"os/exec"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLiteドライバ
)

// ホストOSの電源制御スクリプトのパス
const powerControlScript = "/usr/local/bin/power_control.sh"

// DB接続
var db *sql.DB

// InitDB はデータベース接続を初期化します。
// DSN (Data Source Name) は SQLite ファイルのパスを想定しています。
func InitDB(dsn string) error {
	var err error
	// sqlite3 ドライバを使用
	db, err = sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("error opening database (SQLite): %w", err)
	}

	// 接続確認 (SQLiteではファイルが存在すれば成功する)
	if err = db.Ping(); err != nil {
		return fmt.Errorf("error pinging database (SQLite): %w", err)
	}
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
		return fmt.Errorf("database connection not initialized")
	}
	if config.Name == "" || config.HostIP == "" || config.Port == "" || config.Type == "" {
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
		return fmt.Errorf("failed to save target config to database: %w", err)
	}

	return nil
}


// MonitorTarget は monitor_targets テーブルから読み込まれる設定の構造体です。
// power_control.sh の実行に必要な全情報を含みます。
type MonitorTarget struct {
	Name        string `json:"name"`        // DB column: name
	Type        string `json:"type"`        // DB column: type ("host" or "container")
	HostIP      string `json:"host_ip"`     // DB column: host_ip
	Port        string `json:"port"`        // DB column: port (死活確認用、SSHポートやHTTPポートなど)
	MacAddress  string `json:"mac_address"` // DB column: mac_address (WOL用, hostのみ使用)
	SSHUser     string `json:"ssh_user"`    // DB column: ssh_user (SSH Shutdown用, hostのみ使用)
	SSHPass     string `json:"ssh_pass"`    // DB column: ssh_pass (SSH Shutdown用, hostのみ使用)
	BroadcastIP string `json:"broadcast_ip"`// DB column: broadcast_ip (WOL用, hostのみ使用)
}

// TargetStatus は、APIエンドポイントで返す監視対象のステータス構造体です。
type TargetStatus struct {
	Type     string `json:"type"`     // "host", "container"
	Name     string `json:"name"`     // ターゲット名
	HostPort string `json:"host_port"`// IP:Port
	Status   string `json:"status"`   // "Running", "Stopped/Unreachable", "Unknown"
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
		return nil, fmt.Errorf("target '%s' not found in database", targetName)
	}
	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}
    
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
			return nil, fmt.Errorf("error scanning row from database: %w", err)
		}
		targets = append(targets, config)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over database rows: %w", err)
	}

	if len(targets) == 0 {
		// データがない場合もエラーとして扱う
		return nil, fmt.Errorf("no monitoring targets found in database")
	}

	return targets, nil
}

// ExecutePowerScript は、すべての電源ON/OFF操作を外部スクリプトに委譲するサービスロジックです。
func ExecutePowerScript(action string, config *MonitorTarget) (string, error) {
	if _, err := os.Stat(powerControlScript); os.IsNotExist(err) {
		return "", fmt.Errorf("error: Power control script not found at %s", powerControlScript)
	}

	// power_control.sh の引数: 
	// $1: action, $2: targetName, $3: IP_ADDR, $4: MAC_ADDR, $5: SSH_USER, $6: SSH_PASS, $7: BROADCAST_IP
	args := []string{
		action,
		config.Name,
		config.HostIP,
		config.MacAddress,
		config.SSHUser,
		config.SSHPass,
		config.BroadcastIP,
	}

	cmd := exec.Command(powerControlScript, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errorMsg := fmt.Sprintf("Script failed: %v. Stderr: %s", err, stderr.String())
		// stderrをAPI応答のScriptOutputとして返すため、ここではstderr.String()も返す
		return stderr.String(), fmt.Errorf(errorMsg)
	}

	return stdout.String(), nil
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
			Type: target.Type,
			Name: target.Name,
			HostPort: fmt.Sprintf("%s:%s", target.HostIP, target.Port),
			Status: status,
		})
	}

	return results, nil
}