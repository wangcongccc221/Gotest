# > /code-review
_Exported on 06/18/2026 at 11:22:50 GMT+8 from Claude via WayLog_


**User**

> /code-review

---


**User**

Review target: `你帮我核对一下鸿蒙的更多设置的几个功能 和48的数据源  指令下发 等 是不是一样  但是你不要修改代码 只需要帮我review 不一样的 告诉我 可以吗？`

`xhigh effort → 5+5 angles × 8 candidates → 1-vote verify → sweep → ≤15 findings`

You are reviewing for **recall** at extra-high effort: catch every real bug. At
this level, catching real bugs matters more than avoiding false positives — a
missed bug ships. Err on the side of surfacing.

## Phase 0 — Gather the diff

Run `git diff @{upstream}...HEAD` (or `git diff main...HEAD` / `git diff HEAD~1`
if there's no upstream) to get the unified diff under review. If there are
uncommitted changes, or the range diff is empty, also run `git diff HEAD` and
include the working-tree changes in scope — the review often runs before the
commit. If a PR number, branch name, or file path was passed as an argument,
review that target instead. Treat this diff as the review scope.

## Phase 1 — Find candidates (5 correctness angles + 3 cleanup angles + 1 altitude angle + 1 conventions angle, up to 8 each)

Run **10 independent finder angles** via the Agent tool. Each
surfaces **up to 8 candidate findings**. Do NOT let one angle's conclusions
suppress another's — if two angles flag the same line for different reasons,
record both.

### Angle A — line-by-line diff scan

Read every hunk in the diff, line by line. Then Read the enclosing function for
each hunk — bugs in unchanged lines of a touched function are in scope (the PR
re-exposes or fails to fix them). For every line ask: what input, state, timing,
or platform makes this line wrong? Look for inverted/wrong conditions,
off-by-one, null/undefined deref, missing `await`, falsy-zero checks,
wrong-variable copy-paste, error swallowed in catch, unescaped regex metachars.

### Angle B — removed-behavior auditor

For every line the diff DELETES or replaces, name the invariant or behavior it
enforced, then search the new code for where that invariant is re-established.
If you can't find it, that's a candidate: a removed guard, a dropped error
path, a narrowed validation, a deleted test that was covering a real case.

### Angle C — cross-file tracer

For each function the diff changes, find its callers (Grep for the symbol) and
check whether the change breaks any call site: a new precondition, a changed
return shape, a new exception, a timing/ordering dependency. Also check callees:
does a parallel change in the same PR make a call unsafe?

### Angle D — language-pitfall specialist

Scan for the classic pitfalls of the diff's language/framework — for example:
JS falsy-zero, `==` coercion, closure-captured loop var; Python mutable default
args, late-binding closures; Go nil-map write, range-var capture; SQL injection;
timezone/DST drift; float equality. Flag any instance the diff introduces.

### Angle E — wrapper/proxy correctness

When the PR adds or modifies a type that wraps another (cache, proxy, decorator,
adapter): check that every method routes to the wrapped instance and not back
through a registry/session/global — e.g. a caching provider holding a
`delegate` field that resolves IDs via `session.get(...)` instead of
`delegate.get(...)` will re-enter the cache or recurse. Also check that the
wrapper forwards all the methods the callers actually use.

### Reuse

The angles above hunt for bugs; this one and the next two hunt for cleanup in
the changed code. Flag new code that re-implements something the codebase
already has — Grep shared/utility modules and files adjacent to the change,
and name the existing helper to call instead.

### Simplification

Flag unnecessary complexity the diff adds: redundant or derivable state,
copy-paste with slight variation, deep nesting, dead code left behind. Name
the simpler form that does the same job.

### Efficiency

Flag wasted work the diff introduces: redundant computation or repeated I/O,
independent operations run sequentially, blocking work added to startup or
hot paths. Also flag long-lived objects built from closures or captured
environments — they keep the entire enclosing scope alive for the object's
lifetime (a memory leak when that scope holds large values); prefer a
class/struct that copies only the fields it needs. Name the cheaper
alternative.

### Altitude

Check that each change is implemented at the right depth, not as a fragile
bandaid. Special cases layered on shared infrastructure are a sign the fix
isn't deep enough — prefer generalizing the underlying mechanism over adding
special cases.

### Conventions (CLAUDE.md)

Find the CLAUDE.md files that govern the changed code: the user-level
~/.claude/CLAUDE.md, the repo-root CLAUDE.md, plus any CLAUDE.md or
CLAUDE.local.md in a directory that is an ancestor of a changed file (a
directory's CLAUDE.md only applies to files at or below it). Read each one
that exists, then check the diff for clear violations of the rules they state.

Only flag a violation when you can quote the exact rule and the exact line
that breaks it — no style preferences, no vague "spirit of the doc"
inferences. In the finding, name the CLAUDE.md path and quote the rule so the
report can cite it. If no CLAUDE.md applies, return nothing for this angle.

Cleanup, altitude, and conventions candidates use the same
`file`/`line`/`summary` shape; in `failure_scenario`, state the concrete
cost (what is duplicated, wasted, harder to maintain, or which CLAUDE.md rule
is broken) instead of a crash. Correctness bugs always outrank cleanup,
altitude, and conventions findings when the output cap forces a cut.

## Phase 2 — Verify (1-vote, 3-state)

Dedup candidates that point at the same line/mechanism, keeping the one with
the most concrete failure scenario. For each remaining candidate, run **one
verifier** via the Agent tool: give it the diff, the relevant
file(s), and the candidate, and have it return exactly one of:

- **CONFIRMED** — can name the inputs/state that trigger it and the wrong
  output or crash. Quote the line.
- **PLAUSIBLE** — mechanism is real, trigger is uncertain (timing, env,
  config). State what would confirm it.
- **REFUTED** — factually wrong (code doesn't say that) or guarded elsewhere.
  Quote the line that proves it.

Keep candidates where the vote is CONFIRMED or PLAUSIBLE.

This is recall mode — a single non-REFUTED vote carries the finding. Do NOT
drop on uncertainty.

## Phase 3 — Sweep for gaps

Run **one more finder** as a fresh reviewer who has the verified list. Re-read
the diff and enclosing functions looking ONLY for defects not already listed.
Do not re-derive or re-confirm anything already there — the job is gaps. Focus
on what the first pass tends to miss: moved/extracted code that dropped a guard
or anchor; second-tier footguns (dataclass default evaluated once, `hash()`
non-determinism, lock-scope shrink, predicate methods with side effects);
setup/teardown asymmetry in tests; config defaults flipped.

Surface **up to 8 additional candidates**, each naming a defect not already on
the list. If nothing new, return an empty sweep — do not pad.

## Output

Return findings as a JSON array of at most 15 objects:

```json
[
  {
    "file": "path/to/file.ext",
    "line": 123,
    "summary": "one-sentence statement of the bug",
    "failure_scenario": "concrete inputs/state → wrong output/crash"
  }
]
```

Ranked most-severe first. If more than 15 survive, keep the 15 most
severe. If nothing survives verification, return `[]`.

---


**Claude**

API Error: Request rejected (429) · Service Unavailable

---


**User**

> /code-review

---


**User**

Review target: `你帮我核对一下鸿蒙的更多设置的几个功能 和48的数据源  指令下发 等 是不是一样  但是你不要修改代码 只需要帮我review 不一样的 告诉我 可以吗？
API Error`

`xhigh effort → 5+5 angles × 8 candidates → 1-vote verify → sweep → ≤15 findings`

You are reviewing for **recall** at extra-high effort: catch every real bug. At
this level, catching real bugs matters more than avoiding false positives — a
missed bug ships. Err on the side of surfacing.

## Phase 0 — Gather the diff

Run `git diff @{upstream}...HEAD` (or `git diff main...HEAD` / `git diff HEAD~1`
if there's no upstream) to get the unified diff under review. If there are
uncommitted changes, or the range diff is empty, also run `git diff HEAD` and
include the working-tree changes in scope — the review often runs before the
commit. If a PR number, branch name, or file path was passed as an argument,
review that target instead. Treat this diff as the review scope.

## Phase 1 — Find candidates (5 correctness angles + 3 cleanup angles + 1 altitude angle + 1 conventions angle, up to 8 each)

Run **10 independent finder angles** via the Agent tool. Each
surfaces **up to 8 candidate findings**. Do NOT let one angle's conclusions
suppress another's — if two angles flag the same line for different reasons,
record both.

### Angle A — line-by-line diff scan

Read every hunk in the diff, line by line. Then Read the enclosing function for
each hunk — bugs in unchanged lines of a touched function are in scope (the PR
re-exposes or fails to fix them). For every line ask: what input, state, timing,
or platform makes this line wrong? Look for inverted/wrong conditions,
off-by-one, null/undefined deref, missing `await`, falsy-zero checks,
wrong-variable copy-paste, error swallowed in catch, unescaped regex metachars.

### Angle B — removed-behavior auditor

For every line the diff DELETES or replaces, name the invariant or behavior it
enforced, then search the new code for where that invariant is re-established.
If you can't find it, that's a candidate: a removed guard, a dropped error
path, a narrowed validation, a deleted test that was covering a real case.

### Angle C — cross-file tracer

For each function the diff changes, find its callers (Grep for the symbol) and
check whether the change breaks any call site: a new precondition, a changed
return shape, a new exception, a timing/ordering dependency. Also check callees:
does a parallel change in the same PR make a call unsafe?

### Angle D — language-pitfall specialist

Scan for the classic pitfalls of the diff's language/framework — for example:
JS falsy-zero, `==` coercion, closure-captured loop var; Python mutable default
args, late-binding closures; Go nil-map write, range-var capture; SQL injection;
timezone/DST drift; float equality. Flag any instance the diff introduces.

### Angle E — wrapper/proxy correctness

When the PR adds or modifies a type that wraps another (cache, proxy, decorator,
adapter): check that every method routes to the wrapped instance and not back
through a registry/session/global — e.g. a caching provider holding a
`delegate` field that resolves IDs via `session.get(...)` instead of
`delegate.get(...)` will re-enter the cache or recurse. Also check that the
wrapper forwards all the methods the callers actually use.

### Reuse

The angles above hunt for bugs; this one and the next two hunt for cleanup in
the changed code. Flag new code that re-implements something the codebase
already has — Grep shared/utility modules and files adjacent to the change,
and name the existing helper to call instead.

### Simplification

Flag unnecessary complexity the diff adds: redundant or derivable state,
copy-paste with slight variation, deep nesting, dead code left behind. Name
the simpler form that does the same job.

### Efficiency

Flag wasted work the diff introduces: redundant computation or repeated I/O,
independent operations run sequentially, blocking work added to startup or
hot paths. Also flag long-lived objects built from closures or captured
environments — they keep the entire enclosing scope alive for the object's
lifetime (a memory leak when that scope holds large values); prefer a
class/struct that copies only the fields it needs. Name the cheaper
alternative.

### Altitude

Check that each change is implemented at the right depth, not as a fragile
bandaid. Special cases layered on shared infrastructure are a sign the fix
isn't deep enough — prefer generalizing the underlying mechanism over adding
special cases.

### Conventions (CLAUDE.md)

Find the CLAUDE.md files that govern the changed code: the user-level
~/.claude/CLAUDE.md, the repo-root CLAUDE.md, plus any CLAUDE.md or
CLAUDE.local.md in a directory that is an ancestor of a changed file (a
directory's CLAUDE.md only applies to files at or below it). Read each one
that exists, then check the diff for clear violations of the rules they state.

Only flag a violation when you can quote the exact rule and the exact line
that breaks it — no style preferences, no vague "spirit of the doc"
inferences. In the finding, name the CLAUDE.md path and quote the rule so the
report can cite it. If no CLAUDE.md applies, return nothing for this angle.

Cleanup, altitude, and conventions candidates use the same
`file`/`line`/`summary` shape; in `failure_scenario`, state the concrete
cost (what is duplicated, wasted, harder to maintain, or which CLAUDE.md rule
is broken) instead of a crash. Correctness bugs always outrank cleanup,
altitude, and conventions findings when the output cap forces a cut.

## Phase 2 — Verify (1-vote, 3-state)

Dedup candidates that point at the same line/mechanism, keeping the one with
the most concrete failure scenario. For each remaining candidate, run **one
verifier** via the Agent tool: give it the diff, the relevant
file(s), and the candidate, and have it return exactly one of:

- **CONFIRMED** — can name the inputs/state that trigger it and the wrong
  output or crash. Quote the line.
- **PLAUSIBLE** — mechanism is real, trigger is uncertain (timing, env,
  config). State what would confirm it.
- **REFUTED** — factually wrong (code doesn't say that) or guarded elsewhere.
  Quote the line that proves it.

Keep candidates where the vote is CONFIRMED or PLAUSIBLE.

This is recall mode — a single non-REFUTED vote carries the finding. Do NOT
drop on uncertainty.

## Phase 3 — Sweep for gaps

Run **one more finder** as a fresh reviewer who has the verified list. Re-read
the diff and enclosing functions looking ONLY for defects not already listed.
Do not re-derive or re-confirm anything already there — the job is gaps. Focus
on what the first pass tends to miss: moved/extracted code that dropped a guard
or anchor; second-tier footguns (dataclass default evaluated once, `hash()`
non-determinism, lock-scope shrink, predicate methods with side effects);
setup/teardown asymmetry in tests; config defaults flipped.

Surface **up to 8 additional candidates**, each naming a defect not already on
the list. If nothing new, return an empty sweep — do not pad.

## Output

Return findings as a JSON array of at most 15 objects:

```json
[
  {
    "file": "path/to/file.ext",
    "line": 123,
    "summary": "one-sentence statement of the bug",
    "failure_scenario": "concrete inputs/state → wrong output/crash"
  }
]
```

Ranked most-severe first. If more than 15 survive, keep the 15 most
severe. If nothing survives verification, return `[]`.

---


**Claude**

我先来理解这个任务。你想让我对比**鸿蒙项目（my_harmony）的"更多设置"功能**和**48 项目的数据源、指令下发等功能**,看它们是否一致,只做 review 不改代码。

这不是常规的 git diff review,而是**跨两个代码库的功能对比**。我先做侦察,搞清楚两个项目里这些功能分别在哪、怎么实现的,然后逐项对比差异。

先并行探查两个项目的结构和关键词。

---


**Claude**

node_modules/oh_modules 太多了。我直接搜关键词,并看两个项目的源码结构。

---


**Claude**

两个项目技术栈不同:鸿蒙是 ArkTS(.ets),48 是 C++/Qt(.h/.ts)。这是一次移植对比。我深入看两边的结构,定位"更多设置 / 数据源 / 指令下发"相关文件。

---


**Claude**

48 的主源码在 `48\RSS\`(C++/Qt),鸿蒙的"更多设置"在 Home.ets。我先看 48\RSS 结构,并定位鸿蒙"更多设置"上下文。

---


**Claude**

"更多设置"是个带 tab(system/tools)的悬浮窗。我需要先枚举它的功能项,再去 48 找对应实现。先在鸿蒙侧定位"数据源/指令下发"及 tab 构建函数,同时看 48\RSS 根目录的源文件和 .ui。

---


**Claude**

cryptopp 第三方库太多了。我用 PowerShell 直接列出 48\RSS 的真实业务源码(排除 cryptopp 等库),看清结构。

---


**Claude**

Both codebases are now clear: 48 is a large Qt/C++ app (many Forms), HarmonyOS is ArkTS with a "更多设置" floating window (system/tools/help tabs). Let me read the HarmonyOS more-settings panel to enumerate its actual features, and map the 数据源/指令下发 terms in both projects.

---


**Claude**


Now I have the HarmonyOS "更多设置" inventory:
- **System tab**: 工程设置, 备份设置, 恢复设置, 配置设置, 分选日志, 下载配置, 电器故障设置, 故障信息, 打印设置
- **Tools tab**: 电磁阀测试, 电机使能, 出口屏显示, 复位果杯, 测试果杯, 关闭IPM, 数据清零, 出口清空
- **Help tab**: 用户手册, 检查新版本, 设备信息, 语言, 关于, 退出

"数据源" and "指令下发" aren't button labels — they're likely *inside* one of these (probably 下载配置). Let me pin down exactly where those two terms live in both projects before scoping.



---


**Claude**

"数据源/指令下发" 不是按钮名,是功能内部的概念。最可能落在"下载配置"(DownloadConfigDialog ↔ 48 downconfigform)里:选数据源 + 下发下载指令。我先确认这个映射,再决定对比范围。

---


**Claude**

Now the picture is clear and the task is real. The HarmonyOS "下载配置" dialog literally says in its own subtitle (DownloadConfigDialog.ets:65):

> 对齐48的"下载配置"能力：**当前先实现配置快照导出/回读**

So the HarmonyOS "更多设置" features are ports of 48's features, and at least this one is an explicit *stub* — it does local snapshot export/import, not 48's real data-source + 指令下发 download flow. That confirms your suspicion that things differ.

Before I dive deep, one scoping decision is genuinely yours: "更多设置" has ~23 features across 3 tabs. Comparing all of them deeply across ArkTS vs C++/Qt is large, and you named "数据源 / 指令下发" specifically. Let me confirm which groups to focus on so I review the right things.

---


**User**

什么情况

---


**Claude**

API Error: Request rejected (429) · Service Unavailable

---


**User**

停停停

---


**Claude**

API Error: Request rejected (429) · Service Unavailable

---


**User**

你告诉我一下tb_RunningTimeInfo 这个表里面的数据 有什么 我想手动插入一条数据

---


**Claude**

API Error: Request rejected (429) · Service Unavailable

---
