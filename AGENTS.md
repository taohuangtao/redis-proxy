# AGENTS.md

本文件为 CodeBuddy（及其他 AI 编码助手）在本仓库中工作时提供指引。

## 项目概述

一个用 Go 编写的 **Redis TCP 代理**：对外暴露一个 Redis 端口，把客户端连接透明转发到**当前主节点**；同时内置一个基于 ZooKeeper 的简易哨兵，可自动完成主从选举与故障切换。项目定位是"零侵入"——客户端只需连代理地址，无需感知主从切换。

## 常用命令

```bash
# 构建（模块名 redis-proxy，Go 1.23.5；Dockerfile 中使用 CGO_ENABLED=0）
go build -o redis-proxy .

# 直接运行（配置全部来自环境变量，见下表）
LOCAL_ADDR=0.0.0.0:6379 REDIS_HOSTS=192.168.1.11:6379,192.168.1.12:6379 \
REDIS_PASSWORD=123456 ZK_HOSTS=192.168.1.21:2181 RUN_REDIS_SENTINEL=true go run .

# 格式化 / 静态检查（仓库无 linter 配置、无测试文件）
gofmt -l . && go vet ./...

# 运行测试（目前没有 _test.go，下面命令会提示 "no test files"）
go test ./...
go test -run <TestName> -v ./src/...      # 运行单个测试

# 构建并推送镜像（见 build_docker.sh，推送到 hub.pengbei.tech:18080/bigdata/redis-proxy）
sh build_docker.sh
docker build -t redis-proxy . --progress=plain

# 查看运行时状态（当前认定的 master 与候选节点列表，JSON）
curl http://localhost:6060/debug/proxy/runinfo

# pprof / 调试端点同样挂在 :6060（main.go 中 _ "net/http/pprof"）
curl http://localhost:6060/debug/pprof/goroutine?debug=1
```

### 环境变量（`src/config/config.go` 在 `init()` 中一次性读取，运行中修改无效）

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `LOCAL_ADDR` | `127.0.0.1:28080` | 代理监听地址 |
| `REDIS_HOSTS` | `127.0.0.1:6379` | 候选 Redis 节点，逗号分隔 |
| `REDIS_PASSWORD` | 空 | Redis 密码（`requirepass`/`masterauth`） |
| `ZK_HOSTS` | `127.0.0.1:2181` | ZK 地址，逗号分隔；仅内置哨兵使用 |
| `RUN_REDIS_SENTINEL` | `false` | 是否启用内置哨兵；已有外部集群/哨兵时必须为 false |
| `DEBUG` | `false` | 打开后会打印大量探测日志（并影响 logger 格式） |

容器化部署参考 `Dockerfile`（ubuntu 构建阶段 + alpine 运行阶段，工作目录 `/data`）与 `master_slave.yaml`（K8s StatefulSet + 无头 Service 部署 3 个 Redis 实例的示例）。

## 架构

### 进程结构

`main.go` 只做两件事：在 goroutine 中启动 `src.RunProxy(...)`，然后在主 goroutine 阻塞运行 `http.ListenAndServe(":6060", nil)`（pprof 默认 mux）。因此**代理逻辑跑在后台协程**，HTTP 端口同时承担 pprof 与 `/debug/proxy/runinfo` 状态查询；该处理器由 `RunProxy` 内部注册。

代码分为三层：

- `src/config/config.go` — 纯配置层。包级变量在 `init()` 中从环境变量读取，全局只读。新增配置项时遵循"环境变量 → getter → 包级变量"的模式。
- `src/redisTools.go` — Redis 探测层。`check()` 建立 `PoolSize: 1` 的 go-redis 客户端（禁用连接池，因为每次探测都是一次性短连接），执行 `PING` 并 `INFO replication`，通过字符串包含 `role:master` 判定角色；`GetMaster()` 线性扫描候选列表，返回**第一个**可用的 master，全部不可用时返回空串。
- `src/proxy.go` — 数据面。`listener.Accept()` 每收到一个连接就调用 `getMaster()` 取当前主节点，然后 `go handleConnection(clientConn, remoteAddr)`。`handleConnection` 用两次 `io.Copy` 做纯字节流双向转发（**不解析 RESP 协议**，所以任何 Redis 命令、协议版本、甚至非 Redis 的 TCP 流量都能透传）。
- `src/redisSentinel.go` — 控制面（可选）。详见下文。

### Master 发现（每个代理实例独立进行）

`RunProxy` 启动 `checkMaster()` goroutine：先同步执行一次 `GetMaster`，随后每 10 秒刷新一次全局 `redisInfo.Master_host`，读写用 `rwLock sync.RWMutex` 保护。新连接只取**建连瞬间**的 master 快照，不做 per-command 路由——这意味着切换期间已建立的连接会跟随旧 master 一起失效，客户端重连才能落到新 master。

### 内置哨兵与 ZK 选主

启用条件：`RUN_REDIS_SENTINEL=true`。多个代理实例可同时运行哨兵逻辑，但通过 ZK 的**临时节点** `/redis_ht_sentinel/task_lock` 做互斥，保证同一时刻只有一个实例真正执行切换；ZK 根路径 `/redis_ht_sentinel` 在首次连接时自动创建；选出的 master 持久化在 `/redis_ht_sentinel/last_master`。

`RunSentinel()` 是无限循环，每轮新建 `Task`（重新连接 ZK、重新抢锁），执行完 `clear()` 关闭连接，再随机睡眠 1~11 秒——随机化用于降低多实例抢锁的惊群。

单次 `Task._run()` 的决策流程：

1. 读取 `last_master`，若该节点仍存活且 `role:master` 则继续沿用（保证稳定性，避免不必要的切换）；
2. 遍历 `REDIS_HOSTS` 全部节点跑 `INFO replication`，收集 slave 列表；若发现**多个** master（说明主从已分裂），保留第一个、对多余的发 `SLAVEOF <master> <port>` 重新挂回去；
3. 若一个 master 都没有，则从 slave 列表中取第一个执行 `SLAVEOF NO ONE` 提升为 master；
4. 把最终 master 写回 `last_master`（不存在则 Create，已存在则 Set）；
5. 遍历所有 slave，若 `INFO replication` 中的 `master_host`/`master_port` 与目标不一致，重新下发 `SLAVEOF`。

### 已知行为与易踩坑点

改动下列位置时需格外小心（均为现有实现的有意或遗留行为）：

- `GetMaster()` 可能返回空串，此时 `handleConnection` 的 `net.Dial` 会失败并静默关闭连接；排查"代理连不上"时先看 `/debug/proxy/runinfo` 的 `Master_host` 是否为空。
- 哨兵在没有节点存活时会访问 `slaveList[0]`，空列表会 panic。
- `Task.lock()` 对 `ErrNodeExists` 之外的错误也返回 `true`（会继续跑 `_run`），严格互斥依赖 ZK 正常。
- 循环体内大量使用 `defer rdb.Close()`，实际在函数返回时才关闭；这是既有风格，改动时注意不要改变语义。
- 根目录下 `sentinelTest.go` 并非 Go 测试文件（内部只是 `func _main()` 手动入口），`go test ./...` 不会执行它；`n.java` 是早期 Netty 版 TCP 代理原型，与 Go 实现无关，均为遗留参考文件。
- 依赖极少：`github.com/go-redis/redis/v8`、`github.com/samuel/go-zookeeper/zk`（及其传递依赖）。新增依赖后需同步确认 `Dockerfile` 中 `go mod download` 使用的 `GOPROXY=https://goproxy.cn` 能拉取。
