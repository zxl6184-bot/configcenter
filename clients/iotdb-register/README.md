# iotdb-register

通过 **REST API** 向配置中心注册服务，并申请一个 **IoTDB** 服务实例。不依赖 SDK，纯标准库 HTTP 调用。

## 为什么用 REST 而不是 SDK gRPC

SDK 内部使用 **h2c（明文 HTTP/2 gRPC）** 与配置中心通信（见 [module.go:129](file:///D:/gopath/src/configcenter-sdk/pkg/module/module.go) `grpc.WithInsecure()`）。配置中心服务端也是 h2c + REST 同端口（10086）复用（见 [configcenter.go:118-120](file:///D:/gopath/src/configcenter/pkg/configcenter/configcenter.go)）。

公网入口 `https://api-configcenter-wiseiot-ensaas.dp.wise-paas.com.cn` 是 **HTTPS 网关**，只代理 REST（HTTP/1.1），**不透传 gRPC/h2c**。因此：

- **`CONFIG_CENTER_ADDRESS` 带 `https://` 前缀** → gRPC 解析器把它当成目标地址，产生 `dial tcp: lookup tcp///...: unknown port` 错误 → 每次 keepalive 都失败 → `Connected → Reconnecting → Connected` 死循环
- **`host:80` 或 `host:443` 明文拨号** → 能建立 TCP 连接，但网关返回 `HTTP 404` (text/plain)，gRPC 握手失败
- **内部 k8s 环境**：`CONFIG_CENTER_ADDRESS=configcenter:10086` 可以正常工作（集群内直连）

## 功能

- 通过 REST `POST /v1/services` 注册服务和一个 API Router
- 申请 `iotdb` 服务实例
- 轮询 `GET /v1/:serviceName/serviceintance` 等待 IoTDB 实例就绪
- 打印返回的 IoTDB DSN

## 环境变量

| 变量 | 必填 | 说明 |
| ---- | ---- | ---- |
| `CONFIG_CENTER_ADDRESS` | 是 | 配置中心地址，支持 `https://host` 或 `host`（自动补 `https://`） |

## 运行

```powershell
$env:CONFIG_CENTER_ADDRESS = "https://api-configcenter-wiseiot-ensaas.dp.wise-paas.com.cn"
cd D:\xl_6184\configcenter\clients\iotdb-register
go run . -name iotdb-client -address http://127.0.0.1:8080
```

参数说明：

| 参数 | 默认值 | 说明 |
| ---- | ------ | ---- |
| `-name` | `iotdb-client` | 注册到配置中心的**服务名** |
| `-router` | `API` | 该服务对外 Router 的名称 |
| `-address` | `http://127.0.0.1:8080` | 该服务对外 Router 的地址 |
| `-timeout` | `60s` | 等待 IoTDB 实例的超时时间 |

## 输出示例

```
register: POST https://api-configcenter-wiseiot-ensaas.dp.wise-paas.com.cn/v1/services
body: {"name":"iotdb-client","routers":{"API":{"name":"API","address":"http://127.0.0.1:8080"}},"serviceInstances":[{"schema":"iotdb","serviceInstanceConfig":{}}]}
register response: HTTP 201
poll: GET https://api-configcenter-wiseiot-ensaas.dp.wise-paas.com.cn/v1/iotdb-client/serviceintance
[14:30:01] iotdb not ready yet, instances: map[]
======== iotdb service instance ========
iotdb://<username>:<password>@<host>:<port>/<database>
========================================
```

## 依赖

- 零外部依赖，仅使用 Go 标准库（`net/http`、`encoding/json`）
- 无需 `GOPRIVATE` 配置，无需 Azure DevOps 访问权限