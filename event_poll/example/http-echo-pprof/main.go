package http_echo_pprof

import (
	"fmt"
	"github.com/zbh255/bilog"
	ddio "github.com/zbh255/nyan/event_poll"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sync/atomic"
	"time"
)

type SimpleHttpEchoServer struct {
}

func (s *SimpleHttpEchoServer) OnInit() ddio.ConnConfig {
	return ddio.ConnConfig{OnDataNBlock: 1}
}

func (s *SimpleHttpEchoServer) OnData(conn *ddio.TCPConn) error {
	buffer := make([]byte, 0, 256)
	buffer = append(buffer, "HTTP/1.1 200 OK\r\nServer: ddio\r\nContent-Type: text/plain\r\nDate: "...)
	buffer = append(buffer, time.Now().AppendFormat([]byte{}, "Mon, 02 Jan 2006 15:04:05 GMT")...)
	buffer = append(buffer, "\r\nContent-Length: 12\r\n\r\nHello World!"...)
	conn.WriteBytes(buffer)
	return nil
}

func (s *SimpleHttpEchoServer) OnClose(ev ddio.Event) error {
	fmt.Println("connection closed")
	return nil
}

func (s *SimpleHttpEchoServer) OnError(ev ddio.Event, err error) {
	fmt.Println("connection error: ", err)
}

var count int64
var logger = bilog.NewLogger(os.Stdout, bilog.DEBUG, bilog.WithTimes(), bilog.WithCaller(), bilog.WithTopBuffer(2))

type CustomBalanced struct {
}

func (c *CustomBalanced) Name() string {
	return "custom-round"
}

func (c *CustomBalanced) Target(connLen, fd int) int {
	atomic.AddInt64(&count, 1)
	return fd % connLen
}

func main() {
	go func() {
		logger.Debug(http.ListenAndServe("0.0.0.0:9090", nil).Error())
	}()
	config := &ddio.EngineConfig{
		ConnHandler: &SimpleHttpEchoServer{},
		NBalance: func() ddio.Balanced {
			return &CustomBalanced{}
		},
		NetPollConfig: &ddio.NetPollConfig{
			Protocol: 0x1,
			IP:       net.ParseIP("192.168.1.150"),
			Port:     8080,
		},
	}
	_, err := ddio.NewEngine(ddio.NewTCPListener(ddio.EVENT_LISTENER), config)
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			time.Sleep(time.Second * 5)
			logger.Debug(fmt.Sprintf("connection count: %d", atomic.LoadInt64(&count)))
		}
	}()
	select {}
}
