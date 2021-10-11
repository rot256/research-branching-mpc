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

func DummyPair() (Connection, Connection) {
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

func NDummies(players int) [][]Connection {
	// create multi-dimensional array of connections
	conns := make([][]Connection, players)
	for p1 := 0; p1 < players; p1++ {
		conns[p1] = make([]Connection, players)
	}

	// create pair-wise connections
	for p1 := 0; p1 < players; p1++ {
		for p2 := 0; p2 < players; p2++ {
			if p1 >= p2 {
				continue
			}

			// create connection
			c1, c2 := DummyPair()
			conns[p1][p2] = c1
			conns[p2][p1] = c2
		}
	}
	return conns
}

func (c *Connection) Encode(v interface{}) error {
	return c.enc.Encode(v)
}

func (c *Connection) Decode(v interface{}) error {
	return c.dec.Decode(v)
}

func NilConnection() Connection {
	return Connection{
		enc: nil,
		dec: nil,
	}
}

func NewConnection(conn io.ReadWriter) Connection {
	return Connection{
		enc: gob.NewEncoder(conn),
		dec: gob.NewDecoder(conn),
	}
}
