//go:build darwin || openbsd || freebsd
package ddio

import (
	"time"
)

type poller struct {
	*kqueue
}

func NewPoller() (*poller,error) {
	return nil,nil
}

func (p poller) Exec(maxEvent int, timeOut time.Duration) ([]Event, error) {
	panic("implement me")
}

func (p poller) Exit() error {
	panic("implement me")
}

func (p poller) With(event *Event) error {
	panic("implement me")
}

func (p poller) Modify(event *Event) error {
	panic("implement me")
}

func (p poller) Cancel(event *Event) error {
	panic("implement me")
}

func (p poller) AllEvents() []Event {
	panic("implement me")
}

