# Tcp events

`events` owns cross-module runtime events:

- buffered CTCP log messages consumed by Harmony UI;
- the WebSocket publishing interface used by receiver, stats, store, and realtime code.

Keep this package independent from `internal/runtime` so future modules can import it without creating cycles.
