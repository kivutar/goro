package render

import (
	"math"
	"testing"
)

func TestInverseViewProjectionRoundTrip(t *testing.T) {
	// Perspective, rotation, and translation: a depth reconstruction must undo
	// all three, including the renderer's conversion from [-1,1] to [0,1].
	matrix := [16]float32{2, 0, 0, 0, 0, 1.4, -0.71, -0.7, 0, -1.4, -0.71, -0.7, -6, 4, 29.8, 30}
	inverse, ok := inverseViewProjection(matrix)
	if !ok {
		t.Fatal("camera matrix is invertible")
	}
	for _, p := range [][4]float64{{0, 0, 0, 1}, {4, 2, -5, 1}, {-3, -2, 8, 1}} {
		var clip [4]float64
		for row := range 4 {
			for col := range 4 {
				clip[row] += float64(matrix[col*4+row]) * p[col]
			}
		}
		depth := (clip[2]/clip[3] + 1) * 0.5
		ndc := [4]float64{clip[0] / clip[3], clip[1] / clip[3], depth*2 - 1, 1}
		var restored [4]float64
		for row := range 4 {
			for col := range 4 {
				restored[row] += float64(inverse[col*4+row]) * ndc[col]
			}
		}
		for i := range 3 {
			if got := restored[i] / restored[3]; math.Abs(got-p[i]) > 0.001 {
				t.Fatalf("reconstructed point[%d]=%g, want %g", i, got, p[i])
			}
		}
	}
}

func TestInverseViewProjectionRejectsInvalidCamera(t *testing.T) {
	for _, matrix := range [][16]float32{{}, {float32(math.NaN())}, {float32(math.Inf(1))}} {
		if _, ok := inverseViewProjection(matrix); ok {
			t.Fatal("accepted singular/non-finite camera")
		}
	}
}
