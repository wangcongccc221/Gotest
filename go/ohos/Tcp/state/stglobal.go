package state

import "sync"

var stGlobalJSONState = struct {
	sync.Mutex
	jsonText string
}{}

func LastStGlobalFullJSON() string {
	stGlobalJSONState.Lock()
	defer stGlobalJSONState.Unlock()
	return stGlobalJSONState.jsonText
}

func SetLastStGlobalFullJSON(jsonText string) {
	stGlobalJSONState.Lock()
	stGlobalJSONState.jsonText = jsonText
	stGlobalJSONState.Unlock()
}
