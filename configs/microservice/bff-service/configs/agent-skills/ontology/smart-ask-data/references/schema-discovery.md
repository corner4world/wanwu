# 步骤 2：Schema 发现（候选对象类与字段）

在 **问数主流程第 2 步** 调用：在已解析的 `kn_id` 下，定位与用户问题相关的 **对象类**（object-type），并取出每个对象类的 **字段** 与 **后端 `dataview-id`**，供编排层 LLM 在第 3 步生成 SQL 使用。

> **执行委托**：本文件仅 **描述** 命令形态；**Never** 由 smart-ask-data 直接执行 `ontology` CLI，实际调用统一由 [ontology-core](../../ontology-core/SKILL.md) 完成。

## 与本流程的衔接

- **输入**：步骤 1 选定的 `kn_id`。
- **产出**：交给编排层 LLM 的"schema 摘要"——
  - 相关 object-type 列表（id / name / description）
  - 每个相关 object-type 的字段清单（properties）
  - 每个 object-type 后端绑定的 `dataview-id`（步骤 4 SQL 执行需要）

## 命令形态（委托 ontology-core 执行）

### 2.1 用语义检索定位相关对象类（推荐起步）

```bash
ontology --user-id <accountId> bkn search <kn-id> "<中文问数意图>" \
  [--max-concepts 10] [--mode keyword_vector_retrieval] [-bd bd_public] [--pretty]
```

- 返回该 KN 内匹配的概念（object-type / relation-type / action-type）。
- 用于在大 KN 中快速锁定 3–5 个相关 object-type，避免直接 `list` 后由 LLM 扫全量。
- `--max-concepts` 默认 10；按问题复杂度调到 5–20 即可。

### 2.2 列出全部对象类（兜底 / 小 KN）

```bash
ontology --user-id <accountId> bkn object-type list <kn-id> [-bd bd_public] [--pretty]
```

- 适合 KN 不大、或 `bkn search` 匹配不准时的兜底；返回 KN 中所有 object-type 的 schema 摘要。

### 2.3 取单个对象类的字段与 dataview-id

对步骤 2.1 / 2.2 选出的每个相关 object-type，取详情：

```bash
ontology --user-id <accountId> bkn object-type get <kn-id> <ot-id> [-bd bd_public] [--pretty]
```

返回（以网关为准）通常包含：

- `id` / `name` / `display_key` / `primary_key`
- `dataview_id`（**关键** — 步骤 4 `dataview query --sql` 必须用它）
- `properties`：字段名 + 类型 + 描述
- `tags` / `comment` / 关联关系类型 等

## LLM 交付契约（给编排层）

smart-ask-data 把以下结构化"schema 摘要"交回给 smart-data-analysis 的 LLM 用于生成 SQL：

```text
KN: <kn_id> (<kn_name>)
相关对象类：
  - <ot-id-1> (<name>)
      dataview_id: <dv-id-1>
      字段: <prop1> (<type>, <desc>), <prop2> (...), ...
  - <ot-id-2> (<name>)
      dataview_id: <dv-id-2>
      字段: ...
关系（如有）:
  - <ot-id-1> -[<rel-name>]→ <ot-id-2>
```

- 字段表必须 **来自命令返回**，不得编造或脑补类型/含义。
- 若 `dataview_id` 缺失或为空，对应 object-type **不能** 用于步骤 4 SQL 执行；需要换 object-type 或换 KN。

## 注意事项

- `--user-id <accountId>` **必传**。
- 一次 `bkn search` 通常足够；只在结果明显不准时再 `bkn object-type list` 全量。
- 若 `bkn search` 返回为空：先复述用户问题、把"业务对象 / 指标 / 时间范围"展开后再搜；仍空则提示用户该 KN 不含相关概念，请换 KN。
- **禁止** 强行在不匹配的 KN 上继续 schema 发现；中止本任务并回到步骤 1 重选 KN，或让用户改换 KN。
- 命令报错时直接如实反馈给用户，不要伪造结果或尝试登录刷新凭证。本部署 ontology CLI **无须 token**。
