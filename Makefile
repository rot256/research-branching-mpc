MP-SPDZ/Programs/Source/bmpc-%.mpc runner-%.go: circuit.py
	python3 ./circuit.py $*.yml # compile from yml description

bmpc-%: runner-%.go $(wildcard mpc/*.go)
	cp runner-$*.go mpc/runner.go
	cd mpc && go build
	cp mpc/bmpc bmpc-$*
	cp null-runner.go mpc/runner.go

MP-SPDZ/Programs/Schedules/bmpc-%s.sch: MP-SPDZ/Programs/Source/bmpc-%s.mpc
	python3 ./MP-SPDZ/compile.py --prime=65537 $<

MP-SPDZ/Programs/Source/rmpc-%s.mpc:
	python3 ./random_branches.py $* $@

MP-SPDZ/Programs/Schedules/rmpc-%s.sch: MP-SPDZ/Programs/Source/rmpc-%s.mpc
	python3 ./MP-SPDZ/compile.py --prime=65537 $<

clean:
	rm -f bmpc-*
	rm -f runner-*.go
	rm -f MP-SPDZ/Programs/Source/bmpc-*.mpc
	rm -f MP-SPDZ/Programs/Source/rmpc-*.mpc
	rm -f MP-SPDZ/Programs/Schedules/bmpc-*.sch
	rm -f MP-SPDZ/Programs/Schedules/rmpc-*.sch
	rm -f bench-*.yml

bench-%.yml: bmpc-% runner.py
	python3 runner.py $*.yml 10

.SECONDARY:

.PHONY: clean
