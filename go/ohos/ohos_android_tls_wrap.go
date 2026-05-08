//go:build android && cgo

package main

/*
#cgo LDFLAGS: -Wl,--wrap=dlsym
#include <dlfcn.h>
#include <string.h>

void *__real_dlsym(void *handle, const char *symbol);

void *__wrap_dlsym(void *handle, const char *symbol) {
	if (symbol != 0 && strcmp(symbol, "android_get_device_api_level") == 0) {
		return 0;
	}
	return __real_dlsym(handle, symbol);
}
*/
import "C"
