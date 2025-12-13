#!/bin/bash

# ────────────────────────
# --- 実行パラメータの取得 ---
# ────────────────────────
# このスクリプトはGo APIから以下の7つの引数を受け取ることを想定しています:
# $1: ACTION (start または stop)
# $2: TARGET_NAME (gpu_serverなど)
# $3: IP_ADDR
# $4: MAC_ADDR
# $5: SSH_USER
# $6: SSH_PASS
# $7: BROADCAST_IP

ACTION=$1
TARGET_NAME=$2
IP_ADDR=$3
MAC_ADDR=$4
SSH_USER=$5
SSH_PASS=$6
BROADCAST_IP=$7

# ────────────────────────
# --- 関数定義 ---
# ────────────────────────
usage() {
  echo "Usage: $0 {start|stop} {host_name} {ip} {mac} {user} {pass} {broadcast_ip}"
  exit 1
}

log_error() {
  # 赤字でエラーを表示
  echo -e "\e[1;31m[*]\e[0m $1" >&2
}

log_warning() {
  # 黄色で警告を表示
  echo -e "\e[1;33m[*]\e[0m $1"
}

log_info() {
  # 青字で情報を表示
  echo -e "\e[1;34m[*]\e[0m $1"
}

log_success() {
  # 緑字で成功を表示
  echo -e "\e[1;32m[*]\e[0m $1"
}

# ホスト到達確認（ping）
ping_host() {
  local ip="$1"
  # 1回 ping で応答確認。応答が無い場合は false
  if ping -c 1 -W 1 "$ip" >/dev/null 2>&1; then
    return 0
  else
    return 1
  fi
}

start_host() {
  # グローバル変数から設定を取得
  local host_name=$TARGET_NAME
  local mac_addr=$MAC_ADDR
  local ip_addr=$IP_ADDR
  local broadcast=$BROADCAST_IP
  
  if [ -z "$mac_addr" ]; then
    log_error "エラー: ホスト '$host_name' のMACアドレスが設定されていません。"
    return 1
  fi

  log_info "[$host_name] を起動します (MAC: $mac_addr)..."

  # ホストが到達できるかを確認
  if [ -n "$ip_addr" ]; then
    if ping_host "$ip_addr"; then
      log_success "[$host_name] はすでに到達可能です。Wake‑On‑LAN はスキップします。"
      return 0
    fi
  fi

  # ここから Wake‑On‑LAN 送信
  if ! wakeonlan -i "$broadcast" "$mac_addr"; then
    log_error "[$host_name] の Wake‑On‑LAN 送信に失敗しました。"
    return 1
  else
    log_success "[$host_name] の Wake‑On‑LAN 送信が完了しました。"
    return 0
  fi
}

stop_host() {
  # グローバル変数から設定を取得
  local host_name=$TARGET_NAME
  local ip_addr=$IP_ADDR
  local ssh_user=$SSH_USER
  local ssh_pass=$SSH_PASS

  if [ -z "$ip_addr" ]; then
    log_error "エラー: ホスト '$host_name' のIPアドレスが設定されていません。"
    return 1
  fi

  log_info "[$host_name] ($ip_addr) をシャットダウンします..."

  # 到達確認
  if ! ping_host "$ip_addr"; then
    log_warning "[$host_name] は到達できません。シャットダウンはスキップします。"
    return 0 # 停止状態であれば成功とみなす
  fi

  # expect を使って自動ログイン ＆ パスワード入力
  expect <<EOF
set timeout 10
spawn ssh ${ssh_user}@${ip_addr} "sudo shutdown -h now"
expect {
  "(yes/no" {
    send "yes\r"
    exp_continue
  }
  "password:" {
    send "$ssh_pass\r"
    exp_continue
  }
  timeout {
    exit 1
  }
}
EOF

  ret=$?  # expect の終了ステータスを取得

  if [ $ret -eq 0 ]; then
    log_info "[$host_name] へのシャットダウンコマンド投入完了"
    return 0
  else
    log_error "[$host_name] はシャットダウン投入できませんでした。(Expect終了コード: $ret)"
    return 1
  fi
}

# ────────────────────────
# --- メイン処理 ---
# ────────────────────────
# 引数のチェック (Go APIからの引数は7個を想定: action, name, ip, mac, user, pass, broadcast)
if [ "$#" -ne 7 ]; then
	log_error "内部エラー: Go APIから予期しない数の引数が渡されました ($#個)。(期待される引数は7個です: action name ip mac user pass broadcast)"
	exit 1
fi

case $ACTION in
  start)
    start_host
    ;;
  stop)
    stop_host
    ;;
  *)
    log_error "不正なアクション: $ACTION"
    exit 1
    ;;
esac

# 処理完了メッセージはGo API側で出力するためここでは省略
exit $?