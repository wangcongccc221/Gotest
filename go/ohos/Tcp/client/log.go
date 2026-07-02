package client

import "gotest/ohos/Tcp/events"

func SetMessageLogger(logger func(string, ...any)) {
	events.SetLogger(logger)
}

func setCTCPServerLastMessage(format string, args ...any) {
}
