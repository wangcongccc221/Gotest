#include "android/log.h"

#include <stdarg.h>
#include <stdio.h>

#include "hilog/log.h"

int __android_log_vprint(int prio, const char *tag, const char *fmt, va_list ap)
{
    (void)prio;

    char buffer[4096];
    vsnprintf(buffer, sizeof(buffer), fmt, ap);
    OH_LOG_DEBUG(LOG_APP, "[%{public}s] %{public}s", tag ? tag : "go", buffer);
    return 0;
}

int __android_log_print(int prio, const char *tag, const char *fmt, ...)
{
    va_list ap;
    va_start(ap, fmt);
    int result = __android_log_vprint(prio, tag, fmt, ap);
    va_end(ap);
    return result;
}
