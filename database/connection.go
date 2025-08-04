package database

import (
	"fmt"
	"go-clean-arch/pkg/log"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type Config interface {
	Host() string
	Port() string
	User() string
	Password() string
	Name() string
	SSLMode() string
	MaxOpenConns() int
	MaxIdleConns() int
	ConnMaxLifetime() time.Duration
	EnableLog() bool
	LogLevel() string
}

func getDSN(cfg Config) string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Host(),
		cfg.User(),
		cfg.Password(),
		cfg.Name(),
		cfg.Port(),
		cfg.SSLMode())
}

func getNamingStrategy() schema.NamingStrategy {
	return schema.NamingStrategy{
		SingularTable: false,                             // Use singular table name, table for `User` would be `user` with this option enabled
		NoLowerCase:   false,                             // Skip the snake_casing of names
		NameReplacer:  strings.NewReplacer("CID", "Cid"), // use name replacer to change struct/field name before convert it to db name
	}
}

func newLogger(l log.Logger, cfg Config) logger.Interface {
	var logLevel logger.LogLevel
	if cfg.EnableLog() {
		switch cfg.LogLevel() {
		case "info":
			logLevel = logger.Info
		case "warn":
			logLevel = logger.Warn
		case "error":
			logLevel = logger.Error
		case "silent":
			logLevel = logger.Silent
		default:
			logLevel = logger.Warn
		}
	} else {
		logLevel = logger.Silent
	}

	loggerConfig := logger.Config{
		SlowThreshold:             time.Second, // Slow SQL threshold
		LogLevel:                  logLevel,    // Log level from config
		IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
		ParameterizedQueries:      false,       // Don't include params in the SQL log
		Colorful:                  true,        // Enable color
	}
	return logger.New(l, loggerConfig)
}

func Connect(cfg Config, l log.Logger) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  getDSN(cfg), // Data source name
		PreferSimpleProtocol: true,        // Disables implicit prepared statement usage
	}), &gorm.Config{
		NamingStrategy: getNamingStrategy(),
		Logger:         newLogger(l, cfg),
	})
	if err != nil {
		return nil, err
	}

	// Set connection pool settings
	sDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sDB.SetMaxIdleConns(cfg.MaxIdleConns())
	sDB.SetMaxOpenConns(cfg.MaxOpenConns())
	sDB.SetConnMaxLifetime(cfg.ConnMaxLifetime())

	return db.Debug(), nil
}
