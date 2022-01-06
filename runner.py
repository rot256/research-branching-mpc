from pwn import *

ADDRESSES = [
    '127.0.0.1:%d' % (i + 7000) for i in range(8)
]

BUILD = True


with open('/tmp/players.txt', 'w') as f:
    for addr in ADDRESSES:
        f.write(addr + '\n')

def start_player(players, n):
    return process(
        'cd MP-SPDZ && ../mpc/bmpc ./semi-party.x -N %s -I -p %s bmpc' % (players, n),
        env = {'PLAYER_ADDRESSES':'/tmp/players.txt'},
        shell=True
    )

def start_random(players, n):
    return process(
        'cd MP-SPDZ && ./semi-party.x -N %s -p %s rmpc' % (players, n),
        shell=True
    )

import time


players = int(sys.argv[1])
branches = int(sys.argv[2])

# generate #branches * #length circuit to simulate naive approach
if sys.argv[3] == 'naive':
    RAND = True
elif sys.argv[3] == 'bmpc':
    RAND = False


length = 1 << 15

print('Random circuit:', RAND)
print('Players:', players)
print('Branches:', branches)

if BUILD and not RAND:

    print('Generate circuit and runner')

    p = process([
        'python3',
        './circuit.py',
        str(length),
        str(branches),
        str(players),
        'MP-SPDZ/Programs/Source/bmpc.mpc',
        'mpc/runner.go'
    ])

    p.wait_for_close()
    print(p.recvall())
    assert p.poll() == 0

    print('Compile runner')

    process('cd ./mpc && go build', shell=True).wait_for_close()

    print('Compile circuit')

    p = process([
        './MP-SPDZ/compile.py',
        '--prime=65537',
        'bmpc'
    ])

    while 1:
        try:
            print(p.recvline())
        except EOFError:
            break

    assert p.poll() == 0

elif RAND:
    p = process([
        'python3',
        './random_branches.py',
        str(branches),
        str(length),
        'MP-SPDZ/Programs/Source/rmpc.mpc',
    ])

    p.wait_for_close()
    print(p.recvall())
    assert p.poll() == 0

    p = process([
        './MP-SPDZ/compile.py',
        '--prime=65537',
        'rmpc'
    ])

    while 1:
        try:
            print(p.recvline())
        except EOFError:
            break

    assert p.poll() == 0


print('Run benchmark')

ses = []

if RAND:
    for p in range(players):
        print('Starting', p)
        time.sleep(0.3)
        ses.append(start_random(players, p))

else:
    for p in range(players):
        print('Starting', p)
        time.sleep(0.3)
        ses.append(start_player(players, p))

start = time.time()

for p in ses:
    p.wait_for_close()
    print(p.recvall())
    assert p.poll() == 0

end = time.time()

print('Time:', end - start)
