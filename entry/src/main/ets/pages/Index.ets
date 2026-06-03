import { hilog } from '@kit.PerformanceAnalysisKit';
import testNapi from 'libentry.so';

const DOMAIN = 0x0000;

@Entry
@Component
struct Index {
  @State stGlobalLayout: string = '点击「StGlobal 布局」从 Go(libohos.so) 读取';

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
        Text('Go 后端服务已随应用启动')
          .fontSize($r('app.float.page_text_font_size'))
          .fontWeight(FontWeight.Bold)
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
