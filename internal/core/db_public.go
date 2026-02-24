package core

import "database/sql"

func OpenSQLiteDB(path string) (*sql.DB, error) {
	return openDB(path)
}
