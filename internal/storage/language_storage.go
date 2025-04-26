package storage

import (
	"encoding/json"
	"errors"
	"os"
	"slices"

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
		return []string{}, errors.New("unable to read file")
	}

	if len(data) == 0 {
		if err := os.WriteFile(s.filePath, []byte("[]"), 0644); err != nil {
			return []string{}, errors.New("unable to initialize empty file")
		}
		return []string{}, nil
	}

	var languages []string
	if err := json.Unmarshal(data, &languages); err != nil {
		return []string{}, errors.New("invalid JSON format")
	}

	return languages, nil
}

func (s *LangStorage) AddLanguage(language string) error {
	languages, err := s.GetLanguages()
	if err != nil {
		return err
	}

	if slices.Contains(languages, language) {
		return errors.New("this language already exists")
	}

	languages = append(languages, language)

	newData, err := json.MarshalIndent(languages, "", "  ")
	if err != nil {
		return errors.New("failed to encode JSON")
	}

	if err := os.WriteFile(s.filePath, newData, 0644); err != nil {
		return errors.New("unable to write to file")
	}

	return nil
}

func (s *LangStorage) EnsureStorageExists() error {
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		err := os.WriteFile(s.filePath, []byte("[]"), 0644)
		if err != nil {
			return errors.New("unable to create file")
		}
	}
	return nil
}
