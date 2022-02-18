import yaml

'''
Automatically generates benchmark descriptions.
'''

def circuit(log_length, branches, per_layer=4096, naive=False):
    return {
        'type': 'layered-naive' if naive else 'layered',
        'parameters': {
            'per_layer': 4096,
            'length': 1 << log_length,
            'branches': branches
        }
    }

def cdn(players, branches, log_length, naive=False):
    name = 'auto-cdn{naive}-l{length}-b{branches}-p{players}.yml'.format(
        naive='-naive' if naive else '',
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
            'circuit': circuit(log_length, branches, naive=naive)
        }, f)

def mascot_semi(players, branches, log_length, naive=False):
    name = 'auto-mascot_semi{naive}-l{length}-b{branches}-p{players}.yml'.format(
        naive='-naive' if naive else '',
        players=players,
        branches=branches,
        length=log_length
    )
    with open(name, 'w') as f:
        yaml.safe_dump({
            'mpc': {
                'type': 'mascot_semi',
                'parties': players,
            },
            'circuit': circuit(log_length, branches, naive=naive)
        }, f)

for branches in range(1, 7):
    cdn(players = 3, branches=1 << branches, log_length=16)
    cdn(players = 3, branches=1 << branches, log_length=16, naive=True)
    mascot_semi(players = 3, branches=1 << branches, log_length=16)
    mascot_semi(players = 3, branches=1 << branches, log_length=16, naive=True)

for players in range(2, 9):
    cdn(players = players, branches=16, log_length=16)
    mascot_semi(players = players, branches=16, log_length=16)
