const NativeMock: Record<string, Object> = {
  'add': (a: number, b: number) => {
    return a + b;
  },
  'hello': () => {
    return 'Hello, HarmonyOS from Go';
  },
  'startServer': () => {
    return 18080;
  },
  'stopServer': () => {
    return 0;
  },
  'startOpcuaServer': () => {
    return 4840;
  },
  'stopOpcuaServer': () => {
    return 0;
  },
  'startModbusServer': () => {
    return 5020;
  },
  'stopModbusServer': () => {
    return 0;
  },
  'initOrm': (_databasePath?: string) => {
    return 0;
  },
  'startTcpServer': () => {
    return 1128;
  },
  'stopTcpServer': () => {
    return 0;
  },
  'startTcpClient': (_remoteIp: string, _remotePort: number, _destId: number, _cmd: number) => {
    return 0;
  },
  'stopTcpClient': () => {
    return 0;
  },
  'tcpSend': (message: string) => {
    return message;
  },
  'nativeLastError': () => {
    return '';
  },
  'tcpServerLastMessage': () => {
    return 'CTCP mock server received 0 bytes';
  },
};

export const add = NativeMock.add as (a: number, b: number) => number;
export const hello = NativeMock.hello as () => string;
export const startServer = NativeMock.startServer as () => number;
export const stopServer = NativeMock.stopServer as () => number;
export const startOpcuaServer = NativeMock.startOpcuaServer as () => number;
export const stopOpcuaServer = NativeMock.stopOpcuaServer as () => number;
export const startModbusServer = NativeMock.startModbusServer as () => number;
export const stopModbusServer = NativeMock.stopModbusServer as () => number;
export const initOrm = NativeMock.initOrm as (databasePath?: string) => number;
export const startTcpServer = NativeMock.startTcpServer as () => number;
export const stopTcpServer = NativeMock.stopTcpServer as () => number;
export const startTcpClient = NativeMock.startTcpClient as (remoteIp: string, remotePort: number, destId: number, cmd: number) => number;
export const stopTcpClient = NativeMock.stopTcpClient as () => number;
export const tcpSend = NativeMock.tcpSend as (message: string) => string;
export const nativeLastError = NativeMock.nativeLastError as () => string;
export const tcpServerLastMessage = NativeMock.tcpServerLastMessage as () => string;

export default NativeMock;
