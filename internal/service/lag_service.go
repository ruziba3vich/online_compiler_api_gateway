package service

import (
	"errors"
	"fmt"
	"slices"

	"github.com/ruziba3vich/online_compiler_api_gateway/internal/dto"
	"github.com/ruziba3vich/online_compiler_api_gateway/internal/storage"
	logger "github.com/ruziba3vich/prodonik_lgger"
	"golang.org/x/crypto/bcrypt"
)

var hashed string = "$2a$10$/zXjJLDbjJZcsdl0zPYIe.lcBf9rrFbjomA86CE72SX.akadbPWfi"

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

func (s *LangService) CreateLanguage(req *dto.Language) error {
	ok, err := compareHashedString(req.Password)
	if err != nil {
		s.logger.Error("failed to compare password", map[string]any{"error": err.Error()})
		return fmt.Errorf("failed to compare password")
	}
	if !ok {
		return errors.New("invalid password")
	}
	if len(req.Name) == 0 {
		s.logger.Error("error while checking storage check", map[string]any{"length": len(req.Name)})
		return errors.New("language name cannot be empty")
	}
	languages, err := s.GetAllLanguages()
	if err != nil {
		return err
	}
	if slices.Contains(languages, req.Name) {
		return fmt.Errorf("%s already exists", req.Name)
	}
	return s.langStorage.AddLanguage(req.Name)
}

func (s *LangService) GetAllLanguages() ([]string, error) {
	languages, err := s.langStorage.GetLanguages()

	if err != nil {
		s.logger.Error("failed to fetch all languages", map[string]any{"error": err.Error()})
		return languages, err
	}
	return languages, nil
}

func compareHashedString(plaintext string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plaintext))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
