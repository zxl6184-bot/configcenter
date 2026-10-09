# 快速验证指南：NATS / Valkey / IoTDB 支持

**功能**: [spec.md](./spec.md) | **日期**: 2026-06-29

## 前置条件

- Go 1.22+ 已安装
- 配置中心已构建（`make build`）
- Postgres 数据库可用（用于元数据存储）
- 以下外部服务可用（测试用）：
  - NATS 服务（`nats://<host>:4222`）
  - Valkey 服务（`valkey://<host>:6379`）
  - IoTDB 服务（`http://<host>:18080`）

## 配置准备

在 `cmd/configcenter/config.toml` 中添加新 Broker 配置：

```toml
# 在现有 [[configCenter.serviceBrokers]] 后追加

[[configCenter.serviceBrokers]]
schema = "nats"
host = "<nats-host>"
port = 4222

[[configCenter.serviceBrokers]]
schema = "valkey"
host = "<valkey-host>"
port = 6379
username = "admin"
password = "secret"
[configCenter.serviceBrokers.dsn]
[configCenter.serviceBrokers.dsn.params]
mode = "cluster"

[[configCenter.serviceBrokers]]
schema = "iotdb"
host = "<iotdb-host>"
port = 18080
username = "root"
password = "root"
```

## 验证场景

### 场景 1：NATS 服务注册

**操作**：

```bash
# 启动配置中心
cd cmd/configcenter && go run .

# 通过 gRPC 或 REST 注册 NATS 服务
curl -X POST http://localhost:<port>/api/v1/services \
  -H "Content-Type: application/json" \
  -d '{
    "serviceName": "test-nats-service",
    "serviceInstances": [{
      "serviceInstanceType": "NATS",
      "serviceInstanceConfig": {}
    }]
  }'

# 查询服务验证 DSN
curl http://localhost:<port>/api/v1/services
```

**预期结果**:
- 返回的 NATS DSN 格式为 `nats://<host>:4222`
- 不含用户名和密码

### 场景 2：Valkey 服务注册（含 mode 参数）

**操作**：

```bash
curl -X POST http://localhost:<port>/api/v1/services \
  -H "Content-Type: application/json" \
  -d '{
    "serviceName": "test-valkey-service",
    "serviceInstances": [{
      "serviceInstanceType": "Valkey",
      "serviceInstanceConfig": {}
    }]
  }'
```

**预期结果**:
- 返回的 Valkey DSN 格式为 `valkey://admin:secret@<host>:6379?mode=cluster`
- 包含部署模式参数

### 场景 3：Redis 服务注册（向后兼容）

**操作**：

```bash
curl -X POST http://localhost:<port>/api/v1/services \
  -H "Content-Type: application/json" \
  -d '{
    "serviceName": "test-redis-service",
    "serviceInstances": [{
      "serviceInstanceType": "Redis",
      "serviceInstanceConfig": {}
    }]
  }'
```

**预期结果**:
- Redis DSN 格式不变（无 mode 参数时保持原样）
- 如 Redis Broker 配置了 mode 参数，则 DSN 包含 `?mode=<value>`

### 场景 4：IoTDB 服务注册与清理

**操作**：

```bash
# 注册 IoTDB 服务
curl -X POST http://localhost:<port>/api/v1/services \
  -H "Content-Type: application/json" \
  -d '{
    "serviceName": "test-iotdb-service",
    "serviceInstances": [{
      "serviceInstanceType": "IoTDB",
      "serviceInstanceConfig": {}
    }]
  }'

# 验证 IoTDB 中数据库已创建
curl -X POST http://<iotdb-host>:18080/rest/v1/sql \
  -d '{"sql": "SHOW DATABASES"}'

# 注销服务
curl -X DELETE http://localhost:<port>/api/v1/services/test-iotdb-service

# 验证 IoTDB 中数据库已删除
curl -X POST http://<iotdb-host>:18080/rest/v1/sql \
  -d '{"sql": "SHOW DATABASES"}'
```

**预期结果**:
- 注册后 IoTDB 中出现新数据库
- DSN 格式为 `iotdb://<user>:<password>@<host>:<port>/<database>`
- 注销后 IoTDB 中数据库和用户被清理

### 场景 5：资源组共享验证

**操作**：
- 在资源组配置中将两个服务关联到同一资源组
- 分别注册两个服务请求同一类型的实例（如 IoTDB）

**预期结果**:
- 两个服务返回相同的 DSN（共享同一实例）

### 场景 6：agent-browser + Swagger 端到端验证（实测)

> 本场景已通过 agent-browser 操作 Swagger UI 实测通过,记录实际的 API 字段与验证要点。

**前置**:配置中心已启动(`make run`,监听 `:10087`),redis/valkey/iotdb/nats 四个 Broker 已在 `config.toml` 中配置。

**操作**:

1. 浏览器打开 Swagger UI:`http://<host>:10087/public/apidoc/`
2. 展开 `POST /v1/services` → `Try it out`,填入请求体后 `Execute`:

```json
{
  "name": "e2e-verify",
  "serviceInstances": [
    { "schema": "redis",  "serviceInstanceConfig": {} },
    { "schema": "valkey", "serviceInstanceConfig": {} },
    { "schema": "iotdb",  "serviceInstanceConfig": {} },
    { "schema": "nats",   "serviceInstanceConfig": {} }
  ]
}
```

3. 展开 `GET /v1/services` → `Execute`,在返回中查看 `e2e-verify.serviceInstances`。

> 注意:实际 API 请求体中数据库类型字段为 `schema`(小写,大小写不敏感),而非 `serviceInstanceType`;`serviceInstances` 为**数组**。

**预期结果**(四种链接均生成,符合 `contracts/dsn-contracts.md`):

| schema | DSN 格式 |
|--------|----------|
| redis  | `redis://admin:testpass@<host>:16379?mode=sentinel` |
| valkey | `valkey://admin:testpass@<host>:16379?mode=cluster` |
| nats   | `nats://<host>:14222`(无账密) |
| iotdb  | `iotdb://<uuid>:<uuid>@<host>:18082/<uuid>`(自动建库/用户) |

**要点 1 — 30s 心跳租约**:服务实例在 KV 中以 30 秒 TTL 存储(每 10s 心跳,丢 3 次清除)。注册超过 30 秒后再 `GET`,该服务的 `serviceInstances` 会变为空 `{}`,属**正常在线状态过期**,并非链接丢失。重新注册后信息恢复。校验链接需在 30 秒内查询,或让客户端周期性重注册。

**要点 2 — 链接幂等不变**:有状态 Broker(iotdb/postgres 等)将生成的库名/用户名/密码持久化在 postgres 的 `service_instances` 表(真正的数据源),注册时以**服务名**(同资源组)为 key 查询已有实例。用同名重复注册,返回的 DSN **完全一致**,不会重复建库:

```bash
# 同名两次注册,iotdb 链接应完全相同
curl -s -X POST http://<host>:10087/v1/services -H "Content-Type: application/json" \
  -d '{"name":"same-name-test","serviceInstances":[{"schema":"iotdb","serviceInstanceConfig":{}}]}'
curl -s http://<host>:10087/v1/services   # 记下 same-name-test.serviceInstances.iotdb
curl -s -X POST http://<host>:10087/v1/services -H "Content-Type: application/json" \
  -d '{"name":"same-name-test","serviceInstances":[{"schema":"iotdb","serviceInstanceConfig":{}}]}'
curl -s http://<host>:10087/v1/services   # iotdb 链接与上一次一致
```

## 回归验证

```bash
# 运行全量测试
make test

# 验证构建
make build
```

**预期结果**:
- `make build` 成功，无编译错误
- `make test` 通过，无回归失败
- 现有 Postgres/MongoDB/Redis/RabbitMQ/InfluxDB/Minio 功能不受影响
