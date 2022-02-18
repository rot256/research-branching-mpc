package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const ENV_PLAYER_ADDRESSES = "PLAYER_ADDRESSES"

var MP_SPDZ = true
var ADDRESSES []*net.TCPAddr

/// load player addresses (only player0 required)
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
	}
}

func connect(me int, player int) (*Connection, error) {
	if player >= len(ADDRESSES) {
		log.Fatal("Player addresses not specified", player)
	}

	// create TCP connection
	conn, err := net.DialTCP("tcp", nil, ADDRESSES[player])
	if err != nil {
		return nil, err
	}

	// wrap in Gob
	c := NewConnection(conn)
	if err := c.Send(me); err != nil {
		return nil, err
	}
	return c, nil
}

func wait_connections(me int, players int) ([]*Connection, error) {
	conns := make([]*Connection, players)

	log.Println("Waiting for connections")

	// listen on all interfaces
	addr, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(ADDRESSES[me].Port))
	if err != nil {
		return nil, err
	}

	ls, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}

	// accept one connection from each player
	for p := 0; p < players; p++ {
		var them int

		if p == me {
			continue
		}

		// accept next connection

		conn, err := ls.AcceptTCP()
		if err != nil {
			panic(err)
		}
		c := NewConnection(conn)

		// receieve remote identity

		if err := c.Recv(&them); err != nil {
			return nil, err
		}

		// check for duplicate connection

		if conns[them] != nil {
			return nil, errors.New("duplicate connection")
		}

		log.Println("Got connection from player", them)
		conns[them] = c
	}

	return conns, nil
}

func apply_mapping(mapping [][]int, inputs []uint64) [][]uint64 {

	out := make([][]uint64, len(mapping))

	for i, m := range mapping {
		perm := make([]uint64, len(m))
		for j, mj := range m {
			perm[j] = inputs[mj]
		}
		out[i] = perm
	}

	return out
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

	// star topology (everybody connects to player 0)

	var conns []*Connection

	if me == 0 {
		log.Println("Wait for connections")
		var err error
		conns, err = wait_connections(me, parties)
		if err != nil {
			panic(err)
		}
	} else {
		log.Println("Connect to player 0")
		conn0, err := connect(me, 0)
		if err != nil {
			panic(err)
		}
		conns = make([]*Connection, parties)
		conns[0] = conn0
	}

	// setup OIP
	oip := NewOIP(
		SetupParams(),
		me,
		conns,
	)

    oip.log = true

	// load inputs for party
	inputs := func() []uint64 {
		if me == 1 {
			inputs := make([]uint64, 100)
			inputs[3] = 0x1
			return inputs
		}

		return random(100)
	}()

	// we can execute multiple reps with the same setup
	for reps := 0; reps < 1; reps++ {

		// start MP-SPDZ
		var mpc *MPC
		var cmd *exec.Cmd
		if MP_SPDZ {
            log.Println("Wrapping MP-SPDZ")
			mpc, cmd = func() (*MPC, *exec.Cmd) {
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
		}

		// run MPC circuit
		log.Println("Start evaluation...")
		output, err := run(me, inputs, mpc, oip)
		if err != nil {
			panic(err)
		}

		if MP_SPDZ {
            log.Println("Waiting for MP-SPDZ to finish")
			// wait for MP-SPDZ to finish
			if err := cmd.Wait(); err != nil {
				panic(err)
			}
		}

		log.Println("Output:", output)
	}

}
