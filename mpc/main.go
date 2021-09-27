package main

import (
	"fmt"
	"os"
	"strconv"

	"os/exec"
)

func main() {
	fmt.Println("starting", os.Args[1:])

	// find which player
	player := func() int {
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

	fmt.Println("Player", player)

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
	mpc := NewMPC(stdout, stdin, 0xffffffffffffffc5)

	a := []uint64{0x0, 0x1}
	b := []uint64{0x0, 0x1}
	g := []uint64{0x1, 0x0}

	try(mpc.Input(g))
	try(mpc.Input(a))
	try(mpc.Input(b))
	mpc.Round()

	n, err := mpc.Output()
	if err != nil {
		panic(err)
	}
	fmt.Println(n)

	if err := cmd.Wait(); err != nil {
		fmt.Println("error", err)
		panic(err)
	}

	// os.Args[0]
	return
}
