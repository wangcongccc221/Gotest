import { AbilityConstant, ConfigurationConstant, UIAbility, Want } from '@kit.AbilityKit';
import { hilog } from '@kit.PerformanceAnalysisKit';
import { window } from '@kit.ArkUI';
import testNapi from 'libentry.so';

const DOMAIN = 0x0000;
const TCP_REMOTE_IP = '127.0.0.1';
const TCP_REMOTE_PORT = 6666;
const TCP_DEST_ID = 0x0100; // CommonFunction::EncodeSubsys(0)
const TCP_CMD = 0x0019; // HC_FSM_COMMAND_TYPE::HC_CMD_DISPLAY_ON

export default class EntryAbility extends UIAbility {
  private startTcpClient(): void {

    if (!testNapi) { //判断有没有动态库
      hilog.error(DOMAIN, 'testTag', 'Native module libentry.so export object is undefined, skip TCP client start.');
      return;
    }

    if (!testNapi.startTcpClient) { //判断有没有这个函数
      hilog.error(DOMAIN, 'testTag', 'Native module libentry.so has no startTcpClient export, skip TCP client start.');
      return;
    }

    const tcpClientResult = testNapi.startTcpClient(TCP_REMOTE_IP, TCP_REMOTE_PORT, TCP_DEST_ID, TCP_CMD); //调用函数
    
    hilog.info(DOMAIN, 'testTag', 'TCP client sync result = %{public}d, target = %{public}s:%{public}d, dest = %{public}d, cmd = %{public}d',
      tcpClientResult, TCP_REMOTE_IP, TCP_REMOTE_PORT, TCP_DEST_ID, TCP_CMD);
    if (tcpClientResult < 0 && testNapi.nativeLastError) {
      hilog.error(DOMAIN, 'testTag', 'TCP client native error: %{public}s', testNapi.nativeLastError());
    }
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
      this.startTcpClient();
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
