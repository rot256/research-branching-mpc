package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"

	"os/exec"
)

func party_address(party int) string {
	return "127.0.0.1:" + strconv.Itoa(party_port(party))
}

func party_port(party int) int {
	return party + 7000
}

type Connection struct {
	/*
		dec   *gob.Decoder
		enc   *gob.Encoder
	*/
	dec *json.Decoder
	enc *json.Encoder
	/*
		dec   *cbor.Decoder
		enc   *cbor.Encoder
	*/
	other int
}

func MeConnection(me int) Connection {
	return Connection{
		dec:   nil,
		enc:   nil,
		other: me,
	}
}

func NewConnection(conn net.Conn, me int) (Connection, error) {
	c := Connection{
		/*
			dec:   cbor.NewDecoder(conn),
			enc:   cbor.NewEncoder(conn),
		*/
		dec:   json.NewDecoder(conn),
		enc:   json.NewEncoder(conn),
		other: 0,
	}

	// identity peers
	if err := c.Send(me); err != nil {
		return c, err
	}
	if err := c.Recv(&c.other); err != nil {
		return c, err
	}
	return c, nil
}

func (c *Connection) Send(v interface{}) error {
	fmt.Println("Send:", v)
	return c.enc.Encode(v)
}

func (c *Connection) Recv(v interface{}) error {
	return c.dec.Decode(v)
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

	addr_me, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(party_port(me)))
	if err != nil {
		panic(err)
	}

	conns := make(chan net.Conn)

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
		for party := 0; party < parties; party++ {
			if party > me {
				// I must connect
				addr, err := net.ResolveTCPAddr("tcp", party_address(party))
				if err != nil {
					panic(err)
				}
				conn, err := net.DialTCP("tcp", nil, addr)
				conn.SetReadBuffer(TCP_BUFFER)
				if err != nil {
					panic(err)
				}
				conns <- conn
			}
		}
	}()

	// get parties - 1 connections and identity
	con := make([]Connection, 0, parties)
	con = append(con, MeConnection(me))
	for i := 1; i < parties; i++ {
		c := <-conns
		fmt.Println("Connection:", i)
		g, err := NewConnection(c, me)
		if err != nil {
			panic(err)
		}
		con = append(con, g)
	}

	// sort connections after
	sort.Slice(con, func(i, j int) bool {
		return con[i].other < con[j].other
	})

	fmt.Println("Connections:", con)

	oip, err := NewIOP(con, me, parties)
	if err != nil {
		panic(err)
	}

	inputs := func() []uint64 {

		if me == 0 {
			inputs := []uint64{
				0x0,
				0x0,
				0x0,
			}
			return inputs
		}

		if me == 1 {
			inputs := []uint64{
				0x0,
				0x1,
			}
			return inputs

		}
		return nil

	}()

	// setup OIP protocol
	run(me, inputs, mpc, oip)

	if err := cmd.Wait(); err != nil {
		fmt.Println("error", err)
		panic(err)
	}

	// os.Args[0]
	return
}
