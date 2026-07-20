# DISPLAY_ON Data Uplink Training PPT Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Generate and verify a restrained 12-slide Chinese PPTX that teaches the real `requestStGlobal -> DISPLAY_ON -> FSM 1128 -> Go parse -> WebSocket -> ArkTS mapping -> UI` chain in about 30 minutes.

**Architecture:** A single Python generator owns slide copy, visual tokens, diagrams, and selected code excerpts. A separate verifier reopens the generated PPTX and checks its structure, required content, minimum readable type, and slide bounds. Application code is read-only evidence and is never changed.

**Tech Stack:** Python 3.13, `python-pptx`, Pillow, PowerPoint Open XML, CodeGraph CLI, PowerShell.

---

## File Structure

- Create: `outputs/build_display_on_training_ppt.py`
  - Owns theme tokens, reusable slide helpers, the 12-slide content, real code excerpts, and PPTX generation.
- Create: `tools/verify_display_on_training_ppt.py`
  - Reopens the deck and validates slide count, required terms, shape bounds, font sizes, and package integrity.
- Create: `outputs/一条数据的旅程-前端到FSM再回到界面.pptx`
  - Final user-facing training deck.
- Read only: Go and ArkTS files listed in the approved design.

### Task 1: Capture Current Code Evidence

**Files:**
- Read: `go/ohos/Tcp/internal/runtime/websocket.go`
- Read: `go/ohos/Tcp/internal/runtime/websocket_handlers.go`
- Read: `go/ohos/Tcp/client/ctcp_client.go`
- Read: `go/ohos/Tcp/internal/runtime/ctcp_server.go`
- Read: `go/ohos/Tcp/internal/runtime/ctcp_protocol.go`
- Read: `go/ohos/Tcp/internal/runtime/ctcp_statistics.go`
- Read: `go/ohos/Tcp/internal/runtime/home_stats.go`
- Read: `entry/src/main/ets/entryability/EntryAbility.ets`
- Read: `E:/new/my_harmony/entry/src/main/ets/utils/network/HarmonyWebSocketClient.ets`
- Read: `E:/new/my_harmony/entry/src/main/ets/protocol/StStatisticsJsonMapper.ets`
- Read: `E:/new/my_harmony/entry/src/main/ets/protocol/GlobalDataInterface.ets`
- Read: `E:/new/my_harmony/entry/src/main/ets/protocol/UIDataSync.ets`

- [ ] **Step 1: Query the complete call path with CodeGraph**

Run:

```powershell
codegraph explore "requestStGlobal DISPLAY_ON SyncRequest cmdFSMConfig cmdFSMStatistics PublishWebSocketJSON complete call path with source"
```

Expected: output includes the WebSocket command route, `RequestStGlobalFromFSM`, CTCP send, 1128 receive, payload parse, and WebSocket publish symbols.

- [ ] **Step 2: Query each slide-facing symbol with current source lines**

Run focused CodeGraph queries for `handleIncoming`, `handleRequestStGlobal`, `SyncRequest`, `recvCTCPSync`, `recvCTCPCommand`, `recvCTCPPayload`, `ParseData`, and `publishLatestStStatisticsSpeed`.

Expected: each query returns verbatim source suitable for an 8-to-15-line excerpt.

- [ ] **Step 3: Read the ArkTS entry, request, dispatch, mapper, and storage consumers**

Run targeted `Get-Content` or `Select-String` reads after CodeGraph identifies the relevant symbols.

Expected evidence:

```text
WebSocket open -> requestStGlobal/requestHomeStats/requestStatistics
messageCallback -> type=data -> use the topic field as a data-category switch
statistics JSON -> StStatistics mapper
GlobalDataInterface/UIDataSync or direct AppStorage -> @StorageLink UI
```

- [ ] **Step 4: Cross-check constants and boundaries**

Confirm from source: `18080`, FSM command port `1279`, return port `1128`, `DISPLAY_ON = 0x0019`, `FSM_CMD_CONFIG = 0x1000`, `FSM_CMD_STATISTICS = 0x1001`, little-endian 16-byte SYNC header, and the actual WebSocket frame fields.

Expected: no slide relies on a value found only in the training notes.

### Task 2: Add a Failing Deck Verifier

**Files:**
- Create: `tools/verify_display_on_training_ppt.py`
- Test: `outputs/一条数据的旅程-前端到FSM再回到界面.pptx`

- [ ] **Step 1: Implement structural validation**

The verifier must use these checks:

```python
from pathlib import Path
from pptx import Presentation

DECK = Path(__file__).resolve().parents[1] / "outputs" / "一条数据的旅程-前端到FSM再回到界面.pptx"
REQUIRED = {
    1: ("一条数据的旅程",),
    3: ("完整链路", "WebSocket", "FSM"),
    7: ("DISPLAY_ON", "SYNC", "0x0019"),
    8: ("1128", "payload"),
    11: ("数据类别字段", "AppStorage"),
    12: ("排查", "复盘"),
}

def all_text(slide) -> str:
    return "\n".join(
        shape.text for shape in slide.shapes
        if getattr(shape, "has_text_frame", False) and shape.text.strip()
    )

def verify() -> None:
    assert DECK.exists(), f"missing deck: {DECK}"
    prs = Presentation(DECK)
    assert len(prs.slides) == 12
    assert round(prs.slide_width / prs.slide_height, 3) == round(16 / 9, 3)
    for number, slide in enumerate(prs.slides, start=1):
        text = all_text(slide)
        for term in REQUIRED.get(number, ()):
            assert term in text, f"slide {number} missing {term}"
        assert str(number) in text, f"slide {number} missing page number"
        for shape in slide.shapes:
            assert shape.left >= 0 and shape.top >= 0
            assert shape.left + shape.width <= prs.slide_width
            assert shape.top + shape.height <= prs.slide_height
            if not getattr(shape, "has_text_frame", False):
                continue
            assert shape.text.strip(), f"slide {number} has empty text box"
            for paragraph in shape.text_frame.paragraphs:
                for run in paragraph.runs:
                    if run.text.strip() and run.font.size is not None:
                        assert run.font.size.pt >= 10

if __name__ == "__main__":
    verify()
    print("PASS: 12-slide DISPLAY_ON training deck verified")
```

The verifier must also identify code by its `Consolas` font and require at least 14pt, require at least 18pt for ordinary body copy, and allow 10pt only for paths, stage labels, page numbers, and speaker cues. It must reject the phrases `主题订阅`, `订阅 topic`, and `发布/订阅`.

- [ ] **Step 2: Run the verifier before generation**

Run `python tools/verify_display_on_training_ppt.py`.

Expected: FAIL because the new deck does not exist.

### Task 3: Build the PPTX Generator

**Files:**
- Create: `outputs/build_display_on_training_ppt.py`

- [ ] **Step 1: Define restrained visual tokens**

Use one neutral theme and semantic colors:

```python
COLORS = {
    "paper": "F7F8FA",
    "white": "FFFFFF",
    "ink": "17202A",
    "muted": "5F6B76",
    "line": "D9DEE5",
    "request": "2563A6",
    "response": "2D7D5A",
    "warning": "B46620",
    "code": "18222D",
}
FONT_UI = "Microsoft YaHei"
FONT_CODE = "Consolas"
```

Set `LAYOUT_WIDE`, 0.55-inch outer margins, 28-to-32pt titles, 18-to-21pt body text, and at least 14pt code text. Use square or subtly rounded corners and no gradients, shadows, decorative imagery, or animation.

- [ ] **Step 2: Implement reusable layout helpers**

Implement and use seven helper boundaries: `add_slide_frame(slide, number, stage)`, `add_title(slide, title, subtitle)`, `add_text_block(slide, x, y, w, h, heading, bullets)`, `add_code_block(slide, x, y, w, h, path, code, highlights)`, `add_actor(slide, x, y, w, title, detail, color)`, `add_flow_arrow(slide, x, y, w, label, color, reverse)`, and `add_io_summary(slide, x, y, w, input_text, process_text, output_text)`.

Each helper must create fixed-position `python-pptx` shapes, explicitly set every text run's font name, size, color, and alignment, and use theme constants rather than one-off colors. `add_code_block` must split source by newline, number each line, and apply the highlight color only to the requested line numbers.

- [ ] **Step 3: Implement the 12 approved slides**

Build exactly this sequence:

```text
1  一条数据的旅程
2  三个角色，一条链路
3  完整链路：连接、请求、指令、返回、显示
4  连接成功，只代表通道已经建立
5  前端发的是 JSON，不是 SYNC
6  Go 按 type 路由请求
7  Go 把 JSON 翻译成 DISPLAY_ON
8  FSM 从 1128 返回，Go 分三段读取
9  二进制解析后先进入缓存
10 Go 把快照封装成 WebSocket data 帧并广播
11 前端按数据类别字段分流，映射状态并显示
12 八个节点定位：数据停在哪里
```

Slides 4 through 11 must each include a real code excerpt and an `输入 / 处理 / 输出` summary. Slide 3 must use a left-to-right flow with blue request arrows and green response arrows. Slide 7 must draw the four 4-byte SYNC header fields. Slide 10 must state that the hub broadcasts to every connected client and does not maintain per-category client lists. Slide 11 must use only code and a state-flow diagram; it must not include screenshots or decorative imagery.

- [ ] **Step 4: Add visible presenter cues**

Because `python-pptx` does not provide a stable notes API, place a short gray `讲解重点` line at the bottom of technical slides. Do not write custom notes XML.

- [ ] **Step 5: Generate the deck**

Run `python outputs/build_display_on_training_ppt.py`.

Expected: creates `outputs/一条数据的旅程-前端到FSM再回到界面.pptx` and reports `12 slides written`.

### Task 4: Verify Content and Layout

**Files:**
- Test: `outputs/一条数据的旅程-前端到FSM再回到界面.pptx`
- Test: `tools/verify_display_on_training_ppt.py`

- [ ] **Step 1: Run the independent verifier**

Run `python tools/verify_display_on_training_ppt.py`.

Expected: `PASS: 12-slide DISPLAY_ON training deck verified`.

- [ ] **Step 2: Reopen and inspect package integrity**

Run:

```powershell
python -c "from pptx import Presentation; p=Presentation(r'outputs/一条数据的旅程-前端到FSM再回到界面.pptx'); print(len(p.slides), p.slide_width, p.slide_height)"
```

Expected: `12` slides and 16:9 dimensions.

- [ ] **Step 3: Inspect all slide text and code density**

Print slide number, title, total text characters, code line count, and minimum font size. Expected: no slide has an empty title, slides 4 through 11 contain focused excerpts, and all fonts meet the design minimums.

- [ ] **Step 4: Check the final diff and output size**

Run:

```powershell
git diff --check -- outputs/build_display_on_training_ppt.py tools/verify_display_on_training_ppt.py
git status --short
Get-Item -LiteralPath 'outputs/一条数据的旅程-前端到FSM再回到界面.pptx' | Select-Object FullName,Length,LastWriteTime
```

Expected: no whitespace errors; only the intended new generator, verifier, and deck are attributable to this work; the PPTX is non-empty.

### Task 5: Record the Deliverable

**Files:**
- Create: `outputs/build_display_on_training_ppt.py`
- Create: `tools/verify_display_on_training_ppt.py`
- Create: `outputs/一条数据的旅程-前端到FSM再回到界面.pptx`

- [ ] **Step 1: Commit only the new training artifacts**

Run:

```powershell
git add -- outputs/build_display_on_training_ppt.py tools/verify_display_on_training_ppt.py 'outputs/一条数据的旅程-前端到FSM再回到界面.pptx' docs/superpowers/plans/2026-07-20-display-on-data-uplink-training-ppt.md
git commit -m "docs: add DISPLAY_ON data uplink training deck"
```

Expected: one commit containing only the plan, generator, verifier, and generated PPTX; unrelated working-tree changes remain untouched.
