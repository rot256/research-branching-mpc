CONST_ADD = 0
CONST_MUL = 1

wire_cnt = -1

def wire(i):
    return 'w{num}'.format(num=i)

def wires(vs):
    return [wire(i) for i in vs]

CIRCUIT_PREAMBLE = '''def output(r):
    f = ''
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
        selector: list[int],       # selector wire
        branches: list[list[Gate]] # branches
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
        self.prog('        {elem}[i] = ({elem}[i] + {tmp}[i]) % PRIME'.format(
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
        for p in range(self.players):
            self.circ(
                't{player} = sint.get_input_from({player}, size={size})'.format(
                    player=p,
                    size=size
                )
            )

        mask = ' + '.join(['t{player}'.format(player=p) for p in range(self.players)])
        self.circ(
            '{name} = {mask}'.format(
                name=name,
                mask=mask
            )
        )
        self.prog('mpc.TryInput({name})'.format(name=name))

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

    def compile(self, gates: list[Gate]):
        self.prog('package main')
        self.prog('')
        self.prog('func run(player  int, inputs []uint64, mpc *MPC, oip *OIP) {')
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

                    self.circ('z = (1 - g[{num}]) * (l + r) + g[{num}] * (l * r)'.format(
                        num=i
                    ))
                    self.circ('u[{next_idx}] = (z + out[{next_idx}]).reveal()'.format(
                        next_idx=next_idx
                    ))
                    next_idx += 1

                self.prog('}()')



            else:
                assert False

        self.prog('}')

def export(name, ls):
    s = '\n'.join(ls)
    if name is None:
        print(s)
    else:
        with open(name, 'w') as f:
            f.write(s)

if __name__ == '__main__':

    import sys

    circuit = sys.argv[1] if len(sys.argv) > 1 else None
    runner  = sys.argv[2] if len(sys.argv) > 2 else None

    p = [
        Input(0),
        Input(0),
        Input(0),
        Input(1),
        Input(1)
    ]

    a = [Add(0, 1), Mul(1, 2), Add(5, 0)]
    b = [Mul(0, 1), Add(1, 2), Add(6, 1)]

    disj = Disjunction([3, 4], [a, b])


    ctx = Ctx(2, prime=65537)
    ctx.compile(p + [disj])

    export(circuit, ctx.circuit)
    export(runner, ctx.runner)
