# Tcp state

`state` owns in-memory snapshots shared between receiver, WebSocket handlers, stats, and realtime save code.

The first extracted snapshots are grade, exit info, motor info, and the latest StGlobal JSON text. More runtime caches can move here as the remaining modules are split.
