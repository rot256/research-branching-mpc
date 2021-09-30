package ckks

import (
	"fmt"
	"math"
	"sort"
)

// PrecisionStats is a struct storing statistic about the precision of a CKKS plaintext
type PrecisionStats struct {
	MaxDelta        complex128
	MinDelta        complex128
	MaxPrecision    complex128
	MinPrecision    complex128
	MeanDelta       complex128
	MeanPrecision   complex128
	MedianDelta     complex128
	MedianPrecision complex128
	STDFreq         float64
	STDTime         float64

	RealDist, ImagDist []struct {
		Prec  float64
		Count int
	}

	cdfResol int
}

func (prec PrecisionStats) String() string {
	return fmt.Sprintf("\nMIN Prec : (%.2f, %.2f) Log2 \n", real(prec.MinPrecision), imag(prec.MinPrecision)) +
		fmt.Sprintf("MAX Prec : (%.2f, %.2f) Log2 \n", real(prec.MaxPrecision), imag(prec.MaxPrecision)) +
		fmt.Sprintf("AVG Prec : (%.2f, %.2f) Log2 \n", real(prec.MeanPrecision), imag(prec.MeanPrecision)) +
		fmt.Sprintf("MED Prec : (%.2f, %.2f) Log2 \n", real(prec.MedianPrecision), imag(prec.MedianPrecision)) +
		fmt.Sprintf("Err stdF : %5.2f Log2 \n", math.Log2(prec.STDFreq)) +
		fmt.Sprintf("Err stdT : %5.2f Log2 \n", math.Log2(prec.STDTime))

}

// GetPrecisionStats generates a PrecisionStats struct from the reference values and the decrypted values
func GetPrecisionStats(params Parameters, encoder Encoder, decryptor Decryptor, valuesWant []complex128, element interface{}, logSlots int, sigma float64) (prec PrecisionStats) {

	var valuesTest []complex128

	slots := uint64(1 << logSlots)

	switch element := element.(type) {
	case *Ciphertext:
		valuesTest = encoder.DecodePublic(decryptor.DecryptNew(element), logSlots, sigma)
	case *Plaintext:
		valuesTest = encoder.DecodePublic(element, logSlots, sigma)
	case []complex128:
		valuesTest = element
	}

	var deltaReal, deltaImag float64

	var delta complex128

	diff := make([]complex128, slots)

	prec.MaxDelta = complex(0, 0)
	prec.MinDelta = complex(1, 1)

	prec.MeanDelta = complex(0, 0)

	prec.cdfResol = 500

	prec.RealDist = make([]struct {
		Prec  float64
		Count int
	}, prec.cdfResol)
	prec.ImagDist = make([]struct {
		Prec  float64
		Count int
	}, prec.cdfResol)

	precReal := make([]float64, len(valuesWant))
	precImag := make([]float64, len(valuesWant))

	for i := range valuesWant {

		delta = valuesTest[i] - valuesWant[i]
		deltaReal = math.Abs(real(delta))
		deltaImag = math.Abs(imag(delta))
		precReal[i] = math.Log2(1 / deltaReal)
		precImag[i] = math.Log2(1 / deltaImag)

		diff[i] += complex(deltaReal, deltaImag)

		prec.MeanDelta += diff[i]

		if deltaReal > real(prec.MaxDelta) {
			prec.MaxDelta = complex(deltaReal, imag(prec.MaxDelta))
		}

		if deltaImag > imag(prec.MaxDelta) {
			prec.MaxDelta = complex(real(prec.MaxDelta), deltaImag)
		}

		if deltaReal < real(prec.MinDelta) {
			prec.MinDelta = complex(deltaReal, imag(prec.MinDelta))
		}

		if deltaImag < imag(prec.MinDelta) {
			prec.MinDelta = complex(real(prec.MinDelta), deltaImag)
		}
	}

	prec.calcCDF(precReal, prec.RealDist)
	prec.calcCDF(precImag, prec.ImagDist)

	prec.MinPrecision = deltaToPrecision(prec.MaxDelta)
	prec.MaxPrecision = deltaToPrecision(prec.MinDelta)
	prec.MeanDelta /= complex(float64(slots), 0)
	prec.MeanPrecision = deltaToPrecision(prec.MeanDelta)
	prec.MedianDelta = calcmedian(diff)
	prec.MedianPrecision = deltaToPrecision(prec.MedianDelta)
	prec.STDFreq = encoder.GetErrSTDSlotDomain(valuesWant[:], valuesTest[:], params.Scale())
	prec.STDTime = encoder.GetErrSTDCoeffDomain(valuesWant, valuesTest, params.Scale())
	return prec
}

func deltaToPrecision(c complex128) complex128 {
	return complex(math.Log2(1/real(c)), math.Log2(1/imag(c)))
}

func (prec *PrecisionStats) calcCDF(precs []float64, res []struct {
	Prec  float64
	Count int
}) {
	sortedPrecs := make([]float64, len(precs))
	copy(sortedPrecs, precs)
	sort.Float64s(sortedPrecs)
	minPrec := sortedPrecs[0]
	maxPrec := sortedPrecs[len(sortedPrecs)-1]
	for i := 0; i < prec.cdfResol; i++ {
		curPrec := minPrec + float64(i)*(maxPrec-minPrec)/float64(prec.cdfResol)
		for countSmaller, p := range sortedPrecs {
			if p >= curPrec {
				res[i].Prec = curPrec
				res[i].Count = countSmaller
				break
			}
		}
	}
}

func calcmedian(values []complex128) (median complex128) {

	tmp := make([]float64, len(values))

	for i := range values {
		tmp[i] = real(values[i])
	}

	sort.Float64s(tmp)

	for i := range values {
		values[i] = complex(tmp[i], imag(values[i]))
	}

	for i := range values {
		tmp[i] = imag(values[i])
	}

	sort.Float64s(tmp)

	for i := range values {
		values[i] = complex(real(values[i]), tmp[i])
	}

	index := len(values) / 2

	if len(values)&1 == 1 {
		return values[index]
	}

	if index+1 == len(values) {
		return values[index]
	}

	return (values[index] + values[index+1]) / 2
}
