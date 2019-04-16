# Activity Journals

The `activity` package maintains a collection of journals organized by journal ID. Each journal is stored in a separate file in the user's cache directory under `wash/activity/<id>.log`.

The journal ID should correspond to a universal unique identifier associated with whatever triggered any activity. This is usually a process ID and start time for that process.

Journals are kept open for several seconds after use then closed; they can be re-opened as necessary.
