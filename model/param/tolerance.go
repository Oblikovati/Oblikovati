// SPDX-License-Identifier: GPL-2.0-only

package param

// ModelValueType selects which value within an engineering tolerance band the
// model actually consumes (contract: ModelValueTypeEnum). Stable explicit ids.
type ModelValueType uint8

const (
	// Nominal uses the authored value, ignoring the tolerance band.
	Nominal ModelValueType = 0
	// Upper uses nominal + the upper deviation.
	Upper ModelValueType = 1
	// Lower uses nominal + the lower deviation.
	Lower ModelValueType = 2
	// Median uses nominal + the midpoint of the deviation band.
	Median ModelValueType = 3
)

// Tolerance is an engineering tolerance: a deviation band (Upper, Lower, in
// database units) plus the [ModelValueType] that decides which value the model
// uses. The zero Tolerance is symmetric-zero with type Nominal — i.e. model
// value equals nominal.
type Tolerance struct {
	Upper float64
	Lower float64
	Type  ModelValueType
}

// ModelValue applies the tolerance to a nominal value, returning the value the
// model consumes (all in database units).
func (t Tolerance) ModelValue(nominal float64) float64 {
	switch t.Type {
	case Upper:
		return nominal + t.Upper
	case Lower:
		return nominal + t.Lower
	case Median:
		return nominal + (t.Upper+t.Lower)/2
	default:
		return nominal
	}
}

// ParameterDisplayFormat controls how a parameter is presented (contract:
// ParameterDisplayFormatEnum). It affects display only, never the stored or
// model value.
type ParameterDisplayFormat uint8

const (
	// ShowExpression displays the authored expression text.
	ShowExpression ParameterDisplayFormat = 0
	// ShowValue displays the evaluated value in the preferred unit.
	ShowValue ParameterDisplayFormat = 1
	// ShowTolerance displays the value with its tolerance band.
	ShowTolerance ParameterDisplayFormat = 2
)
