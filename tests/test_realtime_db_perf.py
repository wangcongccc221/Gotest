import importlib.util
import struct
import sys
import unittest
from pathlib import Path


SCRIPT_PATH = Path(__file__).resolve().parents[1] / "tools" / "realtime_db_perf_test.py"


def load_module():
    tools_dir = str(SCRIPT_PATH.parent)
    if tools_dir not in sys.path:
        sys.path.insert(0, tools_dir)
    spec = importlib.util.spec_from_file_location("realtime_db_perf_test", SCRIPT_PATH)
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class RealtimeDbPerfTest(unittest.TestCase):
    def test_build_ctcp_packet_uses_sync_and_little_endian_head(self):
        module = load_module()

        packet = module.build_ctcp_packet(0x0100, 0x1000, 0x1001, b"abc")

        self.assertEqual(packet[:4], b"SYNC")
        self.assertEqual(struct.unpack("<iii", packet[4:16]), (0x0100, 0x1000, 0x1001))
        self.assertEqual(packet[16:], b"abc")

    def test_build_st_statistics_payload_sets_realtime_counter_fields(self):
        module = load_module()

        payload = module.build_st_statistics_payload(
            total_count=123,
            total_weight=456000,
            subsys_id=1,
            speed_per_minute=300,
            cup_count=246,
            exit_count=120,
        )

        self.assertEqual(len(payload), module.ST_STATISTICS_SIZE)
        self.assertEqual(struct.unpack_from("<Q", payload, module.ST_STAT_N_CHANNEL_TOTAL_COUNT_OFFSET)[0], 123)
        self.assertEqual(struct.unpack_from("<Q", payload, module.ST_STAT_N_CHANNEL_WEIGHT_COUNT_OFFSET)[0], 456000)
        self.assertEqual(struct.unpack_from("<i", payload, module.ST_STAT_N_SUBSYS_ID_OFFSET)[0], 1)
        self.assertEqual(struct.unpack_from("<i", payload, module.ST_STAT_N_TOTAL_CUP_NUM_OFFSET)[0], 246)
        self.assertEqual(struct.unpack_from("<i", payload, module.ST_STAT_N_INTERVAL_SUM_PER_MINUTE_OFFSET)[0], 300)

    def test_build_minimal_st_global_payload_enables_subsystem_for_realtime_save(self):
        module = load_module()

        payload = module.build_minimal_st_global_payload(subsys_num=2, channels=3, exits=48)

        self.assertEqual(len(payload), module.ST_GLOBAL_SIZE)
        self.assertEqual(payload[module.ST_GLOBAL_SYS_N_CHANNEL_INFO_OFFSET], 3)
        self.assertEqual(payload[module.ST_GLOBAL_SYS_N_CHANNEL_INFO_OFFSET + 1], 3)
        self.assertEqual(payload[module.ST_GLOBAL_SYS_N_SUBSYS_NUM_OFFSET], 2)
        self.assertEqual(payload[module.ST_GLOBAL_SYS_N_EXIT_NUM_OFFSET], 48)


if __name__ == "__main__":
    unittest.main()
