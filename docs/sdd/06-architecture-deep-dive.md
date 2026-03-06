# 06. 架构深潜：整体代码流程、核心对象与接口职责

> 本文用于补强 SDD 在“架构级理解”上的输入质量，帮助 AI 在改动前先定位调用链、对象边界与契约位置。

## 1. 端到端总体流程（运行时）

### 1.1 进程与命令入口

1. `cmd/influxd/main.go` 中 `Main.Run` 解析子命令，默认进入 `run`。  
2. `run.NewCommand().Run(...)` 负责加载配置、初始化 logger、组装 `run.Server`。  
3. 主循环监听 `SIGTERM` / `SIGHUP`，触发优雅关闭或配置重载。  

**架构含义**：

- 改启动逻辑时应优先检查 `Main.Run` 与 `Command.Run` 的协作边界。
- 动态配置能力以 `ReloadConfig -> Server.ApplyReloadedConfig` 为准，不应默认假设“所有配置可热更新”。

### 1.2 Server 装配与服务注册

`run.NewServer` 会在启动阶段组装关键依赖：

- 基础对象：`MetaClient`、`TSDBStore`、`PointsWriter`、`QueryExecutor`、`Monitor`。
- 协议服务：HTTP、UDP、Graphite、Collectd、OpenTSDB 等通过 `append*Service` 注入。
- 统一生命周期：所有服务遵循 `Service` 接口（`WithLogger/Open/Close`）。

**架构含义**：

- 这是 InfluxDB 1.x 的“组合根（Composition Root）”。
- 新服务或跨模块依赖注入，应优先在 Server 装配层完成，避免在 handler 内部做隐式初始化。

---

## 2. 写入链路（HTTP 为例）

### 2.1 请求路径

`httpd.Handler` 在 `serveWriteV1/serveWrite` 处理写入请求，随后调用 `PointsWriter` 执行协调写入。

典型调用链：

`HTTP /write -> Handler.serveWrite -> PointsWriter.WritePointsPrivileged -> writeToShard -> TSDBStore.WriteToShard -> Shard.WritePoints`

### 2.2 核心对象职责

- **`services/httpd.Handler`**：协议适配、参数解析、鉴权与错误码映射。
- **`coordinator.PointsWriter`**：按 shard 映射点集、并发写入、超时控制、部分失败语义。
- **`tsdb.Store`**：按 shard 落盘写入，处理 store/shard 生命周期与 epoch 冲突保护。

### 2.3 关键语义点（规格必须显式声明）

- `PointsWriter` 可能返回 `ErrTimeout` / `ErrWriteFailed` / `PartialWriteError`。
- `Store.WriteToShard` 在 shard 不存在时返回 `ErrShardNotFound`，上层可能触发“先建 shard 再重试”。
- 写入上下文 `tsdb.WriteContext` 记录用户来源（鉴权、系统写入来源、指标分桶）。

---

## 3. 查询链路（HTTP `/query` 为例）

### 3.1 请求路径

`Handler.serveQuery` 执行参数解析、查询解析、鉴权，然后交给 `query.Executor.ExecuteQuery`。

典型调用链：

`HTTP /query -> Handler.serveQuery -> QueryExecutor.ExecuteQuery -> StatementExecutor.ExecuteStatement -> TSDBStore/MetaClient`

### 3.2 核心对象职责

- **`query.Executor`**：查询任务编排、并发与超时控制、panic 恢复统计。
- **`coordinator.StatementExecutor`**：按语句类型（DDL/DML/SELECT 等）执行，连接元数据与存储。
- **`query.TaskManager`**：查询生命周期管理（并发上限、超时、慢查询追踪）。

### 3.3 授权模型

- **`CoarseAuthorizer`**：数据库级权限判断。
- **`FineAuthorizer`**：序列级读写授权（OSS 下通常为开放实现）。

**SDD 要求**：涉及授权变化时，规格要明确“粗粒度/细粒度”分别受影响的行为。

---

## 4. 对象与接口目录（建议写入规格 `design`）

## 4.1 Server 侧生命周期接口

- `run.Service`
  - `WithLogger(log)`
  - `Open() error`
  - `Close() error`

该接口定义了所有附加服务的统一生命周期契约。

### 4.2 查询执行接口

- `query.StatementExecutor`
  - `ExecuteStatement(ctx, stmt)`
- `query.StatementNormalizer`
  - `NormalizeStatement(stmt, db, rp)`

这组接口把“解析后语句的执行”与“语句归一化”解耦，便于替换执行策略。

### 4.3 协调层存储接口

- `coordinator.TSDBStore`
  - 包含 `CreateShard` / `WriteToShard` / 删除 / 基数统计等能力。

该接口是 coordinator 与 tsdb 的核心边界，跨层改动建议优先通过该接口演进（避免直接耦合具体实现）。

### 4.4 HTTP 层存储接口（Flux/Storage 桥接）

- `services/httpd.Store`
  - `ReadFilter` / `Delete` / `DeleteRetentionPolicy`

用于 HTTP 层访问 storage 子系统的最小能力抽象。

---

## 5. 关键状态与并发控制点

### 5.1 启停状态

- 进程级：`Main.Run` 的信号循环控制优雅退出。
- 服务级：`Server.Open/Close` 管理 `Services` 列表中所有服务生命周期。

### 5.2 写入并发

- `PointsWriter` 对 shard 进行并发 fan-out，受 `WriteTimeout` 控制。
- `tsdb.Store` 通过 epoch tracker 序列化可能冲突的写/删操作，降低并发冲突风险。

### 5.3 查询并发

- `TaskManager` 管理最大并发查询数与超时。
- 终止信号时可输出当前运行查询（受配置控制）。

---

## 6. 规格编写时的“架构映射”模板

在 `specs/<yyyy>/<id>.yaml` 中建议新增/补齐：

```yaml
architecture_mapping:
  runtime_flow:
    - "influxd/main -> run.Command -> run.Server"
    - "httpd.Handler -> coordinator -> tsdb"
  touched_objects:
    - name: "PointsWriter"
      role: "shard fan-out + timeout + partial write semantics"
    - name: "StatementExecutor"
      role: "query statement dispatcher"
  touched_interfaces:
    - name: "coordinator.TSDBStore"
      change: "none|additive|breaking"
  concurrency_model:
    - "write fan-out + timeout"
    - "task manager concurrency limit"
```

这样可以让 AI 在编码前先完成架构对齐，降低“改对文件但改错层”的风险。

---

## 7. 架构改动的 SDD 审查清单

1. 是否改动了 `run.Server` 装配顺序？
2. 是否触及 `coordinator.TSDBStore` 接口契约？
3. 写入错误语义是否变化（400/500/partial write/timeout）？
4. 查询授权路径是否变化（coarse/fine）？
5. 是否补充了并发、性能、回滚验证？

若任一项为“是”，建议至少按 L2 规格执行，并补契约测试。
