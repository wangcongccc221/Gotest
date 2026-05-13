import { common } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { http } from '@kit.NetworkKit';
import testNapi from 'libentry.so';

const DOMAIN = 0x0000;

@Entry
@Component
struct Index {
  @State message: string = 'Hello World';
  @State stGlobalLayout: string = '点击「StGlobal 布局」从 Go(libohos.so) 读取';

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

  private dumpLayoutToConsole(text: string): void {
    console.log('=== StGlobalLayoutReport begin（可全选在 HiLog 里复制）===');
    const lines = text.split('\n');
    for (let i = 0; i < lines.length; i++) {
      console.log(lines[i]);
    }
    console.log('=== StGlobalLayoutReport end ===');
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
        Button('StGlobal 布局（设备上 libohos.so）')
          .margin({ top: 12 })
          .onClick(() => {
            let report = testNapi.stGlobalLayoutReport();
            if (report.length === 0) {
              report = '空：请先用 go/ohos/build_ohos.ps1 编译 libohos.so，或 nativeLastError：' +
                testNapi.nativeLastError();
            }
            this.stGlobalLayout = report;
            this.dumpLayoutToConsole(report);
            hilog.info(DOMAIN, 'testTag', 'StGlobal layout chars = %{public}d', report.length);
          })
        Scroll() {
          Text(this.stGlobalLayout)
            .fontSize(9)
            .fontFamily('monospace')
            .width('100%')
        }
        .width('100%')
        .layoutWeight(1)
        .margin({ top: 8 })
      }
      .width('100%')
      .height('100%')
    }
    .height('100%')
  }
}
