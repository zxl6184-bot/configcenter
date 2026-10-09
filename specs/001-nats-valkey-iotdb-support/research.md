# 技术研究：NATS / Valkey / IoTDB 支持及 Redis 部署模式参数

**功能**: [spec.md](./spec.md) | **日期**: 2026-06-29

## 研究 1: NATS Broker 实现方案

**决策**: NATS Broker 不实例化 NATS 客户端，仅记录元数据并返回连接串。

**理由**:
- NATS 不需要创建数据库/用户等远程资源，仅提供连接信息
- 与 Redis Broker 模式一致：Redis 虽然创建了 go-redis 客户端，但 NewInstance 中也仅记录 ServiceInstance 元数据，不在 Redis 中创建任何资源
- 简化实现，避免引入不必要的 NATS Go SDK 依赖

**替代方案评估**:
- 使用 `github.com/nats-io/nats.go` SDK：增加依赖但无实际管理操作需求，过度设计
- NATS 的 DSN 格式使用标准 NATS URL：`nats://<host>:<port>`

## 研究 2: Valkey 客户端库选择

**决策**: 使用 `github.com/redis/go-redis/v9`，与 Redis Broker 使用相同客户端库。

**理由**:
- Valkey 与 Redis 使用相同的 RESP 协议，go-redis 客户端完全兼容
- Valkey 作为独立 broker 包实现（`pkg/configcenter/servicebroker/valkey/`），遵循项目"每种数据库一个子包"的架构模式
- 代码结构与 Redis Broker 类似，但独立演进

**替代方案评估**:
- 复用 Redis broker 仅改 schema 名称：不符合架构模式，无法独立演进
- 使用 valkey-go 专用客户端：生态不如 go-redis 成熟，且 Valkey 官方推荐 go-redis

## 研究 3: IoTDB 管理 API

**决策**: 使用 IoTDB 原生 REST API（HTTP + SQL 语句）。

**理由**:
- IoTDB 提供 REST API 端点（默认端口 18080），支持通过 HTTP POST 发送 SQL 语句进行数据库/用户管理
- 与 InfluxDB Broker 实现模式一致（HTTP Client + SQL 查询）
- 无需引入额外的 IoTDB 原生驱动依赖
- SQL 管理语句：
  - 创建数据库：`CREATE DATABASE <dbName>`
  - 创建用户：`CREATE USER <username> '<password>'`
  - 授权：`GRANT ALL ON DATABASE <dbName> TO USER <username>`
  - 删除数据库：`DELETE DATABASE <dbName>`
  - 删除用户：`DELETE USER <username>`

**替代方案评估**:
- IoTDB Go SDK (Thrift)：需要引入 thrift 依赖，与现有 broker 模式不一致
- JDBC 桥接：不适合 Go 项目

## 研究 4: Protobuf 枚举扩展

**决策**: 在 `configcenter-sdk` 的 `ServiceInstance_ServiceInstanceType` 枚举中添加三个新值。

**理由**:
- 现有枚举值：Postgres(0), Mongodb(1), Redis(2), RabbitMQ(3), Influxdb(4), Minio(5)
- 新增：NATS(6), Valkey(7), IoTDB(8)
- 遵循 protobuf 向后兼容规则：仅追加，不修改已有值
- 需要更新 `configcenter-sdk` 仓库并重新生成 Go 代码

**影响范围**:
- `configcenter-sdk/pkg/configcenter/configcenter.pb.go` — 枚举定义
- `.proto` 源文件 — 添加枚举值
- 所有引用 `ServiceInstance_*` 常量的代码 — 无需修改（仅追加）

## 研究 5: 部署模式参数实现

**决策**: 在 Redis 和 Valkey Broker 的 `BuildDSN` 中读取 `ConnectInfo.Params["mode"]` 并追加到 DSN 查询参数。

**理由**:
- 与 RabbitMQ Broker 现有模式一致（`BuildDSN` 读取 `connectInfo.Params` 追加为 URL 查询参数）
- 透传机制：不校验 mode 值，由运维配置决定
- 向后兼容：当 Params 为空或 mode 不存在时，DSN 格式不变

**实现要点**:
- Redis `BuildDSN` 当前输出：`redis://<user>:<password>@<host>:<port>`
- 新增 mode 后输出：`redis://<user>:<password>@<host>:<port>?mode=<mode>`
- Valkey `BuildDSN` 同理：`valkey://<user>:<password>@<host>:<port>?mode=<mode>`
