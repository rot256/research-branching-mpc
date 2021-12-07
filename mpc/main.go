package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

const ENV_PLAYER_ADDRESSES = "PLAYER_ADDRESSES"

type ConnectionIdentify struct {
	other int
	conn  Connection
}

type ConnectionPair struct {
	send *Connection
	recv *Connection
}

type Networker struct {
	me      int
	players int
	conns   []ConnectionPair
	ready   sync.WaitGroup
}

var ADDRESSES []*net.TCPAddr

/// load player addresses
func init() {
	var path string
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if pair[0] == ENV_PLAYER_ADDRESSES {
			path = pair[1]
		}
	}

	if path == "" {
		log.Println("No peer addresses loaded")
		return
	}

	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		addr, err := net.ResolveTCPAddr("tcp", scanner.Text())
		if err != nil {
			log.Fatal(err)
		}
		ADDRESSES = append(ADDRESSES, addr)
		fmt.Println("Peer address:", addr)
	}
}

func (n *Networker) Connect(player int) error {
	if player >= len(ADDRESSES) {
		log.Fatal("Player addresses not specified", player)
	}

	// create TCP connection
	conn, err := net.DialTCP("tcp", nil, ADDRESSES[player])
	if err != nil {
		return err
	}

	// wrap in Gob
	if n.conns[player].send != nil {
		log.Fatal("Double connection to:", player)
	}
	n.conns[player].send = NewConnection(conn)
	n.ready.Done()

	// tell the other side who we are
	return n.conns[player].send.Encode(n.me)
}

func NewNetworker(players, me int) Networker {
	if me >= len(ADDRESSES) {
		log.Fatal("Player addresses not specified", me)
	}

	//
	var n Networker
	n.me = me
	n.players = players
	n.conns = make([]ConnectionPair, players)
	n.ready.Add(2 * (players - 1))

	// accept connections indefintely
	go func(n *Networker) {
		ls, err := net.ListenTCP("tcp", ADDRESSES[me])
		if err != nil {
			panic(err)
		}

		for {
			// accept new connection
			conn, err := ls.AcceptTCP()
			if err != nil {
				panic(err)
			}
			c := NewConnection(conn)

			// identify counter party
			var them int
			if err := c.Decode(&them); err != nil {
				panic(err)
			}

			// add connection
			if n.conns[them].recv != nil {
				log.Fatal("Double connection from:", them)
			}
			n.conns[them].recv = c
			n.ready.Done()

			// connect back to higher player number (he is now online, we know)
			if them > me {
				if err := n.Connect(them); err != nil {
					log.Fatalln("Failed to connect to", them, err)
				}
			}
		}
	}(&n)

	// connect to lower players
	for them := 0; them < me; them++ {
		if err := n.Connect(them); err != nil {
			log.Fatalln("Failed to connect to", them, err)
		}
	}

	// wait for 2 connections from each player
	n.ready.Wait()
	return n
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

	if parties > len(ADDRESSES) {
		log.Fatal("Too few player addresses specified")
	}

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

	// block and wait for connections
	networker := NewNetworker(parties, me)

	// setup OIP
	oip, err := NewOIP(
		networker.conns,
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
