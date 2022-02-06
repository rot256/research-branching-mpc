import sys
import yaml

def load(name):
    return yaml.safe_load(open(name, 'r'))

def avg(bench):
    return sum(bench['times']) / len(bench['times'])

def main():
    players = list(range(1, 17))

    for p in players:
        t = avg(load('bench-cdn-l16-b16-p%d.yml' % p))
        print(p, t)

if __name__ == '__main__':
    main()
