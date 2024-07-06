package lib

import "database/sql"

func defineCore(db *sql.DB) error {
	sql := ddlSchema("core")

	_, err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil
}
