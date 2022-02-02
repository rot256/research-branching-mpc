#!/usr/bin/env python3

CONST_ADD = 0
CONST_MUL = 1

wire_cnt = -1

def wire(i):
    return 'w{num}'.format(num=i)

def wires(vs):
    return [wire(i) for i in vs]

CIRCUIT_PREAMBLE = '''def output(r):
    f = 'Output: '
    for _ in range(len(r)):
        f += '%s '
    f = f[:-1]
    print_ln(f, *list(r))
'''.split('\n')

'''
Computes:
'''
class Gate:
    def __init__(self):
        self.output = Wire()

    def inputs(self):
        assert False, 'undefined'

class Input(Gate):
    def __init__(self, player, dim=1):
        self.dim = dim
        self.player = player

class Output(Gate):
    def __init__(self, wire, dim=1):
        self.wire = wire
        self.dim = dim

class Mul(Gate):
    def __init__(self, l: int, r: int):
        self.l = l
        self.r = r

    def inputs(self):
        return (self.l, self.r)

class Add(Gate):
    def __init__(self, l: int, r: int):
        self.l = l
        self.r = r

    def inputs(self):
        return (self.l, self.r)

class Universal(Gate):
    def __init__(self, g: int, l: int, r: int):
        self.g = g
        self.l = l
        self.r = r

    def inputs(self):
        return (self.g, self.l, self.r)

def program(g: Gate, perm=[], prog=[]):
    if isinstance(g, Add) or isinstance(g, Mul):
        # program(g.l, perm, prog)
        # program(g.r, perm, prog)

        perm.append(g.l)
        perm.append(g.r)

        if isinstance(g, Add):
            prog.append(CONST_ADD)
        else:
            prog.append(CONST_MUL)

    else:
        print(g)
        assert False, 'gate not supported in disjunction'

class Disjunction(Gate):
    def translate(
        self,
        start # start index, first gate index in branch
    ):
        end = start + self.branch_size
        ext = set([])

        for branch in self.branches:
            for gate in branch:
                for inp in gate.inputs():
                    if inp < start:
                        ext.add(inp)

        ext = sorted(list(ext))
        ext_l = { w: i for (i, w) in enumerate(ext) }

        # translates a wire reference in the branch to an output label
        def t_wire(w):
            try:
                return ext_l[w]
            except KeyError:
                assert w < end
                return (w - start) + len(ext_l)

        self.progs = []
        self.perms = []
        self.gate_wires = [wire(i) for i in range(start, end)]
        self.disj_inputs = ext

        for branch in self.branches:
            prog = []
            perm = []

            max_wire = len(ext_l)

            for i, gate in enumerate(branch):
                # translate inputs
                for inp in gate.inputs(): perm.append(t_wire(inp))

                # translate programming to bits
                if isinstance(gate, Add) or isinstance(gate, Mul):
                    prog.append(CONST_ADD if isinstance(gate, Add) else CONST_MUL)
                else:
                    assert False, 'gate not supported in disjunction'

            assert len(perm) % 2 == 0

            self.progs.append(prog)
            self.perms.append(perm)

        # compute levels accross branche
        self.levels = []
        max_wire = len(ext_l)
        for i in range(self.branch_size):
            for perm in self.perms:
                a = perm[i*2]
                b = perm[i*2+1]
                if a > max_wire or b > max_wire:
                    self.levels.append(i - 1)
                    max_wire = (i - 1) + len(ext_l)

        self.levels.append(self.branch_size - 1)
        print('Levels:', self.levels, len(self.levels))

        if self.fixed_levels:
            self.levels = self.fixed_levels

        assert len(self.levels) <= self.branch_size

    def __init__(
        self,
        selector,
        branches,
        fixed_levels = None
    ):
        assert len(selector) == len(branches)

        branch_size = len(branches[0])

        self.perms = [] # input permutations
        self.progs = [] # programming bits
        self.inputs = []
        self.selector = selector
        self.branches = branches
        self.branch_size = branch_size
        self.fixed_levels = fixed_levels

class Ctx:
    def __init__(self, players, prime):
        self.prime = prime
        self.players = players
        self.circuit = list(CIRCUIT_PREAMBLE) # MPC circuit description
        self.runner = []  # go program
        self.n = 0

    def append(self, name, elems, offset=0):
        for i, elem in enumerate(elems):
            self.circ('{name}[{i}] = {elem}'.format(
                name=name,
                i=i+offset,
                elem=elem
            ))

    def pack(self, name, elems):
        self.circ('{name} = sint.Array(size={dim})'.format(name=name, dim=len(elems)))
        self.append(name, elems)

    def additive_output(self, elem, size, tmp='t'):
        self.additive_random(tmp, size)
        self.circ('output(({elem} - {tmp}).reveal())'.format(
            elem=elem,
            tmp=tmp
        ))

        # output into runner
        self.prog('{elem} := mpc.TryOutput({size})'.format(
            elem=elem,
            size=size
        ))
        self.prog('if player == 0 {')
        self.prog('    for i := 0; i < {size}; i++ {{'.format(size=size))
        self.prog('        {elem}[i] = add({elem}[i], {tmp}[i])'.format(
            elem=elem,
            tmp=tmp,
        ))
        self.prog('    }')
        self.prog('} else {')
        self.prog('    for i := 0; i < {size}; i++ {{'.format(size=size))
        self.prog('        {elem}[i] = {tmp}[i]'.format(
            elem=elem,
            tmp=tmp,
        ))
        self.prog('    }')
        self.prog('}')

    def additive_input(self, name, size):
        # obtain sharings from each player
        for p in range(self.players):
            self.circ(
                't{player} = sint.get_input_from({player}, size={size})'.format(
                    player=p,
                    size=size
                )
            )

        # runner inputs the shares
        self.prog('mpc.TryInput({name})'.format(name=name))

        # add sharings
        mask = ' + '.join(['t{player}'.format(player=p) for p in range(self.players)])
        self.circ(
            '{name} = {mask}'.format(
                name=name,
                mask=mask
            )
        )

    def additive_random(self, name, size):
        # each player picks a bunch of random integers
        self.prog('{name} := random({size})'.format(
            name=name,
            size=size
        ))

        self.additive_input(name, size)

    def circ(self, l=''):
        self.circuit.append(l)

    def prog(self, l=''):
        self.runner.append(l)

    def compile_cdn(self, gates):
        self.prog('package main')
        self.prog('')
        self.prog('func init() { MP_SPDZ = false }') # disable MP-SPDZ functionality
        self.prog('')
        self.prog('func run(me int, inputs []FieldElem, mpc *MPC, oip *OIP) ([]Share, error) {')
        self.prog('    if mpc != nil { panic("MP-SPDZ Enabled") }')
        self.prog('    nxt_input := 0')
        self.prog('    output := make([]FieldElem, 0, 16)')
        self.prog('    wires := make([]Share, 0, 1 << 17)')
        self.prog('    cdn := NewCDN(oip)')

        for (w, g) in enumerate(gates):
            if isinstance(g, Input):
                self.prog('    if me == {player} {{'.format(player=g.player))
                self.prog('        wires = append(wires, inputs[nxt_input])'.format(num=w))
                self.prog('        nxt_input += 1')
                self.prog('    } else { ')
                self.prog('        wires = append(wires, 0)'.format(num=w))
                self.prog('    }')

            elif isinstance(g, Output):
                self.prog('    out{idx}, err := cdn.Reconstruct([]Share{{ wires[{num}] }})'.format(num=g.wire, idx=w))
                self.prog('    if err != nil { return nil, err }')
                self.prog('    output = append(output, out{idx}...)'.format(idx=w))
                self.prog('    wires = append(wires, 0)'.format(num=w)) # the output gate yields 0

            elif isinstance(g, Mul):
                # we only implemented adds / muls in disjunctions
                raise ValueError('Unimplemented')

            elif isinstance(g, Add):
                # we only implemented adds / muls in disjunctions
                raise ValueError('Unimplemented')

            elif isinstance(g, Disjunction):
                inputs = {}

                # translate branch sub-circuits to position-dependent disjunction meta-circuit
                g.translate(w)

                self.prog('err := func() error {')

                out_dim = len(g.disj_inputs) + g.branch_size
                in_dim = g.branch_size * 2

                # export permutation to runner
                self.prog('mapping := [][]int{')
                for perm in g.perms:
                    self.prog('    {' + ','.join(map(str, perm)) +'},')
                self.prog('}')

                # export programming to runner
                self.prog('programming := [][]bool{')
                for perm in g.progs:
                    self.prog('    {' + ','.join(['true' if p else 'false' for p in perm]) +'},')
                self.prog('}')

                # pack selectors
                self.prog('selectors := []Share{' + ','.join(['wires[%d]' % i for i in g.selector]) + '}')

                # pack inputs
                self.prog('inputs := []Share{' + ','.join(['wires[%d]' % i for i in g.disj_inputs]) + '}')

                # use cdn.Disjunction helper from CDN implemenation
                self.prog('''   w, err := cdn.Disjunction(
        []int{{{levels}}},   // branch evaluation levels
        {mapping},      // mapping
        {inputs},       // inputs to all branches
        {sel},          // selectors
        {gate_programs}, // gate programmings
    )
                '''.format(
                    levels=','.join(map(str, g.levels)),
                    mapping='mapping',
                    inputs='inputs',
                    sel='selectors',
                    gate_programs='programming'
                ))
                self.prog('if err != nil { return err }')
                self.prog('wires = append(wires, w...)')

                # copy branch outputs back to wires slice


                self.prog('return nil')
                self.prog('}()')

                # handle possible error in disjunction
                self.prog('if err != nil { return nil, err }')

        self.prog('return output, nil')
        self.prog('}')


    def compile(self, gates):
        self.prog('package main')
        self.prog('')
        self.prog('func run(player  int, inputs []uint64, mpc *MPC, oip *OIP) ([]uint64, error) {')
        self.prog('output := make([]uint64, 0, 128)')
        self.prog('nxt := 0')

        def flush_inputs(inputs):
            for player in sorted(inputs.keys()):
                length = 0
                for (w, g) in inputs[player]: # length of input
                    self.circ(
                        '{out} = sint.get_input_from({player}, size={dim})'.format(
                            out=wire(w),
                            player=g.player,
                            dim=g.dim
                        )
                    )
                    length += g.dim

                # deliver single input size
                self.prog('if player == {player} {{'.format(player=player))
                self.prog('    mpc.TryInput(inputs[nxt:nxt+{length}])'.format(length=length))
                self.prog('    nxt += {length}'.format(length=length))
                self.prog('}')

        inputs = {}


        for (w, g) in enumerate(gates):
            if isinstance(g, Input):
                try:
                    inputs[g.player].append((w, g))
                except KeyError:
                    inputs[g.player] = [(w, g)]

            elif isinstance(g, Output):
                self.circ('output({wire}.reveal())'.format(
                    wire=wire(g.wire)
                ))
                self.prog('output = append(output, mpc.TryOutput({dim})...)'.format(
                    dim=g.dim
                ))

            elif isinstance(g, Mul):
                self.circ(
                    '{out} = {left} * {right}'.format(
                        out=wire(w),
                        left=wire(g.l),
                        right=wire(g.r),
                    )
                )

            elif isinstance(g, Add):
                self.circ(
                    '{out} = {left} + {right}'.format(
                        out=wire(w),
                        left=wire(g.l),
                        right=wire(g.r)
                    )
                )

            elif isinstance(g, Universal):
                self.circ(
                    '{out} = universal({gate}, {left}, {right})'.format(
                        out=wire(w),
                        gate=g.g,
                        left=g.l,
                        right=g.r
                    )
                )

            elif isinstance(g, Disjunction):
                flush_inputs(inputs)
                inputs = {}

                g.translate(w)
                self.prog('err := func() error {')

                self.circ('')
                self.circ('# pack selection wires')
                self.pack('b', wires(g.selector))

                # compute the gate programming (linear combination)
                self.circ('')
                self.circ('# compute gate programming')
                self.circ('g = sint.Array(size={dim})'.format(dim=g.branch_size))
                for i in range(g.branch_size):
                    select = ['b[%d]' % j for (j, (sel, prog)) in enumerate(zip(g.selector, g.progs)) if prog[i]]

                    if len(select) == 0:
                        self.circ('g[{num}] = 0'.format(num=i))
                    elif len(select) == len(g.selector):
                        self.circ('g[{num}] = 1'.format(num=i))
                    else:
                        self.circ('g[{num}] = {sel}'.format(
                            num=i,
                            sel=' + '.join(select)
                        ))

                out_dim = len(g.disj_inputs) + g.branch_size
                in_dim = g.branch_size * 2

                # export permutation to runner
                self.prog('mapping := [][]int{')
                for perm in g.perms:
                    self.prog('    {' + ','.join(map(str, perm)) +'},')
                self.prog('}')

                # execute OIP protocol on random mask
                self.additive_random('out', size=out_dim)       # export
                self.additive_output('b', size=len(g.selector)) # export
                self.prog('v := apply_mapping(mapping, out)')
                self.prog('D, err := oip.Select(b, v)')
                self.prog('if err != nil { return err }')

                # input back into the MPC
                self.additive_input('D', size=in_dim)

                #
                self.circ('')
                self.circ('# pack outputs to the disjunction')
                self.circ('u = cint.Array(size={dim})'.format(dim=out_dim))
                for i, w in enumerate(g.disj_inputs):
                    self.circ('u[{num}] = (out[{num}] + {wire}).reveal()'.format(
                        num=i,
                        wire=wire(w)
                    ))


                next_idx = len(g.disj_inputs)
                for i in range(g.branch_size):
                    self.circ('')
                    self.circ('# gate number %d' % i)
                    ws = (2*i, 2*i+1)
                    ns = ('l', 'r')
                    for n, w in zip(ns, ws):
                        summation = ['(b[{num}] * u[{idx}])'.format(num=j, idx=perm[w]) for (j, perm) in enumerate(g.perms)]
                        self.circ('{name} = {summation} - D[{idx}]'.format(
                            summation=' + '.join(summation),
                            name=n,
                            idx=w
                        ))

                    self.circ('{wire} = (1 - g[{num}]) * (l + r) + g[{num}] * (l * r)'.format(
                        wire=g.gate_wires[i],
                        num=i
                    ))
                    self.circ('u[{next_idx}] = ({wire} + out[{next_idx}]).reveal()'.format(
                        wire=g.gate_wires[i],
                        next_idx=next_idx
                    ))
                    next_idx += 1

                self.prog('return nil')
                self.prog('}()')
                self.prog('if err != nil { return nil, err }')

            else:
                assert False

        flush_inputs(inputs)
        inputs = {}

        self.prog('return output, nil')
        self.prog('}')

def export(name, ls):
    s = '\n'.join(ls)
    print('Write: %s (%d lines, %d chars)' % (name, len(ls), len(s)))
    if name is None:
        print(s)
    else:
        with open(name, 'w') as f:
            f.write(s)

def split(prog):
    out = []
    disj = []
    for gate in prog:
        if isinstance(gate, tuple):
            disj.append(gate)

        else:
            if disj:
                wires = disj[0]
                branches = list(zip(*disj[1:]))
                out.append(
                    Disjunction(
                        wires,
                        branches
                    )
                )
                disj = []
            out.append(gate)
    return out

def comparison(start, a, b):
    assert len(a) == len(b)
    pass

from itertools import cycle

def random_circuit(wires, start, blocks=cycle([1]), length=4096, leveled=False):
    gates = []
    inputs = list(wires)
    outputs = []
    next_block = next(blocks)
    for i in range(length):
        left   = random.choice(inputs)
        right  = random.choice(inputs)
        choice = random.randrange(2)

        if choice == 0:
            gates.append(Add(left, right))
        if choice == 1:
            gates.append(Mul(left, right))

        # add outputs to inputs
        outputs.append(i + start)
        if len(outputs) >= next_block:
            if leveled:
                inputs = outputs
            else:
                inputs += outputs

            outputs = []
            try:
                next_block = next(blocks)
            except StopIteration:
                next_block = None

    inputs += outputs

    return gates

def random_disj(sel, wires, start, blocks=cycle([4096]), length=1<<15):
    return Disjunction(
        sel,
        [random_circuit(wires, start, blocks=blocks, length=length) for _ in range(len(sel))]
    )

def random_leveled(sel, wires, start, log_length=16):
    blocks = [1 << i for i in range(log_length+1)][::-1] # starts wide
    length = (1 << (log_length+1)) - 1
    return Disjunction(sel, [
        random_circuit(wires, start, blocks=iter(blocks), length=length, leveled=False) for _ in range(len(sel))
    ])

if __name__ == '__main__':

    import os
    import sys
    import yaml
    import random

    random.seed(0x3333) # reproducable results

    args = iter(sys.argv[1:])
    path = sys.argv[1]
    name = os.path.basename(path)
    assert name.endswith('.yml')
    name = name[:-len('.yml')]

    with open(sys.argv[1], 'r') as f:
        config = yaml.safe_load(f)

    def opt():
        global args
        try:
            return next(args)
        except StopIteration:
            return None

    '''
    prog = split([
        Input(0), #0
        Input(0), #1
        Input(0), #2
        Input(1), #3
        Input(1), #4
        (3, 4),
        (Add(0, 1), Mul(0, 1)), #4
        (Mul(1, 2), Add(1, 2)), #5
        (Add(5, 0), Add(6, 1)), #7
        Output(7)
    ])
    '''

    mpc = config['mpc']
    circuit = config['circuit']

    prog = [
        Input(0), #0
        Input(0), #1
        Input(0), #2
        Input(1), #4
        Input(1), #5
        Input(1), #6
    ]

    if circuit['type'] == 'layered':
        param = circuit['parameters']

        #
        inputs = list(range(len(prog)))

        # wire indexes of branch selectors (unary representation)
        selectors = list(range(len(prog), len(prog)+param['branches']))

        # append selectors (player 1 selects the active branch in our benchmark for simplicity)
        prog += [Input(1)] * param['branches']

        # create a random layered disjunction
        prog.append(
            random_disj(
                selectors,
                wires=inputs,
                start=len(prog),
                blocks=cycle([param['per_layer']]),
                length=param['length']
            )
        )

        # output the last value in the branch
        prog.append(Output(len(prog)))

    else:
        raise ValueError('Unknown circuit type')

    # prog.append(random_disj(sel, wires=list(range(len(prog))), start=len(prog), length=length))
    # prog.append(random_leveled(sel, wires=list(range(len(prog))), start=len(prog), log_length=16))

    ctx = Ctx(mpc['parties'], prime=65537)

    if mpc['type'] == 'cdn':
        # compile to CDN execution
        ctx.compile_cdn(prog)

        # it should yield the empty circuit
        # (since CDN it is implemented in Go, not MP-SPDZ)
        assert ctx.circuit == CIRCUIT_PREAMBLE, '\n'.join(ctx.circuit)

        # export the runner
        export('runner-%s.go' % name, ctx.runner)

    else:
        # compile generic MPC
        ctx.compile(prog)

        # export runner and MP-SPDZ circuit
        export('runner-%s.go' % name, ctx.runner)
        export(
            os.path.join('MP-SPDZ/Programs/Source/', 'bmpc-%s.mpc' % name),
            ctx.circuit
        )
