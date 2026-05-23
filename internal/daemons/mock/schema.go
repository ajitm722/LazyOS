package mock

import "github.com/ajitm722/lazyos/internal/daemons"

// MockTables contains the catalog of tables used in mock tests.
var MockTables = []daemons.TableSchema{
	{Name: "processes", Description: "Mock processes.", Columns: "pid, name, path, cmdline, state, cwd, root, uid, gid, on_disk, resident_size, total_size"},
	{Name: "users", Description: "Mock users.", Columns: "uid, gid, uid_signed, gid_signed, username, description, directory, shell, uuid"},
	{Name: "empty", Description: "Table that always returns 0 rows.", Columns: "pid, name, state"},
}
