# 任务清单：NATS / Valkey / IoTDB 支持及 Redis 部署模式参数

**输入**: 设计文档来自 `/specs/001-nats-valkey-iotdb-support/`

**前置条件**: plan.md（必需）、spec.md（必需）、research.md、data-model.md、contracts/

**测试**: 本功能规格未明确要求单元测试，以下任务不包含测试任务。实现完成后通过 `make build` + `make test` 验证无回归。

## 格式：`[ID] [P?] [Story] 描述`

- **[P]**: 可并行执行（不同文件，无依赖）
- **[Story]**: 所属用户故事（US1=NATS, US2=Valkey, US3=Mode参数, US4=IoTDB）
- 描述中包含确切文件路径

---

## Phase 1: 环境准备（Setup）

**目的**: SDK 依赖更新和项目初始化

- [x] T001 在 sdk/configcenter 的 `dev.azure.com/Advantech-EBO/IoTSuite-Core-Component/_git/configcenter.git` 的 protobuf 定义中添加 `NATS=6`、`Valkey=7`、`IoTDB=8` 枚举值，重新生成 Go 代码并发布新版本（如 `v0.17.0`）。⚠️ 此任务为外部阻塞项，若 SDK 发版周期较长，可先在 `go.mod` 中使用 `replace` 指令指向本地 SDK 路径进行开发
- [x] T002 在 `go.mod` 中添加 `dev.azure.com/Advantech-EBO/IoTSuite-Core-Component/_git/configcenter.git` 依赖版本至包含新枚举值的版本，执行 `go mod tidy`

---

## Phase 2: 基础设施（Foundational）

**目的**: 核心导入注册，必须在任何用户故事之前完成

**⚠️ 关键**: 此阶段完成前不能开始任何用户故事

- [x] T003 在 `cmd/configcenter/main.go` 中添加三个新 broker 的 `import _` 注册（`_ "configcenter/pkg/configcenter/servicebroker/nats"`、`_ "configcenter/pkg/configcenter/servicebroker/valkey"`、`_ "configcenter/pkg/configcenter/servicebroker/iotdb"`）

**检查点**: 基础就绪 — 可开始并行实现用户故事

---

## Phase 3: 用户故事 1 — NATS 消息中间件支持（优先级：P1）🎯 MVP

**目标**: 配置中心支持 NATS 服务注册，返回 `nats://<host>:<port>` 格式连接串，无需用户名密码

**独立测试**: 注册 NATS 类型服务，验证返回的 DSN 格式为 `nats://<host>:<port>`，不包含认证信息

### 实现

- [x] T004 [US1] 创建 NATS ServiceBroker 包 `pkg/configcenter/servicebroker/nats/nats.go`，包含：
  - `init()` 调用 `RegisterServiceBrokerBuilder(configcenter.ServiceInstance_NATS.String(), NewNatsServiceBroker)`
  - `NatsServiceBroker` 结构体（host, port, db, connInfo 字段）
  - `NewNatsServiceBroker(connInfo, db)` 构造函数
  - `NewInstance(serviceName, params, resource)` — 在 GORM 中创建 ServiceInstance 记录（Username/Password/Database 为空字符串），返回 `BuildDSN`
  - `DropInstance(instance)` — 调用 `DeleteInstanceInDatabase` 删除元数据
  - `BuildDSN(instance)` — 返回 `fmt.Sprintf("nats://%s:%d", host, port)`。**注意：无论 connInfo 中是否配置了 Username/Password，NATS DSN 始终不包含认证信息（用户明确要求"不用账密"）**
- [x] T005 [US1] 验证：执行 `make build` 确认编译通过，NATS broker 正确注册。注意：NATS 无远程管理操作，SC-001（3 秒内获取 DSN）自然满足（仅 GORM 写入 + 字符串拼接）

**检查点**: NATS 服务注册/发现/注销完整可用

---

## Phase 4: 用户故事 2 — Valkey 键值存储支持（优先级：P1）

**目标**: 配置中心支持 Valkey 服务注册，作为独立 broker 包，使用 go-redis/v9 客户端

**独立测试**: 注册 Valkey 类型服务，验证 DSN 包含正确的认证信息

### 实现

- [x] T006 [US2] 创建 Valkey ServiceBroker 包 `pkg/configcenter/servicebroker/valkey/valkey.go`，包含：
  - `init()` 调用 `RegisterServiceBrokerBuilder(configcenter.ServiceInstance_Valkey.String(), NewValkeyServiceBroker)`
  - `ValkeyServiceBroker` 结构体（valkeyClient *redis.Client, db, host, port, connInfo 字段）
  - `NewValkeyServiceBroker(connInfo, db)` — 使用 `redis.NewClient` 创建 Valkey 客户端（`Addr: host:port`, `Password: connInfo.Password`, `Username: connInfo.Username`）。注意：Valkey 与 Redis 使用相同的 RESP 协议，因此复用 `go-redis/v9` 客户端库（详见 research.md）
  - `NewInstance(serviceName, params, resource)` — 在 GORM 中创建 ServiceInstance 记录（Username/Password 从 connInfo 获取），返回 `BuildDSN`
  - `DropInstance(instance)` — 调用 `DeleteInstanceInDatabase` 删除元数据
  - `BuildDSN(instance)` — 返回 `fmt.Sprintf("valkey://%s:%s@%s:%d", username, password, host, port)`；当 Username 为空时格式为 `valkey://:%s@%s:%d`
- [x] T007 [US2] 验证：执行 `make build` 确认编译通过，Valkey broker 正确注册

**检查点**: Valkey 服务注册/发现/注销完整可用

---

## Phase 5: 用户故事 4 — IoTDB 时序数据库支持（优先级：P1）

**目标**: 配置中心支持 IoTDB 服务注册，创建独立数据库和用户实例（类似 Postgres/InfluxDB 模式）

**独立测试**: 注册 IoTDB 类型服务，验证 GORM 中记录了实例元数据，且 DSN 格式正确

### 实现

- [x] T008 [US4] 创建 IoTDB ServiceBroker 包 `pkg/configcenter/servicebroker/iotdb/iotdb.go`，包含：
  - `init()` 调用 `RegisterServiceBrokerBuilder(configcenter.ServiceInstance_IoTDB.String(), NewIotdbServiceBroker)`
  - `IotdbServiceBroker` 结构体（host, port, db *gorm.DB, httpClient *http.Client, connInfo 字段）
  - `NewIotdbServiceBroker(connInfo, db)` — 创建 HTTP 客户端，IoTDB REST API 基础 URL 为 `http://<host>:<port>/rest/v1/sql`
  - `createDatabase(dbName)` — HTTP POST `{"sql": "CREATE DATABASE <dbName>"}` 到 IoTDB REST API
  - `dropDatabase(dbName)` — HTTP POST `{"sql": "DELETE DATABASE <dbName>"}`
  - `createUser(username, password)` — HTTP POST `{"sql": "CREATE USER <username> '<password>'"}`
  - `dropUser(username)` — HTTP POST `{"sql": "DELETE USER <username>"}`
  - `grantPrivilege(dbName, username)` — HTTP POST `{"sql": "GRANT ALL ON DATABASE <dbName> TO USER <username>"}`
  - `newInstance(dbName, username, password)` — 依次调用 createDatabase → createUser → grantPrivilege，任一步失败则回滚已创建资源
  - `NewInstance(serviceName, params, resource)` — 生成 UUID 作为 dbName/username/password，GORM 事务中创建 ServiceInstance → 调用 newInstance → 失败则 Rollback → 成功则 Commit，返回 `BuildDSN`
  - `DropInstance(instance)` — 调用 dropUser + dropDatabase，然后 `DeleteInstanceInDatabase`
  - `BuildDSN(instance)` — 返回 `fmt.Sprintf("iotdb://%s:%s@%s:%d/%s", username, password, host, port, database)`
- [x] T009 [US4] 验证：执行 `make build` 确认编译通过，IoTDB broker 正确注册。完整的 IoTDB 数据库/用户创建和清理验证（SC-004、SC-005）在 Phase 7 的 T015（quickstart 场景 4）中覆盖

**检查点**: IoTDB 服务注册（含数据库/用户创建）/发现/注销（含资源清理）完整可用

---

## Phase 6: 用户故事 3 — Redis / Valkey 部署模式参数（优先级：P2）

**目标**: Redis 和 Valkey 的 DSN 支持可选的 `mode` 部署模式查询参数

**独立测试**: 配置 mode 参数后注册服务，验证 DSN 中包含 `?mode=<value>`；不配置时验证 DSN 不变

### 实现

- [x] T010 [US3] 修改 `pkg/configcenter/servicebroker/redis/redis.go` 的 `BuildDSN` 方法：
  - 读取 `r.connInfo.Params["mode"]`
  - 若 mode 不为空 → 追加 `?mode=<value>` 到 DSN：`fmt.Sprintf("redis://%s:%s@%s:%d?mode=%s", ...)`
  - 若 mode 为空 → 保持现有格式不变（向后兼容）
- [x] T011 [US3] 修改 `pkg/configcenter/servicebroker/valkey/valkey.go` 的 `BuildDSN` 方法：
  - 同 T010 逻辑，读取 `v.connInfo.Params["mode"]`
  - 若 mode 不为空 → 追加 `?mode=<value>` 到 DSN
  - 若 mode 为空 → 保持现有格式不变
- [x] T012 [US3] 验证：执行 `make build` + `make test`，确认 Redis 现有 DSN 格式未变化（零回归），mode 参数透传正确

**检查点**: Redis/Valkey DSN 支持 mode 参数，且向后兼容

---

## Phase 7: 收尾与交叉关注点（Polish）

**目的**: 全量验证、代码质量检查

- [x] T013 [P] 执行 `make build` 全量构建，确认无编译错误
- [x] T014 [P] 执行 `make test` 全量测试，确认无回归失败
- [x] T015 按 `quickstart.md` 执行 5 个验证场景，确认所有场景通过
- [x] T016 [P] 更新 `README.md` 或项目文档，标注新增支持的 NATS / Valkey / IoTDB 类型

---

## 依赖关系与执行顺序

### 阶段依赖

- **Phase 1 (Setup)**: 无依赖 — 可立即开始
- **Phase 2 (Foundational)**: 依赖 Phase 1 完成（需要新枚举值）— **阻塞所有用户故事**
- **Phase 3 (US1 NATS)**: 依赖 Phase 2 完成 — 可独立于其他故事
- **Phase 4 (US2 Valkey)**: 依赖 Phase 2 完成 — 可独立于其他故事
- **Phase 5 (US4 IoTDB)**: 依赖 Phase 2 完成 — 可独立于其他故事
- **Phase 6 (US3 Mode)**: 依赖 Phase 4 (Valkey broker 存在) — 修改已有文件
- **Phase 7 (Polish)**: 依赖所有用户故事完成

### 用户故事依赖

- **US1 (NATS)**: 无其他故事依赖，可独立开始
- **US2 (Valkey)**: 无其他故事依赖，可独立开始
- **US4 (IoTDB)**: 无其他故事依赖，可独立开始
- **US3 (Mode)**: 依赖 US2 的 Valkey broker 文件已存在（T010 修改已有 redis.go，T011 修改 US2 创建的 valkey.go）

### 故事内依赖

- 每个 broker 包先创建完整文件，再验证构建
- US3 依赖 valkey.go 已存在

### 并行机会

- Phase 3/4/5 的三个新 broker 包（T004、T006、T008）可并行开发（不同文件，无交叉依赖）
- Phase 7 的构建和测试（T013、T014）可并行执行
- Phase 1 的 T001（SDK 更新）可能与本地开发并行（若使用已发布的 SDK 版本）

---

## 并行示例

```bash
# Phase 3/4/5: 三个新 broker 可同时创建
Task: "创建 NATS broker 在 pkg/configcenter/servicebroker/nats/nats.go"
Task: "创建 Valkey broker 在 pkg/configcenter/servicebroker/valkey/valkey.go"
Task: "创建 IoTDB broker 在 pkg/configcenter/servicebroker/iotdb/iotdb.go"

# Phase 7: 构建和测试可同时运行
Task: "make build"
Task: "make test"
```

---

## 实施策略

### MVP 优先（仅 US1 NATS）

1. 完成 Phase 1: Setup（SDK 枚举 + go.mod）
2. 完成 Phase 2: Foundational（main.go 导入）
3. 完成 Phase 3: US1 NATS
4. **停止并验证**: 独立测试 NATS 服务注册
5. 可部署/演示

### 增量交付

1. Setup + Foundational → 基础就绪
2. 添加 US1 (NATS) → 测试 → 部署（MVP!）
3. 添加 US2 (Valkey) → 测试 → 部署
4. 添加 US4 (IoTDB) → 测试 → 部署
5. 添加 US3 (Mode 参数) → 测试 → 部署
6. 每个故事独立增值，不破坏已有功能

### 并行团队策略

多开发者场景：
1. 团队共同完成 Setup + Foundational
2. Foundational 完成后：
   - 开发者 A: US1 (NATS)
   - 开发者 B: US2 (Valkey)
   - 开发者 C: US4 (IoTDB)
3. US3 (Mode) 在 US2 完成后由任意开发者接手

---

## 备注

- [P] 任务 = 不同文件，无依赖，可并行
- [Story] 标签将任务映射到特定用户故事，便于追踪
- 每个用户故事应可独立完成和测试
- 每个任务或逻辑组完成后提交
- 在检查点停止以独立验证故事
- 避免：模糊任务、同文件冲突、破坏独立性的跨故事依赖
