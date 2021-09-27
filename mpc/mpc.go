package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"

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

func try(err error) {
	if err != nil {
		fmt.Println("Failed:", err)
		panic(err)
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
	return nil
}

func (m *MPC) Round() error {
	// terminate with newline
	if _, err := m.out.WriteRune('\n'); err != nil {
		return err
	}
	return m.out.Flush()
}

func (m *MPC) InputRound(elems []uint64) error {
	if err := m.Input(elems); err != nil {
		return err
	}
	return m.Round()
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
