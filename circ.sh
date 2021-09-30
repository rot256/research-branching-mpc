#!/bin/bash

python3 circuit.py MP-SPDZ/Programs/Source/bmpc.mpc mpc/runner.go

cd mpc && go build && cd ..

cd MP-SPDZ && ./compile.py --prime=65537 bmpc && cd ..
