# 02. 规格 Schema（AI 自动编码输入契约）

本文件定义 SDD 规格的**语义字段**，并配套机器校验文件：

- `docs/sdd/schema/spec.schema.json`

建议每个需求对应：`specs/<yyyy>/<spec-id>.yaml`。

## 1. 字段设计原则

- `scope`：约束 AI 可修改范围（防止越界修改）
- `contracts`：显式声明 API/配置/数据契约
- `nfr`：将性能、可靠性、安全、可观测要求结构化
- `validation`：把测试计划写入规格本体
- `release_and_rollback`：强制发布与回退策略

## 2. 规格 YAML 示例骨架

```yaml
id: "SPEC-YYYY-NNN"
title: "一句话描述"
status: draft|review|approved|implemented|verified|released
owner: "team-or-person"
reviewers: ["role-a", "role-b"]

context:
  background: "背景问题"
  goals: ["目标1", "目标2"]
  non_goals: ["非目标1"]

scope:
  in_scope: ["services/httpd", "coordinator"]
  out_of_scope: ["tsdb format changes"]

codebase_mapping:
  entrypoints: ["cmd/influxd"]
  packages: ["services/httpd", "coordinator", "query"]
  configs: ["[http].*"]

contracts:
  api:
    - name: "POST /write"
      change_type: additive|breaking|internal
      request_contract: "参数与限制"
      response_contract: "状态码与错误语义"
  data:
    - name: "存储/索引契约"
      compatibility: backward|forward|none
      migration: "迁移策略"
  config:
    - key: "[section].field"
      default: "默认值"
      dynamic_reload: true|false
      compatibility: "兼容性说明"

nfr:
  performance:
    baseline: "基线"
    target: "目标"
  reliability:
    slo: "SLO"
  security:
    authn_authz: "影响"
    data_safety: "影响"
  observability:
    metrics: ["metric_a"]
    logs: ["log_a"]
    traces: ["trace_a"]

design:
  approach: "方案"
  alternatives:
    - option: "备选"
      pros: ["..."]
      cons: ["..."]
  risks:
    - risk: "风险"
      mitigation: "缓解"

implementation_plan:
  milestones:
    - name: "M1"
      tasks: ["task-1", "task-2"]
  feature_flags:
    - name: "flag_name"
      default: false

validation:
  unit_tests: ["..."]
  integration_tests: ["..."]
  performance_tests: ["..."]
  regression_tests: ["..."]
  manual_checks: ["..."]

release_and_rollback:
  rollout: "灰度策略"
  rollback: "回滚步骤"
  data_recovery: "数据恢复"

traceability:
  related_issues: ["#123"]
  related_prs: ["#456"]
  affected_docs: ["README", "配置文档"]
```

## 3. AI 约束（建议）

- AI 必须先读取 `scope.in_scope` 再进行改动。
- AI 必须为每条 `contracts.api` 生成至少 1 条契约测试。
- 若存在 `change_type=breaking`，需触发人工评审并阻断自动合并。
- AI 产出必须引用 `id + milestone`。

## 4. 最小可执行规格（MVP）

最低必填：

- `id/title/context.goals`
- `scope`
- `contracts`（至少一类）
- `implementation_plan.milestones`
- `validation.unit_tests + integration_tests`
- `release_and_rollback`
