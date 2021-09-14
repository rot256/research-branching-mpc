CONST_ADD = 0
CONST_MUL = 1

wire_cnt = -1

class Wire:
    def __init__(self):
        global wire_cnt
        wire_cnt = wire_cnt + 1
        self.n = int(wire_cnt)

    def __str__(self):
        return 'w%d' % self.n

    def __lt__(self, other):
        return self.n < other.n

'''
Computes:

'''
class Gate:
    def __init__(self):
        self.output = Wire()

    def inputs(self):
        assert False, 'undefined'

class UnarySelector(Gate):
    def __init__(self, b: list[Wire], m: list[Wire]):
        self.output = Wire(m[0].dim)


class Input(Gate):
    def __init__(self, player, dim=1):
        self.dim = dim
        self.player = player
        self.output = Wire()

class Mul(Gate):
    def __init__(self, l: Wire, r: Wire):
        self.l = l
        self.r = r
        self.output = Wire()

    def inputs(self):
        return (self.l, self.r)

class Add(Gate):
    def __init__(self, l: Wire, r: Wire):
        self.l = l
        self.r = r
        self.output = Wire()

    def inputs(self):
        return (self.l, self.r)

class Universal(Gate):
    def __init__(self, g: Wire, l: Wire, r: Wire):
        self.g = g
        self.l = l
        self.r = r
        self.output = Wire()

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
    def __init__(
        self,
        s: list[Wire],
        outs: int, # number of outputs
        branches: list[list[Gate]]
    ):
        assert len(s) == len(branches)

        num_gates = None
        dim = len(branches[0])

        self.perms = []
        self.progs = []
        self.wires = []
        self.inputs = set([])

        # compute programming and permutations
        for branch in branches:
            assert len(branch) >= outs
            assert dim == len(branch)

            perm = []
            prog = []
            for gate in branch:
                self.inputs.add(gate.inputs())
                program(gate, perm, prog)

            if num_gates is None:
                num_gates = len(prog)
            else:
                assert len(prog) == num_gates

            self.perms.append(perm)
            self.progs.append(prog)

        self.inputs = sorted(list(self.inputs))
        self.outputs = [Wire() for _ in range(outs)]

        print(self.inputs)

class Ctx:
    def __init__(self):
        self.circuit = [] # MPC circuit description
        self.runner = []  # go program
        self.n = 0

    def compile(self, gates: list[Gate]):
        for  gate in gates:
            self.compile_gate(gate)

    def compile_gate(self, g: Gate):
        if isinstance(g, Input):
            self.circuit.append(
                '{out} = sint.get_input_from({player}, size={dim})'.format(
                    out=g.output,
                    player=g.player,
                    dim=g.dim
                )
            )

        elif isinstance(g, Mul):
            self.circuit.append(
                '{out} = {left} * {right}'.format(
                    out=g.output,
                    left=g.l,
                    right=g.r,
                )
            )

        elif isinstance(g, Add):
            self.circuit.append(
                '{out} = {left} + {right}'.format(
                    out=g.output,
                    left=g.l,
                    right=g.r
                )
            )

        elif isinstance(g, Universal):
            self.circuit.append(
                '{out} = universal({gate}, {left}, {right})'.format(
                    out=g.output,
                    gate=g.g,
                    left=g.l,
                    right=g.r
                )
            )

        elif isinstance(g, Disjunction):
            for i, perm in enumerate(g.perms):
                self.circuit.append('p{num} = [{perm}]'.format(
                    num=i,
                    perm=','.join(map(str, perm))
                ))

            for i, _ in enumerate(g.outputs):
                self.circuit.append('out{num} = get_random()'.format(
                    num=i
                ))

            '''

            '''

            for i, _ in enumerate(g.outputs):
                self.circuit.append('D{num} = out{num} - sel{num}'.format(
                    num=i
                ))

            print(str(sorted(g.inputs)))

        else:
            assert False

ctx = Ctx()

v0 = Input(0)
v1 = Input(0)
v2 = Input(0)

s0 = Input(1)
s1 = Input(1)

a = [Add(v0.output, v1.output), Mul(v1.output, v2.output)]
b = [Mul(v0.output, v1.output), Add(v1.output, v2.output)]

ctx.compile(a)

ctx.compile_gate(
    Disjunction([s0.output, s1.output], 2, [a, b])
)

print(ctx.circuit)




