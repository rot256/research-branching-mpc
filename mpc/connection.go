package main

import (
	"encoding/gob"
	"io"
	"net"
	"os"
	"syscall"
)

type Connection struct {
	enc *gob.Encoder
	dec *gob.Decoder
}

func DummyPair() (*Connection, *Connection) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}

	f1 := os.NewFile(uintptr(fds[1]), "c1")
	c1, err := net.FileConn(f1)
	if err != nil {
		panic(err)
	}

	f2 := os.NewFile(uintptr(fds[0]), "c2")
	c2, err := net.FileConn(f2)
	if err != nil {
		panic(err)
	}
	syscall.SetNonblock(fds[0], false)

	return NewConnection(c1), NewConnection(c2)
}

func StarDummies(players int) [][]*Connection {
	// create multi-dimensional array of connections
	conns := make([][]*Connection, 0)
	for p1 := 0; p1 < players; p1++ {
		p1_conns := make([]*Connection, players)
		conns = append(conns, p1_conns)
	}

	// fill with dummies
	for p1 := 0; p1 < players; p1++ {
		for p2 := p1 + 1; p2 < players; p2++ {
			conns[p1][p2] = &Connection{nil, nil}
			conns[p2][p1] = &Connection{nil, nil}
		}
	}

	// create pair-wise connections
	for p1 := 1; p1 < players; p1++ {
		c1, c2 := DummyPair()
		conns[0][p1] = c1
		conns[p1][0] = c2
	}
	return conns
}

func NDummies(players int) [][]*Connection {
	// create multi-dimensional array of connections
	conns := make([][]*Connection, 0)
	for p1 := 0; p1 < players; p1++ {
		p1_conns := make([]*Connection, players)
		conns = append(conns, p1_conns)
	}

	// create pair-wise connections
	for p1 := 0; p1 < players; p1++ {
		for p2 := p1 + 1; p2 < players; p2++ {
			// create connection
			c1, c2 := DummyPair()
			conns[p1][p2] = c1
			conns[p2][p1] = c2
		}
	}
	return conns
}

func (c *Connection) Send(v interface{}) error {
	return c.enc.Encode(v)
}

func (c *Connection) Recv(v interface{}) error {
	return c.dec.Decode(v)
}

func NewConnection(conn io.ReadWriter) *Connection {
	return &Connection{
		enc: gob.NewEncoder(conn),
		dec: gob.NewDecoder(conn),
	}
}
