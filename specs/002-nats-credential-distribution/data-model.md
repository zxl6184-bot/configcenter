# 数据模型：NATS 用户凭证文件读取与分发

## 概览

本功能**不新增持久化实体**。NATS 用户凭证为「全局一份」的配置态数据，在 broker 内存中缓存，不写入数据库，不改动 `ServiceInstance` 表结构。

## 实体

### 1. NatsCredential（内存态，新增）

broker 构造时解析凭证文件得到的全局凭证缓存。

| 字段 | 类型 | 说明 |
|------|------|------|
| JWT | string | 凭证文件 JWT 块原始内容（Base64 可解码回 NATS 用户 JWT 字符串） |
| Seed | string | 凭证文件 Seed 块原始内容（Base64 可解码回 NKEY Seed） |
| JWTParam | string | `url.QueryEscape(base64.StdEncoding(JWT))`，直接用于 DSN 参数 |
| SeedParam | string | `url.QueryEscape(base64.StdEncoding(Seed))`，直接用于 DSN 参数 |

约束：
- 加载时若 JWT 或 Seed 任一缺失 → 构造失败（返回 error，遵循 FR-008）。
- 缓存为不可变，注册请求不重复读文件。

### 2. NatsServiceBroker（内存态，修改）

现有 broker 结构体新增凭证字段。

| 字段 | 类型 | 说明 |
|------|------|------|
| host / port | string / int | 来自 `dsn.ConnectInfo`，不变 |
| db | *gorm.DB | 元数据持久化，不变 |
| connInfo | dsn.ConnectInfo | 不变 |
| creds | NatsCredential | 新增，构造时加载 |

### 3. ServiceInstance（既有，不变）

NATS 实例的元数据记录仍为：

| 字段 | 值 |
|------|-----|
| ServiceType | "nats" |
| Username / Password / Database | 空（NATS 无库/用户概念；凭证为全局态） |
| Parameters | 空（凭证不写此处，避免与 per-instance 语义混淆） |

### 4. 凭证文件（文件态，外部输入）

- 路径 = `path.Join(NatsCredsPath, NatsCredsName)`。
- 标准 NATS `nsc` credentials 格式，含 JWT 与 Seed 两个区块。
- 挂载来源：K8s Secret 卷 / Docker volume / 二进制普通文件。

## 关系

- `NatsServiceBroker --加载--> NatsCredential`（1:1，构造期完成）
- `NatsServiceBroker --生成--> NATS DSN`（含 jwt/seed 参数，见 contracts）
- 凭证与 `ServiceInstance` 无关联：全局一份，所有服务共享。

## 状态流转

- 凭证无状态流转；加载成功即恒常可用，加载失败即 broker 构造失败（启动失败）。