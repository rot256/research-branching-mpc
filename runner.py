from pwn import *

ADDRESSES = [
    '127.0.0.1:%d' % (i + 7000) for i in range(8)
]

BUILD = False

with open('/tmp/players.txt', 'w') as f:
    for addr in ADDRESSES:
        f.write(addr + '\n')

def start_player(players, n):
    return process(
        [
            './mpc/bmpc',
            './MP-SPDZ/semi-party.x',
            '-N',
            str(players),
            '-I',
            '-p',
            str(n),
            'bmpc'
        ], env = {
            'PLAYER_ADDRESSES':'/tmp/players.txt'
        })

import time

players = 4

if BUILD:

    print('Generate circuit and runner')

    process([
        'python3',
        './circuit.py',
        '32768',
        '2',
        str(players),
        'MP-SPDZ/Programs/Source/bmpc.mpc',
        'mpc/runner.go'
    ]).wait_for_close()

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

print('Run benchmark')

ses = []

for p in range(players):
    print('Starting', p)
    time.sleep(0.3)
    ses.append(start_player(players, p))

for p in ses:
    p.interactive()



