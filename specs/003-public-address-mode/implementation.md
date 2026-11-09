# 公网地址模式：最终约定与实现

日期：2026-11-09。主工作区、服务端及实际 Azure SDK 均使用 `develop-2.1.7` 分支。

## 已确认规则

- `isSuffixMode` 使用布尔开关：默认 `false` 为前缀模式（兼容旧配置），`true` 为后缀模式。不保留整数枚举或扩展性设计。
- 前缀模式沿用 Ingress 地址转换及既有 K8s 启动域名必填校验。
- 后缀地址拼接不判断平台：`domain` 非空时使用去除首尾空白后的完整域名，为空时使用去除首尾空白后的 `GlobalIP`，再按协议、入口端口和前端路径生成地址。
- 后缀模式由 `isSSL` 选择 HTTP/HTTPS 及 `suffixHttpPort`/`suffixHttpsPort`，默认 80/443，对应协议的默认端口省略。
- 前端路径优先取 router 的 `uriPrefix`，为空时按服务名和 router 名精确查 Excel 固定第六列 `uriprefix`。两处都为空时，`publicAddress` 为空，静默处理，不能影响注册和查询请求。
- 后端 `router.address` 的路径、协议、端口不参与后缀地址生成。ConfigCenter 自身前端路径为 `/configcenter`。
- 继续遵循 `autoGeneratePublic` 和已有手动公网地址的优先级。
- public config 新增 `is_suffix_mode`、`is_ssl`、`suffix_http_port`、`suffix_https_port`，沿用字符串映射类型。
- 新增独立的 `iotedge-iotdb` 共享组，仅包含 `iotdb` 资源及 `IoTEdge`、`IoTEdge-Worker` 两个服务；原有共享组保持原样，四份部署配置同步。

## 实现位置

- 服务端：`/home/wh8199/xiaolin.zhang/gopath/src/configcenter`。
- SDK 独立仓库：`/home/wh8199/xiaolin.zhang/gopath/src/configcenter-sdk`，与服务端同级，在 `develop-2.1.7` 分支修改。服务端使用 `../configcenter-sdk` 进行本地联调。
- 服务端说明：`docs/public-address-mode.md`。
- 地址生成逻辑合并在服务端 `pkg/configcenter/services.go`，布尔模式配置位于 `pkg/config/config.go`，K8s 域名校验保留在 `pkg/configcenter/configcenter.go`；不再单独保留 `public_address.go`。
- SDK 新增 protobuf 字段：`Router.uriPrefix = 14`，已重新生成 Go 代码。
- 本地 SDK 联调使用 `go.work`；可通过 `go.work.example` 重建。尚未发布 SDK，也没有伪造已发布版本或修改正式 SDK 版本号。

## 验证记录

当前最小改动整理：`configcenter.go` 已恢复为本次开发前版本（保留此前平台判断和域名必填校验），原前缀地址分支保持原样；布尔配置编译及 Helm/Compose 检查通过。

以下是开发阶段已完成的验证；按当前整理要求，本次新增的测试文件和独立验证脚本已删除，历史验证记录保留。

- 相关单元测试、内存 gRPC 与 REST 集成测试通过。
- 开发阶段已验证旧配置默认前缀、Docker flags 后缀与默认端口、二进制 TOML 后缀与 HTTPS、K8s 后缀、K8s 缺域名拒绝启动；当前按要求改为布尔模式，相关整数枚举和校验已移除。
- 进程验证同时检查无认证 REST/gRPC 注册发现、public config、缺路径返回空、ConfigCenter 路径和 Swagger 可访问性。
- Linux 构建和 Windows 交叉编译均通过；Windows 原生运行及集群网关访问未在本次本地验证中执行。
- Helm 前缀/后缀配置渲染和 Docker Compose 解析通过。
- 三份 Excel 的原有 A–E 列内容逐项对比不变，只新增 `uriprefix` 路径列并预填已确认的 IoTEdge 路径。
- `make test` 已执行：本次相关测试通过；旧的 `TestMergeRouters` 空 map/nil 断言，以及依赖外部 InfluxDB、Minio、MongoDB 的测试仍失败。
- SDK 全套测试已执行：`dccs.TestGetDCCSResponse` 的外部接口返回 Not Found；其余测试与示例编译通过。
- `go vet ./...` 报告原有统一数据源关闭代码丢弃 context cancel，未混入本次改动。

## 发布前事项

先发布含 `uriPrefix` 的 SDK，再升级服务端 `go.mod`/`go.sum` 和 `SDK_VERSION`，在 `GOWORK=off` 下构建。当前工作属于本地开发与验证，没有推送、发布或部署集群。

Docker/二进制实际开放后缀地址时，需要统一网关按路径转发；网关配置和 TLS 不由配置中心自动创建。

## 本次代码提交

- 服务端后缀地址：`4336935`。
- 服务端独立 IoTDB 共享组：`ca3b809`。
- SDK URI 路径与公共配置键：`2970721`。

提交前 SDK 独立构建、服务端构建、资源组及 Swagger JSON 校验均通过。后缀模式按最新约定优先使用 domain，否则使用 GlobalIP，拼接逻辑不判断平台。以上均为 develop-2.1.7 分支的本地提交，未推送或发布。
