package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const DefaultSQLiteDSN = "db/edge-gateway.db"

var (
	db      *gorm.DB
	dbOnce  sync.Once
	initErr error
)

func DB() (*gorm.DB, error) {
	return InitSQLite(DefaultSQLiteDSN)
}

func InitSQLite(dsn string) (*gorm.DB, error) {
	dbOnce.Do(func() {
		db, initErr = OpenSQLiteWithDSN(dsn)
	})
	return db, initErr
}

func OpenSQLite() (*gorm.DB, error) {
	return OpenSQLiteWithDSN(DefaultSQLiteDSN)
}

func OpenSQLiteWithDSN(dsn string) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dsn), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite dir: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	return db, nil
}
