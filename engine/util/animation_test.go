package util

import (
	"math"
	"testing"
)

func TestKeyframeAnimationSkipsSeveralKeyframes(t *testing.T) {
	frames := [][3]float32{{0, 0, 0}, {1, 0, 0}, {2, 0, 0}, {3, 0, 0}, {4, 0, 0}}
	times := []float32{0, 0.1, 0.2, 0.3, 0.4}
	current := 0
	var value [3]float32
	set := func(v [3]float32) { value = v }
	// one big step over three keyframes must land on the interpolated value, not one key behind
	KeyframeAnimation(0.35, &current, frames, times, Lerp3f, set)
	if math.Abs(float64(value[0])-3.5) > 0.001 {
		t.Fatalf("expected 3.5 at t=0.35, got %v", value[0])
	}
	// going back in time (a looping animation restarted) must also work
	KeyframeAnimation(0.05, &current, frames, times, Lerp3f, set)
	if math.Abs(float64(value[0])-0.5) > 0.001 {
		t.Fatalf("expected 0.5 at t=0.05, got %v", value[0])
	}
}

func TestLerpQuatNearlyIdenticalRotations(t *testing.T) {
	one := [4]float32{0, 0.70710677, 0, 0.70710677}
	two := [4]float32{0, 0.7071068, 0, 0.7071067} // differs only by rounding
	for _, factor := range []float64{0, 0.25, 0.5, 1} {
		result := LerpQuat(one, two, factor)
		for _, component := range result {
			if math.IsNaN(float64(component)) || math.Abs(float64(component)) > 1.0001 {
				t.Fatalf("broken rotation for factor %v: %v", factor, result)
			}
		}
	}
	// opposite signs describe the same rotation: interpolation must take the short way
	flipped := [4]float32{-one[0], -one[1], -one[2], -one[3]}
	mid := LerpQuat(one, flipped, 0.5)
	if math.Abs(float64(mid[1]))+math.Abs(float64(mid[3])) < 1.0 {
		t.Fatalf("expected the same rotation halfway, got %v", mid)
	}
}
