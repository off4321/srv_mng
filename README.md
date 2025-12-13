# サーバ管理用資材

## 機能
DBに登録したサーバに対して下記を実施します

### 死活監視
APIを叩いたときにサーバに対してsshでの死活監視を行います
**前提としてテーブルに各情報を登録する必要があります**

#### API例
```bash
# jsonの表示
curl -X GET http://localhost:5001/status

[{"type":"host","name":"server","host_port":"172.16.0.xxx:22","status":"Stopped/Unreachable"}]


# ASCIIの表示
curl -X GET http://localhost:5001/status -H "Accept: text/plain"

SHOW SERVERS AND CONTAINERS STATUS
TYPE     TARGET         HOST:PORT          STATUS
------------------------------------------------------------------------
host        gpu_server               172.16.100.201:22        STOPPED/UNREACHABLE

```

### WOLによる電源起動
WOLを使い、遠隔サーバに対して電源ONを行います。
**前提としてテーブルに各情報を登録する必要があります**
**WOLを受けるサーバにWOLを有効にする必要があります**

#### API例
```bash
curl -X POST http://localhost:5001/power/start \
     -H "Content-Type: application/json" \
     -d '{"target": "server"}'
```

### ssh -> shutdownによる電源断
ssh -> shutdownを使い、遠隔サーバに対して電源OFFを行います。
**前提としてテーブルに各情報を登録する必要があります**
**現時点(2025/12/13)では鍵によるパスワードなしの認証には対応していません**

#### API例
```bash
curl -X POST http://localhost:5001/power/stop \
     -H "Content-Type: application/json" \
     -d '{"target": "server"}'
```

### DB登録
DBに必要情報を登録します。


#### API例
```bash
curl -X POST http://localhost:5001/targets/register \
     -H "Content-Type: application/json" \
     -d '{
         "name": "server",
         "type": "host",
         "host_ip": "172.16.0.xxx",
         "port": "22",
         "mac_address": "01:23:34:56:78:9a",
         "ssh_user": "user",
         "ssh_pass": "password",  
         "broadcast_ip": "172.16.0.255"
     }'
```