package sse_connector

import (
	"context"
	"errors"
	"github.com/UnicomAI/wanwu/pkg/log"
	safe_go_util "github.com/UnicomAI/wanwu/pkg/safe-go-util"
	"github.com/UnicomAI/wanwu/pkg/sse-util/sse-connector/model"
	"github.com/UnicomAI/wanwu/pkg/sse-util/sse-connector/session"
	"github.com/UnicomAI/wanwu/pkg/sse-util/sse-connector/store"
	"strings"
	"time"
)

const (
	SessionMaxTime = 30 * time.Minute
)

type SSEConnector struct {
	sessionMgr map[string]*session.Manager
}

var Connector = &SSEConnector{
	sessionMgr: make(map[string]*session.Manager),
}

func NewSSESession(ctx context.Context, userSession *model.Session, s store.MessageStore) *session.Manager {
	if !userSession.Check() {
		log.Infof("invalid clientID：%s sessionID: %s", userSession.ClientID, userSession.ConversationID)
		return &session.Manager{Invalid: true, Ctx: ctx}
	}
	// 检查会话是否已经存在,如果已经存在，则取消旧会话
	existManager := GetSession(userSession)
	if existManager != nil {
		_ = existManager.Cancel()
		log.Errorf("session already exists session: %s", userSession.SessionID())
	}
	manager := session.NewManager(ctx, s, userSession, func(sessionId string) {
		delete(Connector.sessionMgr, sessionId)
	})
	Connector.sessionMgr[userSession.SessionID()] = manager
	go DelayClose(userSession, SessionMaxTime)
	return manager
}

func GetSession(userSession *model.Session) *session.Manager {
	manager := Connector.sessionMgr[userSession.SessionID()]
	if manager == nil || manager.Invalid {
		return nil
	}
	return manager
}

func Connect[T any](ctx context.Context, userSession *model.Session,
	lineBuilder func(data *model.Message) T) (<-chan T, error) {
	if lineBuilder == nil {
		return nil, errors.New("line builder is nil")
	}
	manager := Connector.sessionMgr[userSession.SessionID()]
	if manager == nil {
		return nil, errors.New("session not found")
	}
	subscriber := manager.Subscribe()
	if subscriber == nil {
		return nil, errors.New("session not subscribed")
	}

	rawCh := make(chan T, 128)

	safe_go_util.SafeGo(func() {
		defer manager.Unsubscribe()
		defer func() {
			close(rawCh)
		}()
		var lastMessageId string
		// 发送历史消息
		history, err := manager.GetHistory()
		if err != nil {
			log.Errorf("get history message error: %v", err)
		}

		// 先发送历史消息
		if len(history) > 0 {
			for _, msg := range history {
				rawCh <- lineBuilder(msg)
				lastMessageId = msg.ID
			}
		}

		// 持续监听新消息
		for {
			select {
			case msg, ok := <-subscriber.Chan:
				if !ok {
					return
				}
				if strings.Compare(msg.ID, lastMessageId) > 0 {
					rawCh <- lineBuilder(msg)
				}
			case <-ctx.Done():
				// 客户端断开连接
				return
			}
		}
	})

	return rawCh, nil
}

func Close(userSession *model.Session) error {
	manager := Connector.sessionMgr[userSession.SessionID()]
	if manager == nil {
		return nil
	}
	return manager.Cancel()
}

// DelayClose 延迟清理数据
func DelayClose(userSession *model.Session, delay time.Duration) {
	time.Sleep(delay)
	err := Close(userSession)
	if err != nil {
		log.Errorf("finish error: %v", err)
	}
}
