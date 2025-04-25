package service

import (
	"errors"

	"github.com/ruziba3vich/online_compiler_api_gateway/internal/storage"
	logger "github.com/ruziba3vich/prodonik_lgger"
)

type LangService struct {
	langStorage *storage.LangStorage
	logger      *logger.Logger
}

func NewLangService(langStorage *storage.LangStorage, logger *logger.Logger) *LangService {
	return &LangService{
		langStorage: langStorage,
		logger:      logger,
	}
}

func (s *LangService) CreateLanguage(language string) error {
	if len(language) == 0 {
		s.logger.Error("error while checking storage check", map[string]any{"length": len(language)})
		return errors.New("language name cannot be empty")
	}
	if err := s.langStorage.EnsureStorageExists(); err != nil {
		s.logger.Error("error while checking storage check", map[string]any{"error": err.Error()})
		return err
	}
	return s.langStorage.AddLanguage(language)
}

func (s *LangService) GetAllLanguages() ([]string, error) {
	if err := s.langStorage.EnsureStorageExists(); err != nil {
		s.logger.Error("error while checking storage check", map[string]any{"error": err.Error()})
		return []string{}, err
	}
	languages, err := s.langStorage.GetLanguages()

	if err != nil {
		s.logger.Error("failed to fetch all languages", map[string]any{"error": err.Error()})
		return languages, err
	}
	return languages, nil
}
