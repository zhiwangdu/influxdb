# 03. SDD + AI 自动编码工作流

## 1. 标准流程

1. **Intake**：把业务诉求转成规格草案（`draft`）。
2. **Spec Review**：评审范围、契约、NFR、风险、回滚（通过后 `approved`）。
3. **Planning**：按里程碑切成小步可提交任务。
4. **Implementation**：AI/人工按里程碑实现，禁止越过 `scope`。
5. **Validation**：执行质量门禁（单测/集成/回归/性能/兼容）。
6. **Release**：灰度发布、监控观测、必要时回滚。
7. **Close**：规格状态更新为 `verified/released`，沉淀复盘。

## 2. InfluxDB 1.x 推荐切片

- Slice A：契约与测试先行（API/配置/错误语义）
- Slice B：核心实现（`services|coordinator|query|tsdb`）
- Slice C：可观测补齐（metrics/logs/traces）
- Slice D：运维与文档（`influx_inspect|influx_tools|docs`）

## 3. 给 AI 的最小提示上下文

- 规格 ID 与当前里程碑
- 允许改动路径白名单
- 禁止改动路径黑名单
- 必跑命令与验收断言
- Out-of-Scope 声明

示例：

```text
执行 SPEC-2026-015 M2
允许改动: services/httpd, coordinator
禁止改动: tsdb/engine, cmd/influx_tools
必须测试: go test ./services/httpd ./coordinator
验收: 默认配置下行为向后兼容，旧请求响应不变
```

## 4. 规格到验证映射规则

- 每个 `contracts.api` → 至少 1 条契约测试
- 每个 `risk` → 至少 1 条缓解验证
- 每个 `nfr.performance` → 至少 1 个 benchmark 或压测任务

## 5. 失败与回流

当实现失败（测试失败/性能回退/兼容性破坏）：

1. 先定位受影响契约；
2. 分类为“实现问题/规格缺陷/环境问题”；
3. 若是规格缺陷，先修规格再继续实现；
4. 失败用例进入回归池。

## 6. Definition of Done

- 规格与代码一致；
- 验证证据完整；
- PR 可追踪到 `Spec ID + Milestone`；
- 发布与回滚方案已演练或可执行。
