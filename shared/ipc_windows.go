package shared

import (
	"net"
	"time"

	winio "github.com/Microsoft/go-winio"
)

func listenIPC(path string) (net.Listener, error) {
	return winio.ListenPipe(path, nil)
}

func dialIPC(path string, timeout time.Duration) (net.Conn, error) {
	return winio.DialPipe(path, &timeout)
}
