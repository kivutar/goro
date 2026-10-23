package render

import (
	"math"
	"testing"
	"time"
)

func TestHallucinationPhaseUsesElapsedTime(t *testing.T) {
	for _, fps := range []int{30, 60, 120, 300} {
		frame := NewFrame(800, 600)
		for tick := 0; tick <= fps; tick++ {
			frame.BeginFrame()
			frame.SetHallucination(time.Duration(tick) * time.Second / time.Duration(fps))
		}
		if !frame.hallucination || math.Abs(float64(frame.hallucinationPhase)-math.Pi/3) > 0.00001 {
			t.Fatalf("%d fps: phase=%g, want 60 degrees after one second", fps, frame.hallucinationPhase)
		}
		frame.BeginFrame()
		if frame.hallucination {
			t.Fatal("distortion leaked into a later frame without the status")
		}
	}
}
