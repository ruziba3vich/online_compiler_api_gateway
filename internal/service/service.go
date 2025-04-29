package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/ruziba3vich/online_compiler_api_gateway/genprotos/genprotos/compiler_service"
	"github.com/ruziba3vich/online_compiler_api_gateway/internal/repos"
	"github.com/ruziba3vich/online_compiler_api_gateway/pkg/lgg"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// WsMessage represents the JSON payload received over WebSocket.
type WsMessage struct {
	Language string `json:"language,omitempty"`
	Code     string `json:"code,omitempty"`
	Input    string `json:"input,omitempty"`
}

// CodeExecutor defines the interface for language-specific gRPC clients.
type CodeExecutor interface {
	Execute(ctx context.Context) (compiler_service.CodeExecutor_ExecuteClient, error)
}

// Service manages WebSocket connections and routes code execution to language-specific gRPC services.
type Service struct {
	mx        *sync.Mutex
	logger    *lgg.Logger
	dangerous map[string][]string
	executors map[string]CodeExecutor
}

// NewService initializes the service with a registry of language executors.
func NewService(
	mx *sync.Mutex,
	logger *lgg.Logger,
	pythonClient repos.Python,
	javaClient repos.Java,
	cppClient repos.Cpp) *Service {
	dangerous := map[string][]string{
		"python": {
			"import os", "import subprocess", "__import__",
			"import sys", "import shutil", "exec(",
			"os.system", "subprocess", "importlib",
			"open(",
		},
		"java": {
			"Runtime.getRuntime().exec(",
			"new ProcessBuilder(",
			"ProcessBuilder",
			"Runtime.exec(",

			"java.io.File",
			"new File(",
			".delete()",
			".mkdir()",
			".renameTo(",
			"java.io.FileOutputStream",
			"java.io.FileInputStream",
			"java.io.RandomAccessFile",
			"java.nio.file.Files",
			"java.nio.file.Paths",
			"Files.write(",
			"Files.readAllBytes(",
			"Files.delete(",
			"Files.copy(",
			"Files.move(",

			"java.net.Socket",
			"new Socket(",
			"java.net.ServerSocket",
			"new ServerSocket(",
			"java.net.URL",
			".openConnection(",
			".openStream(",
			"java.net.DatagramSocket",
			"java.nio.channels.SocketChannel",
			"java.nio.channels.ServerSocketChannel",

			"java.lang.reflect",
			"Class.forName(",
			".setAccessible(true)",
			"Method.invoke(",
			"Field.set(",

			"System.exit(",
			"System.load(",
			"System.loadLibrary(",
			"System.getenv(",
			"System.getProperty(",
			"System.setProperty(",
			"System.getSecurityManager(",
			"System.setSecurityManager(",
			"java.lang.ClassLoader",
			"URLClassLoader",
			"new Thread(",
		},
		"cpp": {
			"system(",
			"popen(",
			"exec(",
			"execl(",
			"execle(",
			"execlp(",
			"execv(",
			"execve(",
			"execvp(",
			"fork(",
			"vfork(",
			"spawn(",

			"fopen(",
			"freopen(",
			"fdopen(",
			"fclose(",
			"remove(",
			"rename(",
			"tmpfile(",
			"tmpnam(",
			"unlink(",
			"mkdir(",
			"rmdir(",

			"std::fstream",
			"std::ifstream",
			"std::ofstream",
			"std::filebuf",
			"std::filesystem::create_directory(",
			"std::filesystem::remove(",
			"std::filesystem::remove_all(",
			"std::filesystem::rename(",
			"std::filesystem::copy(",
			"std::filesystem::copy_file(",
			"std::filesystem::resize_file(",

			"std::getenv(",
			"std::setenv(",
			"std::putenv(",
			"std::system(",
			"std::abort(",
			"std::exit(",
			"std::quick_exit(",
			"std::terminate(",

			"socket(",
			"bind(",
			"listen(",
			"accept(",
			"connect(",
			"send(",
			"sendto(",
			"recv(",
			"recvfrom(",
			"gethostbyname(",
			"gethostbyaddr(",
			"getaddrinfo(",
			"std::net::socket",

			"malloc(",
			"calloc(",
			"realloc(",
			"free(",
			"std::allocator",
			"std::memcpy(",
			"std::memmove(",
			"std::memset(",
			"std::raw_storage_iterator",

			"dlopen(",
			"dlsym(",
			"dlclose(",
			"dlerror(",

			"std::thread",
			"std::async(",
			"std::mutex",
			"std::lock_guard",
			"std::unique_lock",
			"pthread_create(",
			"pthread_join(",
			"pthread_detach(",

			"asm",
			"__asm__",
			"inline asm",
			"volatile",
			"std::signal(",
			"std::raise(",
			"std::setjmp(",
			"std::longjmp(",

			"#include <cstdlib>",
			"#include <cstdio>",
			"#include <fstream>",
			"#include <filesystem>",
			"#include <sys/socket.h>",
			"#include <netinet/in.h>",
			"#include <arpa/inet.h>",
			"#include <netdb.h>",
			"#include <dlfcn.h>",
			"#include <pthread.h>",
			"#include <signal.h>",
			"#include <unistd.h>",
			"#include <sys/stat.h>",
			"#include <sys/types.h>",

			"operator new",
			"operator delete",
			"std::unique_ptr",
			"std::shared_ptr",
			"std::weak_ptr",
			"std::dynamic_pointer_cast(",
			"std::static_pointer_cast(",
			"std::const_pointer_cast(",
		},
	}

	executors := map[string]CodeExecutor{
		"python": &Compiler{client: pythonClient},
		"java":   &Compiler{client: javaClient},
		"cpp":    &Compiler{client: cppClient},
	}

	return &Service{
		mx:        mx,
		logger:    logger,
		dangerous: dangerous,
		executors: executors,
	}
}

// PythonExecutor wraps the Python gRPC client to implement CodeExecutor.
type Compiler struct {
	client compiler_service.CodeExecutorClient
}

func (p *Compiler) Execute(ctx context.Context) (compiler_service.CodeExecutor_ExecuteClient, error) {
	return p.client.Execute(ctx)
}

// ExecuteWithWs handles WebSocket connections and routes code execution to the appropriate language service.
func (s *Service) ExecuteWithWs(ctx context.Context, conn *websocket.Conn, sessionID string) error {
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
					if !strings.Contains(payload.Error.ErrorText, "--- Cleaned up") {
						msgToSend = []byte("Error: " + payload.Error.ErrorText)
					}
					s.logger.Error("Received Error", map[string]any{"session_id": sessionID, "error": payload.Error.ErrorText})
				case *compiler_service.ExecuteResponse_Status:
					msgToSend = []byte("Status: " + payload.Status.State)
					s.logger.Info("Received Status", map[string]any{"session_id": sessionID, "status": payload.Status.State})
				default:
					s.logger.Warn("Received unknown payload type from gRPC", map[string]any{"session_id": sessionID})
					continue
				}

				if len(msgToSend) > 0 {
					if err := s.publishMessage(conn, msgType, msgToSend); err != nil {
						s.logger.Error("Error writing to WebSocket", map[string]any{"session_id": sessionID, "error": err})
						return
					}
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

		var wsMsg WsMessage
		if err := json.Unmarshal(payload, &wsMsg); err != nil {
			s.logger.Warn("Invalid JSON message", map[string]any{"session_id": sessionID, "error": err})
			s.publishMessage(conn, websocket.TextMessage, []byte(fmt.Sprintf("Error: Invalid JSON: %v", err)))
			continue
		}
		s.logger.Debug("Received WebSocket JSON message", map[string]any{"session_id": sessionID, "message": wsMsg})

		if wsMsg.Language != "" && wsMsg.Code != "" {
			s.logger.Info("Received new code submission", map[string]any{"session_id": sessionID, "language": wsMsg.Language, "code_length": len(wsMsg.Code)})

			executor, ok := s.executors[strings.ToLower(wsMsg.Language)]
			if !ok {
				s.logger.Warn("Unsupported language", map[string]any{"session_id": sessionID, "language": wsMsg.Language})
				s.publishMessage(conn, websocket.TextMessage, []byte(fmt.Sprintf("Error: Language '%s' is not supported", wsMsg.Language)))
				continue
			}

			dangerousKeywords, exists := s.dangerous[strings.ToLower(wsMsg.Language)]
			if !exists {
				dangerousKeywords = []string{}
			}
			for _, keyword := range dangerousKeywords {
				if strings.Contains(wsMsg.Code, keyword) {
					s.logger.Warn("Dangerous code detected", map[string]any{"session_id": sessionID, "language": wsMsg.Language})
					s.publishMessage(conn, websocket.TextMessage, []byte("Error: dangerous script detected"))
					return errors.New("unsafe code detected")
				}
			}

			cleanupStream()

			sessionID = uuid.NewString()
			s.logger.Info("Generated new session ID for code submission", map[string]any{"session_id": sessionID})

			ctx, cancel := context.WithCancel(ctx)
			currentCancel = cancel
			currentStream, err = executor.Execute(ctx)
			if err != nil {
				s.logger.Error("Failed to start gRPC stream", map[string]any{"session_id": sessionID, "language": wsMsg.Language, "error": err})
				s.publishMessage(conn, websocket.TextMessage, []byte(fmt.Sprintf("Error: Failed to connect to %s execution service: %v", wsMsg.Language, err)))
				return err
			}
			s.logger.Info("Started new gRPC stream", map[string]any{"session_id": sessionID, "language": wsMsg.Language})

			startStreamReader()

			req := &compiler_service.ExecuteRequest{
				SessionId: sessionID,
				Payload: &compiler_service.ExecuteRequest_Code{
					Code: &compiler_service.Code{
						Language:   wsMsg.Language,
						SourceCode: wsMsg.Code,
					},
				},
			}
			if err := currentStream.Send(req); err != nil {
				s.logger.Error("Failed to send code request to gRPC", map[string]any{"session_id": sessionID, "language": wsMsg.Language, "error": err})
				s.publishMessage(conn, websocket.TextMessage, []byte(fmt.Sprintf("Error: Failed to send code: %v", err)))
				cleanupStream()
				return err
			}
			s.logger.Info("Sent code to gRPC", map[string]any{"session_id": sessionID, "language": wsMsg.Language, "bytes": len(wsMsg.Code)})
		} else if wsMsg.Input != "" && currentStream != nil {
			s.logger.Info("Received input", map[string]any{"session_id": sessionID, "input": wsMsg.Input})
			req := &compiler_service.ExecuteRequest{
				SessionId: sessionID,
				Payload: &compiler_service.ExecuteRequest_Input{
					Input: &compiler_service.Input{
						InputText: wsMsg.Input,
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
			s.logger.Warn("Invalid or unexpected JSON message", map[string]any{"session_id": sessionID, "message": wsMsg})
			s.publishMessage(conn, websocket.TextMessage, []byte("Error: Invalid message. Send JSON with 'language' and 'code' or 'input' for active session."))
		}
	}
}

func (s *Service) publishMessage(conn *websocket.Conn, messageType int, data []byte) error {
	s.mx.Lock()
	defer s.mx.Unlock()
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return conn.WriteMessage(messageType, data)
}
