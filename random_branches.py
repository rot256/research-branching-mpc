import sys
import random

branches = int(sys.argv[1])

length = int(sys.argv[2])

def random_branch(wires, length):
    circ = []

    w = list(wires)

    for i in range(length):
        l = random.choice(w)
        r = random.choice(w)
        op = random.choice(['+', '*'])

        out = 'out%d' % i
        circ.append('%s = %s %s %s' % (out, l, op, r))
        w.append(out)

    # final sum (avoid deadcode elim)
    circ.append('r = 0')
    for i in range(length):
        circ.append('r = r + out%d' % i)

    circ.append('r.reveal()')

    return circ


if __name__ == '__main__':

    wires = ['t%d' % i for i in range(6+branches)]

    circ = [
        't{num} = sint.get_input_from({player}, size={size})'.format(
            num=num, player=0, size=1
        )
        for num in range(len(wires))
    ]

    for b in range(branches):
        for ins in random_branch(wires, length):
            circ.append(ins)

    with open(sys.argv[3], 'w') as f:
        for c in circ:
            f.write(c + '\n')




