package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
)

func party_address(party int) string {
	return "127.0.0.1:" + strconv.Itoa(party_port(party))
}

func party_port(party int) int {
	return party + 7000
}

type Connection struct {
	conn  *bufio.ReadWriter
	other int
}

func (c *Connection) Write(s []byte) error {
	_, err := c.conn.Write(s)
	if err != nil {
		return err
	}
	return c.conn.Flush()
}

func (c *Connection) Read(dst []byte) error {
	_, err := io.ReadFull(c.conn, dst)
	return err
}

func MeConnection(me int) Connection {
	return Connection{
		conn:  nil,
		other: me,
	}
}

func NewConnection(conn net.Conn, me int) (Connection, error) {
	c := Connection{
		/*
			dec: cbor.NewDecoder(conn),
			enc: cbor.NewEncoder(conn),
				dec:   json.NewDecoder(conn),
				enc:   json.NewEncoder(conn),
		*/
		conn: bufio.NewReadWriter(
			bufio.NewReader(conn),
			bufio.NewWriter(conn),
		),
		other: 0,
	}

	// identity peers
	if err := c.WriteInt(me); err != nil {
		return c, err
	}

	other, err := c.ReadInt()
	if err != nil {
		return c, err
	}
	c.other = other
	return c, nil
}

func MakeConnections(batches, parties, me int) [][]Connection {
	addr := ":" + strconv.Itoa(party_port(me))
	fmt.Println("Making connections:", addr)

	conns := make(chan net.Conn)

	addr_me, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		panic(err)
	}

	// listen for connections
	go func() {
		ls, err := net.ListenTCP("tcp", addr_me)
		if err != nil {
			panic(err)
		}
		for {
			conn, err := ls.AcceptTCP()
			conn.SetReadBuffer(TCP_BUFFER)
			if err != nil {
				panic(err)
			}
			conns <- conn
		}
	}()

	// create TCP connections to higher parties
	go func() {
		for batch := 0; batch < batches; batch++ {
			for party := 0; party < parties; party++ {
				if party > me {
					// I must connect
					addr, err := net.ResolveTCPAddr("tcp", party_address(party))
					if err != nil {
						panic(err)
					}
					conn, err := net.DialTCP("tcp", nil, addr)
					if err != nil {
						panic(err)
					}
					conn.SetReadBuffer(TCP_BUFFER)
					conns <- conn
				}
			}
		}
	}()

	// get parties - 1 connections and identity
	num_conns := parties * batches

	con := make([]Connection, 0, num_conns)
	for batch := 0; batch < batches; batch++ {
		for i := 0; i < parties; i++ {
			if i == 0 {
				con = append(con, MeConnection(me))
			} else {
				c := <-conns
				fmt.Println("Connection:", i)
				g, err := NewConnection(c, me)
				if err != nil {
					panic(err)
				}
				con = append(con, g)
			}
		}
	}

	// sort connections after id
	sort.Slice(con, func(i, j int) bool {
		return con[i].other < con[j].other
	})

	// split into batches
	bcon := make([][]Connection, batches)
	next := 0
	for i := 0; i < parties; i++ {
		for batch := 0; batch < batches; batch++ {
			bcon[batch] = append(bcon[batch], con[next])
			next += 1
		}
	}

	//
	fmt.Println("Connections:", bcon)
	return bcon
}

func main() {
	fmt.Println("starting", os.Args[1:])

	// find number of players
	parties := func() int {
		for i := 1; i < len(os.Args); i++ {
			if os.Args[i] == "-N" || os.Args[i] == "--nparties" {
				parties, err := strconv.Atoi(os.Args[i+1])
				if err != nil {
					panic(err)
				}
				return parties
			}
		}
		panic("Parties not specified")
	}()

	// find which player
	me := func() int {
		for i := 1; i < len(os.Args); i++ {
			if os.Args[i] == "-p" || os.Args[i] == "--player" {
				player, err := strconv.Atoi(os.Args[i+1])
				if err != nil {
					panic(err)
				}
				return player
			}
		}
		panic("Player not specified")
	}()

	fmt.Println("Player:", me)
	fmt.Println("Parties:", parties)

	// pass arguments to MP-SPDZ command
	cmd := exec.Command(os.Args[1], os.Args[2:]...)

	// get stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("failed to open stdout:", err)
		panic(err)
	}

	// get stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Println("failed to open in:", err)
		panic(err)

	}

	// start MP-SPDZ
	if err := cmd.Start(); err != nil {
		fmt.Println("failed to start:", err)
		panic(err)
	}

	// wrap in MPC abstraction
	mpc := NewMPC(stdout, stdin)
	fmt.Println(mpc)

	// make TCP connections
	bcon := MakeConnections(2, parties, me)

	oip, err := NewOIP(bcon, me, parties)
	if err != nil {
		panic(err)
	}

	inputs := func() []uint64 {
		if me == 1 {
			inputs := make([]uint64, 100)
			inputs[0] = 0x1
			return inputs
		}

		return random(100)
	}()

	// setup OIP protocol
	output := run(me, inputs, mpc, oip)

	if err := cmd.Wait(); err != nil {
		fmt.Println("error", err)
		panic(err)
	}

	fmt.Println("Output:", output)

	// os.Args[0]
	return
}
