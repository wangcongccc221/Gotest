import { common } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { http } from '@kit.NetworkKit';
import testNapi from 'libentry.so';

const DOMAIN = 0x0000;

@Entry
@Component
struct Index {
  @State message: string = 'Hello World';

  private async requestGin(port: number): Promise<string> {
    const client = http.createHttp();
    try {
      const response = await client.request(`http://127.0.0.1:${port}/ping`, {
        method: http.RequestMethod.GET,
        expectDataType: http.HttpDataType.STRING,
        connectTimeout: 3000,
        readTimeout: 3000,
      });
      return `${response.result}`;
    } catch (err) {
      const message = JSON.stringify(err);
      hilog.error(DOMAIN, 'testTag', 'Gin /ping failed: %{public}s', message);
      return `Gin request failed: ${message}`;
    } finally {
      client.destroy();
    }
  }

  build() {
    Row() {
      Column() {
        Text(this.message)
          .fontSize($r('app.float.page_text_font_size'))
          .fontWeight(FontWeight.Bold)
          .onClick(async () => {
            const port = testNapi.startServer();
            const opcuaServerPort = testNapi.startOpcuaServer();
            const modbusServerPort = testNapi.startModbusServer();
            const appContext = getContext(this) as common.UIAbilityContext;
            const ormDatabasePath = `${appContext.databaseDir}/harmony_go_orm.db`;

            const ormResult = testNapi.initOrm(ormDatabasePath);
            const tcpServerPort = testNapi.startTcpServer();
            const sum = testNapi.add(2, 3);
            const pong = await this.requestGin(port);
            const tcpResponse = testNapi.tcpSend('hello tcp');
            this.message = `${pong}\n${tcpResponse}`;
            hilog.info(DOMAIN, 'testTag', 'Go NAPI 2 + 3 = %{public}s, Gin port = %{public}d', `${sum}`, port);
            hilog.info(DOMAIN, 'testTag', 'Gin /ping = %{public}s', pong);
            hilog.info(DOMAIN, 'testTag', 'OPC UA server port = %{public}d', opcuaServerPort);
            hilog.info(DOMAIN, 'testTag', 'Modbus TCP server port = %{public}d', modbusServerPort);
            hilog.info(DOMAIN, 'testTag', 'GORM init result = %{public}d, database = %{public}s',
              ormResult, ormDatabasePath);
            hilog.info(DOMAIN, 'testTag', 'TCP server port = %{public}d, TCP response = %{public}s',
              tcpServerPort, tcpResponse);
          })
      }
      .width('100%')
    }
    .height('100%')
  }
}
