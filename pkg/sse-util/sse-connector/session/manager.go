package session

import (
	"context"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/sse-util/sse-connector/model"
	"github.com/UnicomAI/wanwu/pkg/sse-util/sse-connector/store"
	trace_util "github.com/UnicomAI/wanwu/pkg/trace-util"
	"github.com/UnicomAI/wanwu/pkg/util"
	"sync"
	"time"
)

// Subscriber 订阅者，代表一个 SSE 连接
type Subscriber struct {
	Chan chan *model.Message
}

// Manager 会话管理器
type Manager struct {
	Ctx         context.Context
	cancel      context.CancelFunc
	Invalid     bool //会话失效，目前是clientID 或者 conversationID 为空,为了简化接入流程，所以参数不符合预期目前不报错，只设置invalid为true
	store       store.MessageStore
	mu          sync.RWMutex
	subscriber  *Subscriber
	userSession *model.Session
	writeDone   bool //是否已写完
	callback    func(sessionId string)
}

func NewManager(ctx context.Context, s store.MessageStore, userSession *model.Session, callback func(sessionId string)) *Manager {
	detachContext := trace_util.DetachContext(ctx)
	ctx, cancel := context.WithCancel(detachContext)
	return &Manager{
		Ctx:         ctx,
		cancel:      cancel,
		store:       s,
		userSession: userSession,
		callback:    callback,
	}
}

func (m *Manager) GetBgContext() context.Context {
	return m.Ctx
}

// AddExt 添加扩展信息
func (m *Manager) AddExt(extMap map[string]interface{}) {
	if m.Invalid {
		return
	}
	_ = m.store.AddExtMessage(extMap, m.userSession)
}

// GetExt 查询扩展信息
func (m *Manager) GetExt() map[string]interface{} {
	if m.Invalid {
		return nil
	}
	return m.store.GetExtMessage(m.userSession)
}

func (m *Manager) InvalidManager() {
	m.Invalid = true
}

// Subscribe 订阅会话消息
func (m *Manager) Subscribe() *Subscriber {
	if m.Invalid {
		return nil
	}
	if m.writeDone {
		return nil
	}
	sub := &Subscriber{Chan: make(chan *model.Message, 128)}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.subscriber = sub

	//防止用户误操作的情况，内存占用死掉了，15min强制取消订阅
	go m.DelayUnsubscribe(15 * time.Minute)
	return sub
}

// Unsubscribe 取消订阅
func (m *Manager) Unsubscribe() {
	if m.Invalid {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	subscriber := m.subscriber
	if subscriber == nil {
		return
	}
	if subscriber.Chan != nil {
		close(subscriber.Chan)
		m.subscriber = nil
	}
}

// DelayUnsubscribe 延迟清理会话
func (m *Manager) DelayUnsubscribe(delay time.Duration) {
	time.Sleep(delay)
	m.Unsubscribe()
}

// Publish 发布消息给所有订阅者
func (m *Manager) Publish(msg *model.Message, compactProcessor func(currentMsg *model.Message, lastMsg *model.Message) (bool, *model.Message)) error {
	if m.Invalid {
		return nil
	}
	//生成消息ID,保证id 递增
	msg.ID = util.NewID()

	if compactProcessor != nil { //处理消息合并
		lastMsg, err := m.store.GetCurrentMessage(m.userSession)
		if err != nil {
			return err
		}
		var skip = true
		var compactMsg *model.Message
		if lastMsg != nil {
			skip, compactMsg = compactProcessor(msg, lastMsg)
		}
		if skip { // 存储消息
			err = m.store.AddMessage(msg, m.userSession)
			if err != nil {
				return err
			}
		} else {
			err = m.store.CompactMessage(compactMsg, m.userSession)
			if err != nil {
				return err
			}
		}
	} else { // 存储消息
		err := m.store.AddMessage(msg, m.userSession)
		if err != nil {
			return err
		}
	}

	//发送消息
	m.mu.RLock()
	subscriber := m.subscriber
	m.mu.RUnlock()
	if subscriber != nil {
		//发送消息
		select {
		case subscriber.Chan <- msg:
		default:
			log.Warnf("Publish %s subscriber channel full, dropping message", m.userSession.SessionID())
		}
	}
	return nil
}

// Cancel 写入消息取消
func (m *Manager) Cancel() error {
	if m.cancel != nil {
		m.cancel()
	}
	return m.Finish()
}

// Finish 写入消息完成
func (m *Manager) Finish() error {
	if m.Invalid {
		log.Errorf("session %s is invalid", m.userSession.SessionID())
	}
	m.mu.RLock()
	defer func() {
		m.mu.RUnlock()
		if m.callback != nil {
			m.callback(m.userSession.SessionID())
		}
	}()
	m.writeDone = true
	return m.store.DeleteSession(m.userSession)
}

// GetHistory 获取历史消息
func (m *Manager) GetHistory() ([]*model.Message, error) {
	if m.Invalid {
		return make([]*model.Message, 0), nil
	}
	return m.store.GetMessages(m.userSession)
}
