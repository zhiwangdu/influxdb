# 02. 规格 Schema（面向 AI 自动编码）

本文定义统一的 SDD 规格结构，供人类与 AI 共用。建议每个需求对应一个规格文件：`specs/<yyyy>/<spec-id>.yaml`。

## 1. 规格字段定义

```yaml
id: "SPEC-YYYY-NNN"
title: "一句话描述"
status: draft|review|approved|implemented|verified|released
owner: "team-or-person"
reviewers: ["role-a", "role-b"]

context:
  background: "背景问题"
  goals:
    - "目标1"
    - "目标2"
  non_goals:
    - "非目标1"

scope:
  in_scope:
    - "模块/包/接口"
  out_of_scope:
    - "明确排除项"

codebase_mapping:
  entrypoints:
    - "cmd/influxd"
  packages:
    - "services/httpd"
    - "coordinator"
    - "tsdb"
  configs:
    - "配置键路径"

contracts:
  api:
    - name: "HTTP API or CLI"
      change_type: additive|breaking|internal
      request_contract: "参数/限制"
      response_contract: "状态码/字段/错误语义"
  data:
    - name: "存储或索引契约"
      compatibility: backward|forward|none
      migration: "迁移方式"
  config:
    - key: "[section].field"
      default: "默认值"
      dynamic_reload: true|false
      compatibility: "兼容性说明"

nfr:
  performance:
    baseline: "基线说明"
    target: "目标阈值"
  reliability:
    slo: "可用性/错误率目标"
  security:
    authn_authz: "认证授权影响"
    data_safety: "数据安全影响"
  observability:
    metrics:
      - "新增指标"
    logs:
      - "新增日志"
    traces:
      - "链路追踪要点"

design:
  approach: "核心方案"
  alternatives:
    - option: "备选方案"
      pros: ["..."]
      cons: ["..."]
  risks:
    - risk: "风险描述"
      mitigation: "缓解措施"

implementation_plan:
  milestones:
    - name: "M1"
      tasks:
        - "task-1"
        - "task-2"
  feature_flags:
    - name: "flag_name"
      default: false

validation:
  unit_tests:
    - "包级单测"
  integration_tests:
    - "跨模块验证"
  performance_tests:
    - "压测或 benchmark"
  regression_tests:
    - "历史缺陷回归"
  manual_checks:
    - "必要人工验证"

release_and_rollback:
  rollout: "灰度/分批策略"
  rollback: "回滚步骤"
  data_recovery: "数据恢复策略"

traceability:
  related_issues: ["#123"]
  related_prs: ["#456"]
  affected_docs:
    - "README/配置文档/运维手册"
```

## 2. AI 执行约束（建议）

- AI 必须先解析 `scope` 与 `codebase_mapping`，不得越界改动。
- AI 必须根据 `contracts` 生成验证用例骨架。
- AI 必须在提交说明中引用 `id` 与 `milestone`。
- `change_type=breaking` 时，AI 需阻断自动合并并要求人工评审。

## 3. 最小可执行规格（MVP）

若需求紧急，可最少填写：

- `id/title/context.goals`
- `scope.in_scope/out_of_scope`
- `contracts`（至少一个）
- `implementation_plan.milestones`
- `validation.unit_tests + integration_tests`
- `release_and_rollback`

缺失上述字段时，不建议允许 AI 自动提交。
