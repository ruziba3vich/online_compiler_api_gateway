package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ruziba3vich/online_compiler_api_gateway/genprotos/genprotos/compiler_service"
	"github.com/ruziba3vich/online_compiler_api_gateway/pkg/lgg"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	mx     *sync.Mutex
	logger *lgg.Logger
}

func NewService(mx *sync.Mutex, logger lgg.Logger) *Service {
	return &Service{
		mx:     mx,
		logger: &logger,
	}
}

func (s *Service) ExecuteWithWs(conn *websocket.Conn,
	stream compiler_service.CodeExecutor_ExecuteClient,
	cancel context.CancelFunc, sessionID string) error {
	defer s.cleanup(conn, cancel, sessionID)

	go func() {
		defer func() {
			s.logger.Info("gRPC stream reader stopped", map[string]any{"session_id": sessionID})
			s.cleanup(conn, cancel, sessionID)
		}()

		for {
			resp, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					s.logger.Info("gRPC stream closed cleanly by server (EOF)", map[string]any{"session_id": sessionID})
					s.publishMessage(conn, websocket.TextMessage, []byte("INFO: Execution stream closed by server."))
				} else if status.Code(err) == codes.Canceled {
					s.logger.Warn("gRPC stream cancelled", map[string]any{"session_id": sessionID})
				} else {
					s.logger.Warn("Error receiving from gRPC stream", map[string]any{"session_id": sessionID, "error": err})
					s.publishMessage(conn, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, fmt.Sprintf("gRPC stream error: %v", err)))
				}
				return
			}

			var msgToSend []byte
			msgType := websocket.TextMessage

			switch payload := resp.Payload.(type) {
			case *compiler_service.ExecuteResponse_Output:
				msgToSend = []byte(payload.Output.OutputText)
				s.logger.Info("Received Output", map[string]any{"session_id": sessionID, "output": payload.Output.OutputText})
				if strings.HasSuffix(strings.TrimSpace(payload.Output.OutputText), ":") || strings.HasSuffix(strings.TrimSpace(payload.Output.OutputText), "?") {
					s.publishMessage(conn, websocket.TextMessage, []byte("Status: WAITING_FOR_INPUT"))
					s.logger.Info("Detected input prompt, sent WAITING_FOR_INPUT", map[string]any{"session_id": sessionID})
				}
			case *compiler_service.ExecuteResponse_Error:
				msgToSend = []byte("Error: " + payload.Error.ErrorText)
				s.logger.Error("Received Error", map[string]any{"session_id": sessionID, "error": payload.Error.ErrorText})
			case *compiler_service.ExecuteResponse_Status:
				msgToSend = []byte("Status: " + payload.Status.State)
				s.logger.Info("Received Status", map[string]any{"session_id": sessionID, "status": payload.Status.State})
			default:
				s.logger.Warn("Received unknown payload type from gRPC", map[string]any{"session_id": sessionID})
				continue
			}

			if err := s.publishMessage(conn, msgType, msgToSend); err != nil {
				s.logger.Error("Error writing to WebSocket", map[string]any{"session_id": sessionID, "error": err})
				return
			}
		}
	}()

	msgType, codePayload, err := conn.ReadMessage()
	if err != nil {
		s.logger.Error("Error reading initial code from WebSocket", map[string]any{"session_id": sessionID, "error": err})
		return err
	}
	if msgType != websocket.TextMessage {
		s.logger.Error("Initial message not text (code expected)", map[string]any{"session_id": sessionID})
		s.publishMessage(conn, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "Initial message must be Python code"))
		return fmt.Errorf("unknown message type")
	}

	s.logger.Info("Sending Code to gRPC", map[string]any{"session_id": sessionID, "bytes": len(codePayload)})
	initialReq := &compiler_service.ExecuteRequest{
		SessionId: sessionID,
		Payload: &compiler_service.ExecuteRequest_Code{
			Code: &compiler_service.Code{
				Language:   "python",
				SourceCode: string(codePayload),
			},
		},
	}
	if err := stream.Send(initialReq); err != nil {
		s.logger.Error("Failed to send initial code request to gRPC", map[string]any{"session_id": sessionID, "error": err})
		s.publishMessage(conn, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, fmt.Sprintf("Failed to send code: %v", err)))
		return err
	}

	for {
		msgType, inputPayload, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				s.logger.Error("Error reading from WebSocket", map[string]any{"session_id": sessionID, "error": err})
			} else {
				s.logger.Warn("WebSocket closed", map[string]any{"session_id": sessionID, "error": err})
			}
			return err
		}

		if msgType != websocket.TextMessage {
			s.logger.Warn("Ignoring non-text message from WebSocket", map[string]any{"session_id": sessionID})
			continue
		}

		s.logger.Info("Sending Input to gRPC", map[string]any{"session_id": sessionID, "input": string(inputPayload)})
		inputReq := &compiler_service.ExecuteRequest{
			SessionId: sessionID,
			Payload: &compiler_service.ExecuteRequest_Input{
				Input: &compiler_service.Input{
					InputText: string(inputPayload),
				},
			},
		}
		if err := stream.Send(inputReq); err != nil {
			s.logger.Error("Failed to send input request to gRPC", map[string]any{"session_id": sessionID, "error": err})
			s.publishMessage(conn, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, fmt.Sprintf("Failed to send input: %v", err)))
			return err
		}
	}
}

func (s *Service) publishMessage(conn *websocket.Conn, messageType int, data []byte) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return conn.WriteMessage(messageType, data)
}

func (s *Service) cleanup(conn *websocket.Conn, cancel context.CancelFunc, sessionID string) {
	s.logger.Info("Cleaning up session", map[string]any{"session_id": sessionID})
	cancel()
	if err := conn.Close(); err != nil {
		s.logger.Warn("Error closing WebSocket", map[string]any{"session_id": sessionID, "error": err})
	}
}
