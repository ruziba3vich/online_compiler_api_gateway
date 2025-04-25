package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ruziba3vich/online_compiler_api_gateway/internal/service"
	logger "github.com/ruziba3vich/prodonik_lgger"
)

type LangHandler struct {
	langService *service.LangService
	logger      *logger.Logger
}

func NewLangHandler(langService *service.LangService, logger *logger.Logger) *LangHandler {
	return &LangHandler{
		langService: langService,
		logger:      logger,
	}
}

func (h *LangHandler) CreateLanguage(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to decode request body", map[string]any{"error": err.Error()})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.langService.CreateLanguage(req.Name); err != nil {
		h.logger.Error("Failed to add language", map[string]any{"error": err.Error(), "language": req.Name})
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Added new language", map[string]any{"language": req.Name})
	c.JSON(http.StatusCreated, gin.H{"name": req.Name})
}

func (h *LangHandler) GetAllLanguages(c *gin.Context) {
	languages, err := h.langService.GetAllLanguages()
	if err != nil {
		h.logger.Error("Failed to get languages", map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Retrieved all languages", map[string]any{"count": len(languages)})
	c.JSON(http.StatusOK, languages)
}
