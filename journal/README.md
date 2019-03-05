# Action Journal

The `journal` package maintains a collection of logs organized by action ID. Each log is stored in a separate file in the user's cache directory under `wash/journal/<id>.log`.

The action ID should correspond to a universal unique identifier associated with whatever triggered the action. This is usually a process ID.

Logs are kept open for several seconds after use then closed; they can be re-opened as necessary.
