#!/usr/bin/env python3
"""验证累加重量实时落库。

场景：模拟下位机每 1s 发一包 StStatistics，BatchWeight 累加（如 1000000 -> 1050000 -> 1100000 ...）。
后端每 3s 落一次库。脚本每 3s 查一次 tb_Sys_FruitInfo，比对已发送的累加值是否真正写进数据库。

链路：CTCP(1128) -> Go 后端 -> 周期落 SQLite(tb_Sys_FruitInfo) -> WebSocket querySysFruitInfo 查回。
纯标准库，复用 realtime_db_perf_test 的发包/查询逻辑。
"""

from __future__ import annotations

import argparse
import json
import math
import sys
import time
from pathlib import Path
from typing import Any
SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

from db_ws_stability_test import SimpleWebSocket, StabilityError, build_ws_url, json_safe, now_text
from realtime_db_perf_test import (
    HomeStatsMonitor,
    JsonlLogger,
    latest_sys_row,
    send_global_config,
    send_statistics,
    write_json,
)


DEFAULT_BASE_URL = "http://127.0.0.1:2778"
DEFAULT_CTCP_HOST = "127.0.0.1"
DEFAULT_CTCP_STAT_PORT = 1128
DEFAULT_OUTPUT_DIR = Path("reports") / "weight_accumulate_save"

CTCP_DEFAULT_FSM_ID = 0x0400  # 子系统4，避免和真实设备(0x0100/0x0200/0x0300)冲突


def safe_run_id() -> str:
    return __import__("datetime").datetime.now().strftime("%Y%m%d-%H%M%S")


def query_sys_fruit_info_by_index(args: argparse.Namespace, system_id: int = 0) -> tuple[dict[str, Any], float]:
    """查 tb_Sys_FruitInfo，SystemID 用后端写入的索引值（单子系统时为 0）。

    后端 realtimeSaveReplaceSysFruitInfos 按 subsys 索引 i 写 SystemID=i，
    与发包用的 --subsys-id 不同，这里必须传索引 0。
    """
    ws_url = build_ws_url(args.base_url, args.ws_path)
    request_id = f"{args.run_id}-sys-{int(time.time() * 1000)}"
    query: dict[str, Any] = {
        "SystemID": system_id,
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


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="验证累加重量每 3s 落库 tb_Sys_FruitInfo。")
    parser.add_argument("--base-url", default=DEFAULT_BASE_URL, help="HTTP base URL (hdc fport 后)。")
    parser.add_argument("--ws-path", default="/ws")
    parser.add_argument("--ws-timeout", type=int, default=10)
    parser.add_argument("--ctcp-host", default=DEFAULT_CTCP_HOST)
    parser.add_argument("--ctcp-port", type=int, default=DEFAULT_CTCP_STAT_PORT)
    parser.add_argument("--ctcp-timeout", type=float, default=3.0)
    parser.add_argument("--source-id", type=lambda t: int(t, 0), default=CTCP_DEFAULT_FSM_ID)
    parser.add_argument("--subsys-id", type=int, default=4)
    parser.add_argument("--subsys-num", type=int, default=4)
    parser.add_argument("--channels", type=int, default=1)
    parser.add_argument("--exits", type=int, default=48)
    parser.add_argument("--speed-per-minute", type=int, default=300)
    parser.add_argument("--skip-global-config", action="store_true", help="后端已有真实 CONFIG 缓存时跳过。")
    parser.add_argument("--after-global-sleep", type=float, default=0.2)
    parser.add_argument("--start-count", type=int, default=0)
    parser.add_argument("--start-weight", type=int, default=1000000, help="初始累计重量。")
    parser.add_argument("--count-step", type=int, default=1, help="每包累加数量（1s 一包）。")
    parser.add_argument("--weight-step", type=int, default=50000, help="每包累加重量。")
    parser.add_argument("--send-interval", type=float, default=1.0, help="发包间隔秒，对应程序 1s 一次。")
    parser.add_argument("--verify-interval", type=float, default=4.0, help="查库间隔秒，需 > 后端落库间隔(3s)。")
    parser.add_argument("--duration-seconds", type=float, default=60.0, help="总运行时长秒。")
    parser.add_argument("--final-wait", type=float, default=5.0, help="结束后再等多久做最后一次查库。")
    parser.add_argument("--no-ws-listen", action="store_true", help="不订阅 homeStats 推送。")
    parser.add_argument("--run-id", default=safe_run_id())
    parser.add_argument("--output-dir", type=Path, default=DEFAULT_OUTPUT_DIR)
    args = parser.parse_args(argv)
    if args.duration_seconds <= 0:
        raise SystemExit("--duration-seconds must be > 0")
    if args.send_interval <= 0 or args.verify_interval <= 0:
        raise SystemExit("--send-interval 和 --verify-interval 必须 > 0")
    if args.weight_step < 0 or args.count_step < 0:
        raise SystemExit("--weight-step 和 --count-step 必须 >= 0")
    return args


def fmt_money(n: int) -> str:
    return f"{n:,}"


def classify_lag(weight_lag: int, tolerance: int) -> str:
    """持续发包时查库必然有滞后，按容差分类。

    OK        完全追平（最终查库期望）
    OK_LAG    滞后在容差内（持续发包时正常）
    LAG       滞后超过容差（可能落库延迟）
    AHEAD     DB 值大于发送值（异常，可能混入其他数据源）
    """
    if weight_lag == 0:
        return "OK"
    if weight_lag > 0:
        return "OK_LAG" if weight_lag <= tolerance else "LAG"
    return "AHEAD"


class SaveLatencyTracker:
    """Tracks when sent cumulative counts are first observed in the database."""

    def __init__(self) -> None:
        self._pending: dict[int, float] = {}
        self._max_sent_count = 0
        self._latencies_ms: list[float] = []

    def record_send(self, count: int, sent_at: float) -> None:
        self._pending[count] = sent_at
        self._max_sent_count = max(self._max_sent_count, count)

    def observe_db_count(self, db_count: int, observed_at: float) -> list[float]:
        if db_count > self._max_sent_count:
            return []

        confirmed = [count for count in self._pending if count <= db_count]
        confirmed.sort()
        observed_latencies: list[float] = []
        for count in confirmed:
            sent_at = self._pending.pop(count)
            latency_ms = max(0.0, (observed_at - sent_at) * 1000.0)
            self._latencies_ms.append(latency_ms)
            observed_latencies.append(latency_ms)
        return observed_latencies

    def summary(self) -> dict[str, Any]:
        samples = sorted(self._latencies_ms)
        return {
            "save_latency_observed_packets": len(samples),
            "save_latency_pending_packets": len(self._pending),
            "save_latency_min_ms": round(samples[0], 3) if samples else None,
            "save_latency_p50_ms": self._percentile(samples, 50),
            "save_latency_p95_ms": self._percentile(samples, 95),
            "save_latency_max_ms": round(samples[-1], 3) if samples else None,
        }

    @staticmethod
    def _percentile(samples: list[float], percentile: int) -> float | None:
        if not samples:
            return None
        index = max(0, min(len(samples) - 1, math.ceil(len(samples) * percentile / 100) - 1))
        return round(samples[index], 3)


def update_save_latency_stats(stats: dict[str, Any], tracker: SaveLatencyTracker) -> None:
    stats.update(tracker.summary())


def run(args: argparse.Namespace) -> int:
    output_dir = args.output_dir / args.run_id
    events_path = output_dir / "events.jsonl"
    summary_path = output_dir / "summary.json"
    logger = JsonlLogger(events_path)

    stats: dict[str, Any] = {
        "started_at": now_text(),
        "run_id": args.run_id,
        "base_url": args.base_url,
        "ctcp_target": f"{args.ctcp_host}:{args.ctcp_port}",
        "send_interval": args.send_interval,
        "verify_interval": args.verify_interval,
        "packets_sent": 0,
        "db_queries": 0,
        "db_query_failures": 0,
        "save_observed": 0,
        "max_count_lag": 0,
        "max_weight_lag": 0,
        "last_sent_count": args.start_count,
        "last_sent_weight": args.start_weight,
        "last_db_count": None,
        "last_db_weight": None,
        "last_error": "",
        "save_latency_observed_packets": 0,
        "save_latency_pending_packets": 0,
        "save_latency_min_ms": None,
        "save_latency_p50_ms": None,
        "save_latency_p95_ms": None,
        "save_latency_max_ms": None,
        "mismatches": [],
    }

    monitor: HomeStatsMonitor | None = None
    if not args.no_ws_listen:
        from realtime_db_perf_test import build_ws_url  # lazy import
        monitor = HomeStatsMonitor(args.base_url, args.ws_path, args.ws_timeout)
        monitor.start()
        logger.write("ws_listen_started", ws_url=build_ws_url(args.base_url, args.ws_path))

    count = args.start_count
    weight = args.start_weight
    exit_code = 0
    latency_tracker = SaveLatencyTracker()
    # 一个落库周期内的理论增量：发包频率 × 落库间隔。
    # 持续发包时查库必然滞后这个量，不算异常。
    cycle_gain = args.weight_step * max(1, args.verify_interval / max(0.1, args.send_interval))
    # 容差：一个周期增量 + 10% 余量
    weight_tolerance = int(cycle_gain * 1.1)
    try:
        if not args.skip_global_config:
            elapsed_ms = send_global_config(args)
            logger.write("global_config_sent", elapsed_ms=elapsed_ms)
            time.sleep(args.after_global_sleep)

        deadline = time.monotonic() + args.duration_seconds
        next_send = time.monotonic()
        next_verify = time.monotonic() + args.verify_interval

        print(
            f"[{now_text()}] 开始累加发送: 起始 weight={fmt_money(weight)} "
            f"+{fmt_money(args.weight_step)}/包, 发={args.send_interval}s 查={args.verify_interval}s "
            f"时长={args.duration_seconds}s",
            flush=True,
        )

        while time.monotonic() < deadline:
            now = time.monotonic()
            if now < next_send:
                time.sleep(min(0.05, next_send - now))
                continue

            count += args.count_step
            weight += args.weight_step
            elapsed_ms = send_statistics(args, count, weight)
            latency_tracker.record_send(count, time.monotonic())
            stats["packets_sent"] += 1
            stats["last_sent_count"] = count
            stats["last_sent_weight"] = weight
            logger.write("statistics_sent", count=count, weight=weight, elapsed_ms=elapsed_ms)
            print(
                f"[{now_text()}] 发 #{stats['packets_sent']:04d} count={count} weight={fmt_money(weight)} ({elapsed_ms:.1f}ms)",
                flush=True,
            )

            if time.monotonic() >= next_verify:
                db_row = do_verify(args, stats, logger, monitor)
                if db_row is not None:
                    db_count = int(db_row.get("BatchNumber") or 0)
                    db_weight = int(db_row.get("BatchWeight") or 0)
                    observed_latencies = latency_tracker.observe_db_count(db_count, time.monotonic())
                    if observed_latencies:
                        logger.write(
                            "save_latency_observed",
                            db_count=db_count,
                            observed_packets=len(observed_latencies),
                            max_latency_ms=max(observed_latencies),
                        )
                    stats["last_db_count"] = db_count
                    stats["last_db_weight"] = db_weight
                    count_lag = count - db_count
                    weight_lag = weight - db_weight
                    stats["max_count_lag"] = max(int(stats["max_count_lag"]), count_lag)
                    stats["max_weight_lag"] = max(int(stats["max_weight_lag"]), weight_lag)
                    mark = classify_lag(weight_lag, weight_tolerance)
                    print(
                        f"[{now_text()}] 查库 #{stats['db_queries']:04d} {mark} "
                        f"db_count={db_count} db_weight={fmt_money(db_weight)} "
                        f"lagCount={count_lag} lagWeight={fmt_money(weight_lag)} "
                        f"(容差={fmt_money(weight_tolerance)})",
                        flush=True,
                    )
                    if mark in ("OK", "OK_LAG"):
                        stats["save_observed"] += 1
                    else:
                        stats["mismatches"].append({
                            "at": now_text(),
                            "sent_count": count, "sent_weight": weight,
                            "db_count": db_count, "db_weight": db_weight,
                            "count_lag": count_lag, "weight_lag": weight_lag,
                        })
                    update_save_latency_stats(stats, latency_tracker)
                next_verify = time.monotonic() + args.verify_interval

            update_save_latency_stats(stats, latency_tracker)
            write_json(summary_path, stats)
            next_send += args.send_interval

        if args.final_wait > 0:
            print(f"[{now_text()}] 等待 {args.final_wait}s 后做最终查库...", flush=True)
            time.sleep(args.final_wait)
            db_row = do_verify(args, stats, logger, monitor)
            if db_row is not None:
                db_count = int(db_row.get("BatchNumber") or 0)
                db_weight = int(db_row.get("BatchWeight") or 0)
                observed_latencies = latency_tracker.observe_db_count(db_count, time.monotonic())
                if observed_latencies:
                    logger.write(
                        "save_latency_observed",
                        db_count=db_count,
                        observed_packets=len(observed_latencies),
                        max_latency_ms=max(observed_latencies),
                    )
                stats["last_db_count"] = db_count
                stats["last_db_weight"] = db_weight
                count_lag = count - db_count
                weight_lag = weight - db_weight
                stats["max_count_lag"] = max(int(stats["max_count_lag"]), count_lag)
                stats["max_weight_lag"] = max(int(stats["max_weight_lag"]), weight_lag)
                mark = classify_lag(weight_lag, 0)  # 最终查库要求完全追平
                print(
                    f"[{now_text()}] 最终查库 {mark} "
                    f"db_count={db_count} db_weight={fmt_money(db_weight)} "
                    f"lagCount={count_lag} lagWeight={fmt_money(weight_lag)}",
                    flush=True,
                )
                if mark == "OK":
                    stats["save_observed"] += 1
                update_save_latency_stats(stats, latency_tracker)

    except StabilityError as exc:
        stats["last_error"] = str(exc)
        logger.write("run_failed", error=str(exc))
        print(f"ERROR: {exc}", file=sys.stderr)
        exit_code = 2
    except Exception as exc:
        stats["last_error"] = repr(exc)
        logger.write("run_crashed", error=repr(exc))
        print(f"ERROR: {repr(exc)}", file=sys.stderr)
        exit_code = 2
    finally:
        if monitor is not None:
            monitor.stop()
        update_save_latency_stats(stats, latency_tracker)
        stats["finished_at"] = now_text()
        write_json(summary_path, stats)
        print(f"\nSummary: {summary_path}")
        print(f"Events:  {events_path}")
        print(
            f"结果: 发包={stats['packets_sent']} 查库={stats['db_queries']} "
            f"完全落库次数={stats['save_observed']} "
            f"最大重量lag={fmt_money(stats['max_weight_lag'])} "
            f"落库延迟p95={stats['save_latency_p95_ms']}ms "
            f"max={stats['save_latency_max_ms']}ms"
        )
    return exit_code


def do_verify(
    args: argparse.Namespace,
    stats: dict[str, Any],
    logger: JsonlLogger,
    monitor: HomeStatsMonitor | None,
) -> dict[str, Any] | None:
    try:
        query_data, query_ms = query_sys_fruit_info_by_index(args, system_id=3)
        db_row = latest_sys_row(query_data)
        stats["db_queries"] += 1
        logger.write("db_verified", db_row=db_row, query_ms=query_ms)
        return db_row
    except Exception as exc:
        stats["db_query_failures"] += 1
        stats["last_error"] = str(exc)
        logger.write("db_query_failed", error=str(exc))
        print(f"[{now_text()}] 查库失败: {exc}", file=sys.stderr)
        return None


def main(argv: list[str] | None = None) -> int:
    return run(parse_args(argv))


if __name__ == "__main__":
    raise SystemExit(main())
