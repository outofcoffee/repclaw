# Connections

A connection is a saved gateway target (name + URL). The list is persisted to `~/.lucinate/connections.json` and managed through the connections picker (`viewConnections`) — accessible as the entry view on first run, or via `/connections` mid-session.

Today every connection is an OpenClaw gateway. The `ConnectionType` enum (`internal/config/connections.go`) exists so the persisted shape and picker UI don't churn when additional backend types land.

## Startup decision tree

`config.ResolveEntryConnection()` (`internal/config/startup.go`) decides what the TUI's entry view is:

1. If `OPENCLAW_GATEWAY_URL` is set, find or auto-add a matching connection.
2. Else if a saved `defaultId` resolves to a known entry → use it (last-used = default).
3. Else if exactly one connection is stored → auto-pick it.
4. Else → open the connections picker.

Auto-add (step 1) mutates the in-memory store but does **not** persist it until a successful connect, so a typo in the env URL doesn't accumulate ghost entries.

## Connection lifecycle

The TUI owns the lifecycle in managed mode (`RunOptions.Store != nil`, see `app/app.go`):

- A successful connect calls `OnClientChanged(client)` so the app driver in `app/app.go` rewires the events pump and supervisor onto the new client.
- Picking a different connection mid-session (`/connections` or any other `showConnectionsMsg`) publishes `nil` to the driver, which closes the active client before binding to whatever the next pick produces.
- The driver owns Close — TUI code never closes the client itself once it's published.

Legacy mode (`RunOptions.Client != nil`, no `Store`) is for native-platform embedders that manage the connection elsewhere; the connections picker and `/connections` are unavailable, and Close is the caller's responsibility.

## Connections form

`internal/tui/connections.go` implements the picker. The list view supports add / edit / delete / set-default actions, and the form has Name and Gateway URL fields.

Delete confirmation is exposed as a sub-state with confirm/cancel pairs in `Actions()`, so native-platform embedders render them as buttons rather than relying on inline `y/n` keys.
