#!/usr/bin/env python3
"""Long-running Harmony/Go WebSocket + SQLite stability test.

The script uses only Python's standard library so it can run on a fresh
Windows machine without installing extra packages.
"""

from __future__ import annotations

import argparse
import base64
import dataclasses
import hashlib
import json
import os
import random
import secrets
import signal
import socket
import struct
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


DEFAULT_BASE_URL = "http://127.0.0.1:2778"
DEFAULT_DURATION_SECONDS = 8 * 60 * 60
DEFAULT_BATCH_SIZE = 5000
DEFAULT_INTERVAL_SECONDS = 60
DEFAULT_CUSTOMER_ID_START = 9000
DEFAULT_CUSTOMER_ID_END = 9999
DEFAULT_HTTP_TIMEOUT_SECONDS = 30
DEFAULT_WS_TIMEOUT_SECONDS = 10
DEFAULT_FAILURE_BACKOFF_SECONDS = 120
DEFAULT_MAX_CONSECUTIVE_FAILURES = 5
DEFAULT_STARTUP_RETRIES = 12
DEFAULT_STARTUP_RETRY_SECONDS = 10
DEFAULT_OUTPUT_DIR = Path("reports") / "db_ws_stability"

FRUIT_COMPARE_FIELDS = (
    "CustomerID",
    "SystemID",
    "BatchNumber",
    "BatchWeight",
)


class StabilityError(Exception):
    """Raised when a test step fails in a controlled way."""


@dataclasses.dataclass
class StepTiming:
    name: str
    seconds: float
    ok: bool
    detail: str = ""


@dataclasses.dataclass
class RunStats:
    started_at: str
    base_url: str
    ws_url: str
    run_id: str
    customer_id: int | None = None
    planned_duration_seconds: int = 0
    batch_size: int = 0
    interval_seconds: int = 0
    loops_started: int = 0
    loops_ok: int = 0
    loops_failed: int = 0
    rows_attempted: int = 0
    rows_verified: int = 0
    http_failures: int = 0
    ws_failures: int = 0
    compare_failures: int = 0
    min_loop_seconds: float | None = None
    max_loop_seconds: float | None = None
    total_loop_seconds: float = 0.0
    last_error: str = ""
    stopped_reason: str = ""

    def record_loop_duration(self, seconds: float) -> None:
        if self.min_loop_seconds is None or seconds < self.min_loop_seconds:
            self.min_loop_seconds = seconds
        if self.max_loop_seconds is None or seconds > self.max_loop_seconds:
            self.max_loop_seconds = seconds
        self.total_loop_seconds += seconds

    @property
    def average_loop_seconds(self) -> float:
        if self.loops_started == 0:
            return 0.0
        return self.total_loop_seconds / self.loops_started


class ApiClient:
    def __init__(self, base_url: str, timeout: int) -> None:
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout

    def get_json(self, path: str) -> Any:
        url = self.base_url + path
        request = urllib.request.Request(url, method="GET")
        return self._open_json(request, path)

    def post_api(self, path: str, payload: Any) -> Any:
        url = self.base_url + path
        body = json.dumps(payload, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
        request = urllib.request.Request(
            url,
            data=body,
            method="POST",
            headers={"Content-Type": "application/json"},
        )
        response_body = self._open_bytes(request, path)
        return parse_api_envelope(response_body, path)

    def post_raw_json(self, path: str, payload: Any) -> Any:
        url = self.base_url + path
        body = json.dumps(payload, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
        request = urllib.request.Request(
            url,
            data=body,
            method="POST",
            headers={"Content-Type": "application/json"},
        )
        response_body = self._open_bytes(request, path)
        return json.loads(response_body.decode("utf-8"))

    def _open_json(self, request: urllib.request.Request, label: str) -> Any:
        return json.loads(self._open_bytes(request, label).decode("utf-8"))

    def _open_bytes(self, request: urllib.request.Request, label: str) -> bytes:
        started = time.perf_counter()
        try:
            with urllib.request.urlopen(request, timeout=self.timeout) as response:
                body = response.read()
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode("utf-8", errors="replace")
            raise StabilityError(f"{label} HTTP {exc.code}: {detail}") from exc
        except urllib.error.URLError as exc:
            raise StabilityError(f"{label} network error: {exc.reason}") from exc
        except TimeoutError as exc:
            raise StabilityError(f"{label} timeout after {self.timeout}s") from exc
        except OSError as exc:
            raise StabilityError(f"{label} network error: {exc}") from exc
        elapsed = time.perf_counter() - started
        if elapsed > self.timeout:
            raise StabilityError(f"{label} took {elapsed:.3f}s, longer than timeout budget")
        return body


class JsonlLogger:
    def __init__(self, path: Path) -> None:
        self.path = path
        self.path.parent.mkdir(parents=True, exist_ok=True)

    def write(self, event: str, **fields: Any) -> None:
        record = {
            "at": now_text(),
            "event": event,
            **fields,
        }
        with self.path.open("a", encoding="utf-8") as file:
            file.write(json.dumps(json_safe(record), ensure_ascii=False, separators=(",", ":")) + "\n")


class SimpleWebSocket:
    """Minimal RFC 6455 client for text frames, enough for this test."""

    def __init__(self, url: str, timeout: int) -> None:
        self.url = url
        self.timeout = timeout
        self.sock: socket.socket | None = None
        self.host = ""
        self.port = 0
        self.path = ""

    def __enter__(self) -> "SimpleWebSocket":
        self.connect()
        return self

    def __exit__(self, exc_type: Any, exc: Any, tb: Any) -> None:
        self.close()

    def connect(self) -> None:
        parsed = urllib.parse.urlparse(self.url)
        if parsed.scheme not in {"ws", "wss"}:
            raise StabilityError(f"unsupported websocket scheme: {parsed.scheme}")
        if parsed.scheme == "wss":
            raise StabilityError("wss is not supported by the standard-library test client")
        self.host = parsed.hostname or ""
        self.port = parsed.port or 80
        self.path = parsed.path or "/"
        if parsed.query:
            self.path += "?" + parsed.query
        if not self.host:
            raise StabilityError(f"invalid websocket url: {self.url}")

        sock = socket.create_connection((self.host, self.port), timeout=self.timeout)
        sock.settimeout(self.timeout)
        key = base64.b64encode(secrets.token_bytes(16)).decode("ascii")
        request = (
            f"GET {self.path} HTTP/1.1\r\n"
            f"Host: {self.host}:{self.port}\r\n"
            "Upgrade: websocket\r\n"
            "Connection: Upgrade\r\n"
            f"Sec-WebSocket-Key: {key}\r\n"
            "Sec-WebSocket-Version: 13\r\n"
            "\r\n"
        ).encode("ascii")
        sock.sendall(request)
        header = self._read_http_header(sock)
        if b" 101 " not in header.split(b"\r\n", 1)[0]:
            raise StabilityError(f"websocket upgrade failed: {header[:200]!r}")
        if not websocket_accept_is_valid(header, key):
            raise StabilityError("websocket upgrade accepted with unexpected Sec-WebSocket-Accept")
        self.sock = sock

    def send_text(self, text: str) -> None:
        if self.sock is None:
            raise StabilityError("websocket is not connected")
        payload = text.encode("utf-8")
        if len(payload) < 126:
            header = bytearray([0x81, 0x80 | len(payload)])
        elif len(payload) <= 0xFFFF:
            header = bytearray([0x81, 0x80 | 126])
            header.extend(struct.pack("!H", len(payload)))
        else:
            header = bytearray([0x81, 0x80 | 127])
            header.extend(struct.pack("!Q", len(payload)))
        mask = secrets.token_bytes(4)
        masked = bytes(byte ^ mask[index % 4] for index, byte in enumerate(payload))
        self.sock.sendall(bytes(header) + mask + masked)

    def recv_json_until(self, predicate: Any, timeout: int | None = None) -> dict[str, Any]:
        deadline = time.monotonic() + (timeout or self.timeout)
        last_text = ""
        while time.monotonic() < deadline:
            remaining = max(0.2, deadline - time.monotonic())
            text = self.recv_text(timeout=remaining)
            last_text = text
            try:
                frame = json.loads(text)
            except json.JSONDecodeError:
                continue
            if predicate(frame):
                return frame
        raise StabilityError(f"websocket expected frame not received; last={last_text[:200]!r}")

    def recv_text(self, timeout: float | None = None) -> str:
        if self.sock is None:
            raise StabilityError("websocket is not connected")
        old_timeout = self.sock.gettimeout()
        if timeout is not None:
            self.sock.settimeout(timeout)
        try:
            while True:
                opcode, payload = self._recv_frame()
                if opcode == 0x1:
                    return payload.decode("utf-8", errors="replace")
                if opcode == 0x8:
                    raise StabilityError("websocket closed by server")
                if opcode == 0x9:
                    self._send_pong(payload)
        finally:
            if timeout is not None:
                self.sock.settimeout(old_timeout)

    def close(self) -> None:
        sock = self.sock
        self.sock = None
        if sock is not None:
            try:
                sock.sendall(b"\x88\x80" + secrets.token_bytes(4))
            except OSError:
                pass
            try:
                sock.close()
            except OSError:
                pass

    def _send_pong(self, payload: bytes) -> None:
        if self.sock is None:
            return
        mask = secrets.token_bytes(4)
        masked = bytes(byte ^ mask[index % 4] for index, byte in enumerate(payload))
        self.sock.sendall(bytes([0x8A, 0x80 | len(payload)]) + mask + masked)

    def _recv_frame(self) -> tuple[int, bytes]:
        if self.sock is None:
            raise StabilityError("websocket is not connected")
        first = self._recv_exact(2)
        opcode = first[0] & 0x0F
        masked = bool(first[1] & 0x80)
        length = first[1] & 0x7F
        if length == 126:
            length = struct.unpack("!H", self._recv_exact(2))[0]
        elif length == 127:
            length = struct.unpack("!Q", self._recv_exact(8))[0]
        mask = self._recv_exact(4) if masked else b""
        payload = self._recv_exact(length)
        if masked:
            payload = bytes(byte ^ mask[index % 4] for index, byte in enumerate(payload))
        return opcode, payload

    def _recv_exact(self, size: int) -> bytes:
        if self.sock is None:
            raise StabilityError("websocket is not connected")
        chunks = bytearray()
        while len(chunks) < size:
            part = self.sock.recv(size - len(chunks))
            if not part:
                raise StabilityError("socket closed while reading")
            chunks.extend(part)
        return bytes(chunks)

    @staticmethod
    def _read_http_header(sock: socket.socket) -> bytes:
        chunks = bytearray()
        while b"\r\n\r\n" not in chunks:
            part = sock.recv(4096)
            if not part:
                break
            chunks.extend(part)
            if len(chunks) > 32768:
                break
        return bytes(chunks)


def now_text() -> str:
    return datetime.now().astimezone().isoformat(timespec="seconds")


def websocket_accept_is_valid(header: bytes, key: str) -> bool:
    expected = base64.b64encode(
        hashlib.sha1((key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11").encode("ascii")).digest()
    ).decode("ascii")
    for line in header.decode("iso-8859-1", errors="replace").split("\r\n"):
        name, separator, value = line.partition(":")
        if separator and name.strip().lower() == "sec-websocket-accept":
            return value.strip() == expected
    return False


def json_safe(value: Any) -> Any:
    if dataclasses.is_dataclass(value):
        return json_safe(dataclasses.asdict(value))
    if isinstance(value, Path):
        return str(value)
    if isinstance(value, dict):
        return {str(key): json_safe(item) for key, item in value.items()}
    if isinstance(value, (list, tuple)):
        return [json_safe(item) for item in value]
    if isinstance(value, (str, int, float, bool)) or value is None:
        return value
    return str(value)


def safe_run_id() -> str:
    return datetime.now().strftime("%Y%m%d-%H%M%S")


def parse_api_envelope(body: bytes, label: str) -> Any:
    try:
        envelope = json.loads(body.decode("utf-8"))
    except json.JSONDecodeError as exc:
        raise StabilityError(f"{label} returned invalid JSON: {exc}") from exc

    if envelope.get("returnCode") != 1:
        message = envelope.get("returnMessage") or "unknown API error"
        raise StabilityError(f"{label} returnCode={envelope.get('returnCode')}: {message}")

    data = envelope.get("data", "")
    if data == "":
        return ""
    if not isinstance(data, str):
        return data
    try:
        return json.loads(data)
    except json.JSONDecodeError as exc:
        raise StabilityError(f"{label} data is not valid JSON: {exc}") from exc


def is_database_busy_or_timeout(text: str) -> bool:
    lowered = text.lower()
    return (
        "sqlite_busy" in lowered
        or "database is locked" in lowered
        or "timeout after" in lowered
        or "longer than timeout budget" in lowered
    )


def failure_kind(text: str) -> str:
    lowered = text.lower()
    if "websocket" in lowered:
        return "websocket"
    if "compare" in lowered:
        return "compare"
    return "http"


def count_failure(stats: RunStats, text: str) -> None:
    kind = failure_kind(text)
    if kind == "websocket":
        stats.ws_failures += 1
    elif kind == "compare":
        stats.compare_failures += 1
    else:
        stats.http_failures += 1


def build_fruit_rows(run_id: str, customer_id: int, count: int, seed: int) -> list[dict[str, Any]]:
    rng = random.Random(seed)
    rows: list[dict[str, Any]] = []
    for index in range(1, count + 1):
        system_id = index
        rows.append(
            {
                "FID": 0,
                "CustomerID": customer_id,
                "SystemID": system_id,
                "BatchNumber": 1000 + (index % 9000) + rng.randrange(0, 100),
                "BatchWeight": 10000 + (index * 137) + rng.randrange(0, 1000),
            }
        )
    return rows


def compare_fruit_rows(expected: list[dict[str, Any]], actual: list[dict[str, Any]]) -> dict[str, Any]:
    def key_of(row: dict[str, Any]) -> tuple[int, int]:
        return (int(row.get("CustomerID") or 0), int(row.get("SystemID") or 0))

    expected_by_key = {key_of(row): row for row in expected}
    actual_by_key = {key_of(row): row for row in actual}
    missing_keys = sorted(set(expected_by_key) - set(actual_by_key))
    extra_keys = sorted(set(actual_by_key) - set(expected_by_key))
    mismatched_keys: list[tuple[int, int]] = []
    mismatch_details: list[dict[str, Any]] = []

    for key in sorted(set(expected_by_key) & set(actual_by_key)):
        expected_row = expected_by_key[key]
        actual_row = actual_by_key[key]
        changed = {}
        for field in FRUIT_COMPARE_FIELDS:
            if expected_row.get(field) != actual_row.get(field):
                changed[field] = {
                    "expected": expected_row.get(field),
                    "actual": actual_row.get(field),
                }
        if changed:
            mismatched_keys.append(key)
            if len(mismatch_details) < 20:
                mismatch_details.append({"CustomerID": key[0], "SystemID": key[1], "fields": changed})

    return {
        "ok": not missing_keys and not extra_keys and not mismatched_keys,
        "expected_count": len(expected),
        "actual_count": len(actual),
        "missing_keys": missing_keys[:50],
        "extra_keys": extra_keys[:50],
        "mismatched_keys": mismatched_keys[:50],
        "mismatch_details": mismatch_details,
        "missing_count": len(missing_keys),
        "extra_count": len(extra_keys),
        "mismatched_count": len(mismatched_keys),
    }


def percentile(values: list[float], p: float) -> float:
    if not values:
        return 0.0
    ordered = sorted(values)
    index = min(len(ordered) - 1, max(0, int(round((len(ordered) - 1) * p))))
    return ordered[index]


def build_ws_url(base_url: str, path: str) -> str:
    parsed = urllib.parse.urlparse(base_url)
    scheme = "ws" if parsed.scheme == "http" else "wss"
    netloc = parsed.netloc
    return urllib.parse.urlunparse((scheme, netloc, path, "", "", ""))


def choose_empty_customer_id(client: ApiClient, start: int, end: int, logger: JsonlLogger) -> int:
    for customer_id in range(start, end + 1):
        result = client.post_api("/Api/SysFruitInfo/GetSysFruitInfo", {"CustomerID": customer_id})
        rows = result.get("Items") if isinstance(result, dict) else result
        if not rows:
            logger.write("customer_id_selected", customer_id=customer_id)
            return customer_id
    raise StabilityError(f"no empty CustomerID found in range {start}..{end}")


def delete_rows_by_customer(client: ApiClient, customer_id: int, logger: JsonlLogger) -> int:
    result = client.post_api("/Api/SysFruitInfo/GetSysFruitInfo", {"CustomerID": customer_id})
    rows = result.get("Items") if isinstance(result, dict) else result
    if not isinstance(rows, list) or not rows:
        return 0
    client.post_api("/Api/SysFruitInfo/DeleteSysFruitInfo", {"CustomerID": customer_id})
    logger.write("cleanup_done", customer_id=customer_id, deleted=len(rows))
    return len(rows)


def check_http_readiness(client: ApiClient) -> dict[str, Any]:
    ping = client.get_json("/ping")
    orm = client.get_json("/orm")
    if ping.get("message") != "pong from Go":
        raise StabilityError(f"/ping unexpected response: {ping}")
    if orm.get("status") != "ready":
        raise StabilityError(f"/orm not ready: {orm}")
    return {"ping": ping, "orm": orm}


def check_http_readiness_with_retries(
    client: ApiClient,
    retries: int,
    retry_seconds: int,
    logger: JsonlLogger,
    timings: list[StepTiming],
) -> dict[str, Any]:
    attempts = max(0, retries) + 1
    for attempt in range(1, attempts + 1):
        try:
            return timed_step("readiness", timings, lambda: check_http_readiness(client))
        except StabilityError as exc:
            if attempt >= attempts or not is_database_busy_or_timeout(str(exc)):
                raise
            logger.write("readiness_retry", attempt=attempt, error=str(exc), sleep_seconds=retry_seconds)
            time.sleep(max(0, retry_seconds))
    raise StabilityError("readiness failed")


def ws_probe(ws_url: str, timeout: int, query_mode: str, request_id: str) -> dict[str, Any]:
    with SimpleWebSocket(ws_url, timeout=timeout) as ws:
        ready = ws.recv_json_until(lambda frame: frame.get("type") == "ready", timeout=timeout)
        started = time.perf_counter()
        ws.send_text(json.dumps({"type": "ping", "requestId": request_id}, separators=(",", ":")))
        pong = ws.recv_json_until(
            lambda frame: frame.get("type") == "pong" and frame.get("requestId") == request_id,
            timeout=timeout,
        )
        ping_ms = (time.perf_counter() - started) * 1000.0
        result: dict[str, Any] = {"ready": ready, "pong": pong, "ping_ms": ping_ms}

        if query_mode == "sort_log":
            query_id = request_id + "-sort"
            query = {
                "type": "querySortLog",
                "requestId": query_id,
                "sortLogQuery": {"StartDate": "2000-01-01", "EndDate": "2099-12-31"},
            }
            started = time.perf_counter()
            ws.send_text(json.dumps(query, ensure_ascii=False, separators=(",", ":")))
            ack = ws.recv_json_until(
                lambda frame: frame.get("type") == "commandAck"
                and frame.get("topic") == "querySortLog"
                and isinstance(frame.get("data"), dict)
                and frame["data"].get("requestId") == query_id,
                timeout=timeout,
            )
            result["query_ms"] = (time.perf_counter() - started) * 1000.0
            result["query_ack"] = ack
        elif query_mode == "fruit_info":
            query_id = request_id + "-fruit"
            query = {
                "type": "queryFruitInfo",
                "requestId": query_id,
                "fruitInfoQuery": {
                    "PageIndex": 1,
                    "PageSize": 20,
                    "SortColumn": "CustomerID",
                    "SortOrder": "desc",
                },
            }
            started = time.perf_counter()
            ws.send_text(json.dumps(query, ensure_ascii=False, separators=(",", ":")))
            ack = ws.recv_json_until(
                lambda frame: frame.get("type") == "commandAck"
                and frame.get("topic") == "queryFruitInfo"
                and isinstance(frame.get("data"), dict)
                and frame["data"].get("requestId") == query_id,
                timeout=timeout,
            )
            result["query_ms"] = (time.perf_counter() - started) * 1000.0
            result["query_ack"] = ack
        return result


def write_json(path: Path, data: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_suffix(path.suffix + ".tmp")
    tmp.write_text(json.dumps(json_safe(data), ensure_ascii=False, indent=2), encoding="utf-8")
    os.replace(tmp, path)


def write_report(path: Path, stats: RunStats, recent_events_path: Path, timings: list[StepTiming]) -> None:
    avg = stats.average_loop_seconds
    lines = [
        "# 数据库 + WebSocket 稳定性测试报告",
        "",
        f"- 更新时间: {now_text()}",
        f"- 开始时间: {stats.started_at}",
        f"- 停止原因: {stats.stopped_reason or 'running'}",
        f"- Base URL: `{stats.base_url}`",
        f"- WebSocket URL: `{stats.ws_url}`",
        f"- Run ID: `{stats.run_id}`",
        f"- 测试 CustomerID: `{stats.customer_id}`",
        f"- 计划时长: {stats.planned_duration_seconds} 秒",
        f"- 每轮数据量: {stats.batch_size} 行",
        f"- 每轮间隔: {stats.interval_seconds} 秒",
        "",
        "## 汇总",
        "",
        f"- 已开始轮次: {stats.loops_started}",
        f"- 成功轮次: {stats.loops_ok}",
        f"- 失败轮次: {stats.loops_failed}",
        f"- 尝试写入行数: {stats.rows_attempted}",
        f"- 已验证一致行数: {stats.rows_verified}",
        f"- HTTP/API 失败: {stats.http_failures}",
        f"- WebSocket 失败: {stats.ws_failures}",
        f"- 数据比对失败: {stats.compare_failures}",
        f"- 最短轮次耗时: {stats.min_loop_seconds or 0:.3f} 秒",
        f"- 最长轮次耗时: {stats.max_loop_seconds or 0:.3f} 秒",
        f"- 平均轮次耗时: {avg:.3f} 秒",
        f"- 最近错误: {stats.last_error or '无'}",
        "",
        "## 最近步骤耗时",
        "",
        "| 步骤 | 耗时秒 | 结果 | 说明 |",
        "| --- | ---: | --- | --- |",
    ]
    for timing in timings[-30:]:
        ok = "OK" if timing.ok else "FAIL"
        detail = timing.detail.replace("|", "\\|")
        lines.append(f"| {timing.name} | {timing.seconds:.3f} | {ok} | {detail} |")
    lines.extend(
        [
            "",
            "## 判断口径",
            "",
            "- 写入一致性: `/Api/SysFruitInfo/SaveSysFruitInfos` 保存测试专用 `CustomerID` 的完整集合后，立即用 `/Api/SysFruitInfo/GetSysFruitInfo` 查回并逐字段比对。",
            "- WebSocket 稳定性: 连接 `/ws`，等待 `ready`，发送 JSON `ping`，并按配置额外发送 `querySortLog` 或 `queryFruitInfo` 数据库查询探针。",
            "- 事件流水: 见下方 JSONL 文件，可用于定位具体失败轮次。",
            "",
            f"事件流水文件: `{recent_events_path}`",
            "",
        ]
    )
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_suffix(path.suffix + ".tmp")
    tmp.write_text("\n".join(lines), encoding="utf-8")
    os.replace(tmp, path)


def stats_to_dict(stats: RunStats) -> dict[str, Any]:
    data = dataclasses.asdict(stats)
    data["average_loop_seconds"] = stats.average_loop_seconds
    return data


def timed_step(name: str, timings: list[StepTiming], func: Any) -> Any:
    started = time.perf_counter()
    try:
        result = func()
    except Exception as exc:
        timings.append(StepTiming(name=name, seconds=time.perf_counter() - started, ok=False, detail=str(exc)))
        raise
    timings.append(StepTiming(name=name, seconds=time.perf_counter() - started, ok=True))
    return result


def run_once(
    client: ApiClient,
    ws_url: str,
    args: argparse.Namespace,
    stats: RunStats,
    logger: JsonlLogger,
    timings: list[StepTiming],
    loop_index: int,
) -> None:
    assert stats.customer_id is not None
    seed = args.seed + loop_index
    expected_rows = build_fruit_rows(args.run_id, stats.customer_id, args.batch_size, seed)
    request_id = f"{args.run_id}-{loop_index:06d}"
    loop_started = time.perf_counter()
    stats.loops_started += 1
    stats.rows_attempted += len(expected_rows)
    logger.write("loop_start", loop=loop_index, rows=len(expected_rows), seed=seed)

    try:
        def save_rows() -> Any:
            return client.post_api("/Api/SysFruitInfo/SaveSysFruitInfos", expected_rows)

        timed_step("save_sys_fruit_infos", timings, save_rows)

        def read_rows() -> list[dict[str, Any]]:
            result = client.post_api("/Api/SysFruitInfo/GetSysFruitInfo", {"CustomerID": stats.customer_id})
            rows = result.get("Items") if isinstance(result, dict) else result
            if not isinstance(rows, list):
                raise StabilityError(f"GetSysFruitInfo returned {type(rows).__name__}")
            return rows

        actual_rows = timed_step("read_sys_fruit_infos", timings, read_rows)
        compare = compare_fruit_rows(expected_rows, actual_rows)
        if not compare["ok"]:
            stats.compare_failures += 1
            raise StabilityError(f"database compare failed: {json.dumps(compare, ensure_ascii=False)[:1000]}")

        stats.rows_verified += len(expected_rows)

        if args.ws_mode != "off":
            def probe() -> dict[str, Any]:
                return ws_probe(ws_url, args.ws_timeout, args.ws_mode, request_id)

            ws_result = timed_step("websocket_probe", timings, probe)
            query_ack = ws_result.get("query_ack")
            if isinstance(query_ack, dict):
                data = query_ack.get("data")
                if isinstance(data, dict) and not data.get("ok", False):
                    raise StabilityError(f"websocket query returned failure: {data}")

        loop_seconds = time.perf_counter() - loop_started
        stats.loops_ok += 1
        logger.write("loop_ok", loop=loop_index, seconds=loop_seconds, rows=len(expected_rows))
    finally:
        stats.record_loop_duration(time.perf_counter() - loop_started)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Run long Harmony Go backend SQLite/WebSocket stability test.")
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL, help="HTTP base URL after hdc fport.")
    parser.add_argument("--duration-seconds", type=int, default=DEFAULT_DURATION_SECONDS)
    parser.add_argument("--batch-size", type=int, default=DEFAULT_BATCH_SIZE)
    parser.add_argument("--interval-seconds", type=int, default=DEFAULT_INTERVAL_SECONDS)
    parser.add_argument("--http-timeout", type=int, default=DEFAULT_HTTP_TIMEOUT_SECONDS)
    parser.add_argument("--ws-timeout", type=int, default=DEFAULT_WS_TIMEOUT_SECONDS)
    parser.add_argument("--ws-path", default="/ws")
    parser.add_argument(
        "--ws-mode",
        choices=("ping", "sort_log", "fruit_info", "off"),
        default="sort_log",
        help="WebSocket probe mode. sort_log and fruit_info also query DB through websocket.",
    )
    parser.add_argument("--customer-id", type=int, default=0, help="Use this CustomerID. Default: auto-select empty id.")
    parser.add_argument("--customer-id-start", type=int, default=DEFAULT_CUSTOMER_ID_START)
    parser.add_argument("--customer-id-end", type=int, default=DEFAULT_CUSTOMER_ID_END)
    parser.add_argument("--run-id", default=safe_run_id())
    parser.add_argument("--seed", type=int, default=20260629)
    parser.add_argument("--output-dir", type=Path, default=DEFAULT_OUTPUT_DIR)
    parser.add_argument("--failure-backoff-seconds", type=int, default=DEFAULT_FAILURE_BACKOFF_SECONDS)
    parser.add_argument("--max-consecutive-failures", type=int, default=DEFAULT_MAX_CONSECUTIVE_FAILURES)
    parser.add_argument("--startup-retries", type=int, default=DEFAULT_STARTUP_RETRIES)
    parser.add_argument("--startup-retry-seconds", type=int, default=DEFAULT_STARTUP_RETRY_SECONDS)
    parser.add_argument("--no-cleanup", action="store_true", help="Leave generated test rows in the database.")
    parser.add_argument("--once", action="store_true", help="Run one loop only.")
    args = parser.parse_args(argv)

    if args.batch_size <= 0:
        raise SystemExit("--batch-size must be > 0")
    if args.interval_seconds < 0:
        raise SystemExit("--interval-seconds must be >= 0")
    if args.failure_backoff_seconds < 0:
        raise SystemExit("--failure-backoff-seconds must be >= 0")
    if args.max_consecutive_failures < 0:
        raise SystemExit("--max-consecutive-failures must be >= 0")
    if args.startup_retries < 0:
        raise SystemExit("--startup-retries must be >= 0")
    if args.startup_retry_seconds < 0:
        raise SystemExit("--startup-retry-seconds must be >= 0")

    output_dir = args.output_dir / args.run_id
    events_path = output_dir / "events.jsonl"
    summary_path = output_dir / "summary.json"
    report_path = output_dir / "report.md"
    logger = JsonlLogger(events_path)
    timings: list[StepTiming] = []
    client = ApiClient(args.base_url, timeout=args.http_timeout)
    ws_url = build_ws_url(args.base_url, args.ws_path)
    stats = RunStats(
        started_at=now_text(),
        base_url=args.base_url,
        ws_url=ws_url,
        run_id=args.run_id,
        planned_duration_seconds=args.duration_seconds,
        batch_size=args.batch_size,
        interval_seconds=args.interval_seconds,
    )
    stop_requested = {"value": False}

    def request_stop(signum: int, frame: Any) -> None:
        stop_requested["value"] = True
        stats.stopped_reason = f"signal {signum}"
        logger.write("stop_requested", signum=signum)

    signal.signal(signal.SIGINT, request_stop)
    if hasattr(signal, "SIGTERM"):
        signal.signal(signal.SIGTERM, request_stop)

    def persist() -> None:
        write_json(summary_path, stats_to_dict(stats))
        write_report(report_path, stats, events_path, timings)

    logger.write("run_start", args=vars(args), ws_url=ws_url)
    cleanup_rows: list[dict[str, Any]] = []
    exit_code = 0
    try:
        readiness = check_http_readiness_with_retries(
            client,
            args.startup_retries,
            args.startup_retry_seconds,
            logger,
            timings,
        )
        logger.write("readiness_ok", readiness=readiness)
        if args.customer_id > 0:
            existing = client.post_api("/Api/SysFruitInfo/GetSysFruitInfo", {"CustomerID": args.customer_id})
            existing_rows = existing.get("Items") if isinstance(existing, dict) else existing
            if existing_rows:
                raise StabilityError(
                    f"configured --customer-id {args.customer_id} already has {len(existing_rows)} rows; "
                    "choose an empty CustomerID to avoid replacing real data"
                )
            stats.customer_id = args.customer_id
        else:
            stats.customer_id = timed_step(
                "choose_empty_customer_id",
                timings,
                lambda: choose_empty_customer_id(client, args.customer_id_start, args.customer_id_end, logger),
            )
        persist()

        deadline = time.monotonic() + args.duration_seconds
        loop_index = 0
        consecutive_failures = 0
        next_sleep_override: int | None = None
        while not stop_requested["value"]:
            loop_index += 1
            next_sleep_override = None
            if loop_index > 1 and time.monotonic() >= deadline:
                stats.stopped_reason = "duration reached"
                break
            if args.once and loop_index > 1:
                stats.stopped_reason = "once complete"
                break
            try:
                run_once(client, ws_url, args, stats, logger, timings, loop_index)
                consecutive_failures = 0
                if stats.customer_id is not None:
                    cleanup_rows = client.post_api("/Api/SysFruitInfo/GetSysFruitInfo", {"CustomerID": stats.customer_id})
            except StabilityError as exc:
                consecutive_failures += 1
                stats.loops_failed += 1
                stats.last_error = str(exc)
                count_failure(stats, str(exc))
                logger.write("loop_failed", loop=loop_index, error=str(exc))
                if is_database_busy_or_timeout(str(exc)):
                    next_sleep_override = args.failure_backoff_seconds
                    logger.write(
                        "database_busy_backoff",
                        loop=loop_index,
                        consecutive_failures=consecutive_failures,
                        sleep_seconds=next_sleep_override,
                    )
                exit_code = 2
            except Exception as exc:
                consecutive_failures += 1
                stats.loops_failed += 1
                stats.last_error = repr(exc)
                logger.write("loop_crashed", loop=loop_index, error=repr(exc))
                exit_code = 2
            finally:
                persist()

            if (
                args.max_consecutive_failures > 0
                and consecutive_failures >= args.max_consecutive_failures
            ):
                stats.stopped_reason = f"stopped after {consecutive_failures} consecutive failures"
                logger.write("too_many_consecutive_failures", consecutive_failures=consecutive_failures)
                persist()
                break
            if args.once:
                continue
            requested_sleep = args.interval_seconds
            if next_sleep_override is not None:
                requested_sleep = max(requested_sleep, next_sleep_override)
            sleep_for = min(requested_sleep, max(0.0, deadline - time.monotonic()))
            if sleep_for <= 0:
                stats.stopped_reason = "duration reached"
                break
            time.sleep(sleep_for)
    except StabilityError as exc:
        stats.last_error = str(exc)
        count_failure(stats, str(exc))
        stats.stopped_reason = stats.stopped_reason or "startup failed"
        logger.write("run_failed", error=str(exc))
        exit_code = 2
    finally:
        if not stats.stopped_reason:
            stats.stopped_reason = "finished"
        if stats.customer_id is not None and not args.no_cleanup:
            try:
                deleted = delete_rows_by_customer(client, stats.customer_id, logger)
                logger.write("cleanup_done", deleted=deleted)
            except Exception as exc:
                stats.last_error = f"{stats.last_error}; cleanup failed: {exc}" if stats.last_error else f"cleanup failed: {exc}"
                logger.write("cleanup_failed", error=str(exc))
                exit_code = 3 if exit_code == 0 else exit_code
        persist()
        print(f"Report: {report_path}")
        print(f"Summary: {summary_path}")
        print(f"Events: {events_path}")
    return exit_code


if __name__ == "__main__":
    raise SystemExit(main())
