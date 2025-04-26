package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/ruziba3vich/online_compiler_api_gateway/pkg/config"
)

type LangStorage struct {
	filePath string
	mu       sync.Mutex
}

func NewLanguageStorage(cfg *config.Config) *LangStorage {
	return &LangStorage{
		filePath: cfg.LangStorageFilePath,
	}
}

func (s *LangStorage) EnsureStorageExists() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := os.Stat(s.filePath)
	if err == nil {
		return nil
	}

	if errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Storage file '%s' not found, creating...\n", s.filePath)
		emptyJSONArray := []byte("[]")
		writeErr := os.WriteFile(s.filePath, emptyJSONArray, 0644)
		if writeErr != nil {
			return fmt.Errorf("failed to create storage file '%s': %w", s.filePath, writeErr)
		}
		return nil
	}

	return fmt.Errorf("failed to check storage file status for '%s': %w", s.filePath, err)
}

func (s *LangStorage) GetLanguages() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("storage file '%s' not found, run EnsureStorageExists or check path: %w", s.filePath, err)
		}
		return nil, fmt.Errorf("failed to read storage file '%s': %w", s.filePath, err)
	}

	var languages []string
	if len(data) == 0 {
		return []string{}, nil
	}

	err = json.Unmarshal(data, &languages)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON from storage file '%s': %w", s.filePath, err)
	}

	return languages, nil
}

func (s *LangStorage) AddLanguage(language string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read storage file '%s' before adding language: %w", s.filePath, err)
	}

	var languages []string
	if len(data) > 0 {
		err = json.Unmarshal(data, &languages)
		if err != nil {
			return fmt.Errorf("failed to parse JSON from storage file '%s' before adding language: %w", s.filePath, err)
		}
	}
	languages = append(languages, language)

	updatedData, err := json.MarshalIndent(languages, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated language list to JSON: %w", err)
	}

	err = os.WriteFile(s.filePath, updatedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write updated language list to storage file '%s': %w", s.filePath, err)
	}

	return nil
}
