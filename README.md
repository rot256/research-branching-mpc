# Resources

## Code

- [Awesome MPC](https://github.com/rdragos/awesome-mpc/blob/master/readme.md)
- [Lattigo](https://github.com/ldsec/lattigo)
- [MP-SPDZ](https://github.com/data61/MP-SPDZ)

## Papers

- [Compressible FHE with Applications to PIR](https://eprint.iacr.org/2019/733)

# Tools

## Building / Running

- [make]()
- [python3]() used to compile branching circuits into executable descriptions.
- [pyyaml]() used to parse test descriptions.
- [MP-SPDZ]() for implementations of generic MPC (not used for CDN).
- [Go (1.17 or later)]() implements OIP, the CDN MPC and wraps/interacts with MP-SPDZ.

## Benchmarking

For automatic benchmarking (as orchestrated by `runner.py`), we require the following tools / libraries:

- [pwntools](https://docs.pwntools.com/en/stable/) used to interact with processes.
- [tcpdump]() used to calculate the amount of network traffic.
- [Traffic Control (tc)]() used to simulate different network conditions (i.e. latency).

## Plotting / Dataanalysis

- [matplotlib]()

# Instructions

How to reproduce benchmarks.

```
sudo tc qdisc add dev lo root handle 1:0 netem delay 100msec

sudo tc qdisc del dev lo root
```

# Plotting

We use a number of scripts to compile the `.yaml` benchmark results into graphs.

