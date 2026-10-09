# 接口契约：NATS / Valkey / IoTDB 支持

**功能**: [spec.md](../spec.md) | **日期**: 2026-06-29

## 概述

本功能所有新增接口均为内部 ServiceBroker 接口实现，不新增外部 API 端点。已有 gRPC/REST 接口（`ServiceRegistration`、`/api/v1/services`）保持不变，通过新增的 schema 名称自动路由到对应 Broker。

## ServiceBroker 接口（内部契约）

所有新增 Broker 实现以下接口（已有，不变）：

```go
type ServiceBroker interface {
    NewInstance(serviceName string, params map[string]string, resource string) (string, error)
    DropInstance(serviceInstance ServiceInstance) error
    BuildDSN(serviceInstance ServiceInstance) string
}
```

## DSN 格式契约

### NATS

```
格式: nats://<host>:<port>
示例: nats://10.0.0.1:4222

规则:
- 无用户名/密码段
- 无数据库路径段
- 无查询参数
```

### Valkey

```
格式: valkey://[<username>:]<password>@<host>:<port>[?mode=<mode>]
示例: valkey://admin:secret@10.0.0.1:6379?mode=cluster
示例: valkey://:secret@10.0.0.1:6379

规则:
- 用户名可选（无用户名时格式为 "valkey://:password@..."）
- mode 参数可选
- 当 ConnInfo.Params["mode"] 为空时不追加 mode 参数
```

### Redis（更新后）

```
格式: redis://[<username>:]<password>@<host>:<port>[?mode=<mode>]
示例: redis://:secret@10.0.0.1:6379?mode=sentinel

规则:
- 与 Valkey 格式相同
- 新增 mode 参数支持
- 向后兼容：mode 为空时格式不变
```

### IoTDB

```
格式: iotdb://<username>:<password>@<host>:<port>/<database>
示例: iotdb://abc123:def456@10.0.0.1:6667/xyz789

规则:
- 用户名和密码为 UUID 生成
- 数据库名为 UUID 生成
- 路径段为数据库名
```

## gRPC 服务注册（已有协议，无变更）

```protobuf
service ConfigCenter {
    rpc ServiceRegistration(ServiceRegistrationRequest) returns (ServiceRegistrationResponse);
    rpc ServiceUnRegistration(ServiceRegistrationRequest) returns (ServiceRegistrationResponse);
}

message ServiceRegistrationRequest {
    string serviceName = 1;
    repeated ServiceInstance serviceInstances = 2;
}

message ServiceInstance {
    ServiceInstanceType serviceInstanceType = 1;
    map<string, string> serviceInstanceConfig = 2;
}
```

### 新增 ServiceInstanceType 枚举值

```protobuf
enum ServiceInstanceType {
    Postgres = 0;
    Mongodb = 1;
    Redis = 2;
    RabbitMQ = 3;
    Influxdb = 4;
    Minio = 5;
    NATS = 6;      // 新增
    Valkey = 7;    // 新增
    IoTDB = 8;     // 新增
}
```

## REST API（已有端点，无变更）

```
GET  /api/v1/services          # 查询所有服务及其 DSN
POST /api/v1/services          # 服务注册（含新 schema 类型）
DELETE /api/v1/services/:name   # 服务注销
```

## Config.toml 配置契约

### NATS Broker 配置示例

```toml
[[configCenter.serviceBrokers]]
schema = "nats"
host = "10.0.0.1"
port = 4222
# 无 username/password
```

### Valkey Broker 配置示例

```toml
[[configCenter.serviceBrokers]]
schema = "valkey"
host = "10.0.0.1"
port = 6379
username = "admin"
password = "secret"
[configCenter.serviceBrokers.dsn]
[configCenter.serviceBrokers.dsn.params]
mode = "cluster"
```

### IoTDB Broker 配置示例

```toml
[[configCenter.serviceBrokers]]
schema = "iotdb"
host = "10.0.0.1"
port = 18080
username = "root"
password = "root"
```
