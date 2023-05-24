package client

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strconv"
)

type result struct {
	status ResultState
	buf    []byte
	end    int
}

func (r *result) OK() bool {
	return r.status == COMMAND_OK
}

var ErrResponse = errors.New("got bad result from pcp backend")

// process backend error message TODO(joe)
func (r *result) processError() error {
	if r.status != BACKEND_ERROR {
		return nil
	}
	return ErrResponse
}

// get the data returned in the response
func (r *result) data() []byte {
	if len(r.buf) > 5 && r.end <= len(r.buf) {
		return r.buf[5:r.end]
	}
	return nil
}

func (r *result) readByte() byte {
	if len(r.buf) > 0 {
		return r.buf[0]
	}
	return 0
}

// get the size of the result response data
func (r *result) rsize() int {
	if len(r.buf[1:]) >= 4 {
		return int(binary.BigEndian.Uint32(r.buf[1:4]))
	}
	return 0
}

type nodeCount struct {
	NumNodes int `json:"num_nodes"`
}

// process command 'L'
func processNodeCount(data []byte) (*nodeCount, error) {
	res := bytes.Split(data, []byte{0})
	if !bytes.Equal(res[0], []byte("CommandComplete")) {
		return nil, ErrResponse
	}
	count, err := strconv.Atoi(string(res[1]))
	if err != nil {
		return nil, err
	}
	return &nodeCount{NumNodes: count}, nil
}
