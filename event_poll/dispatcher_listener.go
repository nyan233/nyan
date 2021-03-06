package ddio

import (
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
)

// ListenerMultiEventDispatcher 主多路事件派发器
type ListenerMultiEventDispatcher struct {
	handler ListenerEventHandler
	poll    EventLoop
	// 与监听事件多路事件派发器绑定的连接多路事件派发器
	connMds []*ConnMultiEventDispatcher
	// 关闭标志
	closed uint64
	// 完成通知
	done chan struct{}
	// 一些主多路事件派发器的配置
	config *ListenerConfig
}

func NewListenerMultiEventDispatcher(handler ListenerEventHandler, config *ListenerConfig) (*ListenerMultiEventDispatcher, error) {
	lmed := &ListenerMultiEventDispatcher{}
	// 启动绑定的从多路事件派发器
	nMds := runtime.NumCPU()
	if nMds > MAX_SLAVE_LOOP_SIZE {
		nMds = MAX_SLAVE_LOOP_SIZE
	}
	// 所有子Goroutine共享的Pool
	pool := sync.Pool{
		New: func() interface{} {
			return make([]byte, BUFFER_SIZE)
		},
	}
	connMds := make([]*ConnMultiEventDispatcher, nMds)
	connConfig := config.ConnEHd.OnInit()
	for i := 0; i < len(connMds); i++ {
		tmp, err := NewConnMultiEventDispatcher(config.ConnEHd, connConfig)
		tmp.bufferPool = &pool
		if err != nil {
			return nil, err
		}
		connMds[i] = tmp
	}
	lmed.done = make(chan struct{},1)
	lmed.connMds = connMds
	lmed.handler = handler
	lmed.config = config
	poller, err := NewPoller()
	if err != nil {
		logger.ErrorFromErr(err)
		return nil, err
	}
	lmed.poll = poller
	initEvent, err := lmed.handler.OnInit(config.NetPollConfig)
	if err != nil {
		return nil, err
	}
	err = lmed.poll.With(*initEvent)
	if err != nil {
		return nil, err
	}
	go lmed.openLoop()
	return lmed, nil
}

func (l *ListenerMultiEventDispatcher) Close() error {
	if !atomic.CompareAndSwapUint64(&l.closed, 0, 1) {
		return ErrorEpollClosed
	}
	<-l.done
	// 触发主多路事件派发器的定义的错误回调函数
	// 因为负责监听连接的Fd只有一个，所以直接取就好
	l.handler.OnError(l.poll.AllEvents()[0], ErrorEpollClosed)
	// 关闭所有子事件派发器
	for _, v := range l.connMds {
		v.Close()
	}
	return l.poll.Exit()
}

func (l *ListenerMultiEventDispatcher) openLoop() {
	defer func() {
		l.done <- struct{}{}
	}()
	receiver := make([]Event, 1)
	for {
		if atomic.LoadUint64(&l.closed) == 1 {
			return
		}
		events, err := l.poll.Exec(receiver, EVENT_LOOP_SLEEP)
		if events == 0 {
			continue
		}
		if err != syscall.EAGAIN && err != nil {
			break
		}
		event := receiver[0]
		connFd, err := l.handler.OnAccept(event)
		if err != nil {
			logger.ErrorFromString("accept error: " + err.Error())
			continue
		}
		connEvent := &Event{
			sysFd: int32(connFd),
			event: EVENT_READ | EVENT_CLOSE | EVENT_ERROR,
		}
		n := l.config.Balance.Target(len(l.connMds), connFd)
		err = l.connMds[n].AddConnEvent(connEvent)
		if err != nil {
			logger.ErrorFromString("add connection event error : " + err.Error())
		}
	}
}
