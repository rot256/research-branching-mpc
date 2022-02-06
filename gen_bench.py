import yaml


def cdn(players, branches, log_length, naive=False):
    name = 'auto-cdn{naive}l{length}-b{branches}-p{players}.yml'.format(
        naive='-naive-' if naive else '',
        players=players,
        branches=branches,
        length=log_length
    )
    with open(name, 'w') as f:
        yaml.safe_dump({
            'mpc': {
                'type': 'cdn',
                'parties': players,
            },
            'circuit': {
                'type': 'layered-naive' if naive else 'layered',
                'parameters': {
                    'per_layer': 4096,
                    'length': 1 << log_length,
                    'branches': branches
                }
            }
        }, f)

for branches in range(1, 7):
    cdn(players = 3, branches=1 << branches, log_length=16)

for branches in range(1, 7):
    cdn(players = 3, branches=1 << branches, log_length=16, naive=True)


