import sys
import yaml

def load(name):
    return yaml.safe_load(open(name, 'r'))

def avg(bench, f):
    t = 0
    for s in bench['samples']:
        t += f(s)
    return t / len(bench['samples'])

def key_parties(bench):
    return bench['config']['mpc']['parties']

def key_branches(bench):
    return bench['config']['circuit']['parameters']['branches']

avg_comm = lambda bench: avg(bench, lambda s: s['comm'])
avg_time = lambda bench: avg(bench, lambda s: s['time'])

def sort(bench, f):
    return sorted(bench, key=lambda b: f(b['config']))

SMALL_SIZE = 12
MEDIUM_SIZE = 14
BIGGER_SIZE = 16

style_time = iter(['#48BF92', '#4E5BAD'])
style_comm = iter(['#f4b757', '#f13d36'])

if __name__ == '__main__':

    arg = iter(sys.argv[1:])
    out = next(arg)

    print('Generating:', out)

    x_axis = next(arg)
    y_axis = next(arg)

    # multiple y-axis supported
    y_axis = y_axis.split(',')
    assert len(y_axis) <= 2, 'at most 2 y-axis'

    import matplotlib.pyplot as plt
    import matplotlib

    fig = plt.figure()
    fig.set_size_inches(10, 6)

    ax = fig.add_subplot(1, 1, 1)
    ax.xaxis.set_tick_params(labelsize=MEDIUM_SIZE)

    plts = []

    # create multi-axies (same x)
    axies = [ax] + [ ax.twinx() for _ in range(1, len(y_axis)) ]
    print('Plot has %d y-axies' % len(axies))

    # sort accoding to x-axis
    if x_axis == 'parties':
        x_map = key_parties
    elif x_axis == 'branches':
        x_map = key_branches
    else:
        raise ValueError('not implemented')

    tot_xs = set([])

    while 1:
        try:
            label = next(arg)
            paths = next(arg)
        except StopIteration:
            break

        # load every file from dataset
        print(label, paths.split(','))
        bench = [load(p) for p in paths.split(',')]

        # sort according to x-axus
        bench = sorted(bench, key=x_map)

        xs = [x_map(b) for b in bench]

        tot_xs = tot_xs.union(set(xs))

        if len(set(xs)) != len(xs):
            raise ValueError('X-coordinates not distinct!')

        for axx, name in zip(axies, y_axis):
            ys = []
            for b in bench:
                if name == 'time':
                    ys.append(int(avg_time(b) * 1000))
                elif name == 'comm':
                    ys.append(int(avg_comm(b) / 1000))

            axx.yaxis.set_tick_params(labelsize=MEDIUM_SIZE)

            if name == 'time':
                color = next(style_time)
                a, = axx.plot(xs, ys, '--', label='Time: %s' % label, color=color, alpha=0.7, marker='D')
                axx.set_ylabel('Wall Time (ms)', fontsize=BIGGER_SIZE)
                axx.set_ylim(ymin=0, ymax=int(1.05 * max(ys)))
            elif name == 'comm':
                color = next(style_comm)
                a, = axx.plot(xs, ys, ':', label='Comm %s' % label, color=color, alpha=0.7, marker='o')
                axx.set_ylabel('Total Communication (KB)', fontsize=BIGGER_SIZE)
                axx.set_ylim(ymin=0, ymax=int(1.05 * max(ys)))
            else:
                ValueError('Not Supported')

            plts.append(a)

    # branches are benchmarked in powers of 2
    if x_axis == 'branches':
        ax.set_xscale('log',base=2)
        ax.set_xticks(sorted(tot_xs))
        ax.set_xlabel('Branches', fontsize=BIGGER_SIZE)
    elif x_axis == 'parties':
        ax.set_xticks(sorted(tot_xs))
        ax.set_xlabel('Number of parties', fontsize=BIGGER_SIZE)

    for axx in axies:
        axx.margins()

    ax.legend(handles=plts, loc='lower right')

    if x_axis == 'parties':
        plt.title()

    plt.tight_layout()
    plt.savefig(out, transparent=True) #), dpi=200)












