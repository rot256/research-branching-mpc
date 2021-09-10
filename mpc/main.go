package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"os/exec"
	"strings"
)

var INPUT_PROMPT = "Please input"

type MPC struct {
	p   uint64
	in  *bufio.Scanner
	out *bufio.Writer
}

func NewMPC(in io.Reader, out io.Writer, prime uint64) *MPC {
	return &MPC{
		p:   prime,
		in:  bufio.NewScanner(in),
		out: bufio.NewWriter(out),
	}
}

// input a value into the MPC
func (m *MPC) Input(elems []uint64) error {

	//
	for i := 0; i < len(elems); i++ {
		_, err := m.out.WriteString(
			strconv.FormatUint(elems[i], 10) + " ",
		)
		if err != nil {
			return err
		}
	}

	// terminate with newline
	if _, err := m.out.WriteRune('\n'); err != nil {
		return err
	}
	return m.out.Flush()
}

// read an output from the MPC
func (m *MPC) Output() ([]uint64, error) {
	// read line
	if ok := m.in.Scan(); !ok {
		return nil, errors.New("no output, EOF")
	}

	// discard junk
	if strings.HasPrefix(m.in.Text(), INPUT_PROMPT) {
		return m.Output()
	}

	fmt.Println(m.in.Text())

	// split on space
	split := strings.Split(m.in.Text(), " ")

	// convert each element from string to uint64
	elems := make([]uint64, len(split))
	for i := 0; i < len(split); i++ {
		n, err := strconv.ParseUint(split[i], 10, 64)
		if err != nil {
			return nil, err
		}
		elems[i] = n
	}
	return elems, nil
}

func (m *MPC) Flush() error {
	return m.out.Flush()
}

func main() {
	fmt.Println("starting", os.Args[1:])

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

	n := []uint64{0x0, 0x1}
	for i := 0; i < 100; i++ {

		if err := mpc.Input(n); err != nil {
			fmt.Println("failed to provide input:", err)
			panic(err)
		}

		fmt.Println("sent", n)

		n, err = mpc.Output()
		if err != nil {
			panic(err)
		}
		n[0] += 1
		n[1] += 1
		fmt.Println(n)
	}

	if err := cmd.Wait(); err != nil {
		fmt.Println("error", err)
		panic(err)
	}

	// os.Args[0]
	return
}
