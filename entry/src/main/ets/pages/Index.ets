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
  @State stGlobalParsedFullJson: string = '点击「StGlobal 全量解析 JSON」从 NAPI 读取（约几十万字符；界面仅预览一段，完整看 console / HiLog）';

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

  private dumpLargeTextToConsole(tag: string, text: string): void {
    console.log(`=== ${tag} begin ===`);
    const lines = text.split('\n');
    for (let i = 0; i < lines.length; i++) {
      console.log(lines[i]);
    }
    console.log(`=== ${tag} end，共 ${text.length} 字符 ===`);
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
        Button('StGlobal 全量解析 JSON（收到 0x1000 后）')
          .margin({ top: 8 })
          .onClick(() => {
            if (!testNapi.lastStGlobalFullJSON) {
              this.stGlobalParsedFullJson = 'NAPI 未导出 lastStGlobalFullJSON，请重新编译 libohos.so + entry';
              return;
            }
            let j = testNapi.lastStGlobalFullJSON();
            if (j.length === 0) {
              j = '空：请先收到 FSM_CMD_CONFIG，或 nativeLastError：' + testNapi.nativeLastError();
              this.stGlobalParsedFullJson = j;
              return;
            }
            this.dumpLargeTextToConsole('StGlobal 全量解析 JSON', j);
            hilog.info(DOMAIN, 'testTag', 'StGlobal 全量 JSON chars = %{public}d', j.length);
            const maxPreview = 20000;
            if (j.length > maxPreview) {
              this.stGlobalParsedFullJson = j.substring(0, maxPreview) +
                `\n\n... （界面只预览前 ${maxPreview} 字符；完整已在 console 按行输出，HiLog 搜「CTCP StGlobal 全量」）`;
            } else {
              this.stGlobalParsedFullJson = j;
            }
          })
        Scroll() {
          Text(this.stGlobalParsedFullJson)
            .fontSize(9)
            .fontFamily('monospace')
            .width('100%')
        }
        .width('100%')
        .height(200)
        .margin({ top: 4 })
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
