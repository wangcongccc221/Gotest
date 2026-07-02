#!/usr/bin/env python3
"""Simulate lower-machine StStatistics traffic and check realtime DB persistence.

The script sends CTCP packets to the Go backend's stat server. It uses the
normal realtime path: CTCP -> homeStats websocket push -> periodic database save.
"""

from __future__ import annotations

import argparse
import json
import os
import socket
import struct
import sys
import threading
import time
from datetime import datetime
from pathlib import Path
from typing import Any

SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

from db_ws_stability_test import SimpleWebSocket, StabilityError, build_ws_url, json_safe, now_text


DEFAULT_BASE_URL = "http://127.0.0.1:2778"
DEFAULT_CTCP_HOST = "127.0.0.1"
DEFAULT_CTCP_STAT_PORT = 1128
DEFAULT_OUTPUT_DIR = Path("reports") / "realtime_db_perf"

CTCP_HC_ID = 0x1000
CTCP_DEFAULT_FSM_ID = 0x0100
CMD_FSM_CONFIG = 0x1000
CMD_FSM_STATISTICS = 0x1001

ST_GLOBAL_SIZE = 28712
ST_STATISTICS_SIZE = 7152

ST_GLOBAL_SYS_N_CHANNEL_INFO_OFFSET = 384
ST_GLOBAL_SYS_N_SUBSYS_NUM_OFFSET = 490
ST_GLOBAL_SYS_N_EXIT_NUM_OFFSET = 491

ST_STAT_N_GRADE_COUNT_OFFSET = 0
ST_STAT_N_WEIGHT_GRADE_COUNT_OFFSET = 2048
ST_STAT_N_EXIT_COUNT_OFFSET = 4096
ST_STAT_N_EXIT_WEIGHT_COUNT_OFFSET = 4480
ST_STAT_N_CHANNEL_TOTAL_COUNT_OFFSET = 4864
ST_STAT_N_CHANNEL_WEIGHT_COUNT_OFFSET = 4872
ST_STAT_N_SUBSYS_ID_OFFSET = 4880
ST_STAT_N_BOX_GRADE_COUNT_OFFSET = 4884
ST_STAT_N_BOX_GRADE_WEIGHT_OFFSET = 5908
ST_STAT_N_TOTAL_CUP_NUM_OFFSET = 6932
ST_STAT_N_INTERVAL_OFFSET = 6936
ST_STAT_N_INTERVAL_SUM_PER_MINUTE_OFFSET = 6940
ST_STAT_N_PULSE_INTERVAL_OFFSET = 6946


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


class HomeStatsMonitor:
    def __init__(self, base_url: str, ws_path: str, timeout: int) -> None:
        self.ws_url = build_ws_url(base_url, ws_path)
        self.timeout = timeout
        self.stop_event = threading.Event()
        self.lock = threading.Lock()
        self.latest: dict[str, Any] | None = None
        self.frames = 0
        self.last_error = ""
        self.thread: threading.Thread | None = None
        self.ws: SimpleWebSocket | None = None

    def start(self) -> None:
        self.thread = threading.Thread(target=self._run, name="homeStats-listener", daemon=True)
        self.thread.start()

    def stop(self) -> None:
        self.stop_event.set()
        if self.ws is not None:
            self.ws.close()
        if self.thread is not None:
            self.thread.join(timeout=2)

    def snapshot(self) -> tuple[dict[str, Any] | None, int, str]:
        with self.lock:
            latest = dict(self.latest) if isinstance(self.latest, dict) else None
            return latest, self.frames, self.last_error

    def _run(self) -> None:
        try:
            with SimpleWebSocket(self.ws_url, timeout=self.timeout) as ws:
                self.ws = ws
                ws.recv_json_until(lambda frame: frame.get("type") == "ready", timeout=self.timeout)
                while not self.stop_event.is_set():
                    try:
                        text = ws.recv_text(timeout=1.0)
                    except (TimeoutError, socket.timeout):
                        continue
                    except OSError as exc:
                        if self.stop_event.is_set():
                            break
                        raise exc
                    try:
                        frame = json.loads(text)
                    except json.JSONDecodeError:
                        continue
                    if frame.get("type") != "data" or frame.get("topic") != "homeStats":
                        continue
                    data = frame.get("data")
                    if not isinstance(data, dict):
                        continue
                    with self.lock:
                        self.latest = data
                        self.frames += 1
        except Exception as exc:
            with self.lock:
                self.last_error = str(exc)
        finally:
            self.ws = None


def safe_run_id() -> str:
    return datetime.now().strftime("%Y%m%d-%H%M%S")


def clamp_byte(value: int) -> int:
    return max(0, min(255, int(value)))


def build_ctcp_packet(src_id: int, dst_id: int, cmd_id: int, payload: bytes) -> bytes:
    return b"SYNC" + struct.pack("<iii", int(src_id), int(dst_id), int(cmd_id)) + payload


def build_minimal_st_global_payload(subsys_num: int, channels: int, exits: int) -> bytes:
    payload = bytearray(ST_GLOBAL_SIZE)
    subsys_num = clamp_byte(subsys_num)
    channels = clamp_byte(channels)
    exits = clamp_byte(exits)
    for index in range(min(4, max(1, subsys_num))):
        payload[ST_GLOBAL_SYS_N_CHANNEL_INFO_OFFSET + index] = channels
    payload[ST_GLOBAL_SYS_N_SUBSYS_NUM_OFFSET] = max(1, subsys_num)
    payload[ST_GLOBAL_SYS_N_EXIT_NUM_OFFSET] = exits
    return bytes(payload)


def build_st_statistics_payload(
    total_count: int,
    total_weight: int,
    subsys_id: int,
    speed_per_minute: int,
    cup_count: int,
    exit_count: int,
) -> bytes:
    payload = bytearray(ST_STATISTICS_SIZE)
    total_count = max(0, int(total_count))
    total_weight = max(0, int(total_weight))
    subsys_id = max(1, int(subsys_id))
    speed_per_minute = max(0, int(speed_per_minute))
    cup_count = max(0, int(cup_count))
    exit_count = max(0, int(exit_count))

    struct.pack_into("<Q", payload, ST_STAT_N_GRADE_COUNT_OFFSET, total_count)
    struct.pack_into("<Q", payload, ST_STAT_N_WEIGHT_GRADE_COUNT_OFFSET, total_weight)
    struct.pack_into("<Q", payload, ST_STAT_N_EXIT_COUNT_OFFSET, exit_count)
    struct.pack_into("<Q", payload, ST_STAT_N_EXIT_WEIGHT_COUNT_OFFSET, total_weight)
    struct.pack_into("<Q", payload, ST_STAT_N_CHANNEL_TOTAL_COUNT_OFFSET, total_count)
    struct.pack_into("<Q", payload, ST_STAT_N_CHANNEL_WEIGHT_COUNT_OFFSET, total_weight)
    struct.pack_into("<i", payload, ST_STAT_N_SUBSYS_ID_OFFSET, subsys_id)
    struct.pack_into("<i", payload, ST_STAT_N_BOX_GRADE_COUNT_OFFSET, min(total_count, 2_147_483_647))
    struct.pack_into("<i", payload, ST_STAT_N_BOX_GRADE_WEIGHT_OFFSET, min(total_weight, 2_147_483_647))
    struct.pack_into("<i", payload, ST_STAT_N_TOTAL_CUP_NUM_OFFSET, min(cup_count, 2_147_483_647))
    struct.pack_into("<i", payload, ST_STAT_N_INTERVAL_OFFSET, 1 if speed_per_minute > 0 else 0)
    struct.pack_into("<i", payload, ST_STAT_N_INTERVAL_SUM_PER_MINUTE_OFFSET, min(speed_per_minute, 2_147_483_647))
    if speed_per_minute > 0:
        pulse_ms = max(1, min(2000, round(60000 / speed_per_minute)))
        struct.pack_into("<H", payload, ST_STAT_N_PULSE_INTERVAL_OFFSET, pulse_ms)
    return bytes(payload)


def send_ctcp_packet(host: str, port: int, packet: bytes, timeout: float) -> float:
    started = time.perf_counter()
    with socket.create_connection((host, port), timeout=timeout) as sock:
        sock.settimeout(timeout)
        sock.sendall(packet)
        try:
            sock.shutdown(socket.SHUT_WR)
        except OSError:
            pass
    return (time.perf_counter() - started) * 1000.0


def send_global_config(args: argparse.Namespace) -> float:
    payload = build_minimal_st_global_payload(args.subsys_num, args.channels, args.exits)
    packet = build_ctcp_packet(args.source_id, CTCP_HC_ID, CMD_FSM_CONFIG, payload)
    return send_ctcp_packet(args.ctcp_host, args.ctcp_port, packet, args.ctcp_timeout)


def send_statistics(args: argparse.Namespace, total_count: int, total_weight: int) -> float:
    payload = build_st_statistics_payload(
        total_count=total_count,
        total_weight=total_weight,
        subsys_id=args.subsys_id,
        speed_per_minute=args.speed_per_minute,
        cup_count=max(total_count * args.channels, total_count),
        exit_count=total_count,
    )
    packet = build_ctcp_packet(args.source_id, CTCP_HC_ID, CMD_FSM_STATISTICS, payload)
    return send_ctcp_packet(args.ctcp_host, args.ctcp_port, packet, args.ctcp_timeout)


def query_sys_fruit_info(args: argparse.Namespace) -> tuple[dict[str, Any], float]:
    ws_url = build_ws_url(args.base_url, args.ws_path)
    request_id = f"{args.run_id}-sys-{int(time.time() * 1000)}"
    query: dict[str, Any] = {
        "SystemID": args.subsys_id,
        "PageIndex": 1,
        "PageSize": 1,
        "SortOrder": "desc",
    }
    message = {
        "type": "querySysFruitInfo",
        "requestId": request_id,
        "sysFruitInfoQuery": query,
    }
    started = time.perf_counter()
    with SimpleWebSocket(ws_url, timeout=args.ws_timeout) as ws:
        ws.recv_json_until(lambda frame: frame.get("type") == "ready", timeout=args.ws_timeout)
        ws.send_text(json.dumps(message, ensure_ascii=False, separators=(",", ":")))
        ack = ws.recv_json_until(
            lambda frame: frame.get("type") == "commandAck"
            and frame.get("topic") == "querySysFruitInfo"
            and isinstance(frame.get("data"), dict)
            and frame["data"].get("requestId") == request_id,
            timeout=args.ws_timeout,
        )
    elapsed_ms = (time.perf_counter() - started) * 1000.0
    data = ack.get("data")
    if not isinstance(data, dict):
        raise StabilityError(f"querySysFruitInfo returned invalid data: {ack}")
    if not data.get("ok", False):
        raise StabilityError(f"querySysFruitInfo failed: {data}")
    return data, elapsed_ms


def latest_sys_row(query_data: dict[str, Any]) -> dict[str, Any] | None:
    items = query_data.get("Items")
    if not isinstance(items, list) or not items:
        return None
    item = items[0]
    return item if isinstance(item, dict) else None


def write_json(path: Path, data: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = path.with_suffix(path.suffix + ".tmp")
    tmp.write_text(json.dumps(json_safe(data), ensure_ascii=False, indent=2), encoding="utf-8")
    os.replace(tmp, path)


def format_progress(
    sent_count: int,
    sent_weight: int,
    db_row: dict[str, Any] | None,
    query_ms: float | None,
    home_stats: dict[str, Any] | None,
    frames: int,
) -> str:
    if db_row is None:
        db_text = "db=<empty>"
    else:
        db_count = int(db_row.get("BatchNumber") or 0)
        db_weight = int(db_row.get("BatchWeight") or 0)
        db_text = (
            f"db CustomerID={db_row.get('CustomerID')} SystemID={db_row.get('SystemID')} "
            f"count={db_count} weight={db_weight} "
            f"lagCount={sent_count - db_count} lagWeight={sent_weight - db_weight}"
        )
    home_text = "homeStats=<none>"
    if home_stats is not None:
        home_text = (
            f"homeStats count={home_stats.get('batchCount')} "
            f"weightTon={home_stats.get('batchWeightTon')} frames={frames}"
        )
    query_text = f"queryMs={query_ms:.1f}" if query_ms is not None else "queryMs=<skip>"
    return f"sent count={sent_count} weight={sent_weight} | {db_text} | {home_text} | {query_text}"


def run(args: argparse.Namespace) -> int:
    output_dir = args.output_dir / args.run_id
    events_path = output_dir / "events.jsonl"
    summary_path = output_dir / "summary.json"
    logger = JsonlLogger(events_path)

    stats: dict[str, Any] = {
        "started_at": now_text(),
        "run_id": args.run_id,
        "base_url": args.base_url,
        "ws_url": build_ws_url(args.base_url, args.ws_path),
        "ctcp_target": f"{args.ctcp_host}:{args.ctcp_port}",
        "packets_sent": 0,
        "db_queries": 0,
        "db_query_failures": 0,
        "max_db_query_ms": 0.0,
        "max_count_lag": 0,
        "max_weight_lag": 0,
        "last_error": "",
        "last_sent_count": args.start_count,
        "last_sent_weight": args.start_weight,
        "last_db_row": None,
        "last_home_stats": None,
    }

    monitor: HomeStatsMonitor | None = None
    if not args.no_ws_listen:
        monitor = HomeStatsMonitor(args.base_url, args.ws_path, args.ws_timeout)
        monitor.start()

    try:
        if not args.skip_global_config:
            elapsed_ms = send_global_config(args)
            logger.write("global_config_sent", elapsed_ms=elapsed_ms, bytes=ST_GLOBAL_SIZE)
            time.sleep(args.after_global_sleep)

        count = args.start_count
        weight = args.start_weight
        deadline = time.monotonic() + args.duration_seconds
        next_send = time.monotonic()
        next_verify = time.monotonic() + args.verify_interval

        while time.monotonic() < deadline:
            now = time.monotonic()
            if now < next_send:
                time.sleep(min(0.05, next_send - now))
                continue

            count += args.count_step
            weight += args.weight_step
            elapsed_ms = send_statistics(args, count, weight)
            stats["packets_sent"] += 1
            stats["last_sent_count"] = count
            stats["last_sent_weight"] = weight
            logger.write("statistics_sent", count=count, weight=weight, elapsed_ms=elapsed_ms)

            if args.once:
                next_verify = time.monotonic()

            if not args.no_db_verify and time.monotonic() >= next_verify:
                query_ms: float | None = None
                db_row: dict[str, Any] | None = None
                try:
                    query_data, query_ms = query_sys_fruit_info(args)
                    db_row = latest_sys_row(query_data)
                    stats["db_queries"] += 1
                    stats["max_db_query_ms"] = max(float(stats["max_db_query_ms"]), query_ms)
                    stats["last_db_row"] = db_row
                    if db_row is not None:
                        db_count = int(db_row.get("BatchNumber") or 0)
                        db_weight = int(db_row.get("BatchWeight") or 0)
                        stats["max_count_lag"] = max(int(stats["max_count_lag"]), count - db_count)
                        stats["max_weight_lag"] = max(int(stats["max_weight_lag"]), weight - db_weight)
                except Exception as exc:
                    stats["db_query_failures"] += 1
                    stats["last_error"] = str(exc)
                    logger.write("db_query_failed", error=str(exc))
                    if args.require_db_verify:
                        raise

                home_stats, frames, home_error = (None, 0, "")
                if monitor is not None:
                    home_stats, frames, home_error = monitor.snapshot()
                    stats["last_home_stats"] = home_stats
                    if home_error:
                        logger.write("home_stats_error", error=home_error)
                print(format_progress(count, weight, db_row, query_ms, home_stats, frames), flush=True)
                logger.write("verify", sent_count=count, sent_weight=weight, db_row=db_row, query_ms=query_ms, home_stats=home_stats)
                next_verify = time.monotonic() + args.verify_interval

            write_json(summary_path, stats)
            if args.once:
                break
            next_send += args.send_interval

        if args.once and not args.no_db_verify and args.final_wait > 0:
            time.sleep(args.final_wait)
            query_data, query_ms = query_sys_fruit_info(args)
            db_row = latest_sys_row(query_data)
            home_stats, frames, _ = monitor.snapshot() if monitor is not None else (None, 0, "")
            print(format_progress(count, weight, db_row, query_ms, home_stats, frames), flush=True)
            stats["db_queries"] += 1
            stats["max_db_query_ms"] = max(float(stats["max_db_query_ms"]), query_ms)
            stats["last_db_row"] = db_row
            stats["last_home_stats"] = home_stats
    except Exception as exc:
        stats["last_error"] = str(exc)
        logger.write("run_failed", error=str(exc))
        write_json(summary_path, stats)
        print(f"ERROR: {exc}", file=sys.stderr)
        return 2
    finally:
        if monitor is not None:
            monitor.stop()
        stats["finished_at"] = now_text()
        write_json(summary_path, stats)
        print(f"Summary: {summary_path}")
        print(f"Events: {events_path}")
    return 0


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Simulate CTCP realtime stats and verify tb_Sys_FruitInfo persistence.")
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL, help="HTTP base URL after fport, used for websocket query/listen.")
    parser.add_argument("--ws-path", default="/ws")
    parser.add_argument("--ctcp-host", default=DEFAULT_CTCP_HOST)
    parser.add_argument("--ctcp-port", type=int, default=DEFAULT_CTCP_STAT_PORT)
    parser.add_argument("--ctcp-timeout", type=float, default=3.0)
    parser.add_argument("--ws-timeout", type=int, default=10)
    parser.add_argument("--duration-seconds", type=float, default=120.0)
    parser.add_argument("--send-interval", type=float, default=0.2)
    parser.add_argument("--verify-interval", type=float, default=2.0)
    parser.add_argument("--final-wait", type=float, default=12.0, help="Extra wait before final query in --once mode.")
    parser.add_argument("--start-count", type=int, default=0)
    parser.add_argument("--start-weight", type=int, default=0)
    parser.add_argument("--count-step", type=int, default=10)
    parser.add_argument("--weight-step", type=int, default=100000)
    parser.add_argument("--speed-per-minute", type=int, default=300)
    parser.add_argument("--source-id", type=lambda text: int(text, 0), default=CTCP_DEFAULT_FSM_ID)
    parser.add_argument("--subsys-id", type=int, default=1)
    parser.add_argument("--subsys-num", type=int, default=1)
    parser.add_argument("--channels", type=int, default=1)
    parser.add_argument("--exits", type=int, default=48)
    parser.add_argument("--skip-global-config", action="store_true", help="Use when backend already has real FSM_CMD_CONFIG cached.")
    parser.add_argument("--after-global-sleep", type=float, default=0.2)
    parser.add_argument("--no-db-verify", action="store_true")
    parser.add_argument("--require-db-verify", action="store_true", default=True)
    parser.add_argument("--no-ws-listen", action="store_true", help="Do not subscribe to homeStats while sending.")
    parser.add_argument("--once", action="store_true", help="Send one statistics packet, then do a final DB query.")
    parser.add_argument("--run-id", default=safe_run_id())
    parser.add_argument("--output-dir", type=Path, default=DEFAULT_OUTPUT_DIR)
    args = parser.parse_args(argv)

    if args.duration_seconds <= 0:
        raise SystemExit("--duration-seconds must be > 0")
    if args.send_interval <= 0:
        raise SystemExit("--send-interval must be > 0")
    if args.verify_interval <= 0:
        raise SystemExit("--verify-interval must be > 0")
    if args.count_step < 0 or args.weight_step < 0:
        raise SystemExit("--count-step and --weight-step must be >= 0")
    if args.subsys_id <= 0 or args.subsys_num <= 0:
        raise SystemExit("--subsys-id and --subsys-num must be > 0")
    return args


def main(argv: list[str] | None = None) -> int:
    return run(parse_args(argv))


if __name__ == "__main__":
    raise SystemExit(main())
