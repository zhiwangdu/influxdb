# 01. 系统分析（master-1.x）

## 1. 代码分层总览

结合当前仓库结构，可将 InfluxDB 1.x 划分为以下层次：

1. **入口与装配层**
   - `cmd/influxd/*`：服务端主进程、配置装配、生命周期管理。
   - `cmd/influx/*`：CLI 交互入口。
   - `cmd/influx_inspect/*`、`cmd/influx_tools/*`：运维与离线工具链。
2. **协议与服务层**
   - `services/httpd`、`services/udp`、`services/graphite`、`services/collectd`、`services/opentsdb`。
   - 负责多协议接入、鉴权、请求限制、响应序列化。
3. **协调与查询层**
   - `coordinator/*`：写入协调、语句执行依赖。
   - `query/*`：查询执行器、任务管理、控制平面。
4. **存储引擎层**
   - `tsdb/*`：分片、索引、缓存、压缩文件（TSM）与 WAL。
5. **元数据与运维层**
   - `services/meta`、`monitor/*`、`prometheus/*`、`pkg/*` 通用能力。
6. **Flux 与扩展层**
   - `flux/*`、`storage/*`：Flux 相关桥接能力与存储接口。

## 2. 启动路径与关键依赖

`influxd` 启动时由命令入口装配 `run.Server`，并在 Server 中初始化：

- TLS 与配置归一化；
- 元数据客户端（`MetaClient`）；
- `TSDBStore` 与引擎选项；
- `PointsWriter`（写入协调）；
- `QueryExecutor` 与 `StatementExecutor`（查询执行）；
- `Monitor` 与统计信息。

这意味着：**规格设计必须显式声明变更所在层，以及是否影响 Server 装配顺序和默认配置**。

## 3. 关键领域模型（用于规格建模）

- **Data Plane**
  - 写入：协议入口 → 解析 → PointsWriter → TSDB shard/WAL/TSM。
  - 查询：HTTP/CLI → QueryExecutor → shard mapper → engine cursor。
- **Control Plane**
  - 配置加载、动态重载、任务并发控制、限流与超时。
- **Meta Plane**
  - DB/RP/shard group 元数据、节点信息、保留策略。
- **Observability Plane**
  - 监控指标、日志、诊断数据、Prometheus 暴露。

## 4. 高风险改动区（应强制规格化）

1. `coordinator/*` 与 `tsdb/*` 的接口变更。
2. `services/httpd` 请求参数、认证、错误码语义调整。
3. 索引引擎（inmem/tsi）切换策略或默认值调整。
4. 影响 `run.Server` 初始化顺序的改动。
5. `cmd/influx_inspect` / `influx_tools` 输出格式变更（运维兼容性风险）。

## 5. 规格拆分建议（按变更粒度）

- **L1 小变更规格**：单包内部行为、无外部契约变化。
- **L2 跨包规格**：2~4 个模块联动，需契约与回归清单。
- **L3 平台级规格**：启动流程/配置模型/存储格式等，需 ADR + 迁移计划。

## 6. 与业界实践对齐点

- 对齐“Architecture Decision Record (ADR)”：平台级改动必须记录决策与备选方案。
- 对齐“Contract Testing”：服务层参数与响应语义变化必须有契约测试。
- 对齐“Performance Budget”：TSDB/Query 变更必须声明性能预算与测量方法。
- 对齐“Progressive Delivery”：默认通过特性开关灰度启用高风险特性。
