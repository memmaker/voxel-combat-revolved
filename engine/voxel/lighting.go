package voxel

import "math"

// Lighting: every air block stores a sun light level (0..15) and a torch light level (0..15).
// Sun light is 15 under the open sky and loses one level per block as it spreads sideways and
// downwards into buildings; torch light spreads the same way from lamp positions. Faces are lit
// by the brighter of both levels of the air block in front of them (see Chunk.getVoxelFace).

const maxLightLevel = 15

// shadowLightLevel is the most light a block in the shade of direct sunlight gets
const shadowLightLevel = 8

// sunDirection points towards the sun, per block of height
var sunDirection = [3]float32{0.55, 1, 0.35}

// seesSun marches from the block's center towards the sun until it leaves the map.
func (m *Map) seesSun(x, y, z, sizeX, sizeY, sizeZ int32) bool {
	px, py, pz := float32(x)+0.5, float32(y)+0.5, float32(z)+0.5
	const step = 0.5
	for {
		px, py, pz = px+sunDirection[0]*step, py+sunDirection[1]*step, pz+sunDirection[2]*step
		bx, by, bz := int32(px), int32(py), int32(pz)
		if px < 0 || pz < 0 || bx >= sizeX || by >= sizeY || bz >= sizeZ {
			return true
		}
		if m.IsSolidBlockAt(bx, by, bz) {
			return false
		}
	}
}

// EnableLighting turns on light propagation for this map (only clients need it) and computes it.
func (m *Map) EnableLighting(lamps []Int3) {
	m.lightingEnabled = true
	m.lamps = lamps
	m.RecomputeLighting()
}

// SetLamps replaces the lamp positions; lighting is recomputed on the next update.
func (m *Map) SetLamps(lamps []Int3) {
	m.lamps = lamps
	m.lightingDirty = m.lightingEnabled
}

// RecomputeLighting recalculates sun and lamp light for the whole map and re-meshes the chunks
// whose lighting changed. It runs after the terrain changes, e.g. when an explosion opens a roof.
func (m *Map) RecomputeLighting() {
	if !m.lightingEnabled {
		return
	}
	m.lightingDirty = false
	sizeX, sizeY, sizeZ := m.width*m.ChunkSizeHorizontal, m.height*m.ChunkSizeHeight, m.depth*m.ChunkSizeHorizontal
	index := func(x, y, z int32) int32 { return x + y*sizeX + z*sizeX*sizeY }
	sun := make([]byte, sizeX*sizeY*sizeZ)
	torch := make([]byte, sizeX*sizeY*sizeZ)

	var queue []Int3
	// sky: straight down from the top until the first solid block
	for x := int32(0); x < sizeX; x++ {
		for z := int32(0); z < sizeZ; z++ {
			for y := sizeY - 1; y >= 0; y-- {
				if m.IsSolidBlockAt(x, y, z) {
					break
				}
				sun[index(x, y, z)] = maxLightLevel
				queue = append(queue, Int3{x, y, z})
			}
		}
	}
	m.spreadLight(sun, queue, index, sizeX, sizeY, sizeZ)
	// direct sunlight: only blocks with a clear slanted ray to the sun keep full light,
	// everything else is in shadow, so walls and crates cast shadows onto the ground
	for x := int32(0); x < sizeX; x++ {
		for y := int32(0); y < sizeY; y++ {
			for z := int32(0); z < sizeZ; z++ {
				i := index(x, y, z)
				if sun[i] > shadowLightLevel && !m.seesSun(x, y, z, sizeX, sizeY, sizeZ) {
					sun[i] = shadowLightLevel
				}
			}
		}
	}

	queue = queue[:0]
	for _, lamp := range m.lamps {
		if m.ContainsGrid(lamp) && !m.IsSolidBlockAt(lamp.X, lamp.Y, lamp.Z) {
			torch[index(lamp.X, lamp.Y, lamp.Z)] = maxLightLevel
			queue = append(queue, lamp)
		}
	}
	m.spreadLight(torch, queue, index, sizeX, sizeY, sizeZ)

	oldSky := m.sky
	m.sky = m.computeSky(sun, index, sizeX, sizeY, sizeZ)

	// store the result; re-mesh only chunks where something changed
	changedChunks := make(map[*Chunk]bool)
	for x := int32(0); x < sizeX; x++ {
		for y := int32(0); y < sizeY; y++ {
			for z := int32(0); z < sizeZ; z++ {
				block, chunk := m.GetBlockAndChunk(Int3{x, y, z})
				if block == nil || chunk == nil {
					continue
				}
				i := index(x, y, z)
				if len(oldSky) != len(m.sky) || oldSky[i] != m.sky[i] {
					changedChunks[chunk] = true
				}
				if block.GetSunLight() != sun[i] || block.GetTorchLight() != torch[i] {
					block.SetSunLight(sun[i])
					block.SetTorchLight(torch[i])
					changedChunks[chunk] = true
				}
			}
		}
	}
	for chunk := range changedChunks {
		chunk.SetDirty()
		// faces on the border of a neighbor chunk read this chunk's air blocks
		for _, neighbor := range []*Chunk{chunk.cXN, chunk.cXP, chunk.cYN, chunk.cYP, chunk.cZN, chunk.cZP} {
			if neighbor != nil {
				neighbor.SetDirty()
			}
		}
	}
}

func (m *Map) spreadLight(levels []byte, queue []Int3, index func(x, y, z int32) int32, sizeX, sizeY, sizeZ int32) {
	neighbors := []Int3{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		level := levels[index(current.X, current.Y, current.Z)]
		if level <= 1 {
			continue
		}
		for _, offset := range neighbors {
			next := current.Add(offset)
			if next.X < 0 || next.Y < 0 || next.Z < 0 || next.X >= sizeX || next.Y >= sizeY || next.Z >= sizeZ {
				continue
			}
			if m.IsSolidBlockAt(next.X, next.Y, next.Z) {
				continue
			}
			i := index(next.X, next.Y, next.Z)
			if levels[i] < level-1 {
				levels[i] = level - 1
				queue = append(queue, next)
			}
		}
	}
}

// LightAt returns the brightness (0..15) at a block, the brighter of sun and lamp light.
// Without lighting it returns full brightness.
func (m *Map) LightAt(position Int3) byte {
	if !m.lightingEnabled {
		return maxLightLevel
	}
	if !m.ContainsGrid(position) {
		return maxLightLevel
	}
	block := m.GetBlockFromVec(position)
	if block == nil {
		return maxLightLevel
	}
	return max(block.GetSunLight(), block.GetTorchLight())
}

// skyRays are the directions sampled for sky openness: straight up and a ring at 45° elevation
var skyRays = func() [][3]float32 {
	rays := [][3]float32{{0, 1, 0}}
	for i := 0; i < 8; i++ {
		angle := float64(i) * math.Pi / 4
		rays = append(rays, [3]float32{float32(math.Cos(angle)) * 0.7071, 0.7071, float32(math.Sin(angle)) * 0.7071})
	}
	return rays
}()

// skyRayLength is how far a sky ray travels before it counts as open; distant walls don't darken
const skyRayLength = 24

// computeSky stores for every lit air block near a solid one how much of the sky it sees (0..15),
// so open squares get more ambient light than alleys, courtyards and spots under overhangs.
func (m *Map) computeSky(sun []byte, index func(x, y, z int32) int32, sizeX, sizeY, sizeZ int32) []byte {
	sky := make([]byte, sizeX*sizeY*sizeZ)
	for x := int32(0); x < sizeX; x++ {
		for y := int32(0); y < sizeY; y++ {
			for z := int32(0); z < sizeZ; z++ {
				i := index(x, y, z)
				if sun[i] == 0 || m.IsSolidBlockAt(x, y, z) || !m.nearSolid(x, y, z) {
					continue
				}
				open := 0
				for _, ray := range skyRays {
					if m.rayEscapes(x, y, z, ray, sizeX, sizeY, sizeZ) {
						open++
					}
				}
				sky[i] = byte(open * maxLightLevel / len(skyRays))
			}
		}
	}
	return sky
}

// nearSolid reports whether any of the 26 blocks around this one is solid (only those carry faces).
func (m *Map) nearSolid(x, y, z int32) bool {
	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 1; dy++ {
			for dz := int32(-1); dz <= 1; dz++ {
				if m.solidInMap(x+dx, y+dy, z+dz) {
					return true
				}
			}
		}
	}
	return false
}

// solidInMap is IsSolidBlockAt that is false outside the map (also for negative coordinates).
func (m *Map) solidInMap(x, y, z int32) bool {
	return m.Contains(x, y, z) && m.IsSolidBlockAt(x, y, z)
}

func (m *Map) rayEscapes(x, y, z int32, ray [3]float32, sizeX, sizeY, sizeZ int32) bool {
	px, py, pz := float32(x)+0.5, float32(y)+0.5, float32(z)+0.5
	const step = 0.5
	for traveled := float32(0); traveled < skyRayLength; traveled += step {
		px, py, pz = px+ray[0]*step, py+ray[1]*step, pz+ray[2]*step
		if px < 0 || pz < 0 || px >= float32(sizeX) || py >= float32(sizeY) || pz >= float32(sizeZ) {
			return true
		}
		if m.IsSolidBlockAt(int32(px), int32(py), int32(pz)) {
			return false
		}
	}
	return true
}

// VertexLight is the lighting of one corner of a face.
type VertexLight struct {
	Light byte // brighter of sun and lamp light (0..15)
	Sky   byte // sky openness (0..15)
	AO    byte // ambient occlusion: 0 = corner fully enclosed .. 3 = no occluders
}

// airLight is the light and sky openness of an air block; outside the map is open sky.
func (m *Map) airLight(x, y, z int32) (light, sky byte) {
	if !m.lightingEnabled || !m.Contains(x, y, z) {
		return maxLightLevel, maxLightLevel
	}
	block := m.GetGlobalBlock(x, y, z)
	if block == nil {
		return maxLightLevel, maxLightLevel
	}
	if m.sky != nil {
		sizeX, sizeY := m.width*m.ChunkSizeHorizontal, m.height*m.ChunkSizeHeight
		sky = m.sky[x+y*sizeX+z*sizeX*sizeY]
	}
	return max(block.GetSunLight(), block.GetTorchLight()), sky
}

// cornerLight lights the face corner that lies towards du and dv from the air block in front of
// the face: occlusion from the two side blocks and the diagonal one, and light and sky openness
// smoothed over the open blocks among them, so the light fades across faces instead of stepping.
func (m *Map) cornerLight(air, du, dv Int3) VertexLight {
	side1, side2, corner := air.Add(du), air.Add(dv), air.Add(du).Add(dv)
	solid1 := m.solidInMap(side1.X, side1.Y, side1.Z)
	solid2 := m.solidInMap(side2.X, side2.Y, side2.Z)
	solidCorner := (solid1 && solid2) || m.solidInMap(corner.X, corner.Y, corner.Z)

	var ao byte = 3
	if solid1 && solid2 {
		ao = 0
	} else {
		for _, solid := range []bool{solid1, solid2, solidCorner} {
			if solid {
				ao--
			}
		}
	}

	light, sky := m.airLight(air.X, air.Y, air.Z)
	lightSum, skySum, count := int(light), int(sky), 1
	for _, open := range []struct {
		pos   Int3
		solid bool
	}{{side1, solid1}, {side2, solid2}, {corner, solidCorner}} {
		if open.solid {
			continue
		}
		light, sky = m.airLight(open.pos.X, open.pos.Y, open.pos.Z)
		lightSum, skySum, count = lightSum+int(light), skySum+int(sky), count+1
	}
	return VertexLight{
		Light: byte((lightSum + count/2) / count),
		Sky:   byte((skySum + count/2) / count),
		AO:    ao,
	}
}
