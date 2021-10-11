package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"sync"
)

func party_address(party int) string {
	return "127.0.0.1:" + strconv.Itoa(party_port(party))
}

func party_port(party int) int {
	return party + 7000
}

type ConnectionIdentify struct {
	other int
	conn  Connection
}

type Networker struct {
	me      int
	players int
	conns   chan ConnectionIdentify
}

func (n *Networker) Connect(player int) error {
	// create TCP connection
	addr, err := net.ResolveTCPAddr("tcp", party_address(player))
	if err != nil {
		return err
	}
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return err
	}

	// identify self to other party
	var c ConnectionIdentify
	c.other = player
	c.conn = NewConnection(conn)
	if err := c.conn.Encode(n.me); err != nil {
		return err
	}
	n.conns <- c
	return nil
}

func NewNetworker(players, me int) Networker {
	// listen for connections
	addr_me, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(party_port(me)))
	if err != nil {
		panic(err)
	}

	//
	var networker Networker
	networker.conns = make(chan ConnectionIdentify, 10)
	networker.me = me
	networker.players = players

	// accept connections indefintely
	go func(conns chan ConnectionIdentify) {
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

			// identify counter party
			var c ConnectionIdentify
			c.conn = NewConnection(conn)
			if err := c.conn.Decode(&c.other); err != nil {
				panic(err)
			}
			conns <- c
		}
	}(networker.conns)

	return networker
}

func (n *Networker) NewConnections() []Connection {
	// make outbound connections
	var wg sync.WaitGroup
	for p := 0; p < n.players; p++ {
		if p > n.me {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := n.Connect(p); err != nil {
					panic(err)
				}
			}()
		}
	}

	// collect sufficient connections
	var id_conns []ConnectionIdentify
	for i := 0; i < n.players; i++ {
		if i == 0 {
			id_conns = append(id_conns, ConnectionIdentify{
				conn:  NilConnection(),
				other: n.me,
			})
		} else {
			id_conns = append(id_conns, <-n.conns)
		}
	}

	// all outbound connections must have been made
	wg.Wait()

	// sort by party
	sort.Slice(id_conns, func(i, j int) bool {
		return id_conns[i].other < id_conns[j].other
	})

	// extract connections
	var conns []Connection
	for i := 0; i < n.players; i++ {
		if id_conns[i].other != i {
			panic("bad connection id")
		}
		conns = append(conns, id_conns[i].conn)
	}
	return conns
}

func main() {
	log.Println("Setting up:", os.Args[1:])

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

	log.Println("Player:", me)
	log.Println("Parties:", parties)

	// start MP-SPDZ
	mpc, cmd := func() (*MPC, *exec.Cmd) {
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

		// start MP-SPDZ instance
		if err := cmd.Start(); err != nil {
			fmt.Println("failed to start:", err)
			panic(err)
		}

		// wrap in MPC abstraction
		return NewMPC(stdout, stdin), cmd
	}()

	// setup OIP
	networker := NewNetworker(parties, me)
	oip, err := NewOIP(
		[][]Connection{
			networker.NewConnections(),
			networker.NewConnections(),
		},
		me,
		parties,
	)
	if err != nil {
		panic(err)
	}

	// load inputs for party
	inputs := func() []uint64 {
		if me == 1 {
			inputs := make([]uint64, 100)
			inputs[0] = 0x1
			return inputs
		}

		return random(100)
	}()

	// run MPC circuit
	log.Println("Start evaluation...")
	output := run(me, inputs, mpc, oip)

	// wait for MP-SPDZ to finish
	if err := cmd.Wait(); err != nil {
		panic(err)
	}
	log.Println("Output:", output)
}
