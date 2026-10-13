package voxel

import "testing"

func TestCornerOcclusionAndSky(t *testing.T) {
	m := NewMap(1, 1, 1, 16, 16)
	m.NewChunk(0, 0, 0)
	for x := int32(0); x < 16; x++ {
		for z := int32(0); z < 16; z++ {
			m.SetBlock(x, 0, z, NewBlock(1)) // floor
		}
	}
	for z := int32(0); z < 16; z++ {
		for y := int32(1); y < 6; y++ {
			m.SetBlock(8, y, z, NewBlock(1)) // wall along z at x = 8
		}
	}
	m.EnableLighting(nil)

	open := m.cornerLight(Int3{3, 1, 3}, Int3{1, 0, 0}, Int3{0, 0, 1})
	if open.AO != 3 {
		t.Errorf("open floor should not be occluded, got %d", open.AO)
	}
	// floor top face at x = 7, corner towards the wall
	atWall := m.cornerLight(Int3{7, 1, 3}, Int3{1, 0, 0}, Int3{0, 0, 1})
	if atWall.AO != 1 {
		t.Errorf("floor corner at the wall base should be occluded, got %d", atWall.AO)
	}
	if atWall.Sky >= open.Sky {
		t.Errorf("the wall should hide part of the sky: %d at the wall, %d in the open", atWall.Sky, open.Sky)
	}
}
