<!--
  Sync Impact Report
  Version change: 1.0.0 → 1.1.0 (新增测试与端到端验证原则，并修正多语言规范)
  Modified principles:
    - IV. 多语言规范 → IV. 多语言规范
    - 新增 VII. 测试与端到端验证
  Added sections:
    - Core Principles 第 VII 项：测试与端到端验证
  Removed sections: N/A
  Templates requiring updates:
    - .specify/templates/plan-template.md ✅ 无需更新（Constitution Check 引用动态生成）
    - .specify/templates/spec-template.md ✅ 无需更新
    - .specify/templates/tasks-template.md ✅ 无需更新
    - .specify/templates/commands/*.md ⚠ 目录不存在，无需更新
    - README.md ✅ 无需更新
  Follow-up TODOs: 无
-->

# 配置中心（ConfigCenter）项目宪法

## 核心原则

### I. Go 语言标准项目布局

所有代码 MUST 使用 Go 语言编写，遵循 Go 标准项目布局：
- `cmd/` 目录存放可执行入口
- `pkg/` 目录存放可复用库代码
- 每个包 MUST 有明确的职责边界
- 包命名 MUST 简洁、描述性强，避免冗余前缀

**理由**: 统一代码组织方式，降低新人上手成本，便于工具链自动化。

### II. 插件化 Broker 架构

数据库/中间件类型的支持 MUST 通过 ServiceBroker 插件机制实现：
- 每种数据库类型在 `init()` 中调用 `RegisterServiceBrokerBuilder` 注册
- `main.go` 通过 `import _` 触发注册
- ServiceBroker 接口 MUST 实现 `NewInstance`、`DropInstance`、`BuildDSN` 方法
- 新增 Broker 不得修改核心 ConfigCenter 逻辑，只添加新的 broker 包

**理由**: 开闭原则 — 对扩展开放，对修改封闭。新增数据库支持不应影响已有功能。

### III. 配置管理

配置 MUST 通过 `pkg/config` 全局单例管理：
- 优先读取 `config.toml`（TOML 格式）
- 回退到命令行 flag 参数
- `GetConfig()` 返回配置值拷贝，内部持有读锁
- 在循环中使用时 MUST 缓存到局部变量以避免重复加锁

**理由**: 集中管理配置来源，确保配置一致性，明确配置优先级。

### IV. 多语言规范

- 项目文档、规格文档和说明性文档 MUST 使用中文
- 代码注释 MUST 使用英文
- Git 提交信息 MUST 使用英文
- 代码标识符（变量名、函数名等）保持英文
- Git 提交 MUST 遵循 Conventional Commits 规范

**理由**: 文档中文化降低团队沟通成本；代码注释和提交英文化与国际社区和工具链兼容。

### V. 语义化版本

版本号 MUST 遵循 Semantic Versioning 2.0.0：
- `MAJOR.MINOR.PATCH` 格式
- 不向下兼容的 API 变更递增 MAJOR
- 新增向下兼容功能递增 MINOR
- 向下兼容的 Bug 修复递增 PATCH
- 预发布版本使用 `-alpha.N`、`-beta.N`、`-rc.N` 后缀
- CHANGELOG.md MUST 保持更新，遵循 Keep a Changelog 格式

**理由**: 明确版本语义，使用户能准确判断升级影响范围。

### VI. 服务注册与发现

- 静态服务在启动时从 JSON 配置文件加载
- 动态服务通过 gRPC `ServiceRegistration` 接口注册
- 数据 MUST 持久化到 KVStore（支持 memory 和 redis 两种后端）
- 服务发现通过 `getAllServices` 返回静态 + 动态服务的合并视图
- gRPC 和 REST API 通过 h2c 在同一端口提供

**理由**: 统一的服务生命周期管理，确保服务注册的一致性和可靠性。

### VII. 测试与端到端验证

功能变更 MUST 提供与风险匹配的测试覆盖：
- 业务逻辑变更 SHOULD 优先添加或更新单元测试
- RESTful API、gRPC 接口和服务启动流程变更 MUST 添加端到端或集成测试
- 端到端测试 MUST 使用明确的测试配置文件启动服务，避免依赖开发者本机环境
- 无 token API 的端到端测试 MUST 验证无需认证即可访问的预期行为
- 涉及 OpenAPI/Swagger 的变更 MUST 验证服务启动后可访问 `/public/apidoc/swagger.json`
- 完成实现前 MUST 运行相关测试；若跳过测试，MUST 在交付说明中解释原因

**理由**: 配置中心承担服务发现和数据库分配职责，端到端验证能发现配置、路由和启动流程问题。

## 技术约束

- **语言**: Go 1.22+
- **通信协议**: gRPC（Protocol Buffers）+ REST API（Gin 框架）
- **配置格式**: TOML（gookit/config）
- **数据库元数据管理**: GORM + Postgres
- **KV 存储后端**: memory（开发/测试）或 Redis（生产）
- **容器编排**: Kubernetes（Ingress 监听）+ Docker
- **构建**: Makefile + 多架构 Docker 镜像（amd64 + arm64）
- **内部依赖**: `dev.azure.com/.../configcenter.git`（protobuf 定义和 DSN 解析）

## 开发工作流

- 功能开发 MUST 从 feature 分支开始
- 每个提交 MUST 只包含一个功能或修复
- Pull Request 合并前 MUST 通过代码审查
- 构建 MUST 通过 `make build` 或 `make image` 完成
- 测试 MUST 通过 `make test` 运行，覆盖率不应下降
- 本地开发使用 `cd cmd/configcenter && go run .`

## Governance

本宪法是项目开发的最高准则，所有设计决策和代码审查 MUST 以此为基准：

- 宪法的修订 MUST 通过 PR 方式，经团队讨论和审批
- 修订后 MUST 更新版本号和修订日期
- 任何偏离宪法的设计 MUST 在实现计划的 "Constitution Check" 中明确记录并说明理由
- 代码审查时 MUST 验证变更是否符合宪法原则
- 新增数据库 Broker 时 MUST 遵循原则 II（插件化 Broker 架构）
- 新增或修改外部接口时 MUST 验证原则 VII（测试与端到端验证）

**Version**: 1.1.0 | **Ratified**: 2026-06-29 | **Last Amended**: 2026-06-30
