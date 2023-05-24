package client

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os/user"
	"unicode"

	"github.com/davecgh/go-spew/spew"
)

const MAX_BUFSZ = 256

type Client struct {
	conn      net.Conn
	status    ConnState
	buf       []byte
	wr_idx    int
	Authorize func() error
	response  *result
}

type ConnState uint8

const (
	OK ConnState = iota + 1
	CONNECTED
	NOT_CONNECTED
	BAD
	AUTH_ERROR
)

type ResultState uint8

const (
	COMMAND_OK ResultState = iota + 1
	COMMAND_COMPLETE
	BAD_RESPONSE
	BACKEND_ERROR
	INCOMPLETE
	ERROR
)

//go:generate stringer -type=ServerRole -output=string.go
type ServerRole uint8

const (
	main ServerRole = iota + 1
	replica
	primary
	standby
)

func NewClient(ctx context.Context, addr, username, password string) (c *Client, err error) {
	// if supplied pcp user is nil, we try to use the default user on the OS
	if username == "" {
		user, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("failed to get default user")
		}
		username = user.Name
	}

	dialer := new(net.Dialer)
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	if c, err = clientWithConn(conn, MAX_BUFSZ, username, password); err != nil {
		return nil, err
	}
	c.response = makeResult(MAX_BUFSZ)
	return c, nil
}

// create a new connection to a pcp backend
func clientWithConn(conn net.Conn, bufsz int, username, password string) (*Client, error) {
	client := &Client{
		conn:   conn,
		status: NOT_CONNECTED,
		buf:    make([]byte, bufsz),
	}
	client.Authorize = client.authorizer(username, password)
	return client, nil
}

func makeResult(size int) *result {
	return &result{
		buf:    make([]byte, size),
		status: INCOMPLETE,
	}
}

// authorizer returns a function that takes care of pcp authorization request
func (c *Client) authorizer(username, password string) func() error {
	return func() (err error) {
		if c.status == CONNECTED {
			return nil
		}

		c.writeByte('M')
		c.wsize(4)
		if err = c.Flush(); err != nil {
			return err
		}

		var res *result
		if res, err = c.responseOf('M'); err != nil {
			return err
		}

		hexPass := encrypt(res.data(), username, password)

		c.writeByte('R')
		c.wsize(len(username) + len(hexPass) + 6)
		c.Write(addNull(username))
		c.Write(addNull(hexPass))
		if err := c.Flush(); err != nil {
			return err
		}
		if res, err = c.responseOf('R'); err != nil {
			return err
		}

		if !res.OK() {
			return c.response.processError()
		}
		c.status = CONNECTED
		return nil
	}
}

func (c *Client) Status() ConnState {
	return c.status
}

func addNull(s string) []byte {
	return append([]byte(s), 0)
}

func encrypt(salt []byte, username, password string) string {
	pass := hash(password)
	crypt := hash(pass + username)
	return hash(crypt + string(salt))
}

func hash(thing string) string {
	t := md5.Sum([]byte(thing))
	return hex.EncodeToString(t[:])
}

// debug shit
func log(v ...any) {
	spew.Dump(v...)
}

// response commands that have their requests as upper their upper case counter parts
var reqWithUpperRes = []byte{'R', 'M', 'I', 'H', 'L', 'W', 'P', 'N', 'B', 'T'}

// get the appropriate response to request command `typ` from the pcp backend
func (c *Client) responseOf(typ byte) (*result, error) {
	n, err := c.conn.Read(c.response.buf)
	if err != nil {
		return nil, err
	}
	log(c.response.buf[:n])
	c.response.end = n
	toc := rune(c.response.readByte())
	if bytes.ContainsRune(reqWithUpperRes, rune(typ)) && unicode.ToUpper(toc) == rune(typ) {
		c.response.status = COMMAND_OK
		return c.response, nil
	} else if toc == 'E' || toc == 'N' {
		// backend error response
		c.response.status = BACKEND_ERROR
		return c.response, nil
	} else if toc == 'c' || toc == 'd' || toc == 'a' || toc == 'z' {
		// command complete shit
		// TODO(joe): there are probably other things to be done over here
		c.response.status = COMMAND_COMPLETE
		return c.response, nil
	}
	c.response.status = BAD_RESPONSE
	return nil, ErrResponse
}

var ErrNoSpace = errors.New("no space left")

func (c *Client) writeByte(b byte) error {
	if c.wr_idx >= len(c.buf) {
		return ErrNoSpace
	}
	c.buf[c.wr_idx] = b
	c.wr_idx++
	return nil
}

// Write into the connection buffer but not the connection itself.
// use c.Flush to write data to the connection
func (c *Client) Write(p []byte) (int, error) {
	if len(c.buf[c.wr_idx:]) < len(p) {
		return 0, ErrNoSpace
	}

	for ii, b := range p {
		c.buf[c.wr_idx+ii] = b
	}

	c.wr_idx += len(p)
	return len(p), nil
}

// add size of message to packet
func (c *Client) wsize(num int) error {
	if c.wr_idx+3 >= len(c.buf) {
		return ErrNoSpace
	}
	binary.BigEndian.PutUint32(c.buf[c.wr_idx:], uint32(num))
	c.wr_idx += 4
	return nil
}

func (c *Client) bytes() []byte {
	return c.buf[:c.wr_idx]
}

// write all the data in the buffer into connection stream
func (c *Client) Flush() error {
	log(c.bytes())
	if _, err := c.conn.Write(c.bytes()); err != nil {
		return err
	}
	// reset write buffer
	c.wr_idx = 0
	return nil
}

var NotAuthorized = errors.New("client not authorized")

func (c *Client) NodeCount() ([]byte, error) {
	if c.status != CONNECTED {
		return nil, NotAuthorized
	}
	c.writeByte('L')
	c.wsize(4)
	if err := c.Flush(); err != nil {
		return nil, err
	}
	if _, err := c.responseOf('L'); err != nil {
		return nil, err
	}
	count, err := processNodeCount(c.response.data())
	if err != nil {
		return nil, err
	}
	log(count)
	return nil, ErrResponse
}

func (c *Client) Close() error {
	c.conn.Close()
	return nil
}
