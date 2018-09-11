// +build linux darwin solaris

package beacon

import (
	"net"
	"time"

	"github.com/felixge/tcpkeepalive"
	"fmt"
)

type Conn struct {
	net.Conn
	IdleTimeout time.Duration
}

func (c *Conn) Write(p []byte) (int, error) {
	c.updateDeadline()
	return c.Conn.Write(p)
}

func (c *Conn) Read(b []byte) (int, error) {
	c.updateDeadline()
	return c.Conn.Read(b)
}

func (c *Conn) updateDeadline() {
	idleDeadline := time.Now().Add(c.IdleTimeout)
	c.Conn.SetDeadline(idleDeadline)
}

func keepaliveDialer(network string, address string, timeout time.Duration) (net.Conn, error) {
	fmt.Println("keepaliveDialer was called")
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return nil, err
	}

	connWithIotimeout := &Conn{
		Conn:        conn,
		IdleTimeout: 1 * time.Minute,
	}

	err = tcpkeepalive.SetKeepAlive(connWithIotimeout, 10*time.Second, 3, 5*time.Second)
	if err != nil {
		println("failed to enable connection keepalive: " + err.Error())
	}

	return connWithIotimeout, nil
}
