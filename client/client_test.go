package client

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"
)

// switch toc {
// case 'r':
//
//	// authentication response
//
// case 'm':
//
//	// hash salt
//
// case 'i':
//
//	// node info
//
// case 'h':
//
//	// health check
//
// case 'l':
//
//	// node count
//
// case 'z':
// case 'a':
//
//	// command complete
//	// command complete
//
// case 'w':
//
//	// watchdog info
//
// case 'p':
//
//	// process info
//
// case 'n':
//
//	// process count
//
// case 'b':
//
//	// pool status
//
// case 't':
//
//		c.response.status = COMMAND_OK
//	}

const (
	password = "pass"
	username = "pass"
)

type dummyConn struct{}

func (c *dummyConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (c *dummyConn) Write(b []byte) (n int, err error) {
	return 0, nil
}

func (c *dummyConn) Close() error {
	return nil
}

// LocalAddr returns the local network address, if known.
func (c *dummyConn) LocalAddr() net.Addr {
	panic("not implemented") // TODO: Implement
}

// RemoteAddr returns the remote network address, if known.
func (c *dummyConn) RemoteAddr() net.Addr {
	panic("not implemented") // TODO: Implement
}

func (c *dummyConn) SetDeadline(t time.Time) error {
	panic("not implemented") // TODO: Implement
}

func (c *dummyConn) SetReadDeadline(t time.Time) error {
	panic("not implemented") // TODO: Implement
}

func (c *dummyConn) SetWriteDeadline(t time.Time) error {
	panic("not implemented") // TODO: Implement
}

func TestClientFailWhenCtxBad(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := NewClient(ctx, "localhost:9090", "user", "pass"); err == nil {
		t.Fail()
	}
}

func makeClient(t *testing.T, size int) *Client {
	t.Helper()
	client, err := clientWithConn(&dummyConn{}, size, username, password)
	if err != nil {
		t.Fail()
	}
	client.response = makeResult(size)
	return client
}

func TestClientAuthorization(t *testing.T) {
	client := makeClient(t, 5)
	if err := client.Authorize(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(client.bytes(), []byte{'M', 0, 0, 0, 4}) {
		t.Fail()
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
}
