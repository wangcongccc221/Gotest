package tcp

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

var (
	cTCPLastStGlobalFullJSONMu sync.Mutex
	cTCPLastStGlobalFullJSON   string
)

// LastCTCPStGlobalFullJSON 返回最近一次 save 写入的 StGlobal 全量 JSON；未调用过 save 或失败清空后为 ""。
func LastCTCPStGlobalFullJSON() string {
	cTCPLastStGlobalFullJSONMu.Lock()
	defer cTCPLastStGlobalFullJSONMu.Unlock()
	return cTCPLastStGlobalFullJSON
}

// saveCTCPStGlobalFullJSON 将 StGlobal 序列化后写入全局缓存（供 HTTP/NAPI 等读取）。返回 JSON 或 ""。
func saveCTCPStGlobalFullJSON(global StGlobal) string {
	full, err := FormatDataFullJSON(global)
	cTCPLastStGlobalFullJSONMu.Lock()
	if err != nil {
		cTCPLastStGlobalFullJSON = ""
	} else {
		cTCPLastStGlobalFullJSON = full
	}
	cTCPLastStGlobalFullJSONMu.Unlock()
	if err != nil {
		return ""
	}
	return full
}

// valueToReflectJSON 将任意值转为可被 encoding/json 序列化的树（含未导出字段的数值/嵌套）。
func valueToReflectJSON(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}

	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return nil
		}
		return valueToReflectJSON(v.Elem())
	case reflect.Struct:
		out := make(map[string]any)
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			sf := t.Field(i)
			out[sf.Name] = valueToReflectJSON(v.Field(i))
		}
		return out
	case reflect.Slice:
		if v.IsNil() {
			return nil
		}
		fallthrough
	case reflect.Array:
		n := v.Len()
		arr := make([]any, n)
		for i := 0; i < n; i++ {
			arr[i] = valueToReflectJSON(v.Index(i))
		}
		return arr
	case reflect.Map:
		if v.IsNil() {
			return nil
		}
		out := make(map[string]any)
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key()
			var keyStr string
			if k.Kind() == reflect.String {
				keyStr = k.String()
			} else {
				keyStr = fmt.Sprint(k)
			}
			out[keyStr] = valueToReflectJSON(iter.Value())
		}
		return out
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.Complex64, reflect.Complex128:
		return fmt.Sprint(v.Complex())
	default:
		return fmt.Sprintf("<unsupported kind %s>", v.Kind())
	}
}

// FormatDataFullJSON 反射全字段转缩进 JSON 字符串（与下位机布局对齐的 struct 可直接传入）。
func FormatDataFullJSON[T any](data T) (string, error) {
	tree := valueToReflectJSON(reflect.ValueOf(data))
	b, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
