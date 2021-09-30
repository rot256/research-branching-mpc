module bmpc

replace github.com/ldsec/lattigo/v2 => ./lattigo

replace github.com/ldsec/lattigo/v2/rlwe => ./lattigo/rlwe

replace github.com/ldsec/lattigo/v2/bfv => ./lattigo/rlwe

require github.com/ldsec/lattigo/v2 v2.2.0

go 1.15
