import yaml
import time

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

def start_cdn(binary, players, n):
    return process(
        './%s -N %s -p %s' % (binary, players, n),
        env = {'PLAYER_ADDRESSES':'/tmp/players.txt'},
        shell=True
    )

def follow(p):
    while 1:
        try:
            print(p.recvline().decode('utf-8').strip())
        except EOFError:
            break
    assert p.poll() == 0

WAIT = 0.05

def main():


    path = sys.argv[1]
    name = os.path.basename(path)
    assert name.endswith('.yml')
    name = name[:-len('.yml')]

    with open(sys.argv[1], 'r') as f:
        config = yaml.safe_load(f)

    repetitions = 1
    if len(sys.argv) > 2:
        repetitions = int(sys.argv[2])


    # build prereqs for benchmark

    mpc = config['mpc']

    if mpc['type'] == 'cdn':
        follow(process([
            'make',
            'bmpc-%s' % name,
        ]))
    else:
        raise ValueError('Not impl')

    times = []

    for _ in range(repetitions):

        # start players

        ses = []

        parties = mpc['parties']

        if mpc['type'] == 'cdn':
            for p in range(parties):
                print('Starting', p)
                time.sleep(WAIT)
                ses.append(start_cdn(
                    'bmpc-%s' % name,
                    parties,
                    p
                ))
        else:
            raise ValueError('Not impl')

        start = time.time()

        # wait for all players to terminate

        for p in ses:
            follow(p)

        end = time.time()

        # save result to benchmark file
        times.append(end - start)

    with open('bench-%s.yml' % name, 'w') as f:
        yaml.safe_dump({
            'times': times
        }, f)



if __name__ == '__main__':
    main()

