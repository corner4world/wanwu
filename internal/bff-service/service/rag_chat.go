package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	err_code "github.com/UnicomAI/wanwu/api/proto/err-code"
	rag_service "github.com/UnicomAI/wanwu/api/proto/rag-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/pkg/ahocorasick"
	ag_ui_util "github.com/UnicomAI/wanwu/pkg/ag-ui-util"
	"github.com/UnicomAI/wanwu/pkg/constant"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	sse_util "github.com/UnicomAI/wanwu/pkg/sse-util"
	"github.com/UnicomAI/wanwu/pkg/util"
	aguievents "github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// --- AG-UI 事件名常量（RAG 专属）---
const (
	// EventNameRagSearchList CUSTOM 事件名：知识库检索命中列表
	EventNameRagSearchList = "rag_search_list"
	// EventNameRagKnowledgeStart CUSTOM 事件名：即将进入知识库检索（前端据此来创建"知识库检索"卡片）
	EventNameRagKnowledgeStart = "rag_knowledge_start"
	// EventNameRagQAStart CUSTOM 事件名：即将进入问答库检索（前端据此来创建"问答库检索"卡片）
	EventNameRagQAStart = "rag_qa_start"
	// EventNameRagQASearchList CUSTOM 事件名：问答库检索结果列表（与 KB 分开，前端独立渲染 QA 卡片）。
	// 即使命中为空也会发一次（payload=[]），以便前端把"问答库检索"卡片从 running 切到 done（未命中）。
	EventNameRagQASearchList = "rag_qa_search_list"
)

// --- rag-service SSE msg_type 常量（对应 rag-service 的 RagMessageType）---
const (
	ragMsgTypeQAStart        = "qa_start"
	ragMsgTypeQAFinish       = "qa_finish"
	ragMsgTypeKnowledgeStart = "knowledge_start"
)

// --- RAG RUN_ERROR 错误码 ---
// 前端通过该 code 查 vue-i18n 文案。新增码时须同步更新
// web/src/mixins/sseMethod.js 的 RAG_ERROR_CODE_I18N 映射表。
const (
	RagErrCodeSensitiveBlock = "sensitive_block" // 上游 finish=2：敏感词拦截
	RagErrCodeUpstream       = "upstream_error"  // 上游返回非零业务错误码
	RagErrCodeUnknown        = "unknown_error"   // 未分类错误（兜底）
)

// --- rag-service 业务返回码（chunk.Code）---
// 0/1：正常（0=成功、1=流式中间帧）；非 0/1 一律视为错误。
// 7：业务失败（如模型调用失败、检索失败等），对应 RagErrCodeUpstream；
//
//	其他非 0/1 的 code 归类为 RagErrCodeUnknown 兜底。
const (
	ragChunkCodeBusinessError = 7
)

// ragChatStreamParams 记录流式请求的过程参数（首 token 延迟、错误标志等）
type ragChatStreamParams struct {
	ctx               *gin.Context
	startTime         time.Time
	firstTokenLatency int64
	hasRecorded       bool
	hasErr            bool
}

// ragChunkData 对应 rag-service / rag-wanwu 返回的每条 SSE JSON 结构
type ragChunkData struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	MsgID   string         `json:"msg_id"`
	MsgType string         `json:"msg_type"`
	Data    *ragChunkInner `json:"data"`
	Finish  int            `json:"finish"`
}

// ragChunkInner 对应 data 字段，SearchList 保持 json.RawMessage 以便原样透传
type ragChunkInner struct {
	Output           string          `json:"output"`
	ReasoningContent string          `json:"reasoning_content"`
	SearchList       json.RawMessage `json:"searchList"`
}

// ChatRagStream RAG 私域问答，流式返回 AG-UI 协议事件
func ChatRagStream(ctx *gin.Context, userId, orgId string, req request.ChatRagRequest, needLatestPublished bool, source string) (err error) {
	streamParams := &ragChatStreamParams{ctx: ctx, startTime: time.Now()}
	defer func() {
		if source != constant.AppStatisticSourceDraft {
			RecordAppStatistic(ctx.Request.Context(), userId, orgId, req.RagID, constant.AppTypeRag, !streamParams.hasErr, true, streamParams.firstTokenLatency, 0, source)
		}
	}()

	chatCh, kbNameMap, err := CallRagChatStream(ctx, userId, orgId, req, needLatestPublished)
	if err != nil {
		streamParams.hasErr = true
		return err
	}

	// AG-UI 协议要求 threadId/runId 每次 run 唯一；RAG 当前无持久化会话概念，
	// 两者均使用 uuid（若后续引入 conversationID，可以把 threadID 换成它）
	runID := uuid.NewString()
	threadID := uuid.NewString()

	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")

	eventCh := convertRagStream2AGUIEvents(ctx.Request.Context(), chatCh, threadID, runID, streamParams, kbNameMap)
	outputCh := ag_ui_util.EventsToJSONChannel(ctx.Request.Context(), eventCh)

	ctx.Stream(func(w io.Writer) bool {
		select {
		case line, ok := <-outputCh:
			if !ok {
				return false
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", line)
			return true
		case <-ctx.Request.Context().Done():
			return false
		}
	})
	return nil
}

// ragStreamConverter 封装 rag-service 原始 SSE → AG-UI 事件的转换器。
//
// 拆分原因：原来 convertRagStream2AGUIEvents 把 "goroutine 骨架 / 事件发射 /
// 首 token 埋点 / chunk 业务分支" 四种关注点揉在 150 行单函数里，嵌 6 层缩进。
// 拆成 struct 后每个方法 5–15 行，handleChunk 可独立单测。
//
// 转换规则：
//  1. 首条事件：RUN_STARTED
//  2. msg_type=qa_start 状态帧：CUSTOM(rag_qa_start) — 通知前端懒创建"问答库检索"卡片
//  3. msg_type=qa_finish：CUSTOM(rag_qa_search_list)（命中/未命中都发，未命中 payload=[]）+ 透传 output
//     —— 不走通用 searchList 分支，QA 结果与 KB 结果在前端独立渲染
//  4. msg_type=knowledge_start 状态帧：CUSTOM(rag_knowledge_start) — 通知前端懒创建"知识库检索"卡片
//  5. 收到第一个非空 searchList（knowledge_content）时：CUSTOM(rag_search_list) — 必须在任何文字事件之前
//  6. reasoning_content 非空：REASONING_MESSAGE_START / CONTENT（逐 token）
//  7. output 非空（reasoning 阶段已结束）：REASONING_MESSAGE_END + TEXT_MESSAGE_START / CONTENT
//  8. finish=1（终止帧）：BaseState.FinishBase —— 自动关闭所有开放消息 + RUN_FINISHED
//  9. 错误 / finish=2（敏感词拦截）：RUN_ERROR（带 code 字段供前端 i18n 查表）
//
// 事件序列化由调用方通过 ag_ui_util.EventsToJSONChannel 完成。
type ragStreamConverter struct {
	ctx                   context.Context
	out                   chan<- aguievents.Event
	state                 *ag_ui_util.BaseState
	runID                 string
	streamParams          *ragChatStreamParams
	kbNameMap             map[string]string
	hasSentSearchList     bool
	hasSentQASearchList   bool
	hasSentKnowledgeStart bool
	hasSentQAStart        bool
	// hasFinalized 标记是否已发出 RUN_FINISHED 或 RUN_ERROR；用于在上游 channel 异常关闭时
	// 兜底补发 RUN_ERROR，避免前端收到无收尾事件的 SSE 流（症状：前端一直 loading）。
	hasFinalized bool
}

// emit 将一批事件写入输出 channel；ctx 取消时返回 false 让调用方提前退出。
func (c *ragStreamConverter) emit(events ...aguievents.Event) bool {
	for _, evt := range events {
		select {
		case c.out <- evt:
		case <-c.ctx.Done():
			return false
		}
	}
	return true
}

// finalizeError 关闭所有活跃消息后发 RUN_ERROR（不发 RUN_FINISHED）。
// 幂等：若已 finalize（RUN_FINISHED/RUN_ERROR 已发）则直接返回，避免重复事件。
func (c *ragStreamConverter) finalizeError(code, msg string) {
	if c.hasFinalized {
		return
	}
	if msg == "" {
		msg = code // 兜底：至少让 Message 非空满足协议 Validate
	}
	c.emit(c.state.EnsureRunStarted()...)
	c.emit(c.state.EndAll()...)
	c.emit(aguievents.NewRunErrorEvent(msg,
		aguievents.WithErrorCode(code),
		aguievents.WithRunID(c.runID)))
	c.hasFinalized = true
}

// finalizeSuccess 正常收尾：发 RUN_FINISHED（经由 BaseState.FinishBase，会自动关闭所有开放消息）。
func (c *ragStreamConverter) finalizeSuccess() {
	if c.hasFinalized {
		return
	}
	c.emit(c.state.FinishBase()...)
	c.hasFinalized = true
}

// recordTTFT 记录首 token 延迟。
// 口径：首个"生成内容" token（reasoning_content 或 output 首字符），
// 不包含连接延迟与检索延迟——业界通行的 TTFT 定义。
// 注：此口径与旧版（首条 SSE 帧即记录）有差异，旧版会把检索时延算进来。
func (c *ragStreamConverter) recordTTFT() {
	if c.streamParams.hasRecorded {
		return
	}
	c.streamParams.firstTokenLatency = time.Since(c.streamParams.startTime).Milliseconds()
	c.streamParams.hasRecorded = true
	if c.streamParams.ctx != nil {
		c.streamParams.ctx.Set(gin_util.FIRST_RESP_LATENCY, c.streamParams.firstTokenLatency)
	}
}

// handleChunk 处理单条 chunk。返回 true 表示流已终止（error / finish=1/2），调用方应退出循环。
func (c *ragStreamConverter) handleChunk(chunk ragChunkData) (done bool) {
	// 非零 code（0外）视为错误
	if chunk.Code != 0 {
		c.streamParams.hasErr = true
		code := RagErrCodeUnknown
		if chunk.Code == ragChunkCodeBusinessError {
			code = RagErrCodeUpstream
		}
		c.finalizeError(code, chunk.Message)
		return true
	}

	// finish=2：敏感词拦截
	if chunk.Finish == 2 {
		c.streamParams.hasErr = true
		c.finalizeError(RagErrCodeSensitiveBlock, "Content blocked by sensitive word filter")
		return true
	}

	// 状态帧（Data 通常为空）：通知前端懒创建对应检索卡片。
	// 必须在 Data==nil 短路之前处理。
	switch chunk.MsgType {
	case ragMsgTypeQAStart:
		c.emitQAStartOnce()
	case ragMsgTypeKnowledgeStart:
		c.emitKnowledgeStartOnce()
	}

	// 纯状态帧：Data 为 nil 时只看 finish 决定是否终止
	if chunk.Data == nil {
		if chunk.Finish == 1 {
			c.finalizeSuccess()
			return true
		}
		return false
	}

	// qa_finish（问答库检索结束）：独立发 QA 搜索列表（未命中 payload=[]，前端据此把卡片从 running 切到 done），
	// 然后透传 output（命中则 output 是答案，未命中且无 KB 时 output 是"无法回答"兜底文案）。
	// 不走通用 searchList 分支，避免 QA 结果混入 KB 的 rag_search_list 事件。
	// 错误路径（chunk.Code 非零 / finish=2）在上方已处理，QA 阶段的错误仍会正常透出。
	if chunk.MsgType == ragMsgTypeQAFinish {
		c.emitQASearchListOnce(chunk.Data.SearchList)
		c.emitOutput(chunk.Data.Output)
		if chunk.Finish == 1 {
			c.emit(c.state.FinishBase()...)
			return true
		}
		return false
	}

	c.emitSearchListOnce(chunk.Data.SearchList)
	c.emitReasoning(chunk.Data.ReasoningContent)
	c.emitOutput(chunk.Data.Output)

	if chunk.Finish == 1 {
		c.emit(c.state.FinishBase()...)
		return true
	}
	return false
}

// emitKnowledgeStartOnce 在首次收到 knowledge_start 状态帧时发 CUSTOM 事件。
// 幂等：即使后端重复发也只发一次，避免前端重复创建卡片。
func (c *ragStreamConverter) emitKnowledgeStartOnce() {
	if c.hasSentKnowledgeStart {
		return
	}
	c.emit(aguievents.NewCustomEvent(EventNameRagKnowledgeStart,
		aguievents.WithValue(json.RawMessage("null"))))
	c.hasSentKnowledgeStart = true
}

// emitQAStartOnce 在首次收到 qa_start 状态帧时发 CUSTOM 事件，通知前端创建"问答库检索"卡片。
func (c *ragStreamConverter) emitQAStartOnce() {
	if c.hasSentQAStart {
		return
	}
	c.emit(aguievents.NewCustomEvent(EventNameRagQAStart,
		aguievents.WithValue(json.RawMessage("null"))))
	c.hasSentQAStart = true
}

// emitQASearchListOnce 在首次收到 qa_finish 时发 QA 搜索列表事件。
// 与 emitSearchListOnce 不同：空数组也发（payload=[]），让前端把"问答库检索"卡片从 running 切到 done（未命中态）。
func (c *ragStreamConverter) emitQASearchListOnce(raw json.RawMessage) {
	if c.hasSentQASearchList {
		return
	}
	// 非空时复用 KB 端的富化逻辑（补 user_kb_name）；空/解析失败回落为空数组。
	payload := enrichSearchListWithUserKbName(raw, c.kbNameMap)
	c.emit(aguievents.NewCustomEvent(EventNameRagQASearchList,
		aguievents.WithValue(payload)))
	c.hasSentQASearchList = true
}

// emitSearchListOnce 在首次收到非空 searchList 时发 CUSTOM 事件。
// 用 raw JSON 长度快速过滤空数组（"[]" 只有 2 字节），避免反序列化两次。
func (c *ragStreamConverter) emitSearchListOnce(raw json.RawMessage) {
	if c.hasSentSearchList || len(raw) <= 2 {
		return
	}
	payload := enrichSearchListWithUserKbName(raw, c.kbNameMap)
	if len(payload) == 0 {
		return
	}
	c.emit(aguievents.NewCustomEvent(EventNameRagSearchList,
		aguievents.WithValue(payload)))
	c.hasSentSearchList = true
}

// emitReasoning 发推理内容事件（若非空）。
func (c *ragStreamConverter) emitReasoning(reasoning string) {
	if reasoning == "" {
		return
	}
	c.recordTTFT()
	c.emit(c.state.StartReasoningMessage()...)
	c.emit(aguievents.NewReasoningMessageContentEvent(
		c.state.ReasoningMessageID(), reasoning))
}

// emitOutput 发正文内容事件（若非空）；首次 output 到达即视为 reasoning 阶段结束。
func (c *ragStreamConverter) emitOutput(output string) {
	if output == "" {
		return
	}
	c.recordTTFT()
	c.emit(c.state.EndReasoningMessage()...)
	c.emit(c.state.StartTextMessage()...)
	c.emit(aguievents.NewTextMessageContentEvent(
		c.state.MessageID(), output))
}

// parseChunkLine 解析一行 SSE 文本为 ragChunkData；不合法或空行返回 ok=false。
// 额外识别 rag-service 的裸 `error:` 前缀行（见 rag_manage_sevice.go 里
// requestRagStreamChat 的错误返回格式），合成一个 business-error chunk
// 交给 handleChunk 走统一的 RUN_ERROR 路径。
func parseChunkLine(line string) (ragChunkData, bool) {
	line = strings.TrimPrefix(line, "data:")
	line = strings.TrimSpace(line)
	if line == "" {
		return ragChunkData{}, false
	}
	if strings.HasPrefix(line, "error:") {
		return ragChunkData{
			Code:    ragChunkCodeBusinessError,
			Message: strings.TrimSpace(strings.TrimPrefix(line, "error:")),
		}, true
	}
	var chunk ragChunkData
	if err := json.Unmarshal([]byte(line), &chunk); err != nil {
		return ragChunkData{}, false
	}
	return chunk, true
}

// convertRagStream2AGUIEvents 将 RAG 原始 SSE channel 转换为 AG-UI 事件 channel。
// 详细转换规则见 ragStreamConverter 文档注释。
func convertRagStream2AGUIEvents(
	ctx context.Context,
	chatCh <-chan string,
	threadID, runID string,
	streamParams *ragChatStreamParams,
	kbNameMap map[string]string,
) <-chan aguievents.Event {
	out := make(chan aguievents.Event, 64)
	// NewBaseState 返回值类型，方法接收者是 *BaseState；取地址避免方法调用时拷贝状态
	state := ag_ui_util.NewBaseState(threadID, runID)
	c := &ragStreamConverter{
		ctx:          ctx,
		out:          out,
		state:        &state,
		runID:        runID,
		streamParams: streamParams,
		kbNameMap:    kbNameMap,
	}

	go func() {
		defer util.PrintPanicStack()
		defer close(out)

		// 首条：RUN_STARTED
		if !c.emit(c.state.EnsureRunStarted()...) {
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			case line, ok := <-chatCh:
				if !ok {
					// 上游 channel 关闭。分两种情况：
					//  - 已 finalize（正常 RUN_FINISHED 或已发过 RUN_ERROR）：直接退出
					//  - 未 finalize（上游异常断流，例如 rag-wanwu 遇到模型不可用
					//    但只关流没发错误帧）：兜底发 RUN_ERROR，避免前端卡 loading
					if !c.hasFinalized {
						c.streamParams.hasErr = true
						c.finalizeError(RagErrCodeUnknown, "upstream stream closed without finish")
					}
					return
				}
				chunk, ok := parseChunkLine(line)
				if !ok {
					continue
				}
				if c.handleChunk(chunk) {
					return
				}
			}
		}
	}()

	return out
}

// ChatRagStreamLegacy 旧版 RAG 流式接口（原样透传 rag-service SSE JSON）。
//
// 历史背景：新分支把 web 端 RAG 流式响应迁移到 AG-UI 协议（ChatRagStream），
// 但 /openapi/rag/chat 是对外暴露给第三方的 OpenAPI，已有外部集成方按旧格式
// 解析 `data: {"code":0,"msg_id":...,"data":{"output":"...","searchList":[...]},"finish":0|1}`，
// 不能随 web 一起改协议。故保留这份旧实现专供 openapi 使用：
//   - web / 草稿预览  → ChatRagStream（AG-UI 事件流）
//   - openapi         → ChatRagStreamLegacy（原始 SSE JSON 透传）
//
// 两者共用同一个底层 CallRagChatStream，仅输出层不同。
func ChatRagStreamLegacy(ctx *gin.Context, userId, orgId string, req request.ChatRagRequest, needLatestPublished bool, source string) (err error) {
	streamParams := &ragChatStreamParams{ctx: ctx, startTime: time.Now()}
	defer func() {
		if source != constant.AppStatisticSourceDraft {
			RecordAppStatistic(ctx.Request.Context(), userId, orgId, req.RagID, constant.AppTypeRag, !streamParams.hasErr, true, streamParams.firstTokenLatency, 0, source)
		}
	}()

	// openapi 不需要 kbNameMap（旧格式没有 user_kb_name 字段），忽略第二个返回值
	chatCh, _, err := CallRagChatStream(ctx, userId, orgId, req, needLatestPublished)
	if err != nil {
		streamParams.hasErr = true
		return err
	}
	// 旧版行处理器：带 data: 前缀的原样透传，error: 开头的转成 {code:-1,...}
	_ = sse_util.NewSSEWriter(ctx, fmt.Sprintf("[RAG] %v user %v org %v", req.RagID, userId, orgId), sse_util.DONE_MSG).
		WriteStream(chatCh, streamParams, buildRagChatRespLineProcessorLegacy(), nil)
	return nil
}

// buildRagChatRespLineProcessorLegacy 旧版 RAG 行处理器（仅 ChatRagStreamLegacy 使用）。
// 用于保证 openapi 输出格式向后兼容
func buildRagChatRespLineProcessorLegacy() func(sse_util.SSEWriterClient[string], string, interface{}) (string, bool, error) {
	return func(c sse_util.SSEWriterClient[string], lineText string, params interface{}) (string, bool, error) {
		if p, ok := params.(*ragChatStreamParams); ok {
			if !p.hasRecorded {
				p.firstTokenLatency = time.Since(p.startTime).Milliseconds()
				p.hasRecorded = true
				if p.ctx != nil {
					p.ctx.Set(gin_util.FIRST_RESP_LATENCY, p.firstTokenLatency)
				}
			}
		}
		if strings.HasPrefix(lineText, "error:") {
			if p, ok := params.(*ragChatStreamParams); ok {
				p.hasErr = true
			}
			errorText := fmt.Sprintf("data: {\"code\": -1, \"message\": \"%s\"}\n\n", strings.TrimPrefix(lineText, "error:"))
			return errorText, false, nil
		}
		if strings.HasPrefix(lineText, "data:") {
			return lineText + "\n\n", false, nil
		}
		return lineText + "\n\n", false, nil
	}
}

// CallRagChatStream 调用 Rag 对话，返回经敏感词处理后的原始 SSE 字符串 channel。
// 第二个返回值 kbNameMap 是 rag 内部 kb_name → 用户可见知识库名的映射，
// 供上层在透传 searchList 前为每个引用段落补填 user_kb_name。
func CallRagChatStream(ctx *gin.Context, userId, orgId string, req request.ChatRagRequest, needLatestPublished bool) (<-chan string, map[string]string, error) {
	// 根据 ragID 获取敏感词配置
	ragInfo, err := rag.GetRagDetail(ctx, &rag_service.RagDetailReq{
		RagId:   req.RagID,
		Publish: util.IfElse(needLatestPublished, int32(1), int32(0)),
	})
	if err != nil {
		return nil, nil, err
	}
	// 构造 kb_name → user_kb_name 映射（失败时仅退化为空 map，不中断对话）
	kbNameMap := buildRagKbNameMap(ctx, userId, ragInfo)

	var matchDicts []ahocorasick.DictConfig
	// 如果 Enable 为 true，则处理敏感词
	matchDicts, err = BuildSensitiveDict(ctx, ragInfo.SensitiveConfig.GetTableIds(), ragInfo.SensitiveConfig.GetEnable())
	if err != nil {
		return nil, nil, err
	}
	matchResults, err := ahocorasick.ContentMatch(req.Question, matchDicts, true)
	if err != nil {
		return nil, nil, grpc_util.ErrorStatus(err_code.Code_BFFSensitiveWordCheck, err.Error())
	}
	if len(matchResults) > 0 {
		if matchResults[0].Reply != "" {
			return nil, nil, grpc_util.ErrorStatusWithKey(err_code.Code_BFFSensitiveWordCheck, "bff_sensitive_check_req", matchResults[0].Reply)
		}
		return nil, nil, grpc_util.ErrorStatusWithKey(err_code.Code_BFFSensitiveWordCheck, "bff_sensitive_check_req_default_reply")
	}

	var ragHistory []*rag_service.HistoryItem
	if len(req.History) > 0 {
		for _, history := range req.History {
			ragHistory = append(ragHistory, &rag_service.HistoryItem{
				Query:       history.Query,
				Response:    history.Response,
				NeedHistory: history.NeedHistory,
			})
		}
	}
	stream, err := rag.ChatRag(ctx, &rag_service.ChatRagReq{
		RagId:    req.RagID,
		Question: req.Question,
		History:  ragHistory,
		Identity: &rag_service.Identity{
			UserId: userId,
			OrgId:  orgId,
		},
		Publish:      util.IfElse(needLatestPublished, int32(1), int32(0)),
		FileInfoList: buildRagFileInfoList(req.FileInfo),
	})
	if err != nil {
		return nil, nil, err
	}

	// 读取 gRPC 流内容到 channel
	SSEReader := &sse_util.SSEReader[rag_service.ChatRagResp]{
		BusinessKey:    "chat_rag",
		StreamReceiver: sse_util.NewGrpcStreamReceiver(stream),
	}
	rawCh, err := SSEReader.ReadStreamWithBuilder(ctx, func(resp *rag_service.ChatRagResp) string {
		return resp.Content
	})
	if err != nil {
		return nil, nil, err
	}
	// 敏感词过滤（必须过滤，全局敏感词）
	retCh := ProcessSensitiveWords(ctx, rawCh, matchDicts, &ragSensitiveService{})
	return retCh, kbNameMap, nil
}

// buildRagKbNameMap 根据 RAG 应用的知识库/问答库绑定，查出每个知识库的用户可见名，
// 返回 rag 内部 kb_name（即 KnowledgeInfo.RagName）→ 用户可见名（KnowledgeInfo.Name）的映射。
// 任何上游错误都降级为空 map，不阻断对话主流程。
func buildRagKbNameMap(ctx *gin.Context, userId string, ragInfo *rag_service.RagInfo) map[string]string {
	if ragInfo == nil {
		return map[string]string{}
	}
	idSet := make(map[string]struct{})
	if kbCfg := ragInfo.GetKnowledgeBaseConfig(); kbCfg != nil {
		for _, per := range kbCfg.GetPerKnowledgeConfigs() {
			if id := per.GetKnowledgeId(); id != "" {
				idSet[id] = struct{}{}
			}
		}
	}
	if qaCfg := ragInfo.GetQAknowledgeBaseConfig(); qaCfg != nil {
		for _, per := range qaCfg.GetPerKnowledgeConfigs() {
			if id := per.GetKnowledgeId(); id != "" {
				idSet[id] = struct{}{}
			}
		}
	}
	if len(idSet) == 0 {
		return map[string]string{}
	}
	idList := make([]string, 0, len(idSet))
	for id := range idSet {
		idList = append(idList, id)
	}
	list, err := selectKnowledgeListByIdList(ctx, &request.KnowledgeBatchSelectReq{
		UserId:          userId,
		KnowledgeIdList: idList,
	})
	if err != nil || list == nil {
		return map[string]string{}
	}
	nameMap := make(map[string]string, len(list.KnowledgeList))
	for _, kb := range list.KnowledgeList {
		if kb == nil || kb.RagName == "" {
			continue
		}
		nameMap[kb.RagName] = kb.Name
	}
	return nameMap
}

// enrichSearchListWithUserKbName 在透传上游 searchList 前，为每个引用段落补填 user_kb_name 字段。
// 实现策略：解析为松散的 []map[string]interface{}，按每项的 kb_name 查 nameMap 写入 user_kb_name。
// 保持其他字段原样透传；解析失败时回退为原始 RawMessage，保证至少不破坏下游渲染。
func enrichSearchListWithUserKbName(raw json.RawMessage, nameMap map[string]string) []map[string]interface{} {
	items := []map[string]interface{}{}
	if len(raw) == 0 {
		return items
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return items
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		// 已有 user_kb_name（上游未来可能直接返回）则不覆盖
		if existing, ok := item["user_kb_name"].(string); ok && existing != "" {
			continue
		}
		kbName, _ := item["kb_name"].(string)
		if kbName == "" {
			// QA 条目没有 kb_name，用 QABase 作 key
			kbName, _ = item["QABase"].(string)
		}
		if kbName == "" {
			continue
		}
		if display, ok := nameMap[kbName]; ok && display != "" {
			item["user_kb_name"] = display
		} else {
			// 兜底：即便映射缺失也让前端能显示一个名字，而不是空白
			item["user_kb_name"] = kbName
		}
	}
	// 老版本 rag-wanwu（如 64 服务器）顶层不返回 score，只在 rerank_info[0].score 里有；
	// 为让前端 Score 徽章始终能显示，顶层缺失时从 rerank_info 抽上来。
	// TODO: 64 服务器 rag-wanwu 后续会更新，届时确认顶层已有 score 后删除此段。
	// for _, item := range items {
	// 	if item == nil {
	// 		continue
	// 	}
	// 	if _, ok := item["score"].(float64); ok {
	// 		continue
	// 	}
	// 	rerank, ok := item["rerank_info"].([]interface{})
	// 	if !ok || len(rerank) == 0 {
	// 		continue
	// 	}
	// 	first, ok := rerank[0].(map[string]interface{})
	// 	if !ok {
	// 		continue
	// 	}
	// 	if s, ok := first["score"].(float64); ok {
	// 		item["score"] = s
	// 	}
	// }
	return items
}

func buildRagFileInfoList(fileInfoList []request.ConversionStreamFile) []*rag_service.FileInfo {
	retList := make([]*rag_service.FileInfo, 0)
	if len(fileInfoList) > 0 {
		for _, fileInfo := range fileInfoList {
			retList = append(retList, &rag_service.FileInfo{
				FileName: fileInfo.FileName,
				FileSize: fileInfo.FileSize,
				FileUrl:  fileInfo.FileUrl,
			})
		}
	}
	return retList
}

// --- ragSensitiveService: 实现 sensitiveService 接口，供 ProcessSensitiveWords 使用 ---

type ragSensitiveService struct{}

func (s *ragSensitiveService) serviceType() string {
	return constant.AppTypeRag
}

func (s *ragSensitiveService) parseContent(raw string) (id, content string) {
	// 1. 清理数据前缀
	raw = strings.TrimPrefix(raw, "data:")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", ""
	}
	// 2. 解析 JSON
	resp := struct {
		MsgID string `json:"msg_id"`
		Data  struct {
			Output string `json:"output"`
		} `json:"data"`
	}{}

	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return "", ""
	}
	// 3. 返回 content
	return resp.MsgID, resp.Data.Output
}

func (s *ragSensitiveService) buildSensitiveResp(id string, content string) []string {
	resp := map[string]interface{}{
		"code":    0,
		"message": "success",
		"msg_id":  id,
		"data": map[string]interface{}{
			"output":     content,
			"searchList": []interface{}{},
		},
		"history": []interface{}{},
		"finish":  1,
	}
	marshal, _ := json.Marshal(resp)
	return []string{"data: " + string(marshal)}
}
