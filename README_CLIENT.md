# Go Proxy Client - Protocol v2

这是一个Go语言实现的代理客户端，使用v2协议与JavaScript WebSocket服务端通信。

## 协议版本

- **Protocol Version**: v2
- **MAGIC**: 0xA5
- **VERSION**: 2

## Frame结构

```
字节0: MAGIC (0xA5)
字节1: VERSION (2)
字节2: CMD (AUTH=0x01, OPEN=0x02, DATA=0x03, CLOSE=0x04)
字节3: FLAGS (预留扩展)
字节4-7: payload_len (32位大端序)
字节8+: payload
```

## 命令类型

- `CMD_AUTH (0x01)`: 认证帧，payload为PSK token
- `CMD_OPEN (0x02)`: 打开连接帧，格式为 "host:port|firstPayload"
- `CMD_DATA (0x03)`: 数据传输帧
- `CMD_CLOSE (0x04)`: 关闭连接帧

## 通信流程

1. 建立WebSocket连接
2. 发送AUTH帧进行身份验证
3. 等待AUTH响应（成功或CLOSE）
4. 发送OPEN帧请求目标连接
5. 等待OPEN响应（成功或CLOSE）
6. 双向DATA帧数据传输
7. CLOSE帧结束连接

## 支持的代理模式

- **SOCKS5**: 标准SOCKS5代理协议
- **HTTP CONNECT**: HTTP CONNECT隧道代理
- **HTTP**: 普通HTTP代理

## 配置

通过环境变量配置：

- `SERVER_URL`: WebSocket服务器地址（默认: wss://example.com/ws）
- `AUTH_TOKEN`: 认证token（默认: your-secret-token）
- `LISTEN_ADDR`: 本地监听地址（默认: 127.0.0.1:1080）
- `PROXY_MODE`: 代理模式 - socks5/http/connect（默认: socks5）

## 编译运行

### 安装依赖

```bash
go mod download
```

### 编译

```bash
go build -o proxy-client client.go
```

### 运行

```bash
# SOCKS5 模式
export SERVER_URL="wss://your-server.com/ws"
export AUTH_TOKEN="your-secret-token"
export LISTEN_ADDR="127.0.0.1:1080"
export PROXY_MODE="socks5"
./proxy-client

# HTTP CONNECT 模式
export PROXY_MODE="connect"
export LISTEN_ADDR="127.0.0.1:8080"
./proxy-client

# HTTP 代理模式
export PROXY_MODE="http"
export LISTEN_ADDR="127.0.0.1:8080"
./proxy-client
```

## 协议升级预留空间

### FLAGS字段

当前FLAGS设置为0，预留用于未来扩展：

- 压缩标记
- 加密标记
- 分片标记
- 其他功能标记

### 版本升级到v3

升级到v3时只需：

1. 修改 `VERSION` 常量为 3
2. 在 `makeFrame()` 中添加新版本的编码逻辑
3. 在 `parseFrame()` 中添加版本兼容判断
4. 根据FLAGS字段添加新功能处理逻辑

示例：

```go
// 升级到v3
const VERSION byte = 3

// 在parseFrame中添加版本兼容
func parseFrame(buffer []byte) (cmd byte, flags byte, payload []byte, ok bool) {
    version := buffer[1]
    switch version {
    case 2:
        // v2 处理逻辑（保持向后兼容）
    case 3:
        // v3 新增功能处理
    default:
        return 0, 0, nil, false
    }
    // ...
}
```

## 错误处理

- **AUTH失败**: 服务端返回CLOSE帧，客户端立即断开
- **OPEN失败**: 服务端返回CLOSE帧，返回代理错误响应
- **连接错误**: 记录日志并关闭相关连接
- **帧解析错误**: 记录日志并终止连接

## 日志输出

客户端保持详细的日志输出便于调试：

- WebSocket连接状态
- 协议握手过程
- 帧收发信息
- 错误详情

## 架构设计

### 主要函数

- `makeFrame()`: 构造v2协议帧
- `parseFrame()`: 解析v2协议帧
- `handleTunnel()`: WebSocket通信核心逻辑
- `dialWebSocketWithECH()`: 建立WebSocket连接（支持ECH）
- `handleSOCKS5()`: SOCKS5代理处理
- `handleHTTPConnect()`: HTTP CONNECT处理
- `handleHTTP()`: HTTP代理处理

### 数据流

```
客户端应用 <--> 本地代理监听 <--> handleConnection
                                      |
                                      +--> 协议解析 (SOCKS5/HTTP)
                                      |
                                      +--> handleTunnel
                                            |
                                            +--> WebSocket (v2 frames)
                                            |
                                            +--> JavaScript服务端
```

## 测试

确保客户端能够：

1. 正常连接到JavaScript服务端
2. 完成AUTH认证
3. 成功建立OPEN连接
4. 双向DATA传输正常
5. 正确处理CLOSE帧
6. 支持三种代理模式

## 与JavaScript服务端对齐

此Go客户端实现完全对齐JavaScript服务端的v2协议规范：

- 相同的Frame结构
- 相同的命令定义
- 相同的OPEN帧格式 ("host:port|firstPayload")
- 相同的错误处理流程
