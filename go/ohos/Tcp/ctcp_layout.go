package tcp

import (
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"strings"
)

// StGlobalLayoutReport 输出 StGlobal 及其可达的嵌套结构体
func StGlobalLayoutReport() string {
	root := reflect.TypeOf(StGlobal{})
	all := make(map[reflect.Type]struct{})
	collectStructTypes(root, all)

	types := make([]reflect.Type, 0, len(all))
	for t := range all {
		types = append(types, t)
	}
	sort.Slice(types, func(i, j int) bool {
		if types[i].Name() == "StGlobal" {
			return true
		}
		if types[j].Name() == "StGlobal" {
			return false
		}
		return types[i].Name() < types[j].Name()
	})

	var b strings.Builder
	fmt.Fprintf(&b, "runtime: GOOS=%s GOARCH=%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&b, "结构体大小")

	for _, t := range types {
		formatOneStruct(&b, t)
	}
	return b.String()
}

func collectStructTypes(typ reflect.Type, out map[reflect.Type]struct{}) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return
	}
	if _, ok := out[typ]; ok {
		return
	}
	out[typ] = struct{}{}
	for i := 0; i < typ.NumField(); i++ {
		ft := typ.Field(i).Type
		switch ft.Kind() {
		case reflect.Struct:
			collectStructTypes(ft, out)
		case reflect.Array:
			if ft.Elem().Kind() == reflect.Struct {
				collectStructTypes(ft.Elem(), out)
			}
		}
	}
}

func formatOneStruct(b *strings.Builder, typ reflect.Type) {
	fmt.Fprintf(b, "=== %s sizeof=%d align=%d ===\n", typ.Name(), typ.Size(), typ.Align())
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		fmt.Fprintf(b, "  %-32s offset=%-5d size=%-6d type=%s\n",
			f.Name, f.Offset, f.Type.Size(), f.Type.String())
	}
	b.WriteByte('\n')
}
