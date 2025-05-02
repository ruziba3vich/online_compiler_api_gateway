// @title Online Compiler API
// @version 1.0
// @description API for managing programming languages and compiling code
// @host compile.prodonik.uz
// @BasePath /api/v1
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

// GetAllLanguages godoc
// @Summary      Retrieve base Hello World scripts for all supported languages
// @Description  Returns language-script pairs
// @Tags         languages
// @Produce      json
// @Success      200  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /languages [get]
func (h *LangHandler) GetAllLanguages(c *gin.Context) {
	c.JSON(http.StatusOK, baseScripts)
}

var baseScripts map[string]string = map[string]string{
	"java":       "public class Main {\n    public static void main(String[] args) {\n        System.out.println(\"Hello, world!\");\n    }\n}",
	"cpp":        "#include <iostream>\nint main() {\n    std::cout << \"Hello, world!\" << std::endl;\n    return 0;\n}",
	"go":         "package main\nimport \"fmt\"\nfunc main() {\n    fmt.Println(\"Hello, world!\")\n}",
	"javascript": "console.log(\"Hello, world!\");",
	"csharp":     "using System;\nclass Program {\n    static void Main() {\n        Console.WriteLine(\"Hello, world!\");\n    }\n}",
	"php":        "<?php\necho \"Hello, world!\";",
	"python":     "print(\"Hello, world!\")",
}
