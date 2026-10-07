package voxel

// Lighting: every air block stores a sun light level (0..15) and a torch light level (0..15).
// Sun light is 15 under the open sky and loses one level per block as it spreads sideways and
// downwards into buildings; torch light spreads the same way from lamp positions. Faces are lit
// by the brighter of both levels of the air block in front of them (see Chunk.getVoxelFace).

const maxLightLevel = 15

// EnableLighting turns on light propagation for this map (only clients need it) and computes it.
func (m *Map) EnableLighting(lamps []Int3) {
	m.lightingEnabled = true
	m.lamps = lamps
	m.RecomputeLighting()
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

	queue = queue[:0]
	for _, lamp := range m.lamps {
		if m.ContainsGrid(lamp) && !m.IsSolidBlockAt(lamp.X, lamp.Y, lamp.Z) {
			torch[index(lamp.X, lamp.Y, lamp.Z)] = maxLightLevel
			queue = append(queue, lamp)
		}
	}
	m.spreadLight(torch, queue, index, sizeX, sizeY, sizeZ)

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

// faceLight is the light level stored in a face's vertices.
func (m *Map) faceLight(neighbor *Block) byte {
	if !m.lightingEnabled || neighbor == nil {
		return maxLightLevel // faces at the map border look into the open
	}
	return max(neighbor.GetSunLight(), neighbor.GetTorchLight())
}
