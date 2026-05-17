package opencode

import "encoding/json"

// ============================================================================
// Opencode 输出类型（导出）
// ============================================================================

// OpencodeEventType 事件类型。
type OpencodeEventType string

// 事件类型常量。
const (
	OpencodeEventTypeStepStart  OpencodeEventType = "step_start"
	OpencodeEventTypeStepFinish OpencodeEventType = "step_finish"
	OpencodeEventTypeText       OpencodeEventType = "text"
	OpencodeEventTypeToolUse    OpencodeEventType = "tool_use"
	OpencodeEventTypeReasoning  OpencodeEventType = "reasoning"
	OpencodeEventTypeSnapshot   OpencodeEventType = "snapshot"
	OpencodeEventTypePatch      OpencodeEventType = "patch"
	OpencodeEventTypeFile       OpencodeEventType = "file"
	OpencodeEventTypeAgent      OpencodeEventType = "agent"
	OpencodeEventTypeRetry      OpencodeEventType = "retry"
	OpencodeEventTypeSubtask    OpencodeEventType = "subtask"
	OpencodeEventTypeCompaction OpencodeEventType = "compaction"
	OpencodeEventTypeError      OpencodeEventType = "error"

	// Question 事件类型（Human-in-the-Loop）
	OpencodeEventTypeQuestionAsked    OpencodeEventType = "question.asked"
	OpencodeEventTypeQuestionReplied  OpencodeEventType = "question.replied"
	OpencodeEventTypeQuestionRejected OpencodeEventType = "question.rejected"
)

// OpencodeEvent 输出事件结构。
type OpencodeEvent struct {
	Type      OpencodeEventType `json:"type"`
	Timestamp int64             `json:"timestamp"`
	SessionID string            `json:"sessionID"`
	Part      json.RawMessage   `json:"part"`
}

// ============================================================================
// 事件部分类型
// ============================================================================

// textPart 文本部分。
type textPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// reasoningPart 推理部分。
type reasoningPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// toolPart 工具调用部分。
type toolPart struct {
	Type   string    `json:"type"`
	CallID string    `json:"callID,omitempty"`
	Tool   string    `json:"tool"`
	State  toolState `json:"state"`
}

// toolState 工具调用状态。
type toolState struct {
	Status string      `json:"status"`
	Input  interface{} `json:"input,omitempty"`
	Output string      `json:"output,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// stepStartPart 步骤开始部分。
type stepStartPart struct {
	Type string `json:"type"`
}

// stepFinishPart 步骤结束部分。
type stepFinishPart struct {
	Type   string               `json:"type"`
	Reason string               `json:"reason,omitempty"`
	Tokens stepFinishPartTokens `json:"tokens,omitempty"`
	Cost   float64              `json:"cost,omitempty"`
}

// stepFinishPartTokens 步骤结束 token 统计。
type stepFinishPartTokens struct {
	Input     float64 `json:"input,omitempty"`
	Output    float64 `json:"output,omitempty"`
	Reasoning float64 `json:"reasoning,omitempty"`
	Cache     struct {
		Read  float64 `json:"read,omitempty"`
		Write float64 `json:"write,omitempty"`
	} `json:"cache,omitempty"`
}

// errorPart 错误部分。
type errorPart struct {
	Error struct {
		Name string `json:"name"`
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	} `json:"error"`
}

// ErrorPart 错误部分（导出）。
type ErrorPart = errorPart

// questionPart 问题部分（Human-in-the-Loop）。
type questionPart struct {
	Type       string         `json:"type"`
	QuestionID string         `json:"questionId"`
	SessionID  string         `json:"sessionID"`
	Questions  []questionItem `json:"questions"`
	Status     string         `json:"status,omitempty"`
	Answers    [][]string     `json:"answers,omitempty"`
}

type questionItem struct {
	Question string           `json:"question"`
	Header   string           `json:"header"`
	Options  []questionOption `json:"options"`
	Multiple bool             `json:"multiple"`
	Custom   bool             `json:"custom"`
}

type questionOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// QuestionPart 问题部分（导出）。
type QuestionPart = questionPart

// ============================================================================
// SSE 事件类型（内部使用）
//
// 基于 opencode == v1.14.51 的 SSE 事件格式。
// opencode 通过 /global/event 端点推送两类事件：
//
// 1. BusEvent（实时事件）- 对应 opencode/src/bus/bus-event.ts
//    格式: { "id": "...", "payload": { "type": "event-type", "properties": { ... } } }
//          ^-- opencode >= v1.14.51 所有 BusEvent 新增 id 字段（上升标识符）。
//              本结构体未声明该字段，Go json.Unmarshal 会跳过未声明的字段，
//              不会报错。后续升级 opencode 时，请确认事件格式无其他重大变化。
//    事件类型:
//    - message.part.delta: 流式文本/推理增量
//    - session.idle: 会话空闲（deprecated，opencode >= v1.14.51 仍向后兼容发送，
//                    但推荐考虑切换至 session.status）
//    - session.status: 会话状态（opencode >= v1.14.51 新增或强化，内含 idle/retry/busy
//                      三种子状态，若后续 session.idle 停止发送则需改用此事件）
//    - session.error: 会话错误
//    - session.compacted: 会话压缩完成（opencode >= v1.14.51 新增，当前未处理）
//
// 2. SyncEvent（持久化事件）- 对应 opencode/src/sync/index.ts
//    格式: { "payload": { "type": "sync", "syncEvent": { "type": "event-type.version", "data": { ... } } } }
//    事件类型:
//    - message.updated.1: 消息创建/更新
//    - message.part.updated.1: Part 状态更新
//    - message.removed.1: 消息删除
//    - message.part.removed.1: Part 删除（opencode >= v1.14.51 新增，当前未处理）
//
// 3. server 事件（通过 /global/event 端点直接发送，不经过 Bus）
//    - server.connected: SSE 连接确认（opencode >= v1.14.51 新增，properties: {}，无需处理）
//    - server.heartbeat: 10 秒心跳保活（opencode >= v1.14.51 新增，properties: {}，无需处理）
// ============================================================================

// sseEvent SSE 事件结构（对应 GlobalEvent）。
type sseEvent struct {
	Directory string          `json:"directory"`
	Payload   sseEventPayload `json:"payload"`
}

// sseEventPayload SSE 事件负载（支持 BusEvent 和 SyncEvent 两种格式）。
type sseEventPayload struct {
	Type       string        `json:"type"`                 // BusEvent: 事件类型; SyncEvent: "sync"
	Properties sseEventProps `json:"properties,omitempty"` // BusEvent 的属性
	SyncEvent  *sseSyncEvent `json:"syncEvent,omitempty"`  // SyncEvent 的事件
}

// sseSyncEvent SyncEvent 事件结构（对应 opencode/src/sync/index.ts Event）。
type sseSyncEvent struct {
	Type        string           `json:"type"`        // 事件类型，如 "message.part.updated.1"
	ID          string           `json:"id"`          // 事件 ID
	Seq         int              `json:"seq"`         // 序列号
	AggregateID string           `json:"aggregateID"` // 聚合 ID（通常是 sessionID）
	Data        sseSyncEventData `json:"data"`        // 事件数据
}

// sseSyncEventData SyncEvent 事件数据（对应不同事件类型的 schema）。
// message.updated.1: { sessionID, info }
// message.part.updated.1: { sessionID, part, time }
type sseSyncEventData struct {
	SessionID string        `json:"sessionID"`
	Part      *sseEventPart `json:"part,omitempty"`
	Time      int64         `json:"time,omitempty"`
	MessageID string        `json:"messageID,omitempty"`
	Info      *sseEventInfo `json:"info,omitempty"`
}

// sseEventProps BusEvent 事件属性（对应 opencode BusEvent.properties）。
// message.part.delta: { sessionID, messageID, partID, field, delta }
// session.idle: { sessionID }
// session.error: { sessionID?, error }
// question.asked: { id, sessionID, questions } - 注意：OpenCode 用 "id" 而非 "questionId"
// question.replied: { sessionID, requestID, answers }
// question.rejected: { sessionID, requestID }
type sseEventProps struct {
	SessionID string             `json:"sessionID,omitempty"`
	Delta     string             `json:"delta,omitempty"`
	Part      sseEventPart       `json:"part,omitempty"`
	Status    sseEventStatus     `json:"status,omitempty"`
	Error     sseEventError      `json:"error,omitempty"`
	Info      sseEventInfo       `json:"info,omitempty"`
	MessageID string             `json:"messageID,omitempty"`
	PartID    string             `json:"partID,omitempty"`
	Field     string             `json:"field,omitempty"`
	ID        string             `json:"id,omitempty"`        // question.asked: 问题ID（OpenCode用"id"）
	RequestID string             `json:"requestID,omitempty"` // question.replied/rejected: 请求ID
	Questions []sseEventQuestion `json:"questions,omitempty"`
	Answers   [][]string         `json:"answers,omitempty"` // question.replied: 答案
}

// sseEventQuestion 问题结构（对应 opencode Question）。
type sseEventQuestion struct {
	Question string                   `json:"question"`
	Header   string                   `json:"header"`
	Options  []sseEventQuestionOption `json:"options"`
	Multiple bool                     `json:"multiple"`
	Custom   *bool                    `json:"custom,omitempty"` // 可选字段，缺失时由 EnableHumanInTheLoopCustom 配置决定
}

// sseEventQuestionOption 问题选项。
type sseEventQuestionOption struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

// sseEventInfo 消息信息（对应 opencode MessageV2.Info）。
type sseEventInfo struct {
	ID   string `json:"id"`
	Role string `json:"role"`
}

// sseEventError 错误信息（对应 opencode MessageV2.APIError）。
type sseEventError struct {
	Name string `json:"name"`
	Data struct {
		Message    string `json:"message"`
		StatusCode int    `json:"statusCode,omitempty"`
	} `json:"data"`
}

// sseEventPart Part 结构（对应 opencode MessageV2.Part）。
// 包含 text, reasoning, tool, step-start, step-finish 等类型。
type sseEventPart struct {
	ID        string       `json:"id"`
	SessionID string       `json:"sessionID"`
	MessageID string       `json:"messageID"`
	Type      string       `json:"type"`
	Text      string       `json:"text"`
	CallID    string       `json:"callID"`
	Tool      string       `json:"tool"`
	State     sseToolState `json:"state"`
	Reason    string       `json:"reason"` // step-finish 的 reason
	Tokens    struct {
		Total     float64 `json:"total"`
		Input     float64 `json:"input"`
		Output    float64 `json:"output"`
		Reasoning float64 `json:"reasoning"`
		Cache     struct {
			Read  float64 `json:"read"`
			Write float64 `json:"write"`
		} `json:"cache"`
	} `json:"tokens"` // step-finish 的 token 统计
	Cost float64 `json:"cost"` // step-finish 的 cost
}

// sseToolState 工具调用状态（对应 opencode MessageV2.ToolState）。
type sseToolState struct {
	Status string      `json:"status"`
	Input  interface{} `json:"input,omitempty"`
	Output string      `json:"output,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// sseEventStatus 会话状态（对应 opencode SessionStatus.Info）。
type sseEventStatus struct {
	Type string `json:"type"`
}
