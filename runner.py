from pwn import *

ADDRESSES = [
    '127.0.0.1:%d' % (i + 7000) for i in range(200)
]

with open('/tmp/players.txt', 'w') as f:
    for addr in ADDRESSES:
        f.write(addr + '\n')

def start_player(players, n, params):
    cmd = 'cd MP-SPDZ && ../bmpc-{params} ./semi-party.x -N {players} -I -p {n} bmpc-{params}'.format(
        players=players,
        n=n,
        params=params
    )
    print(cmd)
    return process(
        cmd,
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

# check if already build

def follow(p):
    while 1:
        try:
            print(p.recvline())
        except EOFError:
            break
    assert p.poll() == 0

length = 1 << 15

print('Random circuit:', RAND)
print('Players:', players)
print('Branches:', branches)

params = '%s-%s-%s' % (length, branches, players)
rparams = '%s-%s' % (length, branches)

if not RAND:

    print('Generate circuit and runner')

    follow(process([
        'make',
        'bmpc-%s' % params,
        'MP-SPDZ/Programs/Schedules/bmpc-%s.sch' % params
    ]))

else:

    follow(process([
        'make',
        'MP-SPDZ/Programs/Schedules/rmpc-%s.sch' % rparams,
    ]))


print('Run benchmark')

ses = []

if RAND:
    for p in range(players):
        print('Starting', p)
        time.sleep(0.05)
        ses.append(start_random(players, p))

else:
    for p in range(players):
        print('Starting', p)
        time.sleep(0.05)
        ses.append(start_player(players, p, params))

start = time.time()

for p in ses:
    follow(p)

end = time.time()

print('Time:', end - start)
