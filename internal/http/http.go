package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ruziba3vich/online_compiler_api_gateway/internal/service"
	"github.com/ruziba3vich/online_compiler_api_gateway/pkg/lgg"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// origin := r.Header.Get("Origin")
		// return origin == "http://compile.prodonik.uz" || origin == "https://compile.prodonik.uz"
		return true
	},
}

type (
	Handler struct {
		srv    *service.Service
		logger *lgg.Logger
	}
)

func NewHandler(srv *service.Service, logger *lgg.Logger) *Handler {
	return &Handler{
		srv:    srv,
		logger: logger,
	}
}

func (h *Handler) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("WebSocket upgrade error", map[string]any{"error": err})
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	sessionID := newID()
	h.logger.Info("WebSocket client connected", map[string]any{"session_id": sessionID})

	h.logger.Info("Started gRPC stream", map[string]any{"session_id": sessionID})

	if err := h.srv.ExecuteWithWs(c.Request.Context(), conn, sessionID); err != nil {
		h.logger.Error("ExecuteWithWs failed", map[string]any{"session_id": sessionID, "error": err})
	}
}

func newID() string {
	timestamp := time.Now().UnixMilli()

	return fmt.Sprintf("%x-%x-%x-%x", timestamp>>32, (timestamp>>16)&0xffff, timestamp&0xffff, time.Now().UnixNano()&0xffff)
}
