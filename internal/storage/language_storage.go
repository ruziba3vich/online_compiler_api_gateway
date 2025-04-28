package storage

import (
	"github.com/ruziba3vich/online_compiler_api_gateway/internal/models"
	"gorm.io/gorm"
)

type LangStorage struct {
	db *gorm.DB
}

func NewLangStorage(db *gorm.DB) *LangStorage {
	return &LangStorage{
		db: db,
	}
}

func (s *LangStorage) AddLanguage(language string) error {
	return s.db.Create(&models.Language{Name: language}).Error
}

func (s *LangStorage) GetLanguages() ([]string, error) {
	var languages []models.Language
	if err := s.db.Find(&languages).Error; err != nil {
		return nil, err
	}

	names := make([]string, len(languages))
	for i, lang := range languages {
		names[i] = lang.Name
	}
	return names, nil
}
