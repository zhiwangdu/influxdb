# 01. `master-1.x` 系统分析（面向 SDD 建模）

## 1. 代码规模快照（基于仓库扫描）

按顶层目录统计 Go 文件数量（用于评估规格影响面）：

- `tsdb`: 149
- `pkg`: 114
- `cmd`: 93
- `services`: 84
- `query`: 56
- `storage`: 43
- 其余目录合计：138
- **总计：677 Go 文件**

> 含义：InfluxDB 1.x 是典型“多子系统强耦合”项目，任何跨层改动都应以规格明确边界和验证策略。

## 2. 分层模型（用于规格里的 `codebase_mapping`）

1. **入口装配层**
   - `cmd/influxd/*`: 服务启动、信号处理、配置装配。
2. **协议服务层**
   - `services/httpd` / `udp` / `graphite` / `collectd` / `opentsdb`。
3. **协调执行层**
   - `coordinator/*`（写入协调）
   - `query/*`（查询执行与任务控制）
4. **存储引擎层**
   - `tsdb/*`（shard/index/WAL/TSM）
5. **生态与可观测层**
   - `monitor/*`、`prometheus/*`、`flux/*`、`storage/*`

## 3. 启动链关键点（规格中应声明影响）

在 `influxd run` 场景中，Server 初始化顺序涉及：

- TLS 配置默认化
- MetaClient 初始化
- TSDBStore 引擎配置
- PointsWriter 与 QueryExecutor 绑定
- Monitor/统计信息注册

若需求影响上述任一阶段，规格需显式写明：

- 启动顺序是否变化
- 失败回滚行为
- 默认配置兼容性

## 4. 高风险改动区域（强制 L2/L3 规格）

- `services/httpd` 请求参数、错误码、鉴权语义变化
- `coordinator` 与 `tsdb` 接口/结构变更
- 索引或存储格式行为（TSI/TSM/WAL）变更
- `cmd/influx_inspect` / `cmd/influx_tools` 输出契约变化

## 5. 规格分级建议

- **L1（包内）**：单包行为改动，无外部契约变化。
- **L2（跨包）**：2~4 个模块联动，必须有契约测试与回归清单。
- **L3（平台级）**：影响启动路径、配置模型或数据兼容，需 ADR + 迁移/回滚计划。

## 6. 与业界实践映射

- ADR（Architecture Decision Record）：L3 必选。
- Contract Testing：服务层改动必选。
- Performance Budget：`query/tsdb` 改动必选。
- Progressive Delivery：高风险特性默认受 feature flag 控制。


## 7. 进一步的架构细节

本文件提供分层与风险视图；若需要按“启动、写入、查询、对象接口、并发状态机”展开，请结合：

- `docs/sdd/06-architecture-deep-dive.md`

建议在 L2/L3 规格中把该文档的对象/接口条目直接映射到 `design` 与 `codebase_mapping` 字段。
