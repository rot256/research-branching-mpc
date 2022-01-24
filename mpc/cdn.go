package main

import (
	"log"
	"sync"
)

const UNINITIALIZED = 0xffffffffffffffff

const CDN_DEBUG = true

type Share = FieldElem

const (
	OperationAdd = iota
	OperationMul
	OperationMulConst
	OperationAddConst
)

type Gate struct {
	op   int
	src1 int    // index of source wire
	src2 uint64 // index or constant
}

type CDN struct {
	oip  *OIP
	midx []int

	mul_left  []Share // batched left multiplication inputs
	mul_right []Share // batched right multiplication inputs

	wires []Share // wire assignments
}

func (e *CDN) ScheduleMul(l Share, r Share) {
	e.mul_left = append(e.mul_left, l)
	e.mul_right = append(e.mul_right, r)
}

func (e *CDN) Mul(l []Share, r []Share) ([]Share, error) {
	return e.oip.Multiply(l, r)
}

func (e *CDN) ResolveMul() ([]Share, error) {
	res, err := e.oip.Multiply(e.mul_left, e.mul_right)
	e.mul_left = e.mul_left[:0]
	e.mul_right = e.mul_right[:0]
	return res, err
}

func NewCDN(oip *OIP) *CDN {
	return &CDN{
		oip:       oip,
		mul_left:  make([]Share, 0, 1<<15),
		mul_right: make([]Share, 0, 1<<15),
		wires:     make([]Share, 0, 1<<16),
	}
}

func (e *CDN) Export(wires []int) []FieldElem {
	values := make([]FieldElem, len(wires))
	for i, j := range wires {
		values[i] = e.wires[j]
	}
	return values
}

// Run a disjunction (mapping and gate program computed by external program)
//
// The programming is relatively complex: the majority of the circuit analysis and work is offloaded to the circuit compiler
func (e *CDN) Disjunction(
	levels []int, // when to multiply and reconstruct block (when to switch level)
	mapping [][]int, // wire mapping for each branch, i.e mapping[b][i] is the map for gate i in branch b
	inputs []Share, // indexes of all inputs to the branch
	sel []Share, // selectors for each branch (indicator variables)
	gate_programs [][]bool, // gate programmings, i.e. gate_program[b][i] = True iff. the i'th gate in branch b is a multiplication
) error {
	branches := len(mapping)

	if branches != len(gate_programs) || branches != len(sel) {
		log.Panicln("Number of branches does not match", len(mapping), len(sel), len(gate_programs))
	}

	branch_size := len(mapping[0])

	// compute gate programming (1 iff the selected branch has a multiplication in that position)
	// this can be done in parallel with the OIP

	var wg sync.WaitGroup

	wg.Add(1)

	programming := make([]FieldElem, branch_size)

	go func() {
		for j, branch := range gate_programs {
			for i, g := range branch {
				if g {
					programming[i] = add(programming[i], sel[j])
				}
			}
		}
		wg.Done()
	}()

	// create a random shared mask and select permutation using OIP

	in_dim := len(inputs)
	out_dim := branch_size + in_dim
	out := random(out_dim)
	vec := apply_mapping(mapping, out)

	D, err := e.oip.Select(sel, vec)
	if err != nil {
		return err
	}

	// fill u with magic to detect unitialized values
	u := make([]FieldElem, out_dim)
	if CDN_DEBUG {
		for i := 0; i < len(u); i++ {
			u[i] = UNINITIALIZED
		}
	}

	// fill start of u with masked inputs
	m_inp, err := e.Reconstruct(add_vec(inputs, out[:in_dim]))
	if err != nil {
		return err
	}
	copy(u[:in_dim], m_inp)

	// returns the next selected masked input
	nxt := in_dim
	next_masked_input := func(idx int) Share {
		var sum FieldElem
		for i := 0; i < branches; i++ {
			ui := u[mapping[i][idx]]
			if CDN_DEBUG && ui == UNINITIALIZED {
				panic("Unitialized u referenced")
			}
			sum += mul(sel[i], ui)
		}
		sum = sub(reduce(sum), D[nxt])
		nxt += 1
		return sum
	}

	// execute branches in levels

	l := make([]Share, 0, 1<<13) // left operant shares of level
	r := make([]Share, 0, 1<<13) // right operant shares of level
	p := make([]Share, 0, 1<<13) // programming shares of level
	s := in_dim                  // start index of level
	level := 0                   // current level

	wg.Wait() // wait for computation of programming to complete

	for g := 0; g < branch_size; g++ {
		// add gate to branch
		l = append(l, next_masked_input(g*2))
		r = append(r, next_masked_input(g*2+1))
		p = append(p, programming[g])

		// check if gate is the last one in the level
		if g >= levels[level] {
			// execute all gates in level
			gates := len(p)

			if len(l) != gates || len(r) != gates {
				panic("Mismatching dimension")
			}

			// FIRST LEVEL: compute (l*r)
			lr, err := e.Mul(l, r)
			if err != nil {
				return err
			}

			// compute left = (l*r) || (l+r)
			// compute right = (p) || (1 - p)
			left := lr
			right := p

			for i := 0; i < gates; i++ {
				left = append(left, add(l[i], r[i]))
				right = append(right, sub(1, p[i]))
			}

			// SECOND LEVEL: compute (l*r)*p and (l+r)*(1-p)
			res, err := e.Mul(left, right)
			if err != nil {
				return err
			}

			// split mul. result into two seperate slices
			l_mul_r_mul_p := res[:gates]
			l_add_r_mul_1p := res[gates:]

			// THIRD LEVEL: reconstruct((1-p)*(l+r) + p*(l*r))
			w, err := e.Reconstruct(add_vec(l_mul_r_mul_p, l_add_r_mul_1p))
			if err != nil {
				return err
			}

			// write back to u
			copy(u[s:s+gates], w)
			s += gates

			// advance to next level of the branch
			level += 1
			l = l[:]
			r = r[:]
			p = p[:]
		}
	}

	if len(l) != 0 || level != len(levels) {
		panic("Last level has not been executed")
	}

	return nil
}

func add_vec(src1, src2 []Share) []Share {
	dst := make([]Share, len(src1))
	for i := 0; i < len(src1); i++ {
		dst[i] = add(src1[i], src2[i])
	}
	return dst
}

// reconstruct to player 0
func (e *CDN) reconstruct0(shares []Share) ([]FieldElem, error) {

	if e.oip.me != 0 {
		return nil, e.oip.Send0(shares)
	}

	recon := make([]FieldElem, len(shares))
	copy(recon, shares)

	// receieve share from every other player

	var tmp []FieldElem
	for p := 1; p < e.oip.n; p++ {
		if err := e.oip.Pi(p).Recv(&tmp); err != nil {
			return nil, err
		}
		for i := 0; i < len(recon); i++ {
			recon[i] = add(recon[i], tmp[i])
		}
	}

	return recon, nil
}

// reconstruct to player 0
func (e *CDN) Reconstruct(shares []Share) ([]FieldElem, error) {
	// reconstuct to player 0
	val, err := e.reconstruct0(shares)
	if err != nil {
		return nil, err
	}

	// player 0 sends the construction to everyone else
	if e.oip.me == 0 {
		return val, e.oip.broadcast(val)
	} else {
		return val, e.oip.Recv0(&val)
	}
}

/*
func (e *CDN) Schedule(ops []Gate) {

	for _, g := range ops {
		switch g.op {
		case OperationAdd:

			// everybody locally adds their shares

			e.wires = append(
				e.wires,
				add(e.wires[g.src1], e.wires[g.src2]),
			)

		case OperationAddConst:

			// player0 add constant to his share

			if e.oip.me == 0 {
				e.wires = append(
					e.wires,
					add(e.wires[g.src1], g.src2),
				)
			}

		case OperationMulConst:

			// execute linear operation locally

			e.wires = append(
				e.wires,
				mul(e.wires[g.src1], g.src2),
			)

		case OperationMul:

			// batch multiplication

			e.left = append(e.left, e.wires[g.src1])
			e.right = append(e.right, e.wires[g.src2])
			e.wires = append(e.wires, UNRESOLVED)
		}
	}
}
*/
