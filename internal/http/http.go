package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/ruziba3vich/online_compiler_api_gateway/genprotos/genprotos/compiler_service"
	"github.com/ruziba3vich/online_compiler_api_gateway/internal/service"
	"github.com/ruziba3vich/online_compiler_api_gateway/pkg/lgg"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type (
	Handler struct {
		client compiler_service.CodeExecutorClient
		srv    *service.Service
		logger *lgg.Logger
	}
)

func NewHandler(client compiler_service.CodeExecutorClient, srv *service.Service, logger *lgg.Logger) *Handler {
	return &Handler{
		client: client,
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

	sessionID := uuid.NewString()
	h.logger.Info("WebSocket client connected", map[string]any{"session_id": sessionID})

	ctx, cancel := context.WithCancel(context.Background())
	stream, err := h.client.Execute(ctx)
	if err != nil {
		h.logger.Error("Failed to start gRPC stream", map[string]any{"session_id": sessionID, "error": err})
		conn.WriteMessage(websocket.TextMessage, fmt.Appendf([]byte{}, "Error: Failed to connect to execution service: %s", err.Error()))
		conn.Close()
		cancel()
		return
	}

	h.logger.Info("Started gRPC stream", map[string]any{"session_id": sessionID})

	if err := h.srv.ExecuteWithWs(conn, stream, cancel, sessionID); err != nil {
		h.logger.Error("ExecuteWithWs failed", map[string]any{"session_id": sessionID, "error": err})
	}
}
