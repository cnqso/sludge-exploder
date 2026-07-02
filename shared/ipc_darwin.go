package shared

import (
	"net"
	"time"
)

func listenIPC(path string) (net.Listener, error) {
	return net.Listen("unix", path)
}

func dialIPC(path string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", path, timeout)
}
