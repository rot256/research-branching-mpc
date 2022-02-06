import yaml


def cdn(players, branches, log_length):
    name = 'auto-cdn-l{length}-b{branches}-p{players}.yml'.format(
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
                'type': 'layered',
                'parameters': {
                    'per_layer': 4096,
                    'length': 1 << log_length,
                    'branches': branches
                }
            }
        }, f)

for branches in range(1, 7):
    cdn(players = 3, branches=1 << branches, log_length=16)


