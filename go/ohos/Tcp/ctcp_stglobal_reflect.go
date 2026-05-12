package tcp

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// valueToReflectJSON 将任意值转为可被 encoding/json 序列化的树（含未导出字段的数值/嵌套），
// 用于在鸿蒙侧输出完整 StGlobal 快照；不处理循环引用。
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

// FormatStGlobalFullJSON 输出 StGlobal 的完整 JSON（缩进），含所有嵌套字段。
func FormatStGlobalFullJSON(g StGlobal) (string, error) {
	tree := valueToReflectJSON(reflect.ValueOf(g))
	b, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
