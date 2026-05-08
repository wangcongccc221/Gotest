export const add: (a: number, b: number) => number;
export const hello: () => string;
export const startServer: () => number;
export const stopServer: () => number;
export const startOpcuaServer: () => number;
export const stopOpcuaServer: () => number;
export const startModbusServer: () => number;
export const stopModbusServer: () => number;
export const initOrm: (databasePath?: string) => number;
export const startTcpServer: () => number;
export const stopTcpServer: () => number;
export const startTcpClient: {
  (): number;
  (remoteIp: string, remotePort: number, localIp?: string): number;
};
export const stopTcpClient: () => number;
export const tcpSend: (message: string) => string;
export const nativeLastError: () => string;

declare const testNapi: {
  add: typeof add;
  hello: typeof hello;
  startServer: typeof startServer;
  stopServer: typeof stopServer;
  startOpcuaServer: typeof startOpcuaServer;
  stopOpcuaServer: typeof stopOpcuaServer;
  startModbusServer: typeof startModbusServer;
  stopModbusServer: typeof stopModbusServer;
  initOrm: typeof initOrm;
  startTcpServer: typeof startTcpServer;
  stopTcpServer: typeof stopTcpServer;
  startTcpClient: typeof startTcpClient;
  stopTcpClient: typeof stopTcpClient;
  tcpSend: typeof tcpSend;
  nativeLastError: typeof nativeLastError;
};

export default testNapi;
