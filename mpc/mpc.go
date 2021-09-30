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
var OUTPUT_PROMPT = "Output: "

type MPC struct {
	in  *bufio.Scanner
	out *bufio.Writer
}

func NewMPC(in io.Reader, out io.Writer) *MPC {
	return &MPC{
		in:  bufio.NewScanner(in),
		out: bufio.NewWriter(out),
	}
}

// input a value into the MPC
func (m *MPC) Input(elems []uint64) error {
	fmt.Println("Input values to MPC:", len(elems))

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

func (m *MPC) TryInput(elems []uint64) {
	if err := m.InputRound(elems); err != nil {
		panic(err)
	}
}

func (m *MPC) TryOutput(size int) []uint64 {
	fmt.Println("Read output from MPC:", size)
	res, err := m.Output(size)
	if err != nil {
		panic(err)
	}
	return res
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
func (m *MPC) Output(size int) ([]uint64, error) {
	elems := make([]uint64, 0, size)
	for len(elems) < size {
		// read line
		if ok := m.in.Scan(); !ok {
			return nil, errors.New("no output, EOF")
		}

		// fmt.Println("Data:", m.in.Text())

		// discard junk
		if strings.HasPrefix(m.in.Text(), OUTPUT_PROMPT) {
			// split on space
			line := m.in.Text()[len(OUTPUT_PROMPT):]
			split := strings.Split(line, " ")

			// convert each element from string to uint64
			for i := 0; i < len(split); i++ {
				n, err := strconv.ParseInt(split[i], 10, 64)
				if err != nil {
					return nil, err
				}
				if n < 0 {
					n += int64(PRIME)
				}
				elems = append(elems, uint64(n))
			}
		}

	}

	if len(elems) != size {
		panic("too many elements")
	}
	fmt.Println("Elems:", len(elems))
	return elems, nil
}

func (m *MPC) Flush() error {
	return m.out.Flush()
}
