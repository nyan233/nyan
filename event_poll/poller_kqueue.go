//go:build darwin || openbsd || freebsd
package ddio

type poller struct {
	*kqueue
}

func (p poller) With(event *ConnectionEvent) error {
	panic("implement me")
}

func (p poller) Modify(event *ConnectionEvent) error {
	panic("implement me")
}

func (p poller) Cancel(event *ConnectionEvent) error {
	panic("implement me")
}
