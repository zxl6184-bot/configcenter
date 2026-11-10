# ConfigCenter 2.1.7 需求说明

## 1. 需求目标

为 IoTEdge 和 IoTEdge-Worker 增加 IoTDB 共享资源支持；在兼容现有公网地址生成方式的基础上，增加基于统一域名和 URI 路径的后缀模式，并通过公共配置向调用方提供模式、协议和入口端口信息。

本文使用以下命名：

- **前缀模式**：沿用现有基于服务子域名的公网地址生成方式。
- **后缀模式**：使用统一域名或 IP，通过不同 URI 路径区分服务，例如 `https://wise.example.com/iotedge/`。

## 2. IoTDB 资源共享

在 resource group 配置中新增独立共享组，使 `IoTEdge` 和 `IoTEdge-Worker` 可以共享 IoTDB 资源。

| 项目 | 要求 |
| --- | --- |
| 共享组名称 | `iotedge-iotdb` |
| 资源类型 | `iotdb` |
| 共享服务 | `IoTEdge`、`IoTEdge-Worker` |

保留原有资源组及其共享关系，避免影响现有功能。各部署方式使用的 resource group 配置需要同步更新。

## 3. 新增后缀模式配置

| 配置参数 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `isSuffixMode` | bool | `false` | 是否启用后缀模式；`false` 沿用现有前缀模式 |
| `suffixHttpPort` | int | `80` | 后缀模式对外 HTTP 入口端口 |
| `suffixHttpsPort` | int | `443` | 后缀模式对外 HTTPS 入口端口 |

复用现有 `isSSL` 配置决定公网地址使用 HTTP 还是 HTTPS，不新增独立的 SSL 开关。后缀模式端口表示对外访问的入口端口，不是服务内部监听端口。

旧配置未提供新增参数时，应保持现有前缀模式行为。相关配置需要在 Kubernetes、Docker Compose 和二进制部署中可用。

## 4. 公网地址生成规则

### 4.1 生成条件与兼容性

- 继续遵循现有 `autoGeneratePublic` 开关和公网地址查询控制条件。
- 保留已有手动指定公网地址的优先级，不因新增模式覆盖手动配置。
- 前缀模式沿用原有地址生成逻辑及平台处理，不扩大本次修改范围。
- 后缀模式按下述规则生成 `publicAddress`。

### 4.2 后缀模式的 Host

1. `Domain` 非空时，使用去除首尾空白后的 `Domain`。
2. `Domain` 为空时，使用去除首尾空白后的 `GlobalIP`。

后缀模式下，`Domain` 表示完整入口域名，不再拼接服务名、命名空间或集群等子域名前缀。地址拼接不根据 Kubernetes、Docker 或二进制平台分支判断。

现有 Kubernetes 启动阶段的域名必填校验继续保留，这与后缀地址拼接本身的平台无关规则分别处理。

### 4.3 协议与端口

| 条件 | 协议 | 入口端口 |
| --- | --- | --- |
| `isSSL=false` | HTTP | `suffixHttpPort` |
| `isSSL=true` | HTTPS | `suffixHttpsPort` |

使用对应协议的默认端口时，地址中省略端口；使用非默认端口时显式携带端口。

### 4.4 URI 路径

每个 router 新增可选字段 `uriPrefix`，表示服务公网入口的 URI 路径前缀。

路径按以下优先级获取：

1. 优先使用服务注册时 router 提供的 `uriPrefix`。
2. 注册值为空时，按服务名和 router 名从 Excel 对应记录获取路径兜底值。
3. 注册值与 Excel 兜底值均为空时，该 router 的 `publicAddress` 返回空字符串，静默处理，不影响服务注册、发现及其他 API 请求。

ConfigCenter 自身使用 `/configcenter` 作为后缀模式路径。

后缀地址由协议、Host、入口端口和上述 URI 路径生成。后端 `router.address` 的协议、端口及业务路径不参与公网地址拼接；服务内部业务路由仍由各服务负责。

### 4.5 示例

| 配置场景 | 生成结果 |
| --- | --- |
| Domain=`wise.example.com`，HTTP 80，路径=`/iotedge/` | `http://wise.example.com/iotedge/` |
| Domain=`wise.example.com`，HTTPS 443，路径=`/iotedge/` | `https://wise.example.com/iotedge/` |
| Domain=`wise.example.com`，HTTPS 8443，路径=`/iotedge/` | `https://wise.example.com:8443/iotedge/` |
| Domain 为空，GlobalIP=`192.168.1.10`，HTTP 8080，路径=`/iotedge/` | `http://192.168.1.10:8080/iotedge/` |
| ConfigCenter，Domain=`wise.example.com`，HTTP 80 | `http://wise.example.com/configcenter` |
| 注册路径和 Excel 路径均为空 | 空字符串 |

## 5. Excel 兜底配置

在现有服务端口 Excel 文件中新增 URI 路径列，保持原有列及其含义不变。

- 按当前实现，固定第六列的列名为 `uriprefix`。
- 按服务名和 router 名精确匹配对应记录。
- 仅在注册的 `uriPrefix` 为空时使用该列。
- 兼容缺少该列或单元格为空的旧数据，按未提供兜底路径处理。
- 同步维护各部署目录使用的 Excel 文件。

此处记录当前实现的列名；此前讨论过 `uri_prefix` 命名，本文不将该讨论视为已完成的改名。

## 6. Public Config 对外字段

在 `GET /v1/public/configs` 的原有返回内容中新增以下四个字段，保留原有字段。接口继续使用字符串形式返回配置值。

| 字段 | 示例值 | 说明 |
| --- | --- | --- |
| `suffix_http_port` | `"80"` | 后缀模式的对外 HTTP 入口端口 |
| `suffix_https_port` | `"443"` | 后缀模式的对外 HTTPS 入口端口 |
| `is_suffix_mode` | `"false"` | 是否启用后缀模式；`false` 为前缀模式，`true` 为后缀模式 |
| `is_ssl` | `"false"` | 是否使用 HTTPS，来源于现有 `isSSL` 配置 |

新增字段示例：

```json
{
  "suffix_http_port": "80",
  "suffix_https_port": "443",
  "is_suffix_mode": "false",
  "is_ssl": "false"
}
```

以上仅展示新增字段，实际响应还包含原有公共配置。

## 7. SDK 与接口同步

- 在独立 `configcenter-sdk` 项目的 Router 数据结构及 protobuf 定义中增加 `uriPrefix`，同步生成代码。
- 服务端注册、存储和查询链路应保留并传递该字段。
- SDK 增加四个新增公共配置字段对应的键常量。
- 同步更新接口说明和部署配置说明。
- 老客户端未传 `uriPrefix` 时继续兼容，按 Excel 兜底或空公网地址规则处理。

## 8. 完整访问流程与部署依赖

完整流程及 DP 环境配置见 [Wiki：DP 环境 DNS、Traefik 与请求转发](../wiki/dp-traefik-networking.md)。公网地址可实际访问，需要配置中心、DNS 和 Traefik 网关共同配合：

1. 服务向配置中心注册实际后端地址及 router 的 `uriPrefix`；配置中心按模式、SSL 和入口端口生成 `publicAddress`。
2. Traefik 作为统一入口网关，通过服务发现获取后端实例及路由信息，维护域名或路径到实际服务的转发规则。
3. 使用域名访问时，需要 DNS 服务将配置的 `domain` 解析到网关入口 IP；直接使用 GlobalIP 访问时，无需域名解析。
4. 客户端取得 `publicAddress`，解析域名后连接网关的 HTTP 或 HTTPS 入口端口。
5. Traefik 根据 Host、URI 路径匹配路由，将请求转发到实际后端服务，再将响应返回客户端。

前缀模式需要对应服务子域名的 DNS 记录，可按部署情况使用通配符解析；后缀模式使用完整入口域名，通过不同 URI 路径区分服务。DNS 只负责域名到 IP 的解析，具体服务选择由 Traefik 完成。HTTPS 入口还需配置匹配域名的证书。

服务发现的对接方式随部署环境确定：Kubernetes 可由 Traefik 读取 Ingress、Service 和 EndpointSlice；Docker Compose 可使用 Docker provider；二进制部署或统一通过配置中心发现服务时，可采用 Wiki 中描述的 HTTP provider 对接方案。Traefik 不会自动识别配置中心现有注册接口，需要提供它支持的发现来源或动态配置格式。

## 9. 范围边界

本次需求负责资源共享配置、服务注册字段、公网地址生成和公共配置输出。Docker Compose、二进制部署实际使用后缀地址时，需要网关提供对应路径转发及入口端口；HTTPS 还需要匹配的证书配置。

配置中心生成 `publicAddress` 不代表自动创建 DNS、Traefik 路由或 TLS 配置。根据注册信息自动生成 Traefik 动态路由，属于后续独立需求。

## 10. 验收标准

1. IoTEdge 与 IoTEdge-Worker 可通过新增共享组使用 IoTDB，原有共享组不变。
2. 不提供新增参数时，系统保持原有前缀模式行为。
3. 后缀模式可根据域名或 GlobalIP、SSL 开关、对应入口端口及 URI 路径生成公网地址。
4. HTTP 80、HTTPS 443 在生成地址中省略，非默认端口显式展示。
5. 注册路径优先于 Excel；注册值为空时使用 Excel 对应记录。
6. 两处路径均为空时返回空 `publicAddress`，不产生注册或查询错误。
7. 手动公网地址和 `autoGeneratePublic` 原有行为保持兼容。
8. Public Config 返回新增四个字段，类型为字符串，原有字段保留。
9. 新旧 SDK 客户端均可按上述兼容规则注册和查询服务。
10. 各部署配置及 Excel 保持同步，后缀拼接逻辑不按平台区分。
