import { AbilityConstant, ConfigurationConstant, UIAbility, Want } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { window } from '@kit.ArkUI';
import testNapi from 'libentry.so';

const DOMAIN = 0x0000;

export default class EntryAbility extends UIAbility {
  private tcpServerLogTimer: number = 0;

  private startTcpServer(): void {
    if (!testNapi) {
      hilog.error(DOMAIN, 'testTag', 'Native module libentry.so export object is undefined, skip TCP server start.');
      return;
    }

    if (!testNapi.startTcpServer) {
      hilog.error(DOMAIN, 'testTag', 'Native module libentry.so has no startTcpServer export, skip TCP server start.');
      return;
    }

    const tcpServerResult = testNapi.startTcpServer();
    hilog.info(DOMAIN, 'testTag', 'TCP server start result = %{public}d', tcpServerResult);
    if (tcpServerResult < 0 && testNapi.nativeLastError) {
      hilog.error(DOMAIN, 'testTag', 'TCP server native error: %{public}s', testNapi.nativeLastError());
    }
    if (tcpServerResult < 0 && testNapi.tcpServerLastMessage) {
      hilog.error(DOMAIN, 'testTag', 'TCP server last message: %{public}s', testNapi.tcpServerLastMessage());
    }
  }

  private startTcpServerLogPolling(): void {
    if (!testNapi || !testNapi.tcpServerLastMessage || this.tcpServerLogTimer !== 0) {
      return;
    }

    const hilogMaxPublic = 4000;

    this.tcpServerLogTimer = setInterval(() => {
      let guard = 0;
      while (guard < 5000) {
        guard++;
        const line = testNapi.tcpServerLastMessage();
        if (line.length === 0) {
          break;
        }
        for (let i = 0; i < line.length; i += hilogMaxPublic) {
          const part = line.substring(i, Math.min(i + hilogMaxPublic, line.length));
          hilog.info(DOMAIN, 'testTag', '%{public}s', part);
        }
      }
    }, 250);
  }

  private stopTcpServer(): void {
    if (this.tcpServerLogTimer !== 0) {
      clearInterval(this.tcpServerLogTimer);
      this.tcpServerLogTimer = 0;
    }

    if (!testNapi || !testNapi.stopTcpServer) {
      return;
    }

    const tcpServerResult = testNapi.stopTcpServer();
    hilog.info(DOMAIN, 'testTag', 'TCP server stop result = %{public}d', tcpServerResult);
  }

  onCreate(want: Want, launchParam: AbilityConstant.LaunchParam): void {
    try {
      this.context.getApplicationContext().setColorMode(ConfigurationConstant.ColorMode.COLOR_MODE_NOT_SET);
    } catch (err) {
      hilog.error(DOMAIN, 'testTag', 'Failed to set colorMode. Cause: %{public}s', JSON.stringify(err));
    }

    hilog.info(DOMAIN, 'testTag', '%{public}s', 'Ability onCreate');
  }

  onDestroy(): void {
    this.stopTcpServer();
    hilog.info(DOMAIN, 'testTag', '%{public}s', 'Ability onDestroy');
  }

  onWindowStageCreate(windowStage: window.WindowStage): void {
    // Main window is created, set main page for this ability
    hilog.info(DOMAIN, 'testTag', '%{public}s', 'Ability onWindowStageCreate');

    windowStage.loadContent('pages/Index', (err) => {
      if (err.code) {
        hilog.error(DOMAIN, 'testTag', 'Failed to load the content. Cause: %{public}s', JSON.stringify(err));
        return;
      }
      hilog.info(DOMAIN, 'testTag', 'Succeeded in loading the content.');
      this.startTcpServer();
      this.startTcpServerLogPolling();
      // 不在 Ability 启动时主动给下位机发 DISPLAY_ON。
      // 现在由前端 WebSocket 连接成功后发送 requestStGlobal，
      // Go 后端收到该信号后再触发 CTCP client，避免前端还没连上就错过 StGlobal。
    });
  }

  onWindowStageDestroy(): void {
    // Main window is destroyed, release UI related resources
    hilog.info(DOMAIN, 'testTag', '%{public}s', 'Ability onWindowStageDestroy');
  }

  onForeground(): void {
    // Ability has brought to foreground
    hilog.info(DOMAIN, 'testTag', '%{public}s', 'Ability onForeground');
  }

  onBackground(): void {
    // Ability has back to background
    hilog.info(DOMAIN, 'testTag', '%{public}s', 'Ability onBackground');
  }
}
