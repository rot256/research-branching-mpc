MP-SPDZ/Programs/Source/bmpc-%.mpc runner-%.go: circuit.py
	python3 ./circuit.py $* MP-SPDZ/Programs/Source/bmpc-$*.mpc runner-$*.go

bmpc-%: runner-%.go $(wildcard mpc/*.go)
	cp runner-$*.go mpc/runner.go
	cd mpc && go build
	cp mpc/bmpc bmpc-$*
	cp runner-default.go mpc/runner.go

MP-SPDZ/Programs/Schedules/bmpc-%.sch: MP-SPDZ/Programs/Source/bmpc-%.mpc
	python3 ./MP-SPDZ/compile.py --prime=65537 $<

MP-SPDZ/Programs/Source/rmpc-%.mpc:
	python3 ./random_branches.py $* $@

MP-SPDZ/Programs/Schedules/rmpc-%.sch: MP-SPDZ/Programs/Source/rmpc-%.mpc
	python3 ./MP-SPDZ/compile.py --prime=65537 $<

clean:
	rm -f bmpc-*
	rm -f runner-*.go
	rm -f MP-SPDZ/Programs/Source/bmpc-*.mpc
	rm -f MP-SPDZ/Programs/Schedules/bmpc-*.sch

.PHONY: clean
