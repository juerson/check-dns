## ECH-Workers 核心


#### 编译 Go 程序

```cmd
go build -o release/ech-workers.exe .
# 或者
go build -ldflags "-s -w" -o release/ech-workers.exe .
```

## 配置说明

配置文件保存在：
- **Windows**: `%APPDATA%\ECHWorkersClient\config.json`
- **macOS**: `~/Library/Application Support/ECHWorkersClient/config.json`
- **Linux**: `~/.config/ECHWorkersClient/config.json`

注意：其他版本自己编译。


### CLI参数说明

#### 必需参数

| 参数 | 说明 | 示例 |
|------|------|------|
| `-f` | 服务端地址（必需） | `-f your-worker.workers.dev:443` |

#### 可选参数

| 参数 | 默认值 | 说明 | 示例 |
|------|--------|------|------|
| `-l` | `127.0.0.1:30000` | 本地监听地址 | `-l 0.0.0.0:30001` |
| `-token` | 空 | 身份验证令牌 | `-token your-token-here` |
| `-ip` | 空 | 指定服务端 IP（绕过 DNS） | `-ip 1.2.3.4` |
| `-dns` | `dns.alidns.com/dns-query` | ECH 查询 DoH 服务器 | `-dns dns.alidns.com/dns-query` |
| `-ech` | `cloudflare-ech.com` | ECH 查询域名 | `-ech cloudflare-ech.com` |
| `-routing` | `global` | 分流模式 | `-routing bypass_cn` |

#### 分流模式说明

| 模式 | 值 | 说明 |
|------|-----|------|
| **全局代理** | `global` | 所有流量都走代理（默认模式） |
| **跳过中国大陆** | `bypass_cn` | 中国 IP 直连，其他走代理 |
| **直连模式** | `none` | 所有流量直连，不设置代理 |

> **注意**: 
> - 使用 `bypass_cn` 模式时，程序会自动下载中国 IP 列表（IPv4/IPv6）
> - 如果 IP 列表文件不存在或为空，程序会自动从 GitHub 下载
> - IP 列表文件保存在程序目录：`chn_ip.txt`（IPv4）和 `chn_ip_v6.txt`（IPv6）

### 使用示例

#### 基本用法

```bash
# Windows
ech-workers.exe -f your-worker.workers.dev:443

# macOS / Linux
./ech-workers -f your-worker.workers.dev:443
```

#### 指定监听地址

```bash
# 监听所有网络接口（适合软路由）
./ech-workers -f your-worker.workers.dev:443 -l 0.0.0.0:30001

# 仅监听本地（默认）
./ech-workers -f your-worker.workers.dev:443 -l 127.0.0.1:30001
```

#### 使用分流模式

```bash
# 全局代理模式（默认）
./ech-workers -f your-worker.workers.dev:443 -routing global

# 跳过中国大陆模式（自动下载 IP 列表）
./ech-workers -f your-worker.workers.dev:443 -routing bypass_cn

# 直连模式
./ech-workers -f your-worker.workers.dev:443 -routing none
```

#### 完整参数示例

```bash
./ech-workers \
  -f your-worker.workers.dev:443 \
  -l 0.0.0.0:30001 \
  -token your-token \
  -ip saas.sin.fan \
  -dns dns.alidns.com/dns-query \
  -ech cloudflare-ech.com \
  -routing bypass_cn
```

#### 查看帮助

```bash
./ech-workers -h
# 或
./ech-workers --help
```

### 后台运行

#### Linux/macOS

**使用 nohup:**
```bash
nohup ./ech-workers -f your-worker.workers.dev:443 -l 127.0.0.1:30001 > ech-workers.log 2>&1 &
```

**使用 screen:**
```bash
screen -S ech-workers
./ech-workers -f your-worker.workers.dev:443 -l 127.0.0.1:30001
# 按 Ctrl+A 然后 D 分离会话
```

**使用 systemd (推荐):**

创建服务文件 `/etc/systemd/system/ech-workers.service`:
```ini
[Unit]
Description=ECH Workers Proxy Client
After=network.target

[Service]
Type=simple
User=your-username
WorkingDirectory=/path/to/ech-workers
ExecStart=/path/to/ech-workers -f your-worker.workers.dev:443 -l 127.0.0.1:30001 -routing bypass_cn
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

启用并启动服务:
```bash
sudo systemctl daemon-reload
sudo systemctl enable ech-workers
sudo systemctl start ech-workers
sudo systemctl status ech-workers
```

查看日志:
```bash
sudo journalctl -u ech-workers -f
```

#### Windows

**使用 PowerShell:**
```powershell
Start-Process -FilePath "ech-workers.exe" `
  -ArgumentList "-f", "your-worker.workers.dev:443", "-l", "127.0.0.1:30001" `
  -WindowStyle Hidden
```

**使用任务计划程序:**
1. 打开"任务计划程序"
2. 创建基本任务
3. 设置触发器为"计算机启动时"
4. 操作选择启动程序：`ech-workers.exe`
5. 添加参数：`-f your-worker.workers.dev:443 -l 127.0.0.1:30001`

### 配置代理客户端

启动代理后，配置应用程序使用 SOCKS5 代理：

- **代理地址**: `127.0.0.1:30001`（或你指定的监听地址）
- **代理类型**: SOCKS5
- **端口**: 30001（或你指定的端口）

#### 浏览器配置

**Chrome/Edge:**
```bash
# Linux/macOS
google-chrome --proxy-server="socks5://127.0.0.1:30001"

# Windows
chrome.exe --proxy-server="socks5://127.0.0.1:30001"
```

**Firefox:**
- 设置 → 网络设置 → 手动代理配置
- SOCKS 主机: `127.0.0.1`
- 端口: `30001`
- SOCKS v5

#### 环境变量配置

**Linux/macOS:**
```bash
export ALL_PROXY=socks5://127.0.0.1:30001
export HTTP_PROXY=socks5://127.0.0.1:30001
export HTTPS_PROXY=socks5://127.0.0.1:30001
```

**Windows (PowerShell):**
```powershell
$env:ALL_PROXY="socks5://127.0.0.1:30001"
$env:HTTP_PROXY="socks5://127.0.0.1:30001"
$env:HTTPS_PROXY="socks5://127.0.0.1:30001"
```

### 日志输出

程序会在控制台输出运行日志，包括：
- 启动信息和 ECH 配置状态
- 分流模式加载状态
- IP 列表下载和加载信息
- 代理连接和错误信息

将输出重定向到文件：
```bash
./ech-workers -f your-worker.workers.dev:443 -l 127.0.0.1:30001 > ech-workers.log 2>&1
```
