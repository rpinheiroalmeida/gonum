// Copyright ©2014 The gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dist

import (
	"math"
	"math/rand"

	"github.com/gonum/floats"
	"github.com/gonum/stat"
)

const (
	// oneOverRoot2Pi is the value of 1/(2Pi)^(1/2)
	// http://www.wolframalpha.com/input/?i=1%2F%282+*+pi%29%5E%281%2F2%29
	oneOverRoot2Pi = 0.39894228040143267793994605993438186847585863116493465766592582967065792589930183850125233390730693643030255886263518268

	//LogRoot2Pi is the value of log(sqrt(2*Pi))
	logRoot2Pi    = 0.91893853320467274178032973640561763986139747363778341281715154048276569592726039769474329863595419762200564662463433744
	negLogRoot2Pi = -logRoot2Pi
	log2Pi        = 1.8378770664093454835606594728112352797227949472755668
)

// UnitNormal is an instantiation of the standard normal distribution
var UnitNormal = Normal{Mu: 0, Sigma: 1}

// Normal respresents a normal (Gaussian) distribution (https://en.wikipedia.org/wiki/Normal_distribution).
type Normal struct {
	Mu     float64 // Mean of the normal distribution
	Sigma  float64 // Standard deviation of the normal distribution
	Source *rand.Rand

	// Needs to be Mu and Sigma and not Mean and StdDev because Normal has functions
	// Mean and StdDev
}

// CDF computes the value of the cumulative density function at x.
func (n Normal) CDF(x float64) float64 {
	return 0.5 * (1 + math.Erf((x-n.Mu)/(n.Sigma*math.Sqrt2)))
}

// DLogProbDX computes the derivative of the log of the probability with respect
// to the input x.
func (n Normal) DLogProbDX(x float64) float64 {
	return -(1 / (2 * n.Sigma * n.Sigma)) * 2 * (x - n.Mu)
}

// DLogProbDParam returns the derivative of the log of the probability with
// respect to the parameters of the distribution. The deriv slice must have length
// equal to the number of parameters of the distribution.
//
// The order is first ∂LogProb / ∂Mu and then ∂LogProb / ∂Sigma
func (n Normal) DLogProbDParam(x float64, deriv []float64) {
	if len(deriv) != n.NumParameters() {
		panic("dist: slice length mismatch")
	}

	deriv[0] = n.Mu * (x - n.Mu) / (n.Sigma * n.Sigma)
	deriv[1] = 1 / n.Sigma * (-1 + (x-n.Mu)*(x-n.Mu)/2.0)

	return
}

// Entropy returns the differential entropy of the distribution.
func (n Normal) Entropy() float64 {
	return 0.5 * (log2Pi + 1 + 2*math.Log(n.Sigma))
}

// ExKurtosis returns the excess kurtosis of the distribution.
func (Normal) ExKurtosis() float64 {
	return 0
}

// Fit sets the parameters of the probability distribution from the
// data samples x with relative weights w. If weights is nil, then all the weights
// are 1. If weights is not nil, then the len(weights) must equal len(samples).
func (n *Normal) Fit(samples []float64, weights []float64) {
	n.FitPrior(samples, weights, nil, nil)
}

// FitPrior fits the distribution with a set of priors for the sufficient
// statistics. If priorValue and priorWeights both have length 0, no prior is used.
// For the normal distribution, there are two prior values. The first is the
// prior guess for the mean, and the second is the effective standard deviation.
// The strength priorWeight is how many effective samples that prior
// is worth assuming a normal-inverse-gamma prior. In other words, this means
// seeing priorStrength[0] samples with mean priorValue[0] and priorStrength[1]
// samples with mean priorValue[0] and standard deviation priorValue[1].
//
// The output is updated values of the prior after observing the input samples.
func (n *Normal) FitPrior(samples []float64, weights []float64, priorValue []float64, priorWeight []float64) (newPriorValue, newPriorWeight []float64) {
	// Error checking and initialization
	lenSamp := len(samples)
	lenPriorValue := len(priorValue)
	lenPriorWeight := len(priorWeight)

	if len(weights) != 0 && len(samples) != len(weights) {
		panic("dist: slice size mismatch")
	}
	if lenPriorValue != lenPriorWeight {
		panic("normal: mismatch in prior lengths")
	}
	if lenPriorValue > 2 {
		panic("normal: too many prior values")
	}
	prior := true
	if lenPriorValue == 0 || lenPriorWeight == 0 {
		if lenPriorValue == 0 && lenPriorWeight == 0 {
			prior = false
		} else if lenPriorValue == 0 && lenPriorWeight != 0 {
			panic("normal: prior weight provided but not the value")
		} else {
			panic("normal: prior value provided but not the weight")
		}
	}

	sampleMean := stat.Mean(samples, weights)
	sampleVariance := stat.Moment(2, samples, sampleMean, weights) // Don't want it corrected

	var sumWeights float64
	if len(weights) == 0 {
		sumWeights = float64(lenSamp)
	} else {
		sumWeights = floats.Sum(weights)
	}

	totalWeight := sumWeights
	totalSum := sampleMean * sumWeights
	if prior {
		totalWeight += priorWeight[0]
		totalSum += priorValue[0] * priorWeight[0]
	}

	n.Mu = totalSum / totalWeight

	totalVariance := sampleVariance * sumWeights
	if prior {
		// Variance from the prior samples
		totalVariance += priorWeight[1] * priorValue[1] * priorValue[1]

		// Cross varaiance from the differences of the means
		meanDiff := (sampleMean - priorValue[0])
		totalVariance += priorWeight[0] * sumWeights * meanDiff * meanDiff / totalWeight
	}

	n.Sigma = math.Sqrt(totalVariance / totalWeight)

	newPriorValue = []float64{n.Mu, n.Sigma}
	newPriorWeight = []float64{sumWeights, sumWeights}
	if prior {
		newPriorWeight[0] += priorWeight[0]
		newPriorWeight[1] += priorWeight[1]
	}
	return newPriorValue, newPriorWeight
}

// LogProb computes the natural logarithm of the value of the probability density function at x.
func (n Normal) LogProb(x float64) float64 {
	return negLogRoot2Pi - math.Log(n.Sigma) - (x-n.Mu)*(x-n.Mu)/(2*n.Sigma*n.Sigma)
}

// MarshalSlice gets the parameters of the distribution.
// The first element of Parameters is Mu, the second is Sigma.
// Panics if the length of the input slice is not equal to the number of parameters.
func (n Normal) MarshalSlice(s []float64) {
	nParam := n.NumParameters()
	if len(s) != nParam {
		panic("exponential: improper parameter length")
	}
	s[0] = n.Mu
	s[1] = n.Sigma
	return
}

// Mean returns the mean of the probability distribution.
func (n Normal) Mean() float64 {
	return n.Mu
}

// Median returns the median of the normal distribution.
func (n Normal) Median() float64 {
	return n.Mu
}

// Mode returns the mode of the normal distribution.
func (n Normal) Mode() float64 {
	return n.Mu
}

// NumParameters returns the number of parameters in the distribution.
func (Normal) NumParameters() int {
	return 2
}

// NormalMap is the parameter mapping for the Uniform distribution.
var NormalMap = map[string]int{"Mu": 0, "Sigma": 1}

// ParameterMap returns a mapping from fields of the distribution to elements
// of the marshaled slice. Do not edit this variable.
func (n Normal) ParameterMap() map[string]int {
	return NormalMap
}

// Prob computes the value of the probability density function at x.
func (n Normal) Prob(x float64) float64 {
	return math.Exp(n.LogProb(x))
}

// Quantile returns the inverse of the cumulative probability distribution.
func (n Normal) Quantile(p float64) float64 {
	if p < 0 || p > 1 {
		panic("dist: percentile out of bounds")
	}
	return n.Mu + n.Sigma*zQuantile(p)
}

// Rand returns a random sample drawn from the distribution.
func (n Normal) Rand() float64 {
	var rnd float64
	if n.Source == nil {
		rnd = rand.NormFloat64()
	} else {
		rnd = n.Source.NormFloat64()
	}
	return rnd*n.Sigma + n.Mu
}

// Skewness returns the skewness of the distribution.
func (Normal) Skewness() float64 {
	return 0
}

// StdDev returns the standard deviation of the probability distribution.
func (n Normal) StdDev() float64 {
	return n.Sigma
}

// Survival returns the survival function (complementary CDF) at x.
func (n Normal) Survival(x float64) float64 {
	return 0.5 * (1 - math.Erf((x-n.Mu)/(n.Sigma*math.Sqrt2)))
}

// UnmarshalSlice sets the parameters of the distribution.
// This sets Mu to be the first element of the slice and Sigma to be the second
// element of the slice.
// Panics if the length of the input slice is not equal to the number of parameters.
func (n *Normal) UnmarshalSlice(s []float64) {
	if len(s) != n.NumParameters() {
		panic("exponential: incorrect number of parameters to set")
	}
	n.Mu = s[0]
	n.Sigma = s[1]
}

// Variance returns the variance of the probability distribution.
func (n Normal) Variance() float64 {
	return n.Sigma * n.Sigma
}

// TODO: Is the right way to compute inverf?
// It seems to me like the precision is not high enough, but I don't
// know the correct version. It would be nice if this were built into the
// math package in the standard library (issue 6359)

/*
Copyright (c) 2012 The Probab Authors. All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

* Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
* Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
* Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

var (
	zQuantSmallA = []float64{3.387132872796366608, 133.14166789178437745, 1971.5909503065514427, 13731.693765509461125, 45921.953931549871457, 67265.770927008700853, 33430.575583588128105, 2509.0809287301226727}
	zQuantSmallB = []float64{1.0, 42.313330701600911252, 687.1870074920579083, 5394.1960214247511077, 21213.794301586595867, 39307.89580009271061, 28729.085735721942674, 5226.495278852854561}
	zQuantInterA = []float64{1.42343711074968357734, 4.6303378461565452959, 5.7694972214606914055, 3.64784832476320460504, 1.27045825245236838258, 0.24178072517745061177, 0.0227238449892691845833, 7.7454501427834140764e-4}
	zQuantInterB = []float64{1.0, 2.05319162663775882187, 1.6763848301838038494, 0.68976733498510000455, 0.14810397642748007459, 0.0151986665636164571966, 5.475938084995344946e-4, 1.05075007164441684324e-9}
	zQuantTailA  = []float64{6.6579046435011037772, 5.4637849111641143699, 1.7848265399172913358, 0.29656057182850489123, 0.026532189526576123093, 0.0012426609473880784386, 2.71155556874348757815e-5, 2.01033439929228813265e-7}
	zQuantTailB  = []float64{1.0, 0.59983220655588793769, 0.13692988092273580531, 0.0148753612908506148525, 7.868691311456132591e-4, 1.8463183175100546818e-5, 1.4215117583164458887e-7, 2.04426310338993978564e-15}
)

func rateval(a []float64, na int64, b []float64, nb int64, x float64) float64 {
	var (
		u, v, r float64
	)
	u = a[na-1]

	for i := na - 1; i > 0; i-- {
		u = x*u + a[i-1]
	}

	v = b[nb-1]

	for j := nb - 1; j > 0; j-- {
		v = x*v + b[j-1]
	}

	r = u / v

	return r
}

func zQuantSmall(q float64) float64 {
	r := 0.180625 - q*q
	return q * rateval(zQuantSmallA, 8, zQuantSmallB, 8, r)
}

func zQuantIntermediate(r float64) float64 {
	return rateval(zQuantInterA, 8, zQuantInterB, 8, (r - 1.6))
}

func zQuantTail(r float64) float64 {
	return rateval(zQuantTailA, 8, zQuantTailB, 8, (r - 5.0))
}

// Compute the quantile in normalized units
func zQuantile(p float64) float64 {
	switch {
	case p == 1.0:
		return math.Inf(1)
	case p == 0.0:
		return math.Inf(-1)
	}
	var r, x, pp, dp float64
	dp = p - 0.5
	if math.Abs(dp) <= 0.425 {
		return zQuantSmall(dp)
	}
	if p < 0.5 {
		pp = p
	} else {
		pp = 1.0 - p
	}
	r = math.Sqrt(-math.Log(pp))
	if r <= 5.0 {
		x = zQuantIntermediate(r)
	} else {
		x = zQuantTail(r)
	}
	if p < 0.5 {
		return -x
	}
	return x
}
