#include "libohos.h"
#include "napi/native_api.h"

#include <dlfcn.h>
#include <mutex>
#include <string>
#include <vector>

using GoAddFunc = double (*)(double, double);
using GoCStringFunc = char *(*)();
using GoFreeCStringFunc = void (*)(char *);
using GoIntFunc = int (*)();
using GoInitORMWithPathFunc = int (*)(char *);
using GoStartTCPClientWithAddressFunc = int (*)(char *, int, char *);
using GoStartCTCPClientFunc = int (*)(char *, int, int, int);
using GoTCPClientSendFunc = char *(*)(char *);

struct GoApi {
    void *handle = nullptr;
    bool ready = false;
    std::string lastError;

    GoAddFunc add = nullptr;
    GoCStringFunc hello = nullptr;
    GoFreeCStringFunc freeCString = nullptr;
    GoIntFunc startServer = nullptr;
    GoIntFunc stopServer = nullptr;
    GoIntFunc startOPCUAServer = nullptr;
    GoIntFunc stopOPCUAServer = nullptr;
    GoIntFunc startModbusServer = nullptr;
    GoIntFunc stopModbusServer = nullptr;
    GoIntFunc initORM = nullptr;
    GoInitORMWithPathFunc initORMWithPath = nullptr;
    GoIntFunc startTCPServer = nullptr;
    GoIntFunc stopTCPServer = nullptr;
    GoIntFunc startTCPClient = nullptr;
    GoStartTCPClientWithAddressFunc startTCPClientWithAddress = nullptr;
    GoStartCTCPClientFunc startCTCPClient = nullptr;
    GoIntFunc stopTCPClient = nullptr;
    GoTCPClientSendFunc tcpClientSend = nullptr;
    GoCStringFunc lastTCPServerMessage = nullptr;
    GoCStringFunc stGlobalLayoutReport = nullptr;
};

static std::mutex g_goApiMutex;
static GoApi g_goApi;

static std::string ReadDlError()
{
    const char *error = dlerror();
    return error == nullptr ? "unknown dlerror" : error;
}

template <typename Symbol>
static bool LoadGoSymbol(void *handle, const char *name, Symbol *out)
{
    dlerror();
    void *symbol = dlsym(handle, name);
    const char *error = dlerror();
    if (error != nullptr || symbol == nullptr) {
        g_goApi.lastError = "dlsym ";
        g_goApi.lastError += name;
        g_goApi.lastError += " failed: ";
        g_goApi.lastError += error == nullptr ? "symbol is null" : error;
        return false;
    }
    *out = reinterpret_cast<Symbol>(symbol);
    return true;
}

static bool LoadGoApi()
{
    std::lock_guard<std::mutex> lock(g_goApiMutex);
    if (g_goApi.ready) {
        return true;
    }

    g_goApi.lastError.clear();
    if (g_goApi.handle == nullptr) {
        g_goApi.handle = dlopen("libohos.so", RTLD_NOW | RTLD_GLOBAL);
    }
    if (g_goApi.handle == nullptr) {
        g_goApi.handle = dlopen("/data/storage/el1/bundle/libs/arm64/libohos.so", RTLD_NOW | RTLD_GLOBAL);
    }
    if (g_goApi.handle == nullptr) {
        g_goApi.lastError = "dlopen libohos.so failed: " + ReadDlError();
        return false;
    }

    if (!LoadGoSymbol(g_goApi.handle, "GoAdd", &g_goApi.add) ||
        !LoadGoSymbol(g_goApi.handle, "GoHello", &g_goApi.hello) ||
        !LoadGoSymbol(g_goApi.handle, "GoFreeCString", &g_goApi.freeCString) ||
        !LoadGoSymbol(g_goApi.handle, "GoStartServer", &g_goApi.startServer) ||
        !LoadGoSymbol(g_goApi.handle, "GoStopServer", &g_goApi.stopServer) ||
        !LoadGoSymbol(g_goApi.handle, "GoStartOPCUAServer", &g_goApi.startOPCUAServer) ||
        !LoadGoSymbol(g_goApi.handle, "GoStopOPCUAServer", &g_goApi.stopOPCUAServer) ||
        !LoadGoSymbol(g_goApi.handle, "GoStartModbusServer", &g_goApi.startModbusServer) ||
        !LoadGoSymbol(g_goApi.handle, "GoStopModbusServer", &g_goApi.stopModbusServer) ||
        !LoadGoSymbol(g_goApi.handle, "GoInitORM", &g_goApi.initORM) ||
        !LoadGoSymbol(g_goApi.handle, "GoInitORMWithPath", &g_goApi.initORMWithPath) ||
        !LoadGoSymbol(g_goApi.handle, "GoStartTCPServer", &g_goApi.startTCPServer) ||
        !LoadGoSymbol(g_goApi.handle, "GoStopTCPServer", &g_goApi.stopTCPServer) ||
        !LoadGoSymbol(g_goApi.handle, "GoStartTCPClient", &g_goApi.startTCPClient) ||
        !LoadGoSymbol(g_goApi.handle, "GoStartTCPClientWithAddress", &g_goApi.startTCPClientWithAddress) ||
        !LoadGoSymbol(g_goApi.handle, "GoStartCTCPClient", &g_goApi.startCTCPClient) ||
        !LoadGoSymbol(g_goApi.handle, "GoStopTCPClient", &g_goApi.stopTCPClient) ||
        !LoadGoSymbol(g_goApi.handle, "GoTCPClientSend", &g_goApi.tcpClientSend) ||
        !LoadGoSymbol(g_goApi.handle, "GoLastTCPServerMessage", &g_goApi.lastTCPServerMessage)) {
        return false;
    }

    dlerror();
    void *layoutSym = dlsym(g_goApi.handle, "GoStGlobalLayoutReport");
    if (layoutSym != nullptr) {
        g_goApi.stGlobalLayoutReport = reinterpret_cast<GoCStringFunc>(layoutSym);
    } else {
        g_goApi.stGlobalLayoutReport = nullptr;
    }

    g_goApi.ready = true;
    return true;
}

static std::string GetGoApiLastError()
{
    std::lock_guard<std::mutex> lock(g_goApiMutex);
    return g_goApi.lastError;
}

static bool ReadNumber(napi_env env, napi_value value, double *out)
{
    napi_valuetype type;
    napi_typeof(env, value, &type);
    if (type != napi_number) {
        napi_throw_type_error(env, nullptr, "Expected a number");
        return false;
    }
    napi_get_value_double(env, value, out);
    return true;
}

static bool ReadString(napi_env env, napi_value value, std::string *out)
{
    napi_valuetype type;
    napi_typeof(env, value, &type);
    if (type != napi_string) {
        napi_throw_type_error(env, nullptr, "Expected a string");
        return false;
    }

    size_t length = 0;
    napi_get_value_string_utf8(env, value, nullptr, 0, &length);
    std::vector<char> buffer(length + 1);
    napi_get_value_string_utf8(env, value, buffer.data(), buffer.size(), &length);
    out->assign(buffer.data(), length);
    return true;
}

static napi_value NapiAdd(napi_env env, napi_callback_info info)
{
    size_t argc = 2;
    napi_value args[2] = {nullptr};
    napi_get_cb_info(env, info, &argc, args, nullptr, nullptr);

    if (argc < 2) {
        napi_throw_type_error(env, nullptr, "add expects two numbers");
        return nullptr;
    }

    double value0;
    double value1;
    if (!ReadNumber(env, args[0], &value0) || !ReadNumber(env, args[1], &value1)) {
        return nullptr;
    }

    double goResult = 0;
    if (LoadGoApi()) {
        goResult = g_goApi.add(value0, value1);
    }

    napi_value sum;
    napi_create_double(env, goResult, &sum);
    return sum;
}

static napi_value NapiHello(napi_env env, napi_callback_info info)
{
    char *message = nullptr;
    if (LoadGoApi()) {
        message = g_goApi.hello();
    }
    if (message == nullptr) {
        napi_value empty;
        napi_create_string_utf8(env, "", NAPI_AUTO_LENGTH, &empty);
        return empty;
    }

    napi_value result;
    napi_create_string_utf8(env, message, NAPI_AUTO_LENGTH, &result);
    g_goApi.freeCString(message);
    return result;
}

static napi_value NapiStartServer(napi_env env, napi_callback_info info)
{
    int goResult = -1;
    if (LoadGoApi()) {
        goResult = g_goApi.startServer();
    }

    napi_value result;
    napi_create_int32(env, goResult, &result);
    return result;
}

static napi_value NapiStopServer(napi_env env, napi_callback_info info)
{
    int goResult = -1;
    if (LoadGoApi()) {
        goResult = g_goApi.stopServer();
    }

    napi_value result;
    napi_create_int32(env, goResult, &result);
    return result;
}

static napi_value NapiStartOPCUAServer(napi_env env, napi_callback_info info)
{
    int goResult = -1;
    if (LoadGoApi()) {
        goResult = g_goApi.startOPCUAServer();
    }

    napi_value result;
    napi_create_int32(env, goResult, &result);
    return result;
}

static napi_value NapiStopOPCUAServer(napi_env env, napi_callback_info info)
{
    int goResult = -1;
    if (LoadGoApi()) {
        goResult = g_goApi.stopOPCUAServer();
    }

    napi_value result;
    napi_create_int32(env, goResult, &result);
    return result;
}

static napi_value NapiStartModbusServer(napi_env env, napi_callback_info info)
{
    int goResult = -1;
    if (LoadGoApi()) {
        goResult = g_goApi.startModbusServer();
    }

    napi_value result;
    napi_create_int32(env, goResult, &result);
    return result;
}

static napi_value NapiStopModbusServer(napi_env env, napi_callback_info info)
{
    int goResult = -1;
    if (LoadGoApi()) {
        goResult = g_goApi.stopModbusServer();
    }

    napi_value result;
    napi_create_int32(env, goResult, &result);
    return result;
}

static napi_value NapiInitORM(napi_env env, napi_callback_info info)
{
    size_t argc = 1;
    napi_value args[1] = {nullptr};
    napi_get_cb_info(env, info, &argc, args, nullptr, nullptr);

    int initResult = -1;
    if (argc > 0) {
        napi_valuetype type;
        napi_typeof(env, args[0], &type);
        if (type != napi_undefined && type != napi_null) {
            std::string databasePath;
            if (!ReadString(env, args[0], &databasePath)) {
                return nullptr;
            }
            if (LoadGoApi()) {
                initResult = g_goApi.initORMWithPath(const_cast<char *>(databasePath.c_str()));
            }
        } else {
            if (LoadGoApi()) {
                initResult = g_goApi.initORM();
            }
        }
    } else {
        if (LoadGoApi()) {
            initResult = g_goApi.initORM();
        }
    }

    napi_value result;
    napi_create_int32(env, initResult, &result);
    return result;
}

static napi_value NapiStartTCPServer(napi_env env, napi_callback_info info)
{
    int goResult = -1;
    if (LoadGoApi()) {
        goResult = g_goApi.startTCPServer();
    }

    napi_value result;
    napi_create_int32(env, goResult, &result);
    return result;
}

static napi_value NapiStopTCPServer(napi_env env, napi_callback_info info)
{
    int goResult = -1;
    if (LoadGoApi()) {
        goResult = g_goApi.stopTCPServer();
    }

    napi_value result;
    napi_create_int32(env, goResult, &result);
    return result;
}

static napi_value NapiStartTCPClient(napi_env env, napi_callback_info info) //StartTCPClient
{
    size_t argc = 4;
    napi_value args[4] = {nullptr};
    napi_get_cb_info(env, info, &argc, args, nullptr, nullptr);

    int goResult = -1;
    if (LoadGoApi()) {
        if (argc < 4) {
            napi_throw_type_error(env, nullptr, "startTcpClient expects remote IP, remote port, destination ID and command");
            return nullptr;
        }

        std::string remoteIp;
        if (!ReadString(env, args[0], &remoteIp)) {
            return nullptr;
        }

        double remotePortValue;
        if (!ReadNumber(env, args[1], &remotePortValue)) {
            return nullptr;
        }

        double destIdValue;
        if (!ReadNumber(env, args[2], &destIdValue)) {
            return nullptr;
        }

        double cmdValue;
        if (!ReadNumber(env, args[3], &cmdValue)) {
            return nullptr;
        }

        goResult = g_goApi.startCTCPClient(
            const_cast<char *>(remoteIp.c_str()),
            static_cast<int>(remotePortValue),
            static_cast<int>(destIdValue),
            static_cast<int>(cmdValue));
    }

    napi_value result;
    napi_create_int32(env, goResult, &result);
    return result;
}

static napi_value NapiStopTCPClient(napi_env env, napi_callback_info info)
{
    int goResult = -1;
    if (LoadGoApi()) {
        goResult = g_goApi.stopTCPClient();
    }

    napi_value result;
    napi_create_int32(env, goResult, &result);
    return result;
}

static napi_value NapiTCPSend(napi_env env, napi_callback_info info)
{
    size_t argc = 1;
    napi_value args[1] = {nullptr};
    napi_get_cb_info(env, info, &argc, args, nullptr, nullptr);

    if (argc < 1) {
        napi_throw_type_error(env, nullptr, "tcpSend expects one string");
        return nullptr;
    }

    std::string message;
    if (!ReadString(env, args[0], &message)) {
        return nullptr;
    }

    char *response = nullptr;
    if (LoadGoApi()) {
        response = g_goApi.tcpClientSend(const_cast<char *>(message.c_str()));
    }
    if (response == nullptr) {
        napi_value empty;
        napi_create_string_utf8(env, "", NAPI_AUTO_LENGTH, &empty);
        return empty;
    }

    napi_value result;
    napi_create_string_utf8(env, response, NAPI_AUTO_LENGTH, &result);
    g_goApi.freeCString(response);
    return result;
}

static napi_value NapiNativeLastError(napi_env env, napi_callback_info info)
{
    std::string error = GetGoApiLastError();
    napi_value result;
    napi_create_string_utf8(env, error.c_str(), NAPI_AUTO_LENGTH, &result);
    return result;
}

static napi_value NapiTCPServerLastMessage(napi_env env, napi_callback_info info)
{
    char *message = nullptr;
    if (LoadGoApi()) {
        message = g_goApi.lastTCPServerMessage();
    }
    if (message == nullptr) {
        napi_value empty;
        napi_create_string_utf8(env, "", NAPI_AUTO_LENGTH, &empty);
        return empty;
    }

    napi_value result;
    napi_create_string_utf8(env, message, NAPI_AUTO_LENGTH, &result);
    g_goApi.freeCString(message);
    return result;
}

static napi_value NapiStGlobalLayoutReport(napi_env env, napi_callback_info info)
{
    if (!LoadGoApi()) {
        napi_value empty;
        napi_create_string_utf8(env, "", NAPI_AUTO_LENGTH, &empty);
        return empty;
    }
    if (g_goApi.stGlobalLayoutReport == nullptr) {
        napi_value msg;
        napi_create_string_utf8(
            env,
            "GoStGlobalLayoutReport 未导出：请运行 go/ohos/build_ohos.ps1 重新编译 libohos.so",
            NAPI_AUTO_LENGTH,
            &msg);
        return msg;
    }

    char *report = g_goApi.stGlobalLayoutReport();
    if (report == nullptr) {
        napi_value empty;
        napi_create_string_utf8(env, "", NAPI_AUTO_LENGTH, &empty);
        return empty;
    }

    napi_value result;
    napi_create_string_utf8(env, report, NAPI_AUTO_LENGTH, &result);
    g_goApi.freeCString(report);
    return result;
}

EXTERN_C_START
static napi_value Init(napi_env env, napi_value exports)
{
    napi_property_descriptor desc[] = {
        { "add", nullptr, NapiAdd, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "hello", nullptr, NapiHello, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "startServer", nullptr, NapiStartServer, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "stopServer", nullptr, NapiStopServer, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "startOpcuaServer", nullptr, NapiStartOPCUAServer, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "stopOpcuaServer", nullptr, NapiStopOPCUAServer, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "startModbusServer", nullptr, NapiStartModbusServer, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "stopModbusServer", nullptr, NapiStopModbusServer, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "initOrm", nullptr, NapiInitORM, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "startTcpServer", nullptr, NapiStartTCPServer, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "stopTcpServer", nullptr, NapiStopTCPServer, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "startTcpClient", nullptr, NapiStartTCPClient, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "stopTcpClient", nullptr, NapiStopTCPClient, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "tcpSend", nullptr, NapiTCPSend, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "nativeLastError", nullptr, NapiNativeLastError, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "tcpServerLastMessage", nullptr, NapiTCPServerLastMessage, nullptr, nullptr, nullptr, napi_default, nullptr },
        { "stGlobalLayoutReport", nullptr, NapiStGlobalLayoutReport, nullptr, nullptr, nullptr, napi_default, nullptr }
    };
    napi_define_properties(env, exports, sizeof(desc) / sizeof(desc[0]), desc);
    return exports;
}
EXTERN_C_END

static napi_module demoModule = {
    .nm_version = 1,
    .nm_flags = 0,
    .nm_filename = nullptr,
    .nm_register_func = Init,
    .nm_modname = "entry",
    .nm_priv = ((void*)0),
    .reserved = { 0 },
};

extern "C" __attribute__((constructor)) void RegisterEntryModule(void)
{
    napi_module_register(&demoModule);
}
