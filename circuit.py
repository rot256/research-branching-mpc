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
            for gate in branch:
                for inp in gate.inputs():
                    perm.append(t_wire(inp))

                if isinstance(gate, Add) or isinstance(gate, Mul):
                    prog.append(CONST_ADD if isinstance(gate, Add) else CONST_MUL)
                else:
                    assert False, 'gate not supported in disjunction'

            assert len(perm) % 2 == 0

            self.progs.append(prog)
            self.perms.append(perm)

    def __init__(
        self,
        selector,
        branches,
    ):
        assert len(selector) == len(branches)

        branch_size = len(branches[0])

        self.perms = [] # input permutations
        self.progs = [] # programming bits
        self.inputs = []
        self.selector = selector
        self.branches = branches
        self.branch_size = branch_size

class Ctx:
    def __init__(self, players, prime):
        self.prime = prime
        self.players = players
        self.circuit = CIRCUIT_PREAMBLE # MPC circuit description
        self.runner = []  # go program
        self.n = 0

    def append(self, name, elems, offset=0):
        for i, elem in enumerate(elems):
            self.circuit.append('{name}[{i}] = {elem}'.format(
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

    def compile(self, gates):
        self.prog('package main')
        self.prog('')
        self.prog('func run(player  int, inputs []uint64, mpc *MPC, oip *OIP) []uint64 {')
        self.prog('output := make([]uint64, 0, 128)')
        self.prog('nxt := 0')
        for (w, g) in enumerate(gates):
            if isinstance(g, Input):
                self.circ(
                    '{out} = sint.get_input_from({player}, size={dim})'.format(
                        out=wire(w),
                        player=g.player,
                        dim=g.dim
                    )
                )
                self.prog('if player == {player} {{'.format(
                    player=g.player
                ))
                self.prog('    mpc.TryInput([]uint64{inputs[nxt]})')
                self.prog('    nxt += 1')
                self.prog('}')

            elif isinstance(g, Output):
                self.circ('output({wire}.reveal())'.format(
                    wire=wire(g.wire)
                ))
                self.prog('output = append(output, mpc.TryOutput({dim})...)'.format(
                    dim=g.dim
                ))

            elif isinstance(g, Mul):
                self.circuit.append(
                    '{out} = {left} * {right}'.format(
                        out=wire(w),
                        left=wire(g.l),
                        right=wire(g.r),
                    )
                )

            elif isinstance(g, Add):
                self.circuit.append(
                    '{out} = {left} + {right}'.format(
                        out=wire(w),
                        left=wire(g.l),
                        right=wire(g.r)
                    )
                )

            elif isinstance(g, Universal):
                self.circuit.append(
                    '{out} = universal({gate}, {left}, {right})'.format(
                        out=wire(w),
                        gate=g.g,
                        left=g.l,
                        right=g.r
                    )
                )

            elif isinstance(g, Disjunction):
                g.translate(w)
                self.prog('func() {')

                self.circ('')
                self.circ('# pack selection wires')
                self.pack('b', wires(g.selector))

                # compute the gate programming (linear combination)
                self.circ('')
                self.circ('# compute gate programming')
                self.circuit.append('g = sint.Array(size={dim})'.format(dim=g.branch_size))
                for i in range(g.branch_size):
                    select = ['b[%d]' % j for (j, (sel, prog)) in enumerate(zip(g.selector, g.progs)) if prog[i]]

                    if len(select) == 0:
                        self.circuit.append('g[{num}] = 0'.format(num=i))
                    elif len(select) == len(g.selector):
                        self.circuit.append('g[{num}] = 1'.format(num=i))
                    else:
                        self.circuit.append('g[{num}] = {sel}'.format(
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

                # execute OIP protocol
                self.additive_random('out', size=out_dim)
                self.additive_output('b', size=len(g.selector))
                self.prog('D := oip.TryOIPMapping(mapping, b, out)')
                self.additive_input('D', size=in_dim)

                #
                self.circ('')
                self.circ('# pack outputs to the disjunction')
                self.circ('u    = cint.Array(size={dim})'.format(dim=out_dim))
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

                self.prog('}()')



            else:
                assert False

        self.prog('return output')
        self.prog('}')

def export(name, ls):
    s = '\n'.join(ls)
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

import random

def random_circuit(wires, start, length=4096):
    out = []
    wires = list(wires)
    for i in range(length):
        left   = random.choice(wires)
        right  = random.choice(wires)
        choice = random.randrange(2)
        if choice == 0:
            out.append(Add(left, right))
        if choice == 1:
            out.append(Mul(left, right))
        wires.append(i + start)
    return out

def random_disj(sel, wires, start, length=4096):
    return Disjunction(sel, [random_circuit(wires, start, length) for _ in range(len(sel))])

if __name__ == '__main__':

    import sys

    args = iter(sys.argv[1:])

    try:
        length = int(next(args))
        branches = int(next(args))
        parties = int(next(args))
    except (StopIteration, ValueError):
        print('%s length branches parties [circuit] [runner]' % sys.argv[0])
        exit(-1)

    def opt():
        global args
        try:
            return next(args)
        except StopIteration:
            return None

    circuit = opt()
    runner = opt()

    print('length:', length)
    print('branches:', branches)
    print('parties:', parties)
    print('circuit:', circuit)
    print('runner:', runner)

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

    prog = [
        Input(0), #0
        Input(0), #1
        Input(0), #2
        Input(1), #4
        Input(1), #5
        Input(1), #6
    ] + [Input(1)] * branches

    sel = list(range(3, 3+branches))
    prog.append(random_disj(sel, wires=list(range(len(prog))), start=len(prog), length=length))
    prog.append(Output(len(prog)))

    ctx = Ctx(parties, prime=65537)
    ctx.compile(prog)

    export(circuit, ctx.circuit)
    export(runner, ctx.runner)
