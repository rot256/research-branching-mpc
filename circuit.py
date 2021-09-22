CONST_ADD = 0
CONST_MUL = 1

wire_cnt = -1

def wire(i):
    return 'w{num}'.format(num=i)

def wires(vs):
    return [wire(i) for i in vs]

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

        # compute programming and permutations
        for branch in branches:
            inputs = []

            assert len(branch) == branch_size

            perm = []
            prog = []
            for gate in branch:
                for inp in gate.inputs():
                    inputs.append(inp)
                program(gate, perm, prog)

            self.perms.append(perm)
            self.progs.append(prog)
            self.inputs.append(inputs)

        # find unique inputs
        disj_inputs = set([])
        for inputs in self.inputs:
            disj_inputs |= set(inputs)
        self.disj_inputs = sorted(list(disj_inputs))

        # re-label inputs
        self.rev_inputs = {}
        for new, org in enumerate(self.disj_inputs):
            self.rev_inputs[org] = new

        self.branch_size = branch_size
        self.num_outputs = branch_size

class Ctx:
    def __init__(self, players):
        self.players = players
        self.circuit = [] # MPC circuit description
        self.runner = []  # go program
        self.n = 0

    def pack(self, name, elems):
        self.circuit.append('{name} = sint.Array({dim})'.format(
            name=name,
            dim=len(elems)
        ))
        for i, elem in enumerate(elems):
            self.circuit.append('{name}[{i}] = {elem}'.format(
                name=name,
                i=i,
                elem=elem
            ))

    def additive_output(self, elems, tmp_name='v'):
        # mask the vector:
        # each party samples a random value
        self.circ('')
        self.circ('# export to additive sharing')
        for p in range(self.players):
            self.circuit.append(
                'm{player} = sint.get_input_from({player}, size={dim})'.format(
                    player=p,
                    dim=len(elems)
                )
            )

        self.pack(tmp_name, elems)
        mask = ' + '.join(['m{player}'.format(player=p) for p in range(self.players)])
        self.circ(
            't = ({mask} + {name}).reveal()'.format(
                mask=mask,
                name=tmp_name
            )
        )
        self.circ('output(t)')

    def additive_input(self, name, dim):
        self.circ()
        self.circ('# import additive sharing (dim = %d)' % dim)
        for p in range(self.players):
            self.circuit.append(
                't{player} = sint.get_input_from({player}, size={dim})'.format(
                    player=p,
                    dim=dim
                )
            )

        mask = ' + '.join(['t{player}'.format(player=p) for p in range(self.players)])
        self.circuit.append(
            '{name} = {mask}'.format(
                name=name,
                mask=mask
            )
        )

    def circ(self, l=''):
        self.circuit.append(l)

    def select(self, s, vec, perms):
        assert len(s) == len(perms)
        self.additive_output(['w%d' % i for i in s + vec])
        self.additive_input('sl', len(vec))

    def compile(self, gates: list[Gate]):
        for (w, g) in enumerate(gates):
            if isinstance(g, Input):
                self.circuit.append(
                    'w{out} = sint.get_input_from({player}, size={dim})'.format(
                        out=w,
                        player=g.player,
                        dim=g.dim
                    )
                )

            elif isinstance(g, Mul):
                self.circuit.append(
                    'w{out} = w{left} * w{right}'.format(
                        out=w,
                        left=g.l,
                        right=g.r,
                    )
                )

            elif isinstance(g, Add):
                self.circuit.append(
                    'w{out} = w{left} + w{right}'.format(
                        out=w,
                        left=g.l,
                        right=g.r
                    )
                )

            elif isinstance(g, Universal):
                self.circuit.append(
                    'w{out} = universal(w{gate}, w{left}, w{right})'.format(
                        out=w,
                        gate=g.g,
                        left=g.l,
                        right=g.r
                    )
                )

            elif isinstance(g, Disjunction):
                self.circ('')
                self.circ('# pack selection wires')
                self.pack('s', wires(g.selector))

                # compute the gate programming (linear combination)
                self.circ('')
                self.circ('# compute gate programming')
                self.circuit.append('g = sint.Array(dim={dim})'.format(dim=g.branch_size))
                for i in range(g.branch_size):
                    select = ['s[%d]' % j for (j, (sel, prog)) in enumerate(zip(g.selector, g.progs)) if prog[i]]

                    if len(select) == 0:
                        self.circuit.append('g[{num}] = 0'.format(num=i))
                    elif len(select) == len(g.selector):
                        self.circuit.append('g[{num}] = 1'.format(num=i))
                    else:
                        self.circuit.append('g[{num}] = {sel}'.format(
                            num=i,
                            sel=' + '.join(select)
                        ))

                self.circ('')
                self.circ('# pack inputs to the disjunction')
                self.pack('inp', wires(g.disj_inputs))

                self.circ('')
                self.circ('# mask inputs and reconstruct u')
                self.circ('in = get_random(dim={dim})'.format(dim=len(g.disj_inputs)))
                self.circ('u = (in + inp).reveal()')

                self.select(g.selector, g.disj_inputs, [0, 0])

                #
                for i, perm in enumerate(g.perms):
                    self.circuit.append('p{num} = [{perm}]'.format(
                        num=i,
                        perm=','.join(map(str, perm))
                    ))

                '''
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
                '''

            else:
                assert False



p = [
    Input(0),
    Input(0),
    Input(0),
    Input(1),
    Input(1)
]

a = [Add(0, 1), Mul(1, 2)]
b = [Mul(0, 1), Add(1, 2)]

disj = Disjunction([3, 4], [a, b])


ctx = Ctx(2)
ctx.compile(p + [disj])

print('\n'.join(ctx.circuit))


