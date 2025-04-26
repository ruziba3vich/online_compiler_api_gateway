package storage

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/ruziba3vich/online_compiler_api_gateway/pkg/config"
)

type LangStorage struct {
	filePath string
}

func NewLanguageStorage(cfg *config.Config) *LangStorage {
	return &LangStorage{
		filePath: cfg.LangStorageFilePath,
	}
}

func (s *LangStorage) GetLanguages() ([]string, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return nil, err
	}

	var languages []string
	if err := json.Unmarshal(data, &languages); err != nil {
		return nil, err
	}

	return languages, nil
}

func (s *LangStorage) AddLanguage(language string) error {
	languages, err := s.GetLanguages()
	if err != nil {
		return err
	}

	languages = append(languages, language)

	data, err := json.MarshalIndent(languages, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

func (s *LangStorage) EnsureStorageExists() error {
	if _, err := os.Stat(s.filePath); errors.Is(err, os.ErrNotExist) {
		emptyArray := []string{}
		data, err := json.Marshal(emptyArray)
		if err != nil {
			return err
		}
		return os.WriteFile(s.filePath, data, 0644)
	}
	return nil
}
