package db

import (
	"github.com/ruziba3vich/online_compiler_api_gateway/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewDB(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&models.Language{})
	if err != nil {
		return nil, err
	}

	return db, nil
}
