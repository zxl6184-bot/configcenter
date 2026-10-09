# 数据模型：NATS / Valkey / IoTDB 支持及 Redis 部署模式参数

**功能**: [spec.md](./spec.md) | **日期**: 2026-06-29

## 实体概览

本功能不引入新的数据库表或实体。所有新增 Broker 复用现有 `ServiceInstance` 表存储元数据。

## 现有实体复用

### ServiceInstance（GORM 模型）

位置：`pkg/configcenter/servicebroker/servicebroker.go`

| 字段 | 类型 | 说明 |
|------|------|------|
| ID | uint | 主键 |
| ServiceName | string | 注册该实例的服务名 |
| Database | string | 数据库/vhost 名称（NATS 为空字符串） |
| Username | string | 用户名（NATS 为空字符串） |
| Password | string | 密码（NATS 为空字符串） |
| ServiceType | string | 服务类型（新值：nats、valkey、iotdb） |
| Parameters | JSON map | 额外参数（如 mode） |

### ServiceInstance_ServiceInstanceType（Protobuf 枚举）

位置：`configcenter-sdk/pkg/configcenter/configcenter.pb.go`

| 枚举值 | 数值 | 说明 |
|--------|------|------|
| Postgres | 0 | 已有 |
| Mongodb | 1 | 已有 |
| Redis | 2 | 已有 |
| RabbitMQ | 3 | 已有 |
| Influxdb | 4 | 已有 |
| Minio | 5 | 已有 |
| **NATS** | **6** | **新增** |
| **Valkey** | **7** | **新增** |
| **IoTDB** | **8** | **新增** |

## 新增 Broker 数据结构

### NATS ServiceBroker

```go
type NatsServiceBroker struct {
    host     string
    port     int
    db       *gorm.DB
    connInfo dsn.ConnectInfo
}
```

- 无外部客户端（不需要管理操作）
- `NewInstance`: 记录 ServiceInstance 到 GORM（Database/Username/Password 为空）
- `DropInstance`: 删除 GORM 中的 ServiceInstance 记录
- `BuildDSN`: 返回 `nats://<host>:<port>`

### Valkey ServiceBroker

```go
type ValkeyServiceBroker struct {
    valkeyClient *redis.Client
    db           *gorm.DB
    host         string
    port         int
    connInfo     dsn.ConnectInfo
}
```

- 使用 go-redis/v9 客户端（仅用于连接验证，不需要管理操作）
- 结构与 RedisServiceBroker 一致
- `BuildDSN` 支持从 `connInfo.Params["mode"]` 读取部署模式

### IoTDB ServiceBroker

```go
type IotdbServiceBroker struct {
    host       string
    port       int
    db         *gorm.DB
    httpClient *http.Client
    connInfo   dsn.ConnectInfo
}
```

- 使用 HTTP 客户端发送 SQL 管理语句
- `NewInstance`: 创建数据库 → 创建用户 → 授权（类似 InfluxDB）
- `DropInstance`: 删除用户 → 删除数据库
- `BuildDSN`: 返回 `iotdb://<user>:<password>@<host>:<port>/<database>`

## 数据流

### 服务注册流程（以 IoTDB 为例）

```
gRPC ServiceRegistration 请求
  → ConfigCenter.NewInstance("iotdb", serviceName, params)
    → ServiceBrokers.NewInstance(schema, serviceName, params)
      → IotdbServiceBroker.NewInstance(serviceName, params, "")
        1. 生成随机 dbName, username, password (UUID)
        2. 在 GORM 中创建 ServiceInstance 记录
        3. HTTP POST IoTDB REST API 创建数据库
        4. HTTP POST IoTDB REST API 创建用户
        5. HTTP POST IoTDB REST API 授权
        6. 提交 GORM 事务
        7. 返回 BuildDSN(instance)
```

### DSN 生成流程（以 Valkey mode 为例）

```
BuildDSN(instance)
  1. 检查 connInfo.Params["mode"] 是否存在
  2. 若存在 → 返回 "valkey://user:pass@host:port?mode=cluster"
  3. 若不存在 → 返回 "valkey://user:pass@host:port"（向后兼容）
```
