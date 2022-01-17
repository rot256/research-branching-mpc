MP-SPDZ/Programs/Source/bmpc-%.mpc runner-%.go: circuit.py
	python3 ./circuit.py $* MP-SPDZ/Programs/Source/bmpc-$*.mpc runner-$*.go

bmpc-%: runner-%.go $(wildcard mpc/*.go)
	cp runner-$*.go mpc/runner.go
	cd mpc && go build
	cp mpc/bmpc bmpc-$*
	rm mpc/runner.go

MP-SPDZ/Programs/Schedules/bmpc-%.sch: MP-SPDZ/Programs/Source/bmpc-%.mpc
	python3 ./MP-SPDZ/compile.py --prime=65537 $<

clean:
	rm -f bmpc-*
	rm -f runner-*.go
	rm -f MP-SPDZ/Programs/Source/bmpc-*.mpc
	rm -f MP-SPDZ/Programs/Schedules/bmpc-*.sch

.PHONY: clean
