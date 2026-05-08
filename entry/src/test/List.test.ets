import localUnitTest from './LocalUnit.test';
import goNativeTest from './GoNative.test';

export default function testsuite() {
  localUnitTest();
  goNativeTest();
}
