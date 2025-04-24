package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/ruziba3vich/online_compiler_api_gateway/genprotos/genprotos/compiler_service"
	"github.com/ruziba3vich/online_compiler_api_gateway/pkg/lgg"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	mx        *sync.Mutex
	logger    *lgg.Logger
	dangerous []string
}

func NewService(mx *sync.Mutex, logger lgg.Logger) *Service {
	return &Service{
		mx:     mx,
		logger: &logger,
		dangerous: []string{
			"import os", "import subprocess", "__import__",
			"import sys", "import shutil", "open(", "eval(", "exec(",
			"input(", "os.system", "subprocess", "importlib",
		},
	}
}

func (s *Service) ExecuteWithWs(ctx context.Context, conn *websocket.Conn, client compiler_service.CodeExecutorClient, sessionID string) error {
	var currentStream compiler_service.CodeExecutor_ExecuteClient
	var currentCancel context.CancelFunc

	cleanupStream := func() {
		if currentCancel != nil {
			s.logger.Info("Cleaning up current stream", map[string]any{"session_id": sessionID})
			currentCancel()
			currentCancel = nil
			currentStream = nil
		}
	}

	startStreamReader := func() {
		go func(stream compiler_service.CodeExecutor_ExecuteClient, cancel context.CancelFunc, sessionID string) {
			defer func() {
				s.logger.Info("gRPC stream reader stopped", map[string]any{"session_id": sessionID})
				cleanupStream()
				s.publishMessage(conn, websocket.TextMessage, []byte("Status: STREAM_CLOSED"))
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
						s.publishMessage(conn, websocket.TextMessage, []byte(fmt.Sprintf("Error: gRPC stream error: %v", err)))
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
		}(currentStream, currentCancel, sessionID)
	}

	for {
		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				s.logger.Error("Error reading from WebSocket", map[string]any{"session_id": sessionID, "error": err})
			} else {
				s.logger.Warn("WebSocket closed", map[string]any{"session_id": sessionID, "error": err})
			}
			cleanupStream()
			return err
		}

		if msgType != websocket.TextMessage {
			s.logger.Warn("Ignoring non-text message from WebSocket", map[string]any{"session_id": sessionID})
			continue
		}

		message := string(payload)
		s.logger.Debug("Received WebSocket message", map[string]any{"session_id": sessionID, "message": message})

		if strings.HasPrefix(message, "CODE:") {
			code := strings.TrimPrefix(message, "CODE:")
			s.logger.Info("Received new code submission", map[string]any{"session_id": sessionID, "code_length": len(code)})

			cleanupStream()
			for _, keyword := range s.dangerous {
				if strings.Contains(code, keyword) {
					return errors.New("unsafe code detected")
				}
			}

			sessionID = uuid.NewString()
			s.logger.Info("Generated new session ID for code submission", map[string]any{"session_id": sessionID})

			ctx, cancel := context.WithCancel(ctx)
			currentCancel = cancel
			currentStream, err = client.Execute(ctx)
			if err != nil {
				s.logger.Error("Failed to start gRPC stream", map[string]any{"session_id": sessionID, "error": err})
				s.publishMessage(conn, websocket.TextMessage, []byte(fmt.Sprintf("Error: Failed to connect to execution service: %v", err)))
				return err
			}
			s.logger.Info("Started new gRPC stream", map[string]any{"session_id": sessionID})

			startStreamReader()

			req := &compiler_service.ExecuteRequest{
				SessionId: sessionID,
				Payload: &compiler_service.ExecuteRequest_Code{
					Code: &compiler_service.Code{
						Language:   "python",
						SourceCode: code,
					},
				},
			}
			if err := currentStream.Send(req); err != nil {
				s.logger.Error("Failed to send code request to gRPC", map[string]any{"session_id": sessionID, "error": err})
				s.publishMessage(conn, websocket.TextMessage, []byte(fmt.Sprintf("Error: Failed to send code: %v", err)))
				cleanupStream()
				return err
			}
			s.logger.Info("Sent code to gRPC", map[string]any{"session_id": sessionID, "bytes": len(code)})
		} else if currentStream != nil {
			s.logger.Info("Received input", map[string]any{"session_id": sessionID, "input": message})
			req := &compiler_service.ExecuteRequest{
				SessionId: sessionID,
				Payload: &compiler_service.ExecuteRequest_Input{
					Input: &compiler_service.Input{
						InputText: message,
					},
				},
			}
			if err := currentStream.Send(req); err != nil {
				s.logger.Error("Failed to send input request to gRPC", map[string]any{"session_id": sessionID, "error": err})
				s.publishMessage(conn, websocket.TextMessage, []byte(fmt.Sprintf("Error: Failed to send input: %v", err)))
				cleanupStream()
				return err
			}
			s.logger.Info("Sent input to gRPC", map[string]any{"session_id": sessionID})
		} else {
			s.logger.Warn("Received message without active stream", map[string]any{"session_id": sessionID, "message": message})
			s.publishMessage(conn, websocket.TextMessage, []byte("Error: No active execution. Send code with 'CODE:' prefix."))
		}
	}
}

func (s *Service) publishMessage(conn *websocket.Conn, messageType int, data []byte) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return conn.WriteMessage(messageType, data)
}
