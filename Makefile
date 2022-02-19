plots: p3-semi-mascot-branches-plot.png p3-cdn-branches-plot.png parties-plot.png

clean:
	rm -f bmpc-*
	rm -f runner-*.go
	rm -f MP-SPDZ/Programs/Source/bmpc-*.mpc
	rm -f MP-SPDZ/Programs/Source/rmpc-*.mpc
	rm -f MP-SPDZ/Programs/Schedules/bmpc-*.sch
	rm -f MP-SPDZ/Programs/Schedules/rmpc-*.sch
	rm -f bench-*.yml
	rm -f *.png

rebench:
	rm -f bench-*.yml

# MP-SPDZ/Programs/Source/bmpc-%.mpc runner-%.go: circuit.py
MP-SPDZ/Programs/Source/bmpc-%.mpc runner-%.go:
	python3 ./circuit.py $*.yml # compile from yml description

bmpc-%: runner-%.go
	cp runner-$*.go mpc/runner.go
	cd mpc && go build
	cp mpc/bmpc bmpc-$*
	cp null-runner.go mpc/runner.go

MP-SPDZ/Programs/Schedules/bmpc-%.sch: MP-SPDZ/Programs/Source/bmpc-%.mpc
	python3 ./MP-SPDZ/compile.py --prime=65537 $<

MP-SPDZ/Programs/Source/rmpc-%.mpc:
	python3 ./random_branches.py $* $@

MP-SPDZ/Programs/Schedules/rmpc-%.sch: MP-SPDZ/Programs/Source/rmpc-%.mpc
	python3 ./MP-SPDZ/compile.py --prime=65537 $<

# bench-%.yml: %.yml bmpc-% runner.py
bench-%.yml:
	python3 runner.py $*.yml 20

# auto-generate required benchmark descriptions
auto-%.yml:
	python3 gen_bench.py

p3-cdn-branches-plot.png: plot.py \
	bench-auto-cdn-l16-b2-p3.yml \
	bench-auto-cdn-l16-b4-p3.yml \
	bench-auto-cdn-l16-b8-p3.yml \
	bench-auto-cdn-l16-b16-p3.yml \
	bench-auto-cdn-l16-b32-p3.yml \
	bench-auto-cdn-l16-b64-p3.yml \
	bench-auto-cdn-naive-l16-b2-p3.yml \
	bench-auto-cdn-naive-l16-b4-p3.yml \
	bench-auto-cdn-naive-l16-b8-p3.yml \
	bench-auto-cdn-naive-l16-b16-p3.yml \
	bench-auto-cdn-naive-l16-b32-p3.yml \
	bench-auto-cdn-naive-l16-b64-p3.yml
	python3 plot.py $@ \
		"Branching MPC with Semi-Honest CDN (3 Parties, 2^16 Gates Per Branch)" \
		branches \
		time,comm \
		"CDN Branching" bench-auto-cdn-l16-b2-p3.yml,bench-auto-cdn-l16-b4-p3.yml,bench-auto-cdn-l16-b8-p3.yml,bench-auto-cdn-l16-b16-p3.yml,bench-auto-cdn-l16-b32-p3.yml,bench-auto-cdn-l16-b64-p3.yml \
		"CDN Parallel" bench-auto-cdn-naive-l16-b2-p3.yml,bench-auto-cdn-naive-l16-b4-p3.yml,bench-auto-cdn-naive-l16-b8-p3.yml,bench-auto-cdn-naive-l16-b16-p3.yml,bench-auto-cdn-naive-l16-b32-p3.yml,bench-auto-cdn-naive-l16-b64-p3.yml

p3-semi-mascot-branches-plot.png: plot.py \
	bench-auto-mascot_semi-l16-b2-p3.yml \
	bench-auto-mascot_semi-l16-b4-p3.yml \
	bench-auto-mascot_semi-l16-b8-p3.yml \
	bench-auto-mascot_semi-l16-b16-p3.yml \
	bench-auto-mascot_semi-l16-b32-p3.yml \
	bench-auto-mascot_semi-l16-b64-p3.yml \
	bench-auto-mascot_semi-naive-l16-b2-p3.yml \
	bench-auto-mascot_semi-naive-l16-b4-p3.yml \
	bench-auto-mascot_semi-naive-l16-b8-p3.yml \
	bench-auto-mascot_semi-naive-l16-b16-p3.yml \
	bench-auto-mascot_semi-naive-l16-b32-p3.yml \
	bench-auto-mascot_semi-naive-l16-b64-p3.yml
	python3 plot.py $@ \
		"Branching MPC with Semi-Honest MASCOT (3 Parties, 2^16 Gates Per Branch)" \
		branches \
		time,comm \
		"Semi-MASCOT Branching" bench-auto-mascot_semi-l16-b2-p3.yml,bench-auto-mascot_semi-l16-b4-p3.yml,bench-auto-mascot_semi-l16-b8-p3.yml,bench-auto-mascot_semi-l16-b16-p3.yml,bench-auto-mascot_semi-l16-b32-p3.yml,bench-auto-mascot_semi-l16-b64-p3.yml \
		"Semi-MASCOT Parallel" bench-auto-mascot_semi-naive-l16-b2-p3.yml,bench-auto-mascot_semi-naive-l16-b4-p3.yml,bench-auto-mascot_semi-naive-l16-b8-p3.yml,bench-auto-mascot_semi-naive-l16-b16-p3.yml,bench-auto-mascot_semi-naive-l16-b32-p3.yml,bench-auto-mascot_semi-naive-l16-b64-p3.yml

parties-plot.png: plot.py \
	bench-auto-cdn-l16-b16-p2.yml \
	bench-auto-cdn-l16-b16-p3.yml \
	bench-auto-cdn-l16-b16-p4.yml \
	bench-auto-cdn-l16-b16-p5.yml \
	bench-auto-cdn-l16-b16-p6.yml \
	bench-auto-cdn-l16-b16-p7.yml \
	bench-auto-cdn-l16-b16-p8.yml \
	bench-auto-mascot_semi-l16-b16-p2.yml \
	bench-auto-mascot_semi-l16-b16-p3.yml \
	bench-auto-mascot_semi-l16-b16-p4.yml \
	bench-auto-mascot_semi-l16-b16-p5.yml \
	bench-auto-mascot_semi-l16-b16-p6.yml \
	bench-auto-mascot_semi-l16-b16-p7.yml \
	bench-auto-mascot_semi-l16-b16-p8.yml
	python3 plot.py $@ \
		"Branching MPC With Different Num. of Parties (16 Branches of 2^16 Gates)" \
		parties \
		time,comm \
		"Branching MPC (CDN)" bench-auto-cdn-l16-b16-p2.yml,bench-auto-cdn-l16-b16-p3.yml,bench-auto-cdn-l16-b16-p4.yml,bench-auto-cdn-l16-b16-p5.yml,bench-auto-cdn-l16-b16-p6.yml,bench-auto-cdn-l16-b16-p7.yml,bench-auto-cdn-l16-b16-p8.yml \
		"Branching MPC (MASCOT)" bench-auto-mascot_semi-l16-b16-p2.yml,bench-auto-mascot_semi-l16-b16-p3.yml,bench-auto-mascot_semi-l16-b16-p4.yml,bench-auto-mascot_semi-l16-b16-p5.yml,bench-auto-mascot_semi-l16-b16-p6.yml,bench-auto-mascot_semi-l16-b16-p7.yml,bench-auto-mascot_semi-l16-b16-p8.yml

.SECONDARY:

.PHONY: rebench clean plots
