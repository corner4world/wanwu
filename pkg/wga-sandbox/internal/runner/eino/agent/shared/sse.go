package shared

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type SSEWriter interface {
	WriteAgentEvent(event *adk.AgentEvent)
}

func marshalAgentEvent(event *adk.AgentEvent) ([]byte, error) {
	jsonData, err := json.Marshal(event)
	if err != nil {
		log.Printf("[SSE] Failed to marshal AgentEvent: %v", err)
		errEvent := &adk.AgentEvent{Err: fmt.Errorf("内部序列化错误")}
		return json.Marshal(errEvent)
	}
	log.Printf("[SSE] >>> AgentEvent: %s", string(jsonData))
	return jsonData, nil
}

// --- HTTP 实现 ---
type httpSSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func NewHTTPSSEWriter(w http.ResponseWriter, flusher http.Flusher) SSEWriter {
	return &httpSSEWriter{w: w, flusher: flusher}
}

func (h *httpSSEWriter) WriteAgentEvent(event *adk.AgentEvent) {
	jsonData, _ := marshalAgentEvent(event)
	_, _ = fmt.Fprintf(h.w, "data: %s\n\n", jsonData)
	h.flusher.Flush()
}

// --- CLI 实现 ---
type stdoutSSEWriter struct{}

func NewStdoutSSEWriter() SSEWriter {
	return &stdoutSSEWriter{}
}

func (s *stdoutSSEWriter) WriteAgentEvent(event *adk.AgentEvent) {
	jsonData, _ := marshalAgentEvent(event)
	_, _ = fmt.Fprintf(os.Stdout, "data: %s\n\n", jsonData)
}

// --- 公共事件处理 ---
func ProcessEvents(iter *adk.AsyncIterator[*adk.AgentEvent], w SSEWriter) (eventCount int, hasError bool) {
	for {
		event, ok := iter.Next()
		if !ok {
			log.Printf("[Events] Iterator closed, total events processed: %d", eventCount)
			return
		}

		if event.Err != nil {
			// 详细的错误日志
			log.Printf("[Events] ========== ERROR EVENT #%d ==========", eventCount)
			log.Printf("[Events] ERROR: %+v", event.Err)
			log.Printf("[Events] AgentName: %s", event.AgentName)
			log.Printf("[Events] RunPath: %v", event.RunPath)

			// 如果有Output信息，记录角色和工具名
			if event.Output != nil && event.Output.MessageOutput != nil {
				role := string(event.Output.MessageOutput.Role)
				toolName := event.Output.MessageOutput.ToolName
				if role != "" {
					log.Printf("[Events] Role: %s", role)
				}
				if toolName != "" {
					log.Printf("[Events] ToolName: %s", toolName)
				}
			}
			log.Printf("[Events] ==========================================")

			// 返回错误事件给客户端（保持现有逻辑）
			w.WriteAgentEvent(event)
			hasError = true
			continue
		}

		eventCount++

		if event.Output == nil || event.Output.MessageOutput == nil {
			log.Printf("[Events] Event #%d: empty output, skipping", eventCount)
			continue
		}

		role := string(event.Output.MessageOutput.Role)
		isStreaming := event.Output.MessageOutput.IsStreaming
		log.Printf("[Events] Event #%d: role=%s, streaming=%v", eventCount, role, isStreaming)

		if isStreaming {
			handleStreaming(event, w, role, eventCount)
		} else {
			handleNonStreaming(event, w, role, eventCount)
		}
	}
}

func handleStreaming(event *adk.AgentEvent, w SSEWriter, _ string, eventNum int) {
	msgStream := event.Output.MessageOutput.MessageStream
	if msgStream == nil {
		log.Printf("[Events] Event #%d: streaming message stream is nil, skipping", eventNum)
		return
	}

	var messages []*schema.Message
	chunkCount := 0
	for {
		msg, err := msgStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			// 详细的错误日志
			log.Printf("[Events] ========== STREAM ERROR - Event #%d ==========", eventNum)
			log.Printf("[Events] ERROR: %+v", err)
			log.Printf("[Events] AgentName: %s", event.AgentName)
			log.Printf("[Events] RunPath: %v", event.RunPath)

			role := string(event.Output.MessageOutput.Role)
			toolName := event.Output.MessageOutput.ToolName
			if role != "" {
				log.Printf("[Events] Role: %s", role)
			}
			if toolName != "" {
				log.Printf("[Events] ToolName: %s", toolName)
			}
			log.Printf("[Events] ====================================================")

			// 构造错误事件并返回给客户端
			errorEvent := &adk.AgentEvent{
				AgentName: event.AgentName,
				RunPath:   event.RunPath,
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{
						IsStreaming: false,
						Role:        event.Output.MessageOutput.Role,
						ToolName:    event.Output.MessageOutput.ToolName,
					},
				},
				Err: fmt.Errorf("stream recv error: %w", err),
			}
			w.WriteAgentEvent(errorEvent)
			break
		}
		if msg == nil {
			continue
		}
		chunkCount++
		messages = append(messages, msg)
	}

	if len(messages) == 0 {
		return
	}

	concatMsg, err := schema.ConcatMessages(messages)
	if err != nil || concatMsg == nil {
		// 详细的错误日志
		log.Printf("[Events] ========== CONCAT ERROR - Event #%d ==========", eventNum)
		if err != nil {
			log.Printf("[Events] ERROR: %+v", err)
		} else {
			log.Printf("[Events] ERROR: concatenated message is nil")
		}
		log.Printf("[Events] AgentName: %s", event.AgentName)
		log.Printf("[Events] RunPath: %v", event.RunPath)
		log.Printf("[Events] Chunks count: %d", chunkCount)

		role := string(event.Output.MessageOutput.Role)
		toolName := event.Output.MessageOutput.ToolName
		if role != "" {
			log.Printf("[Events] Role: %s", role)
		}
		if toolName != "" {
			log.Printf("[Events] ToolName: %s", toolName)
		}
		log.Printf("[Events] ===================================================")

		// 构造错误事件并返回给客户端
		var errMsg error
		if err != nil {
			errMsg = fmt.Errorf("failed to concat messages: %w", err)
		} else {
			errMsg = fmt.Errorf("concatenated message is nil")
		}

		errorEvent := &adk.AgentEvent{
			AgentName: event.AgentName,
			RunPath:   event.RunPath,
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming: false,
					Role:        event.Output.MessageOutput.Role,
					ToolName:    event.Output.MessageOutput.ToolName,
				},
			},
			Err: errMsg,
		}
		w.WriteAgentEvent(errorEvent)
		return
	}

	log.Printf("[Events] Event #%d: streaming done (%d chunks, content=%d bytes, tool_calls=%d)",
		eventNum, chunkCount, len(concatMsg.Content), len(concatMsg.ToolCalls))

	outputEvent := &adk.AgentEvent{
		AgentName: event.AgentName,
		RunPath:   event.RunPath,
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				IsStreaming: false,
				Message:     concatMsg,
				Role:        event.Output.MessageOutput.Role,
				ToolName:    event.Output.MessageOutput.ToolName,
			},
		},
		Action: event.Action,
	}

	w.WriteAgentEvent(outputEvent)
}

func handleNonStreaming(event *adk.AgentEvent, w SSEWriter, _ string, eventNum int) {
	msg, err := event.Output.MessageOutput.GetMessage()
	if err != nil || msg == nil {
		// 详细的错误日志
		log.Printf("[Events] ========== NON-STREAMING ERROR - Event #%d ==========", eventNum)
		if err != nil {
			log.Printf("[Events] ERROR: %+v", err)
		} else {
			log.Printf("[Events] ERROR: message is nil")
		}
		log.Printf("[Events] AgentName: %s", event.AgentName)
		log.Printf("[Events] RunPath: %v", event.RunPath)

		role := string(event.Output.MessageOutput.Role)
		toolName := event.Output.MessageOutput.ToolName
		if role != "" {
			log.Printf("[Events] Role: %s", role)
		}
		if toolName != "" {
			log.Printf("[Events] ToolName: %s", toolName)
		}
		log.Printf("[Events] ===========================================================")

		// 构造错误事件并返回给客户端
		var errMsg error
		if err != nil {
			errMsg = fmt.Errorf("failed to get non-streaming message: %w", err)
		} else {
			errMsg = fmt.Errorf("non-streaming message is nil")
		}

		errorEvent := &adk.AgentEvent{
			AgentName: event.AgentName,
			RunPath:   event.RunPath,
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming: false,
					Role:        event.Output.MessageOutput.Role,
					ToolName:    event.Output.MessageOutput.ToolName,
				},
			},
			Err: errMsg,
		}
		w.WriteAgentEvent(errorEvent)
		return
	}

	log.Printf("[Events] Event #%d: non-streaming (content=%d bytes, tool_calls=%d)",
		eventNum, len(msg.Content), len(msg.ToolCalls))

	outputEvent := &adk.AgentEvent{
		AgentName: event.AgentName,
		RunPath:   event.RunPath,
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				IsStreaming: false,
				Message:     msg,
				Role:        event.Output.MessageOutput.Role,
				ToolName:    event.Output.MessageOutput.ToolName,
			},
		},
		Action: event.Action,
	}

	w.WriteAgentEvent(outputEvent)
}
