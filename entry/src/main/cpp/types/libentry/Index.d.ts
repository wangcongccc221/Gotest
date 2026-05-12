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
export const startTcpClient: (remoteIp: string, remotePort: number, destId: number, cmd: number) => number;
export const stopTcpClient: () => number;
export const tcpSend: (message: string) => string;
export const nativeLastError: () => string;
export const tcpServerLastMessage: () => string;
/** StGlobal 及嵌套结构体在当前 libohos.so 编译目标下的 sizeof/offset（需在设备上重新编译 Go 库后调用）。 */
export const stGlobalLayoutReport: () => string;
/** 最近一次 FSM_CMD_CONFIG(0x1000) 的 StGlobal 完整 JSON；未收到前为空串。字符串可能很大。 */
export const lastStGlobalFullJSON: () => string;

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
  tcpServerLastMessage: typeof tcpServerLastMessage;
  stGlobalLayoutReport: typeof stGlobalLayoutReport;
  lastStGlobalFullJSON: typeof lastStGlobalFullJSON;
};

export default testNapi;
