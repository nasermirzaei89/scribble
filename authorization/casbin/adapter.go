package casbin

import (
	"database/sql"
	"fmt"

	sqladapter "github.com/Blank-Xu/sql-adapter"
)

func NewSQLAdapter(sqlDB *sql.DB, dbType, tableName string) (*sqladapter.Adapter, error) {
	adapter, err := sqladapter.NewAdapter(sqlDB, dbType, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to casbin database: %w", err)
	}

	return adapter, nil
}
