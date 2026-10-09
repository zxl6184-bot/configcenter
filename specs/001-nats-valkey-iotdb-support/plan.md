# 实施计划：NATS / Valkey / IoTDB 支持及 Redis 部署模式参数

**分支**: `001-nats-valkey-iotdb-support` | **日期**: 2026-06-29 | **规格**: [spec.md](./spec.md)

**输入**: `/specs/001-nats-valkey-iotdb-support/spec.md` 功能规格

## 摘要

本功能为配置中心新增三种数据库/中间件类型的 ServiceBroker 支持：
1. **NATS** — 消息中间件，不需要认证，仅返回连接信息
2. **Valkey** — 键值存储（Redis 兼容），作为独立 broker 包，使用 go-redis 客户端
3. **IoTDB** — 时序数据库，需要创建独立数据库/用户实例（类似 Postgres/InfluxDB 模式）

同时为 Redis 和 Valkey 的 DSN 添加可选的 `mode` 部署模式参数（如 `?mode=cluster`）。

## 技术上下文

**语言/版本**: Go 1.22+

**主要依赖**:
- `github.com/redis/go-redis/v9` — Valkey 客户端（Redis 兼容）
- `github.com/nats-io/nats.go` — NATS 客户端（仅用于连接测试，无管理操作）
- IoTDB REST API（HTTP + SQL）— 数据库/用户管理，无需额外 Go SDK
- `google.golang.org/grpc` + `github.com/gin-gonic/gin` — 已有协议层
- `gorm.io/gorm` — 元数据持久化

**存储**: Postgres（GORM 元数据管理，与现有方式一致）

**测试**: `go test`（`make test`）

**目标平台**: Linux 服务器（Docker + K8s）

**项目类型**: 微服务（gRPC + REST API 后端）

**性能目标**: 服务实例注册 < 3 秒（与 SC-001 一致）

**约束**:
- 必须遵循 ServiceBroker 插件注册模式（原则 II）
- 不得修改核心 ConfigCenter 逻辑
- 新增代码仅限新的 broker 包 + main.go import + protobuf 枚举

**规模/范围**: 4 个新 broker 包（nats、valkey、iotdb）+ 1 个已有包修改（redis）

## 宪法检查

*门禁：必须在 Phase 0 研究前通过。Phase 1 设计后重新检查。*

| 原则 | 状态 | 说明 |
|------|------|------|
| I. Go 标准项目布局 | ✅ 通过 | 新增包位于 `pkg/configcenter/servicebroker/<name>/` |
| II. 插件化 Broker 架构 | ✅ 通过 | 每个新类型通过 `init()` + `RegisterServiceBrokerBuilder` 注册 |
| III. 配置管理 | ✅ 通过 | 配置通过 `config.toml` → `dsn.ConnectInfo` 传递 |
| IV. 多语言规范 | ✅ 通过 | 代码注释中文，Git 提交英文 |
| V. 语义化版本 | ✅ 通过 | 新增向下兼容功能 → MINOR 版本递增 |
| VI. 服务注册与发现 | ✅ 通过 | 复用现有 gRPC/REST 接口 |

**Phase 1 后重检**: 无违规项，无需填写复杂度追踪表。

## 项目结构

### 文档（本功能）

```text
specs/001-nats-valkey-iotdb-support/
├── plan.md              # 本文件
├── research.md          # Phase 0 研究输出
├── data-model.md        # Phase 1 数据模型
├── quickstart.md        # Phase 1 验证指南
├── contracts/           # Phase 1 接口契约
│   └── dsn-contracts.md
└── tasks.md             # Phase 2 任务分解（/speckit-tasks 生成）
```

### 源代码（仓库根）

```text
pkg/configcenter/servicebroker/
├── servicebroker.go          # ServiceBroker 接口（已有，不变）
├── nats/                     # [新增] NATS Broker
│   └── nats.go
├── valkey/                   # [新增] Valkey Broker
│   └── valkey.go
├── iotdb/                    # [新增] IoTDB Broker
│   └── iotdb.go
├── redis/                    # [修改] 添加 mode 参数
│   └── redis.go
├── influxdb/                 # 已有，不变
├── minio/                    # 已有，不变
├── mongodb/                  # 已有，不变
├── postgres/                 # 已有，不变
└── rabbitmq/                 # 已有，不变

cmd/configcenter/
└── main.go                   # [修改] 添加 3 个 import _

configcenter-sdk/
└── pkg/configcenter/
    └── configcenter.pb.go    # [修改] 添加 NATS/Valkey/IoTDB 枚举值
```

**结构决策**: 遵循现有单项目结构，每种数据库一个子包。Protobuf 枚举更新在 SDK 仓库中完成。

## 复杂度追踪

> 无宪法违规项，无需填写。
