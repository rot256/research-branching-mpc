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

class NetCapture:
    def __init__(self):
        # remove old capture
        follow(process(['sudo', 'rm', '-f', '/tmp/bench.pcap']))

        # start new capture
        self.p = process(['sudo', 'tcpdump', '--interface', 'lo', '-w', '/tmp/bench.pcap'])
        self.p.recvuntil('tcpdump: listening on lo')

    def stop(self):
        self.p.kill()

        # upper bound
        self.total = os.path.getsize('/tmp/bench.pcap')

        return self.total

        t = process(['tshark', '-nr', '/tmp/bench.pcap', '-T', 'fields', '-e', 'frame.len'])
        # t = process(['tshark', '-r', '/tmp/bench.pcap', '-T', 'text', '-V'])

        total = 0

        while 1:
            try:
                cnt = int(t.recvline().decode('utf-8'))
                total += cnt
                print('total', total)
            except EOFError:
                break

        return total

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

    samples = []

    started = time.time()

    for _ in range(repetitions):

        # start players

        ses = []

        processes = {}

        parties = mpc['parties']

        net = NetCapture()

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

        total = net.stop()

        samples.append({
            'time': end - start,
            'comm': total
        })


    stopped = time.time()

    # log the start/stop time, the samples (of course) and the input configuration
    bench = {
        'started': started,
        'stopped': stopped,
        'samples': samples,
        'config': config
    }

    # get CPU info
    with open('bench-%s.yml' % name, 'w') as f:
        yaml.safe_dump(bench, f)


if __name__ == '__main__':
    main()

