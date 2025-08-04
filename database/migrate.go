package database

import (
	"go-clean-arch/domain"

	"gorm.io/gorm"
)

func MigrateDB(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.User{},
		&domain.UserSession{},
		&domain.File{},
		&domain.FileLink{},
		&domain.EmailLog{},
		&domain.EmailTemplate{},
	)
}
