#!/bin/bash

# ==============================================================================
# Power API テストスクリプト
#
# 使い方:
# ./test_power_api.sh <port> <name> <type> <host_ip> <port> <mac_address> <ssh_user> <ssh_pass> <broadcast_ip>
#
# 例:
# ./test_power_api.sh 5001 gpu_server host 172.16.100.201 22 9c:6b:00:7b:67:24 h-jun hjnk1201 172.16.100.255
# ==============================================================================

# 引数の取得
API_PORT=$1
NAME=$2
TYPE=$3
HOST_IP=$4
PORT=$5
MAC_ADDR=$6
SSH_USER=$7
SSH_PASS=$8
BROADCAST_IP=$9

API_URL="http://localhost:${API_PORT}"

# 引数のバリデーション
if [ $# -ne 9 ]; then
    echo "ERROR: 9つの引数が必要です。"
    echo "Usage: $0 <port> <name> <type> <host_ip> <port> <mac_address> <ssh_user> <ssh_pass> <broadcast_ip>"
    exit 1
fi

# ------------------------------------------------------------------------------
# ターゲット登録関数
# ------------------------------------------------------------------------------
register_target() {
    echo "--- 1. ターゲットの登録: ${NAME} ---"
    
    # JSONペイロードを組み立て
    PAYLOAD=$(cat <<EOF
{
    "name": "${NAME}",
    "type": "${TYPE}",
    "host_ip": "${HOST_IP}",
    "port": "${PORT}",
    "mac_address": "${MAC_ADDR}",
    "ssh_user": "${SSH_USER}",
    "ssh_pass": "${SSH_PASS}",
    "broadcast_ip": "${BROADCAST_IP}"
}
EOF
)

    curl -s -X POST "${API_URL}/targets/register" \
        -H "Content-Type: application/json" \
        -d "${PAYLOAD}"
    echo ""
}

# ------------------------------------------------------------------------------
# ステータス確認関数
# ------------------------------------------------------------------------------
check_status() {
    local accept_header=$1
    echo "--- ステータス確認 (Accept: ${accept_header}) ---"
    curl -s -X GET "${API_URL}/status" \
        -H "Accept: ${accept_header}"
    echo ""
}

# ------------------------------------------------------------------------------
# 電源制御関数
# ------------------------------------------------------------------------------
power_action() {
    local action=$1
    echo "--- ${NAME}: 電源${action} (WOL/SSH) ---"
    
    curl -s -X POST "${API_URL}/power/${action}" \
        -H "Content-Type: application/json" \
        -d '{"target": "'"${NAME}"'"}' | json_pp # json_ppはインストールされていれば整形表示
    echo ""
}

# ------------------------------------------------------------------------------
# メイン処理の実行
# ------------------------------------------------------------------------------

echo "--- Power API テスト開始 ---"
echo "ターゲット: ${NAME}, IP: ${HOST_IP}, MAC: ${MAC_ADDR}"
echo "API URL: ${API_URL}"
echo ""

# 1. ターゲット登録
register_target

# 2. ステータス確認 (プレーンテキスト)
check_status "text/plain"

# 3. 電源投入 (start)
power_action "start"

# 4. ステータス確認 (JSON)
check_status "application/json"

# 5. 電源断 (stop)
power_action "stop"

# 6. ステータス確認 (プレーンテキスト)
check_status "text/plain"

echo "--- Power API テスト完了 ---"