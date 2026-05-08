package gormx

import (
	"log/slog"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func Open(path string) (db *gorm.DB, err error) {
	db, err = gorm.Open(gormlite.Open(path), &gorm.Config{
		SkipDefaultTransaction: true,
		NamingStrategy:         schema.NamingStrategy{SingularTable: true},
		Logger:                 logger.NewSlogLogger(slog.Default(), logger.Config{LogLevel: logger.Info}),
	})
	if err != nil {
		return
	}

	// 开启 WAL 模式, 增加最大连接数为 100
	if err = db.Exec("PRAGMA journal_mode = wal; PRAGMA busy_timeout = 100;").Error; err != nil {
		return
	}

	return
}
