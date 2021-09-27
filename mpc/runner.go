package main

func run(player  int, inputs []uint64, mpc *MPC, oip *OIP) {
nxt := 0
if player == 0 {
    mpc.TryInput([]uint64{inputs[nxt]})
    nxt += 1
}
if player == 0 {
    mpc.TryInput([]uint64{inputs[nxt]})
    nxt += 1
}
if player == 0 {
    mpc.TryInput([]uint64{inputs[nxt]})
    nxt += 1
}
if player == 1 {
    mpc.TryInput([]uint64{inputs[nxt]})
    nxt += 1
}
if player == 1 {
    mpc.TryInput([]uint64{inputs[nxt]})
    nxt += 1
}
func() {
mapping := [][]int{
    {0,1,1,2,3,0},
    {0,1,1,2,4,1},
}
out := random(6)
mpc.TryInput(out)
t := random(2)
mpc.TryInput(t)
b := mpc.TryOutput(2)
if player == 0 {
    for i := 0; i < 2; i++ {
        b[i] = (b[i] + t[i]) % PRIME
    }
} else {
    for i := 0; i < 2; i++ {
        b[i] = t[i]
    }
}
D := oip.TryOIPMapping(mapping, b, out)
mpc.TryInput(D)
}()
}