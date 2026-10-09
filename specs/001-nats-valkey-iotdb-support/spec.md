# 功能规格：NATS / Valkey / IoTDB 支持及 Redis 部署模式参数

**功能分支**: `001-nats-valkey-iotdb-support`

**创建日期**: 2026-06-29

**状态**: 草稿

**输入**: 用户描述："1. 配置中心支持nats(不用账密)，2.支持valkey支持，3.Redis，valkey添加一个参数表示现在是什么方式部署的比如dsn后面加个参数，用于表示部署模式，4.支持IoTDB，这个需要创建实例和其他关系型数据库一样"

## 澄清记录

### 会话 2026-06-29

- Q: Valkey Broker 应该作为独立的 broker 包实现，还是复用现有的 Redis broker？ → A: 独立 broker 包（`pkg/configcenter/servicebroker/valkey/`），使用 go-redis 客户端
- Q: IoTDB 的管理 API 使用什么协议进行数据库和用户的创建/删除操作？ → A: IoTDB 原生 REST API（HTTP + SQL 语句，类似 InfluxDB 1.x）
- Q: `mode` 参数的值是否需要系统进行枚举校验？ → A: 透传任意值，不校验，与 RabbitMQ 现有 params 机制一致

## 用户场景与测试 *(必填)*

### 用户故事 1 — NATS 消息中间件支持（优先级：P1）

作为服务开发者，我需要通过配置中心自动获取 NATS 连接信息，以便我的微服务无需手动配置即可连接到 NATS 消息系统。NATS 部署不需要用户名和密码认证。

**为何是这个优先级**: NATS 是核心消息中间件，被多个微服务依赖，是最基础的服务发现需求。

**独立测试**: 可以通过注册一个 NATS 类型服务并验证返回的 DSN 连接串是否正确来完整测试。

**验收场景**:

1. **Given** 配置中心已部署且 NATS Broker 已配置，**When** 服务通过 gRPC 注册请求 NATS 实例，**Then** 系统返回有效的 NATS DSN 连接串（格式：`nats://<host>:<port>`），无需用户名和密码。
2. **Given** 已有 NATS 服务实例，**When** 另一个同资源组的服务请求 NATS 实例，**Then** 系统返回已有共享实例的 DSN，不创建新实例。
3. **Given** 已有 NATS 服务实例，**When** 服务注销，**Then** NATS 实例被正确删除。

---

### 用户故事 2 — Valkey 键值存储支持（优先级：P1）

作为服务开发者，我需要通过配置中心自动获取 Valkey（Redis 兼容）连接信息，支持有用户名/密码和仅密码两种认证方式，以便微服务可以连接到 Valkey 存储。

**为何是这个优先级**: Valkey 作为 Redis 的替代方案，在内存存储场景中与 Redis 同等重要。

**独立测试**: 可以通过注册 Valkey 类型服务并验证 DSN 连接串来独立测试。

**验收场景**:

1. **Given** 配置中心已部署且 Valkey Broker 已配置，**When** 服务注册请求 Valkey 实例（带用户名密码），**Then** 系统返回有效的 Valkey DSN（格式：`valkey://<user>:<password>@<host>:<port>`）。
2. **Given** Valkey 仅配置密码无用户名，**When** 服务注册请求 Valkey 实例，**Then** 系统返回仅含密码的 DSN（格式：`valkey://:<password>@<host>:<port>`）。
3. **Given** 已有 Valkey 实例，**When** 服务注销，**Then** Valkey 实例被正确清理。

---

### 用户故事 3 — Redis / Valkey 部署模式参数（优先级：P2）

作为运维人员，我需要在 Redis 和 Valkey 的 DSN 中标识当前部署模式（例如单机 `standalone`、哨兵 `sentinel`、集群 `cluster`），以便客户端能根据部署模式选择正确的连接方式。

**为何是这个优先级**: 此参数增强运维可见性和客户端智能路由，但基础的连接功能（US1/US2）更为优先。

**独立测试**: 可以通过注册 Redis/Valkey 服务并检查返回的 DSN 中是否包含 `mode` 参数来独立测试。

**验收场景**:

1. **Given** Redis 配置了部署模式为 `sentinel`，**When** 服务注册 Redis 实例，**Then** 返回的 DSN 中包含 `?mode=sentinel` 参数。
2. **Given** Valkey 配置了部署模式为 `cluster`，**When** 服务注册 Valkey 实例，**Then** 返回的 DSN 中包含 `?mode=cluster` 参数。
3. **Given** Redis 未配置部署模式，**When** 服务注册 Redis 实例，**Then** 返回的 DSN 不包含 `mode` 参数（向后兼容）。
4. **Given** Valkey 未配置部署模式，**When** 服务注册 Valkey 实例，**Then** 返回的 DSN 不包含 `mode` 参数（向后兼容）。

---

### 用户故事 4 — IoTDB 时序数据库支持（优先级：P1）

作为服务开发者，我需要通过配置中心自动创建 IoTDB 数据库实例并获取连接信息，就像使用 Postgres 或 InfluxDB 一样。IoTDB 需要为每个服务创建独立的数据库、用户名和密码。

**为何是这个优先级**: IoTDB 是工业物联网场景的关键时序数据库，与 InfluxDB 同等重要。

**独立测试**: 可以通过注册 IoTDB 服务、验证数据库是否在 IoTDB 中实际创建、并检查返回的 DSN 来独立测试。

**验收场景**:

1. **Given** 配置中心已部署且 IoTDB Broker 已配置，**When** 服务注册请求 IoTDB 实例，**Then** 系统在 IoTDB 中创建新数据库和用户，并返回 DSN（格式：`iotdb://<user>:<password>@<host>:<port>/<database>`）。
2. **Given** 已有 IoTDB 实例，**When** 同资源组的另一个服务请求 IoTDB，**Then** 返回已有共享实例的 DSN。
3. **Given** 已有 IoTDB 实例，**When** 服务注销，**Then** IoTDB 中的数据库和用户被清理，元数据记录被删除。
4. **Given** IoTDB 实例创建过程中数据库创建失败，**When** 事务回滚，**Then** 元数据记录不保留，系统中不留下孤立资源。

---

### 边界情况

- NATS 配置了账密但用户明确说"不用账密"——`BuildDSN` 始终忽略 `connInfo.Username/Password`，连接字符串不包含认证信息，仅 `nats://<host>:<port>`。
- Valkey 的认证方式：支持有用户名/密码和仅密码两种模式，由配置中的 ConnectInfo 决定。
- Redis/Valkey 的 `mode` 参数从哪里来？——来自 `config.toml` 中对应 Broker 的 `dsn.params` 或 ConnectInfo 的 `Params` 字段，与 RabbitMQ 现有的 `params` 机制一致。
- IoTDB 的 API 兼容性：假设 IoTDB 提供与 InfluxDB 类似的管理 API（创建/删除数据库、用户管理）。
- 当多个服务属于同一资源组时，NATS 实例是否共享？——NATS 不创建独立实例（无数据库/用户概念），仅返回连接信息。

## 需求 *(必填)*

### 功能需求

- **FR-001**: 系统 MUST 支持 NATS 作为新的 ServiceBroker 类型，通过 `RegisterServiceBrokerBuilder` 注册。
- **FR-002**: NATS Broker 的 `BuildDSN` MUST 返回格式为 `nats://<host>:<port>` 的连接串（不含用户名密码）。
- **FR-003**: 系统 MUST 支持 Valkey 作为新的独立 ServiceBroker 类型，实现在 `pkg/configcenter/servicebroker/valkey/` 包中（不复用 Redis broker，遵循每种数据库独立子包的架构模式）。
- **FR-004**: Valkey Broker 的 `BuildDSN` MUST 支持有用户名/密码和仅密码两种认证模式的 DSN 格式。
- **FR-005**: Redis 和 Valkey Broker 的 DSN MUST 支持可选的 `mode` 参数（如 `?mode=standalone`、`?mode=sentinel`、`?mode=cluster`），该参数从 ConnectInfo.Params 读取。
- **FR-006**: 当 `mode` 参数未配置时，Redis/Valkey DSN MUST 保持现有格式不变，确保向后兼容。
- **FR-007**: 系统 MUST 支持 IoTDB 作为新的 ServiceBroker 类型，实现 `NewInstance`、`DropInstance`、`BuildDSN` 方法。
- **FR-008**: IoTDB 的 `NewInstance` MUST 在 IoTDB 中创建独立的数据库、用户名和密码（类似 Postgres/InfluxDB 模式）。
- **FR-009**: IoTDB 的 `DropInstance` MUST 清理 IoTDB 中的数据库和用户。
- **FR-010**: IoTDB Broker MUST 在 `init()` 中通过 `RegisterServiceBrokerBuilder` 自动注册。
- **FR-011**: NATS Broker MUST 在 `init()` 中通过 `RegisterServiceBrokerBuilder` 自动注册。
- **FR-012**: Valkey Broker MUST 在 `init()` 中通过 `RegisterServiceBrokerBuilder` 自动注册。
- **FR-013**: 所有新增 Broker MUST 在 `main.go` 中通过 `import _` 触发注册。
- **FR-014**: 新增的 ServiceInstance 枚举值（NATS、Valkey、IoTDB）MUST 在 protobuf 定义中添加，并重新生成 Go 代码。

### 关键实体

- **NATS 连接信息**: 主机地址、端口（无需用户名、密码、数据库）。
- **Valkey 连接信息**: 主机地址、端口、用户名（可选）、密码、部署模式（可选参数）。
- **IoTDB 服务实例**: 数据库名、用户名、密码、主机地址、端口——与 Postgres/InfluxDB 实例结构一致。
- **部署模式参数（mode）**: 附加在 Redis/Valkey DSN 中的可选查询参数，标识部署拓扑（standalone / sentinel / cluster）。

## 成功标准 *(必填)*

### 可衡量成果

- **SC-001**: 用户注册 NATS 服务后，可在 3 秒内获取到有效的 NATS 连接串。
- **SC-002**: 用户注册 Valkey 服务后，连接串包含正确的认证信息和部署模式参数。
- **SC-003**: Redis/Valkey 添加 `mode` 参数后，现有 Redis 服务的 DSN 格式不发生任何变化（零回归）。
- **SC-004**: IoTDB 实例创建成功后，IoTDB 数据库中可查询到对应的数据库和用户记录。
- **SC-005**: IoTDB 实例删除后，IoTDB 中的数据库和用户资源被完全清理，无残留。
- **SC-006**: 四种新增功能全部完成后，`make build` 和 `make test` 通过，无回归错误。

## 假设

- NATS 部署环境不需要认证（无用户名/密码），直接通过 host:port 连接。
- Valkey 的 API 与 Redis 完全兼容，因此可以复用 go-redis 客户端库。
- IoTDB 使用原生 REST API（HTTP + SQL 语句）进行数据库和用户管理，与本项目 InfluxDB Broker 的 HTTP Client + SQL 模式一致。
- 部署模式参数的值由运维人员在 `config.toml` 中配置，系统不做枚举校验，仅透传。
- NATS 不创建独立数据库/用户资源，因此在资源组共享场景下，同一 NATS 集群的服务返回相同的连接信息。

