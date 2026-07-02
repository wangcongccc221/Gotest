import importlib.util
import sys
import unittest
from pathlib import Path


SCRIPT_PATH = Path(__file__).resolve().parents[1] / "tools" / "verify_weight_accumulate_save.py"


def load_module():
    tools_dir = str(SCRIPT_PATH.parent)
    if tools_dir not in sys.path:
        sys.path.insert(0, tools_dir)
    spec = importlib.util.spec_from_file_location("verify_weight_accumulate_save", SCRIPT_PATH)
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class SaveLatencyTrackerTest(unittest.TestCase):
    def test_observe_db_count_records_first_seen_latency_for_pending_packets(self):
        module = load_module()
        tracker = module.SaveLatencyTracker()

        tracker.record_send(1, 10.0)
        tracker.record_send(2, 11.0)
        tracker.record_send(3, 12.0)

        self.assertEqual(tracker.observe_db_count(2, 15.0), [5000.0, 4000.0])
        self.assertEqual(tracker.observe_db_count(2, 18.0), [])
        self.assertEqual(tracker.observe_db_count(3, 20.0), [8000.0])

        summary = tracker.summary()
        self.assertEqual(summary["save_latency_observed_packets"], 3)
        self.assertEqual(summary["save_latency_pending_packets"], 0)
        self.assertEqual(summary["save_latency_min_ms"], 4000.0)
        self.assertEqual(summary["save_latency_p50_ms"], 5000.0)
        self.assertEqual(summary["save_latency_p95_ms"], 8000.0)
        self.assertEqual(summary["save_latency_max_ms"], 8000.0)

    def test_summary_reports_pending_packets_without_latency_samples(self):
        module = load_module()
        tracker = module.SaveLatencyTracker()

        tracker.record_send(10, 1.0)
        tracker.record_send(11, 2.0)

        summary = tracker.summary()

        self.assertEqual(summary["save_latency_observed_packets"], 0)
        self.assertEqual(summary["save_latency_pending_packets"], 2)
        self.assertIsNone(summary["save_latency_min_ms"])
        self.assertIsNone(summary["save_latency_p50_ms"])
        self.assertIsNone(summary["save_latency_p95_ms"])
        self.assertIsNone(summary["save_latency_max_ms"])

    def test_ahead_db_count_does_not_confirm_current_run_packets(self):
        module = load_module()
        tracker = module.SaveLatencyTracker()

        tracker.record_send(1, 10.0)

        self.assertEqual(tracker.observe_db_count(99, 11.0), [])
        self.assertEqual(tracker.summary()["save_latency_pending_packets"], 1)


if __name__ == "__main__":
    unittest.main()
