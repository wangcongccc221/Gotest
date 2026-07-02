import importlib.util
import json
import sys
import unittest
from unittest import mock
from pathlib import Path


SCRIPT_PATH = Path(__file__).resolve().parents[1] / "tools" / "db_ws_stability_test.py"


def load_module():
    spec = importlib.util.spec_from_file_location("db_ws_stability_test", SCRIPT_PATH)
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class DbWsStabilityHelpersTest(unittest.TestCase):
    def test_parse_api_envelope_decodes_json_data_string(self):
        module = load_module()
        payload = json.dumps(
            {
                "returnCode": 1,
                "returnMessage": "",
                "data": json.dumps([{"FCode": "A001", "FName": "alpha"}]),
            }
        ).encode("utf-8")

        data = module.parse_api_envelope(payload, "sample")

        self.assertEqual(data, [{"FCode": "A001", "FName": "alpha"}])

    def test_build_fault_rows_is_deterministic_and_uses_test_type(self):
        module = load_module()

        first = module.build_fault_rows(run_id="night", fault_type=901, count=3, seed=7)
        second = module.build_fault_rows(run_id="night", fault_type=901, count=3, seed=7)

        self.assertEqual(first, second)
        self.assertEqual([row["FID"] for row in first], [0, 0, 0])
        self.assertEqual({row["FType"] for row in first}, {901})
        self.assertEqual(
            [row["FCode"] for row in first],
            [
                "STAB-night-000001",
                "STAB-night-000002",
                "STAB-night-000003",
            ],
        )

    def test_compare_fault_rows_reports_missing_and_mismatched_values(self):
        module = load_module()
        expected = [
            {"FCode": "A", "FName": "alpha", "FMessage": "expected"},
            {"FCode": "B", "FName": "beta", "FMessage": "expected"},
        ]
        actual = [
            {"FCode": "A", "FName": "alpha", "FMessage": "changed"},
            {"FCode": "C", "FName": "gamma", "FMessage": "extra"},
        ]

        result = module.compare_fault_rows(expected, actual)

        self.assertFalse(result["ok"])
        self.assertEqual(result["expected_count"], 2)
        self.assertEqual(result["actual_count"], 2)
        self.assertEqual(result["missing_codes"], ["B"])
        self.assertEqual(result["extra_codes"], ["C"])
        self.assertEqual(result["mismatched_codes"], ["A"])

    def test_json_safe_converts_paths_inside_nested_values(self):
        module = load_module()

        data = module.json_safe({"path": Path("reports") / "x", "items": [Path("a")]})

        self.assertEqual(data, {"path": "reports\\x", "items": ["a"]})

    def test_websocket_accept_validation_keeps_header_value_case_sensitive(self):
        module = load_module()
        key = "dGhlIHNhbXBsZSBub25jZQ=="
        header = (
            b"HTTP/1.1 101 Switching Protocols\r\n"
            b"Upgrade: websocket\r\n"
            b"Connection: Upgrade\r\n"
            b"Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=\r\n"
            b"\r\n"
        )

        self.assertTrue(module.websocket_accept_is_valid(header, key))

    def test_loop_failure_records_duration_for_report_averages(self):
        module = load_module()

        class FailingClient:
            def post_api(self, path, payload):
                if path == "/Api/Fault/SaveBaseFaults":
                    raise module.StabilityError("/Api/Fault/SaveBaseFaults timeout after 30s")
                return []

        class MemoryLogger:
            def __init__(self):
                self.events = []

            def write(self, event, **fields):
                self.events.append((event, fields))

        args = type(
            "Args",
            (),
            {
                "seed": 10,
                "run_id": "unit",
                "batch_size": 2,
                "ws_mode": "off",
                "ws_timeout": 1,
            },
        )()
        stats = module.RunStats(
            started_at="now",
            base_url="http://example",
            ws_url="ws://example/ws",
            run_id="unit",
            fault_type=9000,
            batch_size=2,
        )
        timings = []

        with self.assertRaises(module.StabilityError):
            module.run_once(FailingClient(), "ws://example/ws", args, stats, MemoryLogger(), timings, 1)

        self.assertEqual(stats.loops_started, 1)
        self.assertIsNotNone(stats.min_loop_seconds)
        self.assertGreater(stats.average_loop_seconds, 0)

    def test_busy_or_timeout_errors_are_classified_for_backoff(self):
        module = load_module()

        self.assertTrue(module.is_database_busy_or_timeout("/orm HTTP 500: SQLITE_BUSY"))
        self.assertTrue(module.is_database_busy_or_timeout("database is locked (5)"))
        self.assertTrue(module.is_database_busy_or_timeout("/Api/Fault/SaveBaseFaults timeout after 30s"))
        self.assertFalse(module.is_database_busy_or_timeout("database compare failed"))

    def test_failure_kind_classifies_startup_and_loop_errors(self):
        module = load_module()

        self.assertEqual(module.failure_kind("/ping network error: refused"), "http")
        self.assertEqual(module.failure_kind("/orm HTTP 500: database is locked"), "http")
        self.assertEqual(module.failure_kind("websocket closed by server"), "websocket")
        self.assertEqual(module.failure_kind("database compare failed"), "compare")

    def test_api_client_wraps_connection_reset_as_stability_error(self):
        module = load_module()
        client = module.ApiClient("http://example.test", timeout=1)

        with mock.patch.object(module.urllib.request, "urlopen", side_effect=ConnectionResetError("reset")):
            with self.assertRaisesRegex(module.StabilityError, "/ping network error"):
                client.get_json("/ping")


if __name__ == "__main__":
    unittest.main()
