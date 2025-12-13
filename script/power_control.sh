#!/bin/bash

# ────────────────────────
#  --- 設定 ---
# ────────────────────────
# ホスト名、IPアドレス、MACアドレス、SSHユーザーを連想配列で定義
declare -A HOSTS
HOSTS["gpu_server,ip"]="172.16.100.201"
HOSTS["gpu_server,mac"]="9c:6b:00:7b:67:24"
HOSTS["minipc3,ip"]="172.16.100.103"
HOSTS["minipc3,mac"]="c8:ff:bf:04:0e:0d"
HOSTS["minipc1,ip"]="172.16.100.100"
HOSTS["minipc1,mac"]="68:1d:ef:49:c1:01"
SSH_USER="h-jun"          # 各ホストにSSH接続する際のユーザー名
SSH_PASS="hjnk1201"
BROADCAST="172.16.100.255"

# ────────────────────────
#  --- 関数定義 ---
# ────────────────────────
usage() {
  echo "Usage: $0 {start|stop} {gpu_server|minipc3|minipc1|all}"
  echo "  - start: Wake On LANで電源を起動します"
  echo "  - stop:  SSH経由でシャットダウンします"
  exit 1
}

log_error() {
  # 赤字でエラーを表示
  echo -e "\e[1;31m[*]\e[0m $1"
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
  local host_name=$1
  local mac_addr=${HOSTS["$host_name,mac"]}

  if [ -z "$mac_addr" ]; then
    log_error "エラー: ホスト '$host_name' のMACアドレスが定義されていません。"
    return
  fi

  log_info "[$host_name] を起動します (MAC: $mac_addr)..."

  # ホストが到達できるかを確認 (IPが無ければスキップ)
  local ip_addr=${HOSTS["$host_name,ip"]}
  if [ -n "$ip_addr" ]; then
    if ping_host "$ip_addr"; then
      log_success "[$host_name] はすでに到達可能です。Wake‑On‑LAN はスキップします。"
      return
    fi
  fi

  # ここから Wake‑On‑LAN 送信
  if ! wakeonlan -i "$BROADCAST" "$mac_addr"; then
    log_error "[$host_name] の Wake‑On‑LAN 送信に失敗しました。"
  else
    log_success "[$host_name] の Wake‑On‑LAN 送信が完了しました。"
  fi
}

stop_host() {
  local host_name=$1
  local ip_addr=${HOSTS["$host_name,ip"]}

  if [ -z "$ip_addr" ]; then
    log_error "エラー: ホスト '$host_name' のIPアドレスが定義されていません。"
    return
  fi

  log_info "[$host_name] ($ip_addr) をシャットダウンします..."

  # 到達確認
  if ! ping_host "$ip_addr"; then
    log_warning "[$host_name] は到達できません。シャットダウンはスキップします。"
    return
  fi

  # expect を使って自動ログイン ＆ パスワード入力
  expect <<EOF
set timeout 10
spawn ssh ${SSH_USER}@${ip_addr} "sudo shutdown -h now"
expect {
  "(yes/no" {
     send "yes\r"
     exp_continue
  }
  "password:" {
     send "$SSH_PASS\r"
     exp_continue
  }
}
EOF

  ret=$?  # expect の終了ステータスを取得

  if [ $ret -eq 0 ]; then
    log_info "[$host_name] へのシャットダウンコマンド投入完了"
  else
    log_error "[$host_name] はシャットダウン投入できませんでした。"
  fi
}

# ────────────────────────
#  --- メイン処理 ---
# ────────────────────────
# 引数のチェック
if [ "$#" -ne 2 ]; then
  usage
fi

ACTION=$1
TARGET=$2
HOST_LIST=("gpu_server" "minipc3" "minipc1")

case $ACTION in
  start)
    if [ "$TARGET" == "all" ]; then
      for host in "${HOST_LIST[@]}"; do
        start_host "$host"
      done
    else
      start_host "$TARGET"
    fi
    ;;
  stop)
    if [ "$TARGET" == "all" ]; then
      for host in "${HOST_LIST[@]}"; do
        stop_host "$host"
      done
    else
      stop_host "$TARGET"
    fi
    ;;
  *)
    usage
    ;;
esac

log_success "処理が完了しました。"

