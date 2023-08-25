package client

import "github.com/go-gl/mathgl/mgl32"

// PositionalTolerance is the distance between two opengl float positions that is considered "close enough" to be considered the same position.
// Default is 0.05.
const PositionalTolerance = 0.05

var ColorTechTeal = mgl32.Vec4{float32(47) / float32(255), float32(214) / float32(255), float32(195) / float32(255), 1.0}
