package tcp

import (
	"os"
	"runtime"
	"testing"
)

// 在鸿蒙目标下查看布局：set GOOS=android&& set GOARCH=arm64&& go test ./Tcp -run TestStGlobalLayoutReport -v
func TestStGlobalLayoutReport(t *testing.T) {
	if os.Getenv("GOOS") == "" && os.Getenv("GOARCH") == "" {
		t.Logf("提示: 默认使用本机 %s/%s；要看 OHOS arm64 布局请设置 GOOS=android GOARCH=arm64 后再跑本测试\n", runtime.GOOS, runtime.GOARCH)
	}
	t.Log("\n" + StGlobalLayoutReport())
}
