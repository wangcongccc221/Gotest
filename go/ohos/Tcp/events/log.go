package events

import (
	"fmt"
	"sync"
	"time"
)

const maxBufferedMessages = 2048

var chinaLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

var logState = struct {
	sync.Mutex
	latest   string
	messages []string
	logger   func(string, ...any)
}{
	latest: "CTCP server not started",
}

func SetLogger(logger func(string, ...any)) {
	logState.Lock()
	logState.logger = logger
	logState.Unlock()
}

func Info(format string, args ...any) {
	message := time.Now().In(chinaLocation).Format("15:04:05.000 ") + fmt.Sprintf(format, args...)

	logState.Lock()
	logState.latest = message
	logState.messages = append(logState.messages, message)
	if len(logState.messages) > maxBufferedMessages {
		logState.messages = logState.messages[len(logState.messages)-maxBufferedMessages:]
	}
	logger := logState.logger
	logState.Unlock()

	if logger != nil {
		logger("%s", message)
	}
}

func LastMessage() string {
	logState.Lock()
	defer logState.Unlock()
	return logState.latest
}

func NextMessage() string {
	logState.Lock()
	defer logState.Unlock()
	if len(logState.messages) == 0 {
		return ""
	}
	message := logState.messages[0]
	logState.messages = logState.messages[1:]
	return message
}
