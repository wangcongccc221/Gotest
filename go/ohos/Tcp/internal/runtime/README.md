# Runtime 包说明

这个包保留当前强耦合运行链路，避免拆包时引入 import cycle。

## 服务入口

- `server.go`：HTTP/Gin 服务。
- `ctcp_server.go`：CTCP 接收服务，解析 FSM/WAM/IPM 回包。
- `ctcp_protocol.go`：CTCP 接收侧读包逻辑。
- `client_alias.go`：对 `Tcp/client` 的兼容别名。
- `protocol_alias.go`：对 `Tcp/protocol` 的兼容别名。

## WebSocket

- `websocket.go`：连接、hub、frame、ack。
- `websocket_handlers.go`：前端命令分发。
- `websocket_fsm_commands.go`：FSM 配置指令。
- `websocket_wam_commands.go`：WAM 指令。
- `websocket_ipm_commands.go`：IPM 相机、MAC、Wake-on-LAN。
- `websocket_grade_commands.go`：等级和出口映射。
- `websocket_exit.go`：出口信息、出口屏、附加文字。
- `websocket_project_config.go`：等级辅助配置、果型、项目方案。
- `websocket_database.go`：前端数据库查询/修改。
- `websocket_payload_encode.go`：设备 payload 编码。

## 统计和保存

- `ctcp_statistics.go`：FSM 统计快照和推送。
- `home_stats.go`：主页统计聚合。
- `fsm_stats_csv.go`：统计 CSV 诊断记录。
- `realtime_save*.go`：实时保存数据库数据。
- `end_process.go`：结束加工流程。

## 本地配置和出口

- `exit_*_store.go`：出口信息、出口屏、附加文字本地配置。
- `exit_display_broadcast.go`：出口屏 UDP 广播。
- `fruit_type_config_store.go`：果型配置。
- `level_aux_config_store.go`：等级辅助配置。
- `project_settings_scheme_store.go`：项目方案导入导出。

## 诊断和其它

- `ctcp_config_cache.go`：配置缓存和周期日志。
- `ctcp_stglobal_to_json.go`：StGlobal JSON 快照。
- `fixed_text_gbk_wire.go`：GBK 文本编码。
- `grade_info_log.go`：等级信息日志。
- `ipm_image_publish.go`：IPM 图片发布。
- `about_info_api.go`：关于信息 API。
- `project_auth.go`：项目密码。
- `time_utils.go`：本地时间工具。

