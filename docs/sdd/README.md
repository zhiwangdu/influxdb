# InfluxDB 1.x Specification-Driven Development (SDD) 指南

本目录提供一套**可执行、可评审、可自动校验**的 SDD 文档，目标是让 `master-1.x` 上的人类开发者与 AI Agent 使用统一规格语言协作。

## 1) 适用范围

- 分支：`master-1.x`
- 目标对象：`cmd/*`、`services/*`、`coordinator/*`、`query/*`、`tsdb/*`、`storage/*` 等跨模块改动
- 适用任务：新功能、行为修复、性能优化、兼容性调整

## 2) 文档与资产

- `01-system-analysis.md`：代码库分层分析与高风险区域。
- `02-specification-schema.md`：规格字段定义（面向人类与 AI）。
- `03-workflow-ai-coding.md`：从需求到交付的执行流程。
- `04-quality-gates-and-playbooks.md`：质量门禁与场景化剧本。
- `05-spec-example-write-path.yaml`：完整示例规格。
- `06-architecture-deep-dive.md`：代码流程、对象/接口职责、并发与生命周期深潜。
- `schema/spec.schema.json`：机器可校验 JSON Schema。
- `../../specs/templates/spec.template.yaml`：可复制的规格模板。
- `../../tools/sdd/validate_spec.py`：本地规格校验脚本。

## 3) 最小落地规则（建议纳入团队规范）

1. 任意跨包改动必须先提交规格（`specs/<yyyy>/<spec-id>.yaml`）。
2. PR 标题或正文必须包含 `Spec ID`。
3. 规格必须包含：范围、契约、验证、回滚。
4. 合并前必须做“规格-实现一致性”检查。

## 4) 快速开始

```bash
# 1) 基于模板创建规格
cp specs/templates/spec.template.yaml specs/2026/SPEC-2026-010.yaml

# 2) 校验规格（结构 + 必填字段）
python tools/sdd/validate_spec.py specs/2026/SPEC-2026-010.yaml

# 3) 开发与测试
# 按 03-workflow-ai-coding.md 与 04-quality-gates-and-playbooks.md 执行
```

## 5) 与业界实践对齐

- Spec as Code（规格与代码同仓）
- Contract First（契约先于实现）
- Traceability（需求-规格-代码-测试可追踪）
- Shift-left Validation（前置验证）
- Progressive Delivery（渐进式发布 + 可回滚）
