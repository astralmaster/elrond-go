package Benchmark_TestEllipticCurveScalarMultP224

import (
	"testing"

	mc "github.com/astralmaster/elrond-go/integrationTests/vm/arwen/arwenvm/mandosConverter"
)

func TestMandosConverter_EllipticCurves(t *testing.T) {
	mc.CheckConverter(t, "./elliptic_curves.scen.json")
}

func Benchmark_MandosConverter_EllipticCurves(b *testing.B) {
	mc.BenchmarkMandosSpecificTx(b, "./elliptic_curves.scen.json")
}
