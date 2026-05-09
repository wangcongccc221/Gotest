import { describe, it, expect } from '@ohos/hypium';
import testNapi from 'libentry.so';

export default function goNativeTest() {
  describe('goNativeTest', () => {
    it('calls Go through the native module', 0, () => {
      expect(testNapi.add(2, 3)).assertEqual(5);
      expect(testNapi.hello()).assertEqual('Hello, HarmonyOS from Go');
      expect(testNapi.startServer()).assertEqual(18080);
      expect(testNapi.stopServer()).assertEqual(0);
      expect(testNapi.startOpcuaServer()).assertEqual(4840);
      expect(testNapi.stopOpcuaServer()).assertEqual(0);
      expect(testNapi.startModbusServer()).assertEqual(5020);
      expect(testNapi.stopModbusServer()).assertEqual(0);
      expect(testNapi.initOrm()).assertEqual(0);
      expect(testNapi.startTcpServer()).assertEqual(18081);
      expect(testNapi.startTcpClient('127.0.0.1', 6666, 0x0100, 0x0019)).assertEqual(18082);
      expect(testNapi.tcpSend('hello tcp')).assertEqual('hello tcp');
      expect(testNapi.stopTcpClient()).assertEqual(0);
      expect(testNapi.stopTcpServer()).assertEqual(0);
    });
  });
}
