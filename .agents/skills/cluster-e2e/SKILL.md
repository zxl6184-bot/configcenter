---
name: cluster-e2e
description: 将 configcenter 部署到 admin-ensaas-context 的 xiaobai 命名空间并做端到端验证。用于集群测试、部署新镜像、检查集群版本和排查代码未生效。
---

# Configcenter 集群端到端测试

## 环境

| 项目 | 值 |
| --- | --- |
| context | `admin-ensaas-context` |
| namespace | `xiaobai` |
| release/deployment | `configcenter` |
| pod label | `app.kubernetes.io/instance=configcenter` |
| API | `http://127.0.0.1:10086` |
| registry | `appfactory.wise-paas.com.cn/configcenter/` |

Pod 内只有 `wget` 和 `nc`。节点使用 `imagePullPolicy: IfNotPresent`，代码改变后必须使用新 tag。

## 1. 检查现状

```bash
cd /home/wh8199/xiaolin.zhang/gopath/src/configcenter
kubectl config current-context
helm -n xiaobai list | grep '^configcenter'
kubectl -n xiaobai get deploy configcenter \
  -o jsonpath='{.spec.template.spec.containers[0].image}{"\n"}'
git status --porcelain
```

Go、Dockerfile、Makefile 或静态资源改变时需要新镜像。只改 Helm 配置时直接重新部署。

## 2. 更新版本

构建新镜像前先让用户确认。alpha 版本递增后，同步：

- `Makefile` 的 `TAG`
- `charts/configcenter/Chart.yaml` 的 `version` 和 `appVersion`
- `charts/configcenter/values.yaml` 的 `image.tag`

```bash
grep -nE '^TAG :=|^version:|^appVersion:|^  tag:' \
  Makefile charts/configcenter/Chart.yaml charts/configcenter/values.yaml
```

## 3. 构建推送

```bash
BUILDKIT_PROGRESS=plain make image TARGET=configcenter
```

以普通用户运行 make，不能运行 `sudo make`。构建失败后删除残留的 `vendor/`。自动化终端不能输入 sudo 密码时，让用户在自己的终端运行。不要保存密码。

## 4. 校验镜像

```bash
IMG=appfactory.wise-paas.com.cn/configcenter/configcenter:<TAG>
docker image inspect "$IMG" \
  --format '{{if .RepoDigests}}{{index .RepoDigests 0}}{{else}}NEVER_PUSHED{{end}}'
timeout 60 docker manifest inspect --verbose "$IMG"
```

本地 RepoDigest 和远端 manifest digest 必须一致。部署后 Pod imageID 也必须等于该 digest。

## 5. 部署

release 不存在时执行 `charts/configcenter/install.sh`。release 已存在时，先取得用户许可，再执行：

```bash
cd charts/configcenter
bash uninstall.sh
bash install.sh
```

```bash
kubectl -n xiaobai get pod -l app.kubernetes.io/instance=configcenter -o wide
kubectl -n xiaobai get deploy configcenter \
  -o jsonpath='{.spec.template.spec.containers[0].image}{"\n"}'
POD=$(kubectl -n xiaobai get pod -l app.kubernetes.io/instance=configcenter \
  -o jsonpath='{.items[0].metadata.name}')
kubectl -n xiaobai get pod "$POD" \
  -o jsonpath='{.status.containerStatuses[0].imageID}{"\n"}'
```

不要依赖 `rollout status`，此集群可能因 watch 问题一直等待。

## 6. 功能验证

```bash
POD=$(kubectl -n xiaobai get pod -l app.kubernetes.io/instance=configcenter \
  -o jsonpath='{.items[0].metadata.name}')
BASE=http://127.0.0.1:10086

kubectl -n xiaobai exec "$POD" -- wget -qO- "$BASE/v1/public/configs"
kubectl -n xiaobai exec "$POD" -- wget -qO- "$BASE/v1/credential"
kubectl -n xiaobai exec "$POD" -- wget -qO- "$BASE/v1/services"

kubectl -n xiaobai exec "$POD" -- wget -qO- \
  --header='Content-Type: application/json' \
  --post-data='{"serviceName":"e2e-test","serviceType":"valkey","parameters":{}}' \
  "$BASE/v1/credential"

kubectl -n xiaobai exec "$POD" -- sh -c 'BODY="{\"serviceName\":\"e2e-test\",\"serviceType\":\"valkey\"}";
  printf "DELETE /v1/credential HTTP/1.1\r\nHost: 127.0.0.1\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s" ${#BODY} "$BODY" | nc 127.0.0.1 10086'

kubectl -n xiaobai logs "$POD" --tail=200 | grep -iE 'error|panic|fatal|schema'
```

K8s 平台确认：

- `platform=k8s`
- `enableSrpPortal=true`
- `enableTwinspace=false`
- public configs 不含 `globalIP`
- 日志不出现周期性的 `global ip`

## 7. 输出结果

报告 tag、远端 digest、Pod imageID、Pod 状态、接口结果、错误日志和未覆盖项。

失败时查看 `kubectl describe pod` 和 Pod 日志。不要自行删除失败资源，先说明原因和建议。
