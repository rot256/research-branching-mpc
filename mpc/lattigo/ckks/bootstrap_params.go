package ckks

import (
	"math"

	"github.com/ldsec/lattigo/v2/rlwe"
	"github.com/ldsec/lattigo/v2/utils"
	//"fmt"
)

// SinType is the type of function used during the bootstrapping
// for the homomorphic modular reduction
type SinType uint64

// Sin and Cos are the two proposed functions for SinType
const (
	Sin  = SinType(0) // Standard Chebyshev approximation of (1/2pi) * sin(2pix)
	Cos1 = SinType(1) // Special approximation (Han and Ki) of pow((1/2pi), 1/2^r) * cos(2pi(x-0.25)/2^r)
	Cos2 = SinType(2) // Standard Chebyshev approximation of pow((1/2pi), 1/2^r) * cos(2pi(x-0.25)/2^r)
)

// BootstrappingParameters is a struct for the default bootstrapping parameters
type BootstrappingParameters struct {
	ResidualModuli
	KeySwitchModuli
	SlotsToCoeffsModuli
	SineEvalModuli
	CoeffsToSlotsModuli
	LogN         int
	LogSlots     int
	Scale        float64
	Sigma        float64
	H            int     // Hamming weight of the secret key
	SinType      SinType // Chose betwenn [Sin(2*pi*x)] or [cos(2*pi*x/r) with double angle formula]
	MessageRatio float64 // Ratio between Q0 and m, i.e. Q[0]/|m|
	SinRange     int     // K parameter (interpolation in the range -K to K)
	SinDeg       int     // Degree of the interpolation
	SinRescal    int     // Number of rescale and double angle formula (only applies for cos)
	ArcSineDeg   int     // Degree of the Taylor arcsine composed with f(2*pi*x) (if zero then not used)
	MaxN1N2Ratio float64 // n1/n2 ratio for the bsgs algo for matrix x vector eval
	BitReversed  bool    // Flag for bit-reverseed input to the DFT (with bit-reversed output), by default false.
}

// Params generates a new set of Parameters from the BootstrappingParameters
func (b *BootstrappingParameters) Params() (p Parameters, err error) {
	Qi := append(b.ResidualModuli, b.SlotsToCoeffsModuli.Qi...)
	Qi = append(Qi, b.SineEvalModuli.Qi...)
	Qi = append(Qi, b.CoeffsToSlotsModuli.Qi...)

	if p, err = NewParametersFromLiteral(ParametersLiteral{
		Q:        Qi,
		P:        b.KeySwitchModuli,
		LogN:     b.LogN,
		Sigma:    b.Sigma,
		Scale:    b.Scale,
		LogSlots: b.LogSlots,
	}); err != nil {
		return Parameters{}, err
	}
	return
}

// Copy return a new BootstrappingParameters which is a copy of the target
func (b *BootstrappingParameters) Copy() *BootstrappingParameters {
	paramsCopy := &BootstrappingParameters{
		LogN:         b.LogN,
		LogSlots:     b.LogSlots,
		Scale:        b.Scale,
		Sigma:        b.Sigma,
		H:            b.H,
		SinType:      b.SinType,
		MessageRatio: b.MessageRatio,
		SinRange:     b.SinRange,
		SinDeg:       b.SinDeg,
		SinRescal:    b.SinRescal,
		ArcSineDeg:   b.ArcSineDeg,
		MaxN1N2Ratio: b.MaxN1N2Ratio,
		BitReversed:  b.BitReversed,
	}

	// KeySwitchModuli
	paramsCopy.KeySwitchModuli = make([]uint64, len(b.KeySwitchModuli))
	copy(paramsCopy.KeySwitchModuli, b.KeySwitchModuli)

	// ResidualModuli
	paramsCopy.ResidualModuli = make([]uint64, len(b.ResidualModuli))
	copy(paramsCopy.ResidualModuli, b.ResidualModuli)

	// CoeffsToSlotsModuli
	paramsCopy.CoeffsToSlotsModuli.Qi = make([]uint64, b.CtSDepth(true))
	copy(paramsCopy.CoeffsToSlotsModuli.Qi, b.CoeffsToSlotsModuli.Qi)

	paramsCopy.CoeffsToSlotsModuli.ScalingFactor = make([][]float64, b.CtSDepth(true))
	for i := range paramsCopy.CoeffsToSlotsModuli.ScalingFactor {
		paramsCopy.CoeffsToSlotsModuli.ScalingFactor[i] = make([]float64, len(b.CoeffsToSlotsModuli.ScalingFactor[i]))
		copy(paramsCopy.CoeffsToSlotsModuli.ScalingFactor[i], b.CoeffsToSlotsModuli.ScalingFactor[i])
	}

	// SineEvalModuli
	paramsCopy.SineEvalModuli.Qi = make([]uint64, len(b.SineEvalModuli.Qi))
	copy(paramsCopy.SineEvalModuli.Qi, b.SineEvalModuli.Qi)
	paramsCopy.SineEvalModuli.ScalingFactor = b.SineEvalModuli.ScalingFactor

	// SlotsToCoeffsModuli
	paramsCopy.SlotsToCoeffsModuli.Qi = make([]uint64, b.StCDepth(true))
	copy(paramsCopy.SlotsToCoeffsModuli.Qi, b.SlotsToCoeffsModuli.Qi)

	paramsCopy.SlotsToCoeffsModuli.ScalingFactor = make([][]float64, b.StCDepth(true))
	for i := range paramsCopy.SlotsToCoeffsModuli.ScalingFactor {
		paramsCopy.SlotsToCoeffsModuli.ScalingFactor[i] = make([]float64, len(b.SlotsToCoeffsModuli.ScalingFactor[i]))
		copy(paramsCopy.SlotsToCoeffsModuli.ScalingFactor[i], b.SlotsToCoeffsModuli.ScalingFactor[i])
	}

	return paramsCopy
}

// ResidualModuli is a list of the moduli available after the bootstrapping.
type ResidualModuli []uint64

// KeySwitchModuli is a list of the special moduli used for the key-switching.
type KeySwitchModuli []uint64

// CoeffsToSlotsModuli is a list of the moduli used during he CoeffsToSlots step.
type CoeffsToSlotsModuli struct {
	Qi            []uint64
	ScalingFactor [][]float64
}

// SineEvalModuli is a list of the moduli used during the SineEval step.
type SineEvalModuli struct {
	Qi            []uint64
	ScalingFactor float64
}

// SlotsToCoeffsModuli is a list of the moduli used during the SlotsToCoeffs step.
type SlotsToCoeffsModuli struct {
	Qi            []uint64
	ScalingFactor [][]float64
}

// MaxLevel returns the maximum level of the bootstrapping parameters
func (b *BootstrappingParameters) MaxLevel() int {
	return len(b.ResidualModuli) + len(b.CoeffsToSlotsModuli.Qi) + len(b.SineEvalModuli.Qi) + len(b.SlotsToCoeffsModuli.Qi) - 1
}

// SineEvalDepth returns the depth of the SineEval. If true, then also
// counts the double angle formula.
func (b *BootstrappingParameters) SineEvalDepth(withRescale bool) int {
	depth := int(math.Ceil(math.Log2(float64(b.SinDeg + 1))))

	if withRescale {
		depth += b.SinRescal
	}

	return depth
}

// ArcSineDepth returns the depth of the arcsine polynomial.
func (b *BootstrappingParameters) ArcSineDepth() int {
	return int(math.Ceil(math.Log2(float64(b.ArcSineDeg + 1))))
}

// CtSDepth returns the number of levels allocated to CoeffsToSlots.
// If actual == true then returns the number of moduli consumed, else
// returns the factorization depth.
func (b *BootstrappingParameters) CtSDepth(actual bool) (depth int) {
	if actual {
		depth = len(b.CoeffsToSlotsModuli.ScalingFactor)
	} else {
		for i := range b.CoeffsToSlotsModuli.ScalingFactor {
			for range b.CoeffsToSlotsModuli.ScalingFactor[i] {
				depth++
			}
		}
	}

	return
}

// CtSLevels returns the index of the Qi used int CoeffsToSlots
func (b *BootstrappingParameters) CtSLevels() (ctsLevel []int) {
	ctsLevel = []int{}
	for i := range b.CoeffsToSlotsModuli.Qi {
		for range b.CoeffsToSlotsModuli.ScalingFactor[b.CtSDepth(true)-1-i] {
			ctsLevel = append(ctsLevel, b.MaxLevel()-i)
		}
	}

	return
}

// StCDepth returns the number of levels allocated to SlotToCoeffs.
// If actual == true then returns the number of moduli consumed, else
// returns the factorization depth.
func (b *BootstrappingParameters) StCDepth(actual bool) (depth int) {
	if actual {
		depth = len(b.SlotsToCoeffsModuli.ScalingFactor)
	} else {
		for i := range b.SlotsToCoeffsModuli.ScalingFactor {
			for range b.SlotsToCoeffsModuli.ScalingFactor[i] {
				depth++
			}
		}
	}

	return
}

// StCLevels returns the index of the Qi used in SlotsToCoeffs
func (b *BootstrappingParameters) StCLevels() (stcLevel []int) {
	stcLevel = []int{}
	for i := range b.SlotsToCoeffsModuli.Qi {
		for range b.SlotsToCoeffsModuli.ScalingFactor[b.StCDepth(true)-1-i] {
			stcLevel = append(stcLevel, b.MaxLevel()-b.CtSDepth(true)-b.SineEvalDepth(true)-b.ArcSineDepth()-i)
		}
	}

	return
}

// GenCoeffsToSlotsMatrix generates the factorized encoding matrix
// scaling : constant by witch the all the matrices will be multuplied by
// encoder : ckks.Encoder
func (b *BootstrappingParameters) GenCoeffsToSlotsMatrix(scaling complex128, encoder Encoder) []*PtDiagMatrix {

	logSlots := b.LogSlots
	slots := 1 << logSlots
	depth := b.CtSDepth(false)
	logdSlots := logSlots + 1
	if logdSlots == b.LogN {
		logdSlots--
	}

	roots := computeRoots(slots << 1)
	pow5 := make([]int, (slots<<1)+1)
	pow5[0] = 1
	for i := 1; i < (slots<<1)+1; i++ {
		pow5[i] = pow5[i-1] * 5
		pow5[i] &= (slots << 2) - 1
	}

	ctsLevels := b.CtSLevels()

	// CoeffsToSlots vectors
	pDFTInv := make([]*PtDiagMatrix, len(ctsLevels))
	pVecDFTInv := computeDFTMatrices(logSlots, logdSlots, depth, roots, pow5, scaling, true, b.BitReversed)
	cnt := 0
	for i := range b.CoeffsToSlotsModuli.ScalingFactor {
		for j := range b.CoeffsToSlotsModuli.ScalingFactor[b.CtSDepth(true)-i-1] {
			pDFTInv[cnt] = encoder.EncodeDiagMatrixBSGSAtLvl(ctsLevels[cnt], pVecDFTInv[cnt], b.CoeffsToSlotsModuli.ScalingFactor[b.CtSDepth(true)-i-1][j], b.MaxN1N2Ratio, logdSlots)
			cnt++
		}
	}

	return pDFTInv
}

// GenSlotsToCoeffsMatrix generates the factorized decoding matrix
// scaling : constant by witch the all the matrices will be multuplied by
// encoder : ckks.Encoder
func (b *BootstrappingParameters) GenSlotsToCoeffsMatrix(scaling complex128, encoder Encoder) []*PtDiagMatrix {

	logSlots := b.LogSlots
	slots := 1 << logSlots
	depth := b.StCDepth(false)
	logdSlots := logSlots + 1
	if logdSlots == b.LogN {
		logdSlots--
	}

	roots := computeRoots(slots << 1)
	pow5 := make([]int, (slots<<1)+1)
	pow5[0] = 1
	for i := 1; i < (slots<<1)+1; i++ {
		pow5[i] = pow5[i-1] * 5
		pow5[i] &= (slots << 2) - 1
	}

	stcLevels := b.StCLevels()

	// CoeffsToSlots vectors
	pDFT := make([]*PtDiagMatrix, len(stcLevels))
	pVecDFT := computeDFTMatrices(logSlots, logdSlots, depth, roots, pow5, scaling, false, b.BitReversed)
	cnt := 0
	for i := range b.SlotsToCoeffsModuli.ScalingFactor {
		for j := range b.SlotsToCoeffsModuli.ScalingFactor[b.StCDepth(true)-i-1] {
			pDFT[cnt] = encoder.EncodeDiagMatrixBSGSAtLvl(stcLevels[cnt], pVecDFT[cnt], b.SlotsToCoeffsModuli.ScalingFactor[b.StCDepth(true)-i-1][j], b.MaxN1N2Ratio, logdSlots)
			cnt++
		}
	}

	return pDFT
}

// RotationsForCoeffsToSlots returns the list of rotations performed during the CoeffsToSlot operation.
func (b *BootstrappingParameters) RotationsForCoeffsToSlots(logSlots int) (rotations []int) {
	rotations = []int{}

	slots := 1 << logSlots
	dslots := slots
	if logSlots < b.LogN-1 {
		dslots <<= 1
		rotations = append(rotations, slots)
	}

	indexCtS := computeBootstrappingDFTIndexMap(b.LogN, logSlots, b.CtSDepth(false), true, b.BitReversed)

	// Coeffs to Slots rotations
	for _, pVec := range indexCtS {
		N1 := findbestbabygiantstepsplit(pVec, dslots, b.MaxN1N2Ratio)
		rotations = addMatrixRotToList(pVec, rotations, N1, slots, false)
	}

	return
}

// RotationsForSlotsToCoeffs returns the list of rotations performed during the SlotsToCoeffs operation.
func (b *BootstrappingParameters) RotationsForSlotsToCoeffs(logSlots int) (rotations []int) {
	rotations = []int{}

	slots := 1 << logSlots
	dslots := slots
	if logSlots < b.LogN-1 {
		dslots <<= 1
	}

	indexStC := computeBootstrappingDFTIndexMap(b.LogN, logSlots, b.StCDepth(false), false, b.BitReversed)

	// Slots to Coeffs rotations
	for i, pVec := range indexStC {
		N1 := findbestbabygiantstepsplit(pVec, dslots, b.MaxN1N2Ratio)
		rotations = addMatrixRotToList(pVec, rotations, N1, slots, logSlots < b.LogN-1 && i == 0)
	}

	return
}

// RotationsForBootstrapping returns the list of rotations performed during the Bootstrapping operation.
func (b *BootstrappingParameters) RotationsForBootstrapping(logSlots int) (rotations []int) {

	// List of the rotation key values to needed for the bootstrapp
	rotations = []int{}

	slots := 1 << logSlots
	dslots := slots
	if logSlots < b.LogN-1 {
		dslots <<= 1
	}

	//SubSum rotation needed X -> Y^slots rotations
	for i := logSlots; i < b.LogN-1; i++ {
		if !utils.IsInSliceInt(1<<i, rotations) {
			rotations = append(rotations, 1<<i)
		}
	}

	indexCtS := computeBootstrappingDFTIndexMap(b.LogN, logSlots, b.CtSDepth(false), true, b.BitReversed)

	// Coeffs to Slots rotations
	for _, pVec := range indexCtS {
		N1 := findbestbabygiantstepsplit(pVec, dslots, b.MaxN1N2Ratio)
		rotations = addMatrixRotToList(pVec, rotations, N1, slots, false)
	}

	indexStC := computeBootstrappingDFTIndexMap(b.LogN, logSlots, b.StCDepth(false), false, b.BitReversed)

	// Slots to Coeffs rotations
	for i, pVec := range indexStC {
		N1 := findbestbabygiantstepsplit(pVec, dslots, b.MaxN1N2Ratio)
		rotations = addMatrixRotToList(pVec, rotations, N1, slots, logSlots < b.LogN-1 && i == 0)
	}

	return
}

// DefaultBootstrapParams are default bootstrapping params for the bootstrapping.
var DefaultBootstrapParams = []*BootstrappingParameters{

	// SET I
	// 1546
	{
		LogN:     16,
		LogSlots: 15,
		Scale:    1 << 40,
		Sigma:    rlwe.DefaultSigma,
		ResidualModuli: []uint64{
			0x10000000006e0001, // 60 Q0
			0x10000140001,      // 40
			0xffffe80001,       // 40
			0xffffc40001,       // 40
			0x100003e0001,      // 40
			0xffffb20001,       // 40
			0x10000500001,      // 40
			0xffff940001,       // 40
			0xffff8a0001,       // 40
			0xffff820001,       // 40
		},
		KeySwitchModuli: []uint64{
			0x1fffffffffe00001, // Pi 61
			0x1fffffffffc80001, // Pi 61
			0x1fffffffffb40001, // Pi 61
			0x1fffffffff500001, // Pi 61
			0x1fffffffff420001, // Pi 61
		},
		SlotsToCoeffsModuli: SlotsToCoeffsModuli{
			Qi: []uint64{
				0x7fffe60001, // 39 StC
				0x7fffe40001, // 39 StC
				0x7fffe00001, // 39 StC
			},
			ScalingFactor: [][]float64{
				{0x7fffe60001},
				{0x7fffe40001},
				{0x7fffe00001},
			},
		},
		SineEvalModuli: SineEvalModuli{
			Qi: []uint64{
				0xfffffffff840001,  // 60 Sine (double angle)
				0x1000000000860001, // 60 Sine (double angle)
				0xfffffffff6a0001,  // 60 Sine
				0x1000000000980001, // 60 Sine
				0xfffffffff5a0001,  // 60 Sine
				0x1000000000b00001, // 60 Sine
				0x1000000000ce0001, // 60 Sine
				0xfffffffff2a0001,  // 60 Sine
			},
			ScalingFactor: 1 << 60,
		},
		CoeffsToSlotsModuli: CoeffsToSlotsModuli{
			Qi: []uint64{
				0x100000000060001, // 58 CtS
				0xfffffffff00001,  // 58 CtS
				0xffffffffd80001,  // 58 CtS
				0x1000000002a0001, // 58 CtS
			},
			ScalingFactor: [][]float64{
				{0x100000000060001},
				{0xfffffffff00001},
				{0xffffffffd80001},
				{0x1000000002a0001},
			},
		},
		H:            192,
		SinType:      Cos1,
		MessageRatio: 256.0,
		SinRange:     25,
		SinDeg:       63,
		SinRescal:    2,
		ArcSineDeg:   0,
		MaxN1N2Ratio: 16.0,
		BitReversed:  false,
	},

	// SET II
	// 1547
	{
		LogN:     16,
		LogSlots: 15,
		Scale:    1 << 45,
		Sigma:    rlwe.DefaultSigma,
		ResidualModuli: []uint64{
			0x10000000006e0001, // 60 Q0
			0x2000000a0001,     // 45
			0x2000000e0001,     // 45
			0x1fffffc20001,     // 45
			0x200000440001,     // 45
			0x200000500001,     // 45
		},
		KeySwitchModuli: []uint64{
			0x1fffffffffe00001, // Pi 61
			0x1fffffffffc80001, // Pi 61
			0x1fffffffffb40001, // Pi 61
			0x1fffffffff500001, // Pi 61
		},
		SlotsToCoeffsModuli: SlotsToCoeffsModuli{
			Qi: []uint64{
				0x3ffffe80001, //42 StC
				0x3ffffd20001, //42 StC
				0x3ffffca0001, //42 StC
			},
			ScalingFactor: [][]float64{
				{0x3ffffe80001},
				{0x3ffffd20001},
				{0x3ffffca0001},
			},
		},
		SineEvalModuli: SineEvalModuli{
			Qi: []uint64{
				0xffffffffffc0001,  // ArcSine
				0xfffffffff240001,  // ArcSine
				0x1000000000f00001, // ArcSine
				0xfffffffff840001,  // Double angle
				0x1000000000860001, // Double angle
				0xfffffffff6a0001,  // Sine
				0x1000000000980001, // Sine
				0xfffffffff5a0001,  // Sine
				0x1000000000b00001, // Sine
				0x1000000000ce0001, // Sine
				0xfffffffff2a0001,  // Sine
			},
			ScalingFactor: 1 << 60,
		},
		CoeffsToSlotsModuli: CoeffsToSlotsModuli{
			Qi: []uint64{
				0x400000000360001, // 58 CtS
				0x3ffffffffbe0001, // 58 CtS
				0x400000000660001, // 58 CtS
				0x4000000008a0001, // 58 CtS
			},
			ScalingFactor: [][]float64{
				{0x400000000360001},
				{0x3ffffffffbe0001},
				{0x400000000660001},
				{0x4000000008a0001},
			},
		},
		H:            192,
		SinType:      Cos1,
		MessageRatio: 4.0,
		SinRange:     25,
		SinDeg:       63,
		SinRescal:    2,
		ArcSineDeg:   7,
		MaxN1N2Ratio: 16.0,
		BitReversed:  false,
	},

	// SET III
	// 1553
	{
		LogN:     16,
		LogSlots: 15,
		Scale:    1 << 30,
		Sigma:    rlwe.DefaultSigma,
		ResidualModuli: []uint64{
			0x80000000080001,   // 55 Q0
			0xffffffffffc0001,  // 60
			0x10000000006e0001, // 60
			0xfffffffff840001,  // 60
			0x1000000000860001, // 60
			0xfffffffff6a0001,  // 60
			0x1000000000980001, // 60
			0xfffffffff5a0001,  // 60
		},
		KeySwitchModuli: []uint64{
			0x1fffffffffe00001, // Pi 61
			0x1fffffffffc80001, // Pi 61
			0x1fffffffffb40001, // Pi 61
			0x1fffffffff500001, // Pi 61
			0x1fffffffff420001, // Pi 61
		},
		SlotsToCoeffsModuli: SlotsToCoeffsModuli{
			Qi: []uint64{
				0x1000000000b00001, // 60 StC  (30)
				0x1000000000ce0001, // 60 StC  (30+30)
			},
			ScalingFactor: [][]float64{
				{1073741824.0},
				{1073741824.0062866, 1073741824.0062866},
			},
		},
		SineEvalModuli: SineEvalModuli{
			Qi: []uint64{
				0x80000000440001, // 55 Sine (double angle)
				0x7fffffffba0001, // 55 Sine (double angle)
				0x80000000500001, // 55 Sine
				0x7fffffffaa0001, // 55 Sine
				0x800000005e0001, // 55 Sine
				0x7fffffff7e0001, // 55 Sine
				0x7fffffff380001, // 55 Sine
				0x80000000ca0001, // 55 Sine
			},
			ScalingFactor: 1 << 55,
		},
		CoeffsToSlotsModuli: CoeffsToSlotsModuli{
			Qi: []uint64{
				0x200000000e0001, // 53 CtS
				0x20000000140001, // 53 CtS
				0x20000000280001, // 53 CtS
				0x1fffffffd80001, // 53 CtS
			},
			ScalingFactor: [][]float64{
				{0x200000000e0001},
				{0x20000000140001},
				{0x20000000280001},
				{0x1fffffffd80001},
			},
		},
		H:            192,
		SinType:      Cos1,
		MessageRatio: 256.0,
		SinRange:     25,
		SinDeg:       63,
		SinRescal:    2,
		ArcSineDeg:   0,
		MaxN1N2Ratio: 16.0,
		BitReversed:  false,
	},

	// Set IV
	// 1792
	{
		LogN:     16,
		LogSlots: 15,
		Scale:    1 << 40,
		Sigma:    rlwe.DefaultSigma,
		ResidualModuli: []uint64{
			0x4000000120001, // 60 Q0
			0x10000140001,
			0xffffe80001,
			0xffffc40001,
			0x100003e0001,
			0xffffb20001,
			0x10000500001,
			0xffff940001,
			0xffff8a0001,
			0xffff820001,
		},
		KeySwitchModuli: []uint64{
			0x1fffffffffe00001, // Pi 61
			0x1fffffffffc80001, // Pi 61
			0x1fffffffffb40001, // Pi 61
			0x1fffffffff500001, // Pi 61
			0x1fffffffff420001, // Pi 61
			0x1fffffffff380001, // Pi 61
		},
		SlotsToCoeffsModuli: SlotsToCoeffsModuli{
			Qi: []uint64{
				0x100000000060001, // 56 StC (28 + 28)
				0xffa0001,         // 28 StC
			},
			ScalingFactor: [][]float64{
				{268435456.0007324, 268435456.0007324},
				{0xffa0001},
			},
		},
		SineEvalModuli: SineEvalModuli{
			Qi: []uint64{
				0xffffffffffc0001,  // 60 Sine (double angle)
				0x10000000006e0001, // 60 Sine (double angle)
				0xfffffffff840001,  // 60 Sine (double angle)
				0x1000000000860001, // 60 Sine (double angle)
				0xfffffffff6a0001,  // 60 Sine
				0x1000000000980001, // 60 Sine
				0xfffffffff5a0001,  // 60 Sine
				0x1000000000b00001, // 60 Sine
				0x1000000000ce0001, // 60 Sine
				0xfffffffff2a0001,  // 60 Sine
				0xfffffffff240001,  // 60 Sine
				0x1000000000f00001, // 60 Sine
			},
			ScalingFactor: 1 << 60,
		},
		CoeffsToSlotsModuli: CoeffsToSlotsModuli{
			Qi: []uint64{
				0x200000000e0001, // 53 CtS
				0x20000000140001, // 53 CtS
				0x20000000280001, // 53 CtS
				0x1fffffffd80001, // 53 CtS
			},
			ScalingFactor: [][]float64{
				{0x200000000e0001},
				{0x20000000140001},
				{0x20000000280001},
				{0x1fffffffd80001},
			},
		},
		H:            32768,
		SinType:      Cos2,
		MessageRatio: 256.0,
		SinRange:     325,
		SinDeg:       255,
		SinRescal:    4,
		ArcSineDeg:   0,
		MaxN1N2Ratio: 16.0,
		BitReversed:  false,
	},

	// Set V
	// 768
	{
		LogN:     15,
		LogSlots: 14,
		Scale:    1 << 25,
		Sigma:    rlwe.DefaultSigma,
		ResidualModuli: []uint64{
			0x1fff90001,     // 32 Q0
			0x4000000420001, // 50
			0x1fc0001,       // 25
		},
		KeySwitchModuli: []uint64{
			0x7fffffffe0001, // 51
			0x8000000110001, // 51
		},
		SlotsToCoeffsModuli: SlotsToCoeffsModuli{
			Qi: []uint64{
				0xffffffffffc0001, // 60 StC (30+30)
			},
			ScalingFactor: [][]float64{
				{1073741823.9998779, 1073741823.9998779},
			},
		},
		SineEvalModuli: SineEvalModuli{
			Qi: []uint64{
				0x4000000120001, // 50 Sine
				0x40000001b0001, // 50 Sine
				0x3ffffffdf0001, // 50 Sine
				0x4000000270001, // 50 Sine
				0x3ffffffd20001, // 50 Sine
				0x3ffffffcd0001, // 50 Sine
				0x4000000350001, // 50 Sine
				0x3ffffffc70001, // 50 Sine
			},
			ScalingFactor: 1 << 50,
		},
		CoeffsToSlotsModuli: CoeffsToSlotsModuli{
			Qi: []uint64{
				0x1fffffff50001, // 49 CtS
				0x1ffffffea0001, // 49 CtS
			},
			ScalingFactor: [][]float64{
				{0x1fffffff50001},
				{0x1ffffffea0001},
			},
		},
		H:            192,
		SinType:      Cos1,
		MessageRatio: 256.0,
		SinRange:     25,
		SinDeg:       63,
		SinRescal:    2,
		ArcSineDeg:   0,
		MaxN1N2Ratio: 16.0,
		BitReversed:  false,
	},
}

func computeRoots(N int) (roots []complex128) {

	var angle float64

	m := N << 1

	roots = make([]complex128, m)

	roots[0] = 1

	for i := 1; i < m; i++ {
		angle = 6.283185307179586 * float64(i) / float64(m)
		roots[i] = complex(math.Cos(angle), math.Sin(angle))
	}

	return
}

func fftPlainVec(logN, dslots int, roots []complex128, pow5 []int) (a, b, c [][]complex128) {

	var N, m, index, tt, gap, k, mask, idx1, idx2 int

	N = 1 << logN

	a = make([][]complex128, logN)
	b = make([][]complex128, logN)
	c = make([][]complex128, logN)

	var size int
	if 2*N == dslots {
		size = 2
	} else {
		size = 1
	}

	index = 0
	for m = 2; m <= N; m <<= 1 {

		a[index] = make([]complex128, dslots)
		b[index] = make([]complex128, dslots)
		c[index] = make([]complex128, dslots)

		tt = m >> 1

		for i := 0; i < N; i += m {

			gap = N / m
			mask = (m << 2) - 1

			for j := 0; j < m>>1; j++ {

				k = (pow5[j] & mask) * gap

				idx1 = i + j
				idx2 = i + j + tt

				for u := 0; u < size; u++ {
					a[index][idx1+u*N] = 1
					a[index][idx2+u*N] = -roots[k]
					b[index][idx1+u*N] = roots[k]
					c[index][idx2+u*N] = 1
				}
			}
		}

		index++
	}

	return
}

func fftInvPlainVec(logN, dslots int, roots []complex128, pow5 []int) (a, b, c [][]complex128) {

	var N, m, index, tt, gap, k, mask, idx1, idx2 int

	N = 1 << logN

	a = make([][]complex128, logN)
	b = make([][]complex128, logN)
	c = make([][]complex128, logN)

	var size int
	if 2*N == dslots {
		size = 2
	} else {
		size = 1
	}

	index = 0
	for m = N; m >= 2; m >>= 1 {

		a[index] = make([]complex128, dslots)
		b[index] = make([]complex128, dslots)
		c[index] = make([]complex128, dslots)

		tt = m >> 1

		for i := 0; i < N; i += m {

			gap = N / m
			mask = (m << 2) - 1

			for j := 0; j < m>>1; j++ {

				k = ((m << 2) - (pow5[j] & mask)) * gap

				idx1 = i + j
				idx2 = i + j + tt

				for u := 0; u < size; u++ {

					a[index][idx1+u*N] = 1
					a[index][idx2+u*N] = -roots[k]
					b[index][idx1+u*N] = 1
					c[index][idx2+u*N] = roots[k]
				}
			}
		}

		index++
	}

	return
}

func addMatrixRotToList(pVec map[int]bool, rotations []int, N1, slots int, repack bool) []int {

	if len(pVec) < 3 {
		for j := range pVec {
			if !utils.IsInSliceInt(j, rotations) {
				rotations = append(rotations, j)
			}
		}
	} else {
		var index int
		for j := range pVec {

			index = (j / N1) * N1

			if repack {
				// Sparse repacking, occurring during the first DFT matrix of the CoeffsToSlots.
				index &= (2*slots - 1)
			} else {
				// Other cases
				index &= (slots - 1)
			}

			if index != 0 && !utils.IsInSliceInt(index, rotations) {
				rotations = append(rotations, index)
			}

			index = j & (N1 - 1)

			if index != 0 && !utils.IsInSliceInt(index, rotations) {
				rotations = append(rotations, index)
			}
		}
	}

	return rotations
}

func computeBootstrappingDFTIndexMap(logN, logSlots, maxDepth int, forward, bitreversed bool) (rotationMap []map[int]bool) {

	var level, depth, nextLevel int

	level = logSlots

	rotationMap = make([]map[int]bool, maxDepth)

	// We compute the chain of merge in order or reverse order depending if its DFT or InvDFT because
	// the way the levels are collapsed has an impact on the total number of rotations and keys to be
	// stored. Ex. instead of using 255 + 64 plaintext vectors, we can use 127 + 128 plaintext vectors
	// by reversing the order of the merging.
	merge := make([]int, maxDepth)
	for i := 0; i < maxDepth; i++ {

		depth = int(math.Ceil(float64(level) / float64(maxDepth-i)))

		if forward {
			merge[i] = depth
		} else {
			merge[len(merge)-i-1] = depth

		}

		level -= depth
	}

	level = logSlots
	for i := 0; i < maxDepth; i++ {

		if logSlots < logN-1 && !forward && i == 0 {

			// Special initial matrix for the repacking before SlotsToCoeffs
			rotationMap[i] = genWfftRepackIndexMap(logSlots, level)

			// Merges this special initial matrix with the first layer of SlotsToCoeffs DFT
			rotationMap[i] = nextLevelfftIndexMap(rotationMap[i], logSlots, 2<<logSlots, level, forward, bitreversed)

			// Continues the merging with the next layers if the total depth requires it.
			nextLevel = level - 1
			for j := 0; j < merge[i]-1; j++ {
				rotationMap[i] = nextLevelfftIndexMap(rotationMap[i], logSlots, 2<<logSlots, nextLevel, forward, bitreversed)
				nextLevel--
			}

		} else {
			// First layer of the i-th level of the DFT
			rotationMap[i] = genWfftIndexMap(logSlots, level, forward, bitreversed)

			// Merges the layer with the next levels of the DFT if the total depth requires it.
			nextLevel = level - 1
			for j := 0; j < merge[i]-1; j++ {
				rotationMap[i] = nextLevelfftIndexMap(rotationMap[i], logSlots, 1<<logSlots, nextLevel, forward, bitreversed)
				nextLevel--
			}
		}

		level -= merge[i]
	}

	return
}

func genWfftIndexMap(logL, level int, forward, bitreversed bool) (vectors map[int]bool) {

	var rot int

	if forward && !bitreversed || !forward && bitreversed {
		rot = 1 << (level - 1)
	} else {
		rot = 1 << (logL - level)
	}

	vectors = make(map[int]bool)
	vectors[0] = true
	vectors[rot] = true
	vectors[(1<<logL)-rot] = true
	return
}

func genWfftRepackIndexMap(logL, level int) (vectors map[int]bool) {
	vectors = make(map[int]bool)
	vectors[0] = true
	vectors[(1 << logL)] = true
	return
}

func nextLevelfftIndexMap(vec map[int]bool, logL, N, nextLevel int, forward, bitreversed bool) (newVec map[int]bool) {

	var rot int

	newVec = make(map[int]bool)

	if forward && !bitreversed || !forward && bitreversed {
		rot = (1 << (nextLevel - 1)) & (N - 1)
	} else {
		rot = (1 << (logL - nextLevel)) & (N - 1)
	}

	for i := range vec {
		newVec[i] = true
		newVec[(i+rot)&(N-1)] = true
		newVec[(i-rot)&(N-1)] = true
	}

	return
}

func computeDFTMatrices(logSlots, logdSlots, maxDepth int, roots []complex128, pow5 []int, diffscale complex128, inverse, bitreversed bool) (plainVector []map[int][]complex128) {

	var fftLevel, depth, nextfftLevel int

	fftLevel = logSlots

	var a, b, c [][]complex128

	if inverse {
		a, b, c = fftInvPlainVec(logSlots, 1<<logdSlots, roots, pow5)
	} else {
		a, b, c = fftPlainVec(logSlots, 1<<logdSlots, roots, pow5)
	}

	plainVector = make([]map[int][]complex128, maxDepth)

	// We compute the chain of merge in order or reverse order depending if its DFT or InvDFT because
	// the way the levels are collapsed has an inpact on the total number of rotations and keys to be
	// stored. Ex. instead of using 255 + 64 plaintext vectors, we can use 127 + 128 plaintext vectors
	// by reversing the order of the merging.
	merge := make([]int, maxDepth)
	for i := 0; i < maxDepth; i++ {

		depth = int(math.Ceil(float64(fftLevel) / float64(maxDepth-i)))

		if inverse {
			merge[i] = depth
		} else {
			merge[len(merge)-i-1] = depth

		}

		fftLevel -= depth
	}

	fftLevel = logSlots
	for i := 0; i < maxDepth; i++ {

		if logSlots != logdSlots && !inverse && i == 0 {

			// Special initial matrix for the repacking before SlotsToCoeffs
			plainVector[i] = genRepackMatrix(logSlots, bitreversed)

			// Merges this special initial matrix with the first layer of SlotsToCoeffs DFT
			plainVector[i] = multiplyFFTMatrixWithNextFFTLevel(plainVector[i], logSlots, 2<<logSlots, fftLevel, a[logSlots-fftLevel], b[logSlots-fftLevel], c[logSlots-fftLevel], inverse, bitreversed)

			// Continues the merging with the next layers if the total depth requires it.
			nextfftLevel = fftLevel - 1
			for j := 0; j < merge[i]-1; j++ {
				plainVector[i] = multiplyFFTMatrixWithNextFFTLevel(plainVector[i], logSlots, 2<<logSlots, nextfftLevel, a[logSlots-nextfftLevel], b[logSlots-nextfftLevel], c[logSlots-nextfftLevel], inverse, bitreversed)
				nextfftLevel--
			}

		} else {
			// First layer of the i-th level of the DFT
			plainVector[i] = genFFTDiagMatrix(logSlots, fftLevel, a[logSlots-fftLevel], b[logSlots-fftLevel], c[logSlots-fftLevel], inverse, bitreversed)

			// Merges the layer with the next levels of the DFT if the total depth requires it.
			nextfftLevel = fftLevel - 1
			for j := 0; j < merge[i]-1; j++ {
				plainVector[i] = multiplyFFTMatrixWithNextFFTLevel(plainVector[i], logSlots, 1<<logSlots, nextfftLevel, a[logSlots-nextfftLevel], b[logSlots-nextfftLevel], c[logSlots-nextfftLevel], inverse, bitreversed)
				nextfftLevel--
			}
		}

		fftLevel -= merge[i]
	}

	// Repacking after the CoeffsToSlots (we multiply the last DFT matrix with the vector [1, 1, ..., 1, 1, 0, 0, ..., 0, 0]).
	if logSlots != logdSlots && inverse {
		for j := range plainVector[maxDepth-1] {
			for x := 0; x < 1<<logSlots; x++ {
				plainVector[maxDepth-1][j][x+(1<<logSlots)] = complex(0, 0)
			}
		}
	}

	// Rescaling of the DFT matrix of the SlotsToCoeffs/CoeffsToSlots
	for j := range plainVector {
		for x := range plainVector[j] {
			for i := range plainVector[j][x] {
				plainVector[j][x][i] *= diffscale
			}
		}
	}

	/*
		for j := range plainVector {
			for x := 0; x < 1<<logSlots; x++{
				if plainVector[j][x] != nil{
					fmt.Printf("%d :", x)
					for i := range plainVector[j][x] {
						fmt.Printf("%0.4f", plainVector[j][x][i])
					}
					fmt.Printf("\n")
				}

			}
			fmt.Printf("\n")
		}
	*/

	return
}

func genFFTDiagMatrix(logL, fftLevel int, a, b, c []complex128, forward, bitreversed bool) (vectors map[int][]complex128) {

	var rot int

	if forward && !bitreversed || !forward && bitreversed {
		rot = 1 << (fftLevel - 1)
	} else {
		rot = 1 << (logL - fftLevel)
	}

	vectors = make(map[int][]complex128)

	if bitreversed {
		sliceBitReverseInPlaceComplex128(a, 1<<logL)
		sliceBitReverseInPlaceComplex128(b, 1<<logL)
		sliceBitReverseInPlaceComplex128(c, 1<<logL)

		if len(a) > 1<<logL {
			sliceBitReverseInPlaceComplex128(a[1<<logL:], 1<<logL)
			sliceBitReverseInPlaceComplex128(b[1<<logL:], 1<<logL)
			sliceBitReverseInPlaceComplex128(c[1<<logL:], 1<<logL)
		}
	}

	addToDiagMatrix(vectors, 0, a)
	addToDiagMatrix(vectors, rot, b)
	addToDiagMatrix(vectors, (1<<logL)-rot, c)

	return
}

func genRepackMatrix(logL int, bitreversed bool) (vectors map[int][]complex128) {

	vectors = make(map[int][]complex128)

	a := make([]complex128, 2<<logL)
	b := make([]complex128, 2<<logL)

	for i := 0; i < 1<<logL; i++ {
		a[i] = complex(1, 0)
		a[i+(1<<logL)] = complex(0, 1)

		b[i] = complex(0, 1)
		b[i+(1<<logL)] = complex(1, 0)
	}

	addToDiagMatrix(vectors, 0, a)
	addToDiagMatrix(vectors, (1 << logL), b)

	return
}

func multiplyFFTMatrixWithNextFFTLevel(vec map[int][]complex128, logL, N, nextLevel int, a, b, c []complex128, forward, bitreversed bool) (newVec map[int][]complex128) {

	var rot int

	newVec = make(map[int][]complex128)

	if forward && !bitreversed || !forward && bitreversed {
		rot = (1 << (nextLevel - 1)) & (N - 1)
	} else {
		rot = (1 << (logL - nextLevel)) & (N - 1)
	}

	if bitreversed {
		sliceBitReverseInPlaceComplex128(a, 1<<logL)
		sliceBitReverseInPlaceComplex128(b, 1<<logL)
		sliceBitReverseInPlaceComplex128(c, 1<<logL)

		if len(a) > 1<<logL {
			sliceBitReverseInPlaceComplex128(a[1<<logL:], 1<<logL)
			sliceBitReverseInPlaceComplex128(b[1<<logL:], 1<<logL)
			sliceBitReverseInPlaceComplex128(c[1<<logL:], 1<<logL)
		}
	}

	/*
		fmt.Printf("W[%d]\n", nextLevel)
		for x := 0; x < 1<<logL; x++{
			if vec[x] != nil{
				fmt.Printf("%d :", x)
				for i := range vec[x] {
					fmt.Printf("%0.4f", vec[x][i])
				}
				fmt.Printf("\n")
			}

		}
		fmt.Printf("\n")

		fmt.Printf("W[%d]\n", nextLevel-1)
		fmt.Printf("%d :", 0)
		for i := range a {
			fmt.Printf("%0.4f", a[i])
		}
		fmt.Printf("\n")

		fmt.Printf("%d :", rot)
		for i := range b {
			fmt.Printf("%0.4f", b[i])
		}
		fmt.Printf("\n")

		fmt.Printf("%d :", N-rot)
		for i := range c {
			fmt.Printf("%0.4f", c[i])
		}
		fmt.Printf("\n")
		fmt.Println()
	*/

	for i := range vec {
		addToDiagMatrix(newVec, i, mul(vec[i], a))
		addToDiagMatrix(newVec, (i+rot)&(N-1), mul(rotate(vec[i], rot), b))
		addToDiagMatrix(newVec, (i-rot)&(N-1), mul(rotate(vec[i], -rot), c))
	}

	/*
		fmt.Printf("W[%d] x W[%d]\n", nextLevel, nextLevel-1)
		for x := 0; x < 1<<logL; x++{
			if newVec[x] != nil{
				fmt.Printf("%d :", x)
				for i := range newVec[x] {
					fmt.Printf("%0.4f", newVec[x][i])
				}
				fmt.Printf("\n")
			}

		}
		fmt.Printf("\n")
	*/

	return
}

func transposeDiagMatrix(mat map[int][]complex128, N int) {
	for i := range mat {
		if i < N>>1 {
			mat[i], mat[N-i] = mat[N-i], mat[i]
		}
	}
}

func conjugateDiagMatrix(mat map[int][]complex128) {
	for i := range mat {

		for j := range mat[i] {
			c := mat[i][j]
			mat[i][j] = complex(real(c), -imag(c))
		}
	}
}

func genBitReverseDiagMatrix(logN int) (diagMat map[int][]complex128) {

	var N, iRev, diff int

	diagMat = make(map[int][]complex128)

	N = 1 << logN

	for i := 0; i < N; i++ {
		iRev = int(utils.BitReverse64(uint64(i), uint64(logN)))

		diff = (i - iRev) & (N - 1)

		if diagMat[diff] == nil {
			diagMat[diff] = make([]complex128, N)
		}

		diagMat[diff][iRev] = complex(1, 0)
	}

	return
}

func addToDiagMatrix(diagMat map[int][]complex128, index int, vec []complex128) {
	if diagMat[index] == nil {
		diagMat[index] = vec
	} else {
		diagMat[index] = add(diagMat[index], vec)
	}
}

func rotate(x []complex128, n int) (y []complex128) {

	y = make([]complex128, len(x))

	mask := int(len(x) - 1)

	// Rotates to the left
	for i := 0; i < len(x); i++ {
		y[i] = x[(i+n)&mask]
	}

	return
}

func mul(a, b []complex128) (res []complex128) {

	res = make([]complex128, len(a))

	for i := 0; i < len(a); i++ {
		res[i] = a[i] * b[i]
	}

	return
}

func add(a, b []complex128) (res []complex128) {

	res = make([]complex128, len(a))

	for i := 0; i < len(a); i++ {
		res[i] = a[i] + b[i]
	}

	return
}
