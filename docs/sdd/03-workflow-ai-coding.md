# 03. AI 自动编码工作流（SDD）

## 1. 端到端流程

1. **需求受理（Intake）**
   - 输入：问题陈述、业务目标、限制条件。
   - 输出：规格草案（`status=draft`）。
2. **规格评审（Spec Review）**
   - 校验范围、契约、NFR、风险与回滚。
   - 通过后置为 `approved`。
3. **任务切片（Planning）**
   - 将规格映射为里程碑与可提交最小增量（small PRs）。
4. **AI 编码（Implementation）**
   - 按里程碑逐步生成代码与测试，禁止跨规格漂移。
5. **验证与门禁（Validation）**
   - 单测、集成、回归、性能、兼容性校验。
6. **发布与复盘（Release & Learn）**
   - 灰度发布，监控异常，更新规格状态与经验库。

## 2. 针对 InfluxDB 1.x 的任务切片模板

- Slice A：接口/配置契约（优先）
- Slice B：核心逻辑实现（coordinator/query/tsdb）
- Slice C：观测性补齐（metrics/logs）
- Slice D：工具链与文档更新（inspect/tools/docs）

> 原则：先落契约和测试，再落实现细节。

## 3. AI 提示词（Prompt）规范建议

每次让 AI 执行编码时，传入以下最小上下文：

- 规格 ID 与目标里程碑；
- 允许改动路径白名单；
- 禁止改动路径黑名单；
- 必跑测试命令；
- 验收标准（Given/When/Then 或输入/输出断言）。

示例：

```text
你正在执行 SPEC-2026-012 的 M2。
允许改动：services/httpd, coordinator
禁止改动：tsdb/engine, cmd/influx_tools
必须新增：契约测试 + 回归测试
完成标准：新增参数在默认配置下向后兼容；老请求无行为变化。
```

## 4. 规格到测试的映射规则

- 每条 `contracts.api` 至少对应 1 个契约测试。
- 每条 `nfr.performance` 至少对应 1 个 benchmark 或压测任务。
- 每个 `risk` 至少对应 1 条缓解验证（测试/监控/开关）。

## 5. 失败处理（Failure Handling）

当 AI 实施失败（测试不通过、依赖冲突、性能回退）时：

1. 回到规格定位受影响契约。
2. 标注失败分类：实现缺陷 / 规格不完整 / 环境限制。
3. 如属规格问题，先修订规格并再次评审。
4. 保留失败用例，加入回归测试池。

## 6. Definition of Done（DoD）

- 规格状态更新到 `implemented` 或 `verified`。
- 代码、测试、文档与规格一致。
- 质量门禁全部通过。
- PR 描述能追踪到规格字段与里程碑。
