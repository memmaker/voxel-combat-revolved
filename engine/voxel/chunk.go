package voxel

import (
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
)

// translated from https://github.com/Vercidium/voxel-mesh-generation/blob/master/source/Chunk.cs

type Chunk struct {
	data        []*Block
	m           *Map
	chunkPosX   int32
	chunkPosY   int32
	chunkPosZ   int32
	chunkHelper *ChunkHelper
	cXN         *Chunk
	cXP         *Chunk
	cYN         *Chunk
	cYP         *Chunk
	cZN         *Chunk
	cZP         *Chunk
	isDirty     bool
	meshBuffer  ChunkMesh
}

func NewChunk(chunkShader *glhf.Shader, voxelMap *Map, x, y, z int32) *Chunk {
	c := &Chunk{
		data:      make([]*Block, CHUNK_SIZE_CUBED),
		m:         voxelMap,
		chunkPosX: x,
		chunkPosY: y,
		chunkPosZ: z,
		chunkHelper: &ChunkHelper{
			visitXN: make([]bool, CHUNK_SIZE_CUBED),
			visitXP: make([]bool, CHUNK_SIZE_CUBED),
			visitYN: make([]bool, CHUNK_SIZE_CUBED),
			visitYP: make([]bool, CHUNK_SIZE_CUBED),
			visitZN: make([]bool, CHUNK_SIZE_CUBED),
			visitZP: make([]bool, CHUNK_SIZE_CUBED),
		},
		meshBuffer: NewMeshBuffer(chunkShader),
	}
	for i := int32(0); i < CHUNK_SIZE_CUBED; i++ {
		c.data[i] = NewAirBlock()
	}
	return c
}

func blockIndex(i, j, k int32) int32 {
	return i + j*CHUNK_SIZE + k*CHUNK_SIZE_SQUARED
}
func (c *Chunk) Contains(x, y, z int32) bool {
	return x >= 0 && x < CHUNK_SIZE && y >= 0 && y < CHUNK_SIZE && z >= 0 && z < CHUNK_SIZE
}
func (c *Chunk) GetLocalBlock(i, j, k int32) *Block {
	if !c.Contains(i, j, k) {
		return nil
	}
	return c.data[blockIndex(i, j, k)]
}

func (c *Chunk) SetBlock(x, y, z int32, block *Block) {
	c.data[blockIndex(x, y, z)] = block
	c.isDirty = true
}

type VoxelFace struct {
	transparent  bool
	side         FaceType
	textureIndex byte
	hidden       bool
}

func (v *VoxelFace) EqualForMerge(face *VoxelFace) bool {
	if face.transparent {
		return face.transparent == v.transparent
	}
	return face.transparent == v.transparent && face.textureIndex == v.textureIndex
}

func (c *Chunk) InitNeighbors() {
	if c.chunkPosX > 0 {
		c.cXN = c.m.GetChunk(c.chunkPosX-1, c.chunkPosY, c.chunkPosZ)
	}

	// Positive CollisionX side
	if c.chunkPosX < c.m.width-1 {
		c.cXP = c.m.GetChunk(c.chunkPosX+1, c.chunkPosY, c.chunkPosZ)
	}

	// Negative CollisionY side
	if c.chunkPosY > 0 {
		c.cYN = c.m.GetChunk(c.chunkPosX, c.chunkPosY-1, c.chunkPosZ)
	}

	// Positive CollisionY side
	if c.chunkPosY < c.m.height-1 {
		c.cYP = c.m.GetChunk(c.chunkPosX, c.chunkPosY+1, c.chunkPosZ)
	}

	// Negative Z neighbour
	if c.chunkPosZ > 0 {
		c.cZN = c.m.GetChunk(c.chunkPosX, c.chunkPosY, c.chunkPosZ-1)
	}

	// Positive Z side
	if c.chunkPosZ < c.m.depth-1 {
		c.cZP = c.m.GetChunk(c.chunkPosX, c.chunkPosY, c.chunkPosZ+1)
	}
}

func (c *Chunk) GreedyMeshing() ChunkMesh {
	// adapted from: https://github.com/roboleary/GreedyMesh/blob/master/src/mygame/Main.java

	if !c.isDirty {
		return c.meshBuffer
	}

	c.InitNeighbors()

	c.chunkHelper.Reset()
	c.meshBuffer.Reset()

	var (
		i, j, k, l, w, h, u, v, n int32
		side                      FaceType

		x  = [3]int32{}
		q  = [3]int32{}
		du = [3]int32{}
		dv = [3]int32{}

		mask       = make([]*VoxelFace, CHUNK_SIZE*CHUNK_SIZE)
		voxelFace  *VoxelFace
		voxelFace1 *VoxelFace
	)

	for backFace, b := true, false; b != backFace; backFace, b = backFace && b, !b {
		for d := int32(0); d < 3; d++ {
			u = (d + 1) % 3
			v = (d + 2) % 3

			x[0], x[1], x[2] = 0, 0, 0

			q[0], q[1], q[2] = 0, 0, 0
			q[d] = 1

			switch {
			case d == 0:
				side = map[bool]FaceType{true: xn, false: xp}[backFace]
			case d == 1:
				side = map[bool]FaceType{true: yn, false: yp}[backFace]
			case d == 2:
				side = map[bool]FaceType{true: zn, false: zp}[backFace]
			}

			for x[d] = -1; x[d] < CHUNK_SIZE; {
				n = 0

				for x[v] = 0; x[v] < CHUNK_SIZE; x[v]++ {
					for x[u] = 0; x[u] < CHUNK_SIZE; x[u]++ {
						if x[d] >= 0 {
							voxelFace = c.getVoxelFace(x[0], x[1], x[2], side)
						} else {
							voxelFace = nil
						}

						if x[d] < CHUNK_SIZE-1 {
							voxelFace1 = c.getVoxelFace(x[0]+q[0], x[1]+q[1], x[2]+q[2], side)
						} else {
							voxelFace1 = nil
						}

						if voxelFace != nil && voxelFace1 != nil && voxelFace.EqualForMerge(voxelFace1) {
							mask[n] = nil
						} else {
							if backFace {
								mask[n] = voxelFace1
							} else {
								mask[n] = voxelFace
							}
						}
						n++
					}
				}

				x[d]++

				n = 0
				for j = 0; j < CHUNK_SIZE; j++ {
					for i = 0; i < CHUNK_SIZE; {
						if mask[n] != nil {
							w = 1
							for i+w < CHUNK_SIZE && mask[n+w] != nil && mask[n+w].EqualForMerge(mask[n]) {
								w++
							}

							done := false
							h = 1
							for h+j < CHUNK_SIZE {
								for k = 0; k < w; k++ {
									if mask[n+k+h*CHUNK_SIZE] == nil || !mask[n+k+h*CHUNK_SIZE].EqualForMerge(mask[n]) {
										done = true
										break
									}
								}
								if done {
									break
								}
								h++
							}

							if (!mask[n].transparent) && (!mask[n].hidden) {
								x[u], x[v] = i, j
								du[0], du[1], du[2] = 0, 0, 0
								du[u] = w
								dv[0], dv[1], dv[2] = 0, 0, 0
								dv[v] = h
								//c.combinedMeshBuffer.AppendQuad()
								bottomLeft := Int3{x[0], x[1], x[2]}
								topLeft := Int3{x[0] + du[0], x[1] + du[1], x[2] + du[2]}
								bottomRight := Int3{x[0] + dv[0], x[1] + dv[1], x[2] + dv[2]}
								topRight := Int3{x[0] + du[0] + dv[0], x[1] + du[1] + dv[1], x[2] + du[2] + dv[2]}
								c.meshBuffer.AppendQuad(bottomLeft, topLeft, bottomRight, topRight, mask[n].side, mask[n].textureIndex)
								// we also would have width (w) and height (h) here
								// backface is a bool, true if we are rendering the backface of a block
							}

							for l = 0; l < h; l++ {
								for k = 0; k < w; k++ {
									mask[n+k+l*CHUNK_SIZE] = nil
								}
							}

							i += w
							n += w
						} else {
							i++
							n++
						}
					}
				}
			}
		}
	}

	c.isDirty = false

	println(fmt.Sprintf("[Greedy] Chunk %d,%d,%d was meshed into %d triangles", c.chunkPosX, c.chunkPosY, c.chunkPosZ, c.meshBuffer.TriangleCount()))
	return c.meshBuffer
}

func (c *Chunk) getVoxelFace(x int32, y int32, z int32, side FaceType) *VoxelFace {
	block := c.GetLocalBlock(x, y, z)
	if block == nil {
		return &VoxelFace{transparent: true}
	}
	if block.IsAir() {
		return &VoxelFace{transparent: true}
	}
	var neighbor *Block
	switch side {
	case xn:
		if x == 0 {
			if c.cXN != nil {
				neighbor = c.cXN.GetLocalBlock(CHUNK_SIZE-1, y, z)
			}
		} else {
			neighbor = c.GetLocalBlock(x-1, y, z)
		}
	case xp:
		if x == CHUNK_SIZE-1 {
			if c.cXP != nil {
				neighbor = c.cXP.GetLocalBlock(0, y, z)
			}
		} else {
			neighbor = c.GetLocalBlock(x+1, y, z)
		}
	case yn:
		if y == 0 {
			if c.cYN != nil {
				neighbor = c.cYN.GetLocalBlock(x, CHUNK_SIZE-1, z)
			}
		} else {
			neighbor = c.GetLocalBlock(x, y-1, z)
		}
	case yp:
		if y == CHUNK_SIZE-1 {
			if c.cYP != nil {
				neighbor = c.cYP.GetLocalBlock(x, 0, z)
			}
		} else {
			neighbor = c.GetLocalBlock(x, y+1, z)
		}
	case zn:
		if z == 0 {
			if c.cZN != nil {
				neighbor = c.cZN.GetLocalBlock(x, y, CHUNK_SIZE-1)
			}
		} else {
			neighbor = c.GetLocalBlock(x, y, z-1)
		}
	case zp:
		if z == CHUNK_SIZE-1 {
			if c.cZP != nil {
				neighbor = c.cZP.GetLocalBlock(x, y, 0)
			}
		} else {
			neighbor = c.GetLocalBlock(x, y, z+1)
		}
	}

	face := &VoxelFace{side: side, textureIndex: block.GetTextureIndexForSide(side), transparent: false}
	if neighbor != nil && !neighbor.IsAir() {
		face.hidden = true
		face.transparent = true
	}
	return face
}

type Int3 struct {
	X, Y, Z int32
}

func (i Int3) Add(other Int3) Int3 {
	return Int3{i.X + other.X, i.Y + other.Y, i.Z + other.Z}
}

func (i Int3) Mul(factor int32) Int3 {
	i.X *= factor
	i.Y *= factor
	i.Z *= factor
	return i
}

func (i Int3) ToFloatVec3() mgl32.Vec3 {
	return mgl32.Vec3{float32(i.X), float32(i.Y), float32(i.Z)}
}

func (i Int3) Sub(tr Int3) Int3 {
	return Int3{i.X - tr.X, i.Y - tr.Y, i.Z - tr.Z}
}

func (i Int3) ToVec3() mgl32.Vec3 {
	return mgl32.Vec3{float32(i.X), float32(i.Y), float32(i.Z)}
}

func (i Int3) Div(factor int)Int3 {
	return Int3{i.X / int32(factor), i.Y / int32(factor), i.Z / int32(factor)}
}

func (i Int3) ToBlockCenterVec3() mgl32.Vec3 {
	return mgl32.Vec3{float32(i.X) + 0.5, float32(i.Y), float32(i.Z) + 0.5}
}

func (c *Chunk) CreateRun(b *Block, i, j, k, access int32) {
	//println("CreateRun -> ", i, j, k)
	i1 := i + 1
	j1 := j + 1
	k1 := k + 1

	//health16 := byte(b.health / 16)

	runLengthUpwards := int32(0)

	chunkAccess := int32(0)

	// Left face (CollisionX-)
	if !c.chunkHelper.visitXN[access] && c.VisibleFaceXN(i-1, j, k) {
		// Search upwards to determine run runLengthUpwards
		for q := j; q < CHUNK_SIZE; q++ {
			// Pre-calculate the array lookup as it is used twice
			chunkAccess = i + q*CHUNK_SIZE + k*CHUNK_SIZE_SQUARED

			// If we reach a different block or an empty block, end the run
			if c.DifferentBlock(chunkAccess, b) || !c.VisibleFaceXN(i-1, q, k) {
				break
			}

			// Store that we have visited this block
			c.chunkHelper.visitXN[chunkAccess] = true

			runLengthUpwards++
		}

		if runLengthUpwards > 0 {
			// Create a quad and write it directly to the buffer
			tl := Int3{i, runLengthUpwards + j, k1}
			tr := Int3{i, runLengthUpwards + j, k}
			bl := Int3{i, j, k1}
			br := Int3{i, j, k}
			c.meshBuffer.AppendQuad(bl, tl, br, tr, xn, byte(b.textureIndex))
		}
	}

	runLengthUpwards = 0

	// Same algorithm for right (CollisionX+)
	if !c.chunkHelper.visitXP[access] && c.VisibleFaceXP(i1, j, k) {
		// Search upwards to determine run runLengthUpwards
		for q := j; q < CHUNK_SIZE; q++ {
			chunkAccess = i + q*CHUNK_SIZE + k*CHUNK_SIZE_SQUARED

			if c.DifferentBlock(chunkAccess, b) || !c.VisibleFaceXP(i1, q, k) {
				break
			}

			c.chunkHelper.visitXP[chunkAccess] = true

			runLengthUpwards++
		}

		if runLengthUpwards > 0 {
			tl := Int3{i1, runLengthUpwards + j, k}
			tr := Int3{i1, runLengthUpwards + j, k1}
			bl := Int3{i1, j, k}
			br := Int3{i1, j, k1}
			c.meshBuffer.AppendQuad(bl, tl, br, tr, xp, byte(b.textureIndex))
		}
	}

	runLengthUpwards = int32(0)

	// Same algorithm for back (Z-)
	if !c.chunkHelper.visitZN[access] && c.VisibleFaceZN(i, j, k-1) {
		// Search upwards to determine run runLengthUpwards
		for q := j; q < CHUNK_SIZE; q++ {
			chunkAccess = i + q*CHUNK_SIZE + k*CHUNK_SIZE_SQUARED

			if c.DifferentBlock(chunkAccess, b) || !c.VisibleFaceZN(i, q, k-1) {
				break
			}

			c.chunkHelper.visitZN[chunkAccess] = true

			runLengthUpwards++
		}

		if runLengthUpwards > 0 {
			tl := Int3{i, runLengthUpwards + j, k}
			tr := Int3{i1, runLengthUpwards + j, k}
			bl := Int3{i, j, k}
			br := Int3{i1, j, k}
			c.meshBuffer.AppendQuad(bl, tl, br, tr, zn, byte(b.textureIndex))
		}
	}
	runLengthUpwards = 0
	// Same algorithm for front (Z+)
	if !c.chunkHelper.visitZP[access] && c.VisibleFaceZP(i, j, k1) {
		// Search upwards to determine run runLengthUpwards
		for q := j; q < CHUNK_SIZE; q++ {
			chunkAccess = i + q*CHUNK_SIZE + k*CHUNK_SIZE_SQUARED

			if c.DifferentBlock(chunkAccess, b) || !c.VisibleFaceZP(i, q, k1) {
				break
			}

			c.chunkHelper.visitZP[chunkAccess] = true

			runLengthUpwards++
		}

		if runLengthUpwards > 0 {
			tl := Int3{i1, runLengthUpwards + j, k1}
			tr := Int3{i, runLengthUpwards + j, k1}
			bl := Int3{i1, j, k1}
			br := Int3{i, j, k1}
			c.meshBuffer.AppendQuad(bl, tl, br, tr, zp, byte(b.textureIndex))
		}
	}
	runLengthX := int32(0)
	// Same algorithm for bottom (CollisionY-)
	if !c.chunkHelper.visitYN[access] && c.VisibleFaceYN(i, j-1, k) {
		// Search horizontally to determine run runLengthUpwards
		for q := i; q < CHUNK_SIZE; q++ {
			chunkAccess = q + j*CHUNK_SIZE + k*CHUNK_SIZE_SQUARED

			if c.DifferentBlock(chunkAccess, b) || !c.VisibleFaceYN(q, j-1, k) {
				break
			}

			c.chunkHelper.visitYN[chunkAccess] = true

			runLengthX++
		}

		if runLengthX > 0 {
			tl := Int3{i + runLengthX, j, k1}
			tr := Int3{i, j, k1}
			bl := Int3{i + runLengthX, j, k}
			br := Int3{i, j, k}
			c.meshBuffer.AppendQuad(bl, tl, br, tr, yn, byte(b.textureIndex))
		}
	}
	runLengthX = 0
	// Same algorithm for top (CollisionY+)
	if !c.chunkHelper.visitYP[access] && c.VisibleFaceYP(i, j1, k) {
		// Search horizontally to determine run runLengthUpwards
		for q := i; q < CHUNK_SIZE; q++ {
			chunkAccess = q + j*CHUNK_SIZE + k*CHUNK_SIZE_SQUARED

			if c.DifferentBlock(chunkAccess, b) || !c.VisibleFaceYP(q, j1, k) {
				break
			}

			c.chunkHelper.visitYP[chunkAccess] = true

			runLengthX++
		}

		if runLengthX > 0 {
			tl := Int3{i, j1, k}
			tr := Int3{i + runLengthX, j1, k}
			bl := Int3{i, j1, k1}
			br := Int3{i + runLengthX, j1, k1}
			c.meshBuffer.AppendQuad(bl, tl, br, tr, yp, byte(b.textureIndex))
		}
	}

	runLengthX = 0
	// Same algorithm for back (Z-) - horizontal
	if !c.chunkHelper.visitZN[access] && c.VisibleFaceZN(i, j, k-1) {
		// Search horizontally to determine run runLengthUpwards
		for q := i1; q < CHUNK_SIZE; q++ {
			chunkAccess = q + j*CHUNK_SIZE + k*CHUNK_SIZE_SQUARED

			if c.DifferentBlock(chunkAccess, b) || !c.VisibleFaceZN(q, j, k-1) {
				break
			}

			c.chunkHelper.visitZN[chunkAccess] = true

			runLengthX++
		}

		if runLengthX > 0 {
			tl := Int3{i1, j, k}
			tr := Int3{i1 + runLengthX, j, k}
			bl := Int3{i1, j1, k}
			br := Int3{i1 + runLengthX, j1, k}
			c.meshBuffer.AppendQuad(bl, tl, br, tr, zn, byte(b.textureIndex))
		}
	}
	runLengthX = 0
	// Same algorithm for front (Z+) - horizontal
	if !c.chunkHelper.visitZP[access] && c.VisibleFaceZP(i, j, k1) {
		// Search horizontally to determine run runLengthX
		for q := i1; q < CHUNK_SIZE; q++ {
			chunkAccess = q + j*CHUNK_SIZE + k*CHUNK_SIZE_SQUARED

			if c.DifferentBlock(chunkAccess, b) || !c.VisibleFaceZP(q, j, k1) {
				break
			}

			c.chunkHelper.visitZP[chunkAccess] = true

			runLengthX++
		}

		if runLengthX > 0 {
			tl := Int3{i1 + runLengthX, j, k1}
			tr := Int3{i1, j, k1}
			bl := Int3{i1 + runLengthX, j1, k1}
			br := Int3{i1, j1, k1}
			c.meshBuffer.AppendQuad(bl, tl, br, tr, zp, byte(b.textureIndex))
		}
	}

}
func (c *Chunk) VisibleFaceXN(i, j, k int32) bool {
	if i < 0 {
		if c.cXN == nil {
			return true
		}

		neighborBlock := c.cXN.data[31+j*CHUNK_SIZE+k*CHUNK_SIZE_SQUARED]
		return neighborBlock == nil || neighborBlock.kind == EMPTY
	}

	block := c.data[i+j*CHUNK_SIZE+k*CHUNK_SIZE_SQUARED]
	return block == nil || block.kind == EMPTY
}

func (c *Chunk) VisibleFaceXP(i, j, k int32) bool {
	if i >= CHUNK_SIZE {
		if c.cXP == nil {
			return true
		}

		neighborBlock := c.cXP.data[j*CHUNK_SIZE+k*CHUNK_SIZE_SQUARED]
		return neighborBlock == nil || neighborBlock.kind == EMPTY
	}

	block := c.data[i+j*CHUNK_SIZE+k*CHUNK_SIZE_SQUARED]
	return block == nil || block.kind == EMPTY
}

func (c *Chunk) VisibleFaceZN(i, j, k int32) bool {
	if k < 0 {
		if c.cZN == nil {
			return true
		}

		neighborBlock := c.cZN.data[i+j*CHUNK_SIZE+31*CHUNK_SIZE_SQUARED]
		return neighborBlock == nil || neighborBlock.kind == EMPTY
	}

	block := c.data[i+j*CHUNK_SIZE+k*CHUNK_SIZE_SQUARED]
	return block == nil || block.kind == EMPTY
}

func (c *Chunk) VisibleFaceZP(i, j, k int32) bool {
	if k >= CHUNK_SIZE {
		if c.cZP == nil {
			return true
		}

		neighborBlock := c.cZP.data[i+j*CHUNK_SIZE]
		return neighborBlock == nil || neighborBlock.kind == EMPTY
	}

	block := c.data[i+j*CHUNK_SIZE+k*CHUNK_SIZE_SQUARED]
	return block == nil || block.kind == EMPTY
}

func (c *Chunk) VisibleFaceYN(i, j, k int32) bool {
	if j < 0 {
		if c.cYN == nil {
			return true
		}

		neighborBlock := c.cYN.data[i+31*CHUNK_SIZE+k*CHUNK_SIZE_SQUARED]
		return neighborBlock == nil || neighborBlock.kind == EMPTY
	}

	block := c.data[i+j*CHUNK_SIZE+k*CHUNK_SIZE_SQUARED]
	return block == nil || block.kind == EMPTY
}

func (c *Chunk) VisibleFaceYP(i, j, k int32) bool {
	if j >= CHUNK_SIZE {
		if c.cYP == nil {
			return true
		}

		neighborBlock := c.cYP.data[i+k*CHUNK_SIZE_SQUARED]
		return neighborBlock == nil || neighborBlock.kind == EMPTY
	}

	block := c.data[i+j*CHUNK_SIZE+k*CHUNK_SIZE_SQUARED]
	return block == nil || block.kind == EMPTY
}

func (c *Chunk) DifferentBlock(chunkAccess int32, compare *Block) bool {
	if chunkAccess < 0 || chunkAccess >= int32(CHUNK_SIZE_CUBED) {
		return compare != nil && compare.kind != EMPTY
	}
	b := c.data[chunkAccess]
	if b == nil && compare == nil {
		return false
	} else if b == nil {
		return true
	} else if compare == nil {
		return true
	}
	return b.textureIndex != compare.textureIndex || b.health != compare.health
}

func (c *Chunk) Draw(shader *glhf.Shader, camDirection mgl32.Vec3) {
	if c.meshBuffer == nil || c.meshBuffer.TriangleCount() == 0 {
		return
	}
	shader.SetUniformAttr(2, c.GetMatrix())
	c.meshBuffer.Draw()
}
func (c *Chunk) getDiscreteCamDir(camDir mgl32.Vec3) Int3 {
	intPos := Int3{0,0,0}
	if camDir.X() < 0 {
		intPos.X = 1
	} else if camDir.X() > 0 {
		intPos.X = -1
	}

	if camDir.Y() < 0 {
		intPos.Y = 1
	} else if camDir.Y() > 0 {
		intPos.Y = -1
	}

	if camDir.Z() < 0 {
		intPos.Z = 1
	} else if camDir.Z() > 0 {
		intPos.Z = -1
	}
	return intPos
}


func (c *Chunk) GetMatrix() mgl32.Mat4 {
	return mgl32.Translate3D(float32(c.chunkPosX*CHUNK_SIZE), float32(c.chunkPosY*CHUNK_SIZE), float32(c.chunkPosZ*CHUNK_SIZE))
}

func (c *Chunk) Position() Int3 {
	return Int3{c.chunkPosX, c.chunkPosY, c.chunkPosZ}
}

func (c *Chunk) IsBlockAt(i int32, j int32, k int32) bool {
	block := c.data[i+j*CHUNK_SIZE+k*CHUNK_SIZE_SQUARED]
	return block != nil && !block.IsAir()
}

func (c *Chunk) SetDirty() {
	c.isDirty = true
}

func (c *Chunk) AABBMin() mgl32.Vec3 {
	return mgl32.Vec3{float32(c.chunkPosX * CHUNK_SIZE), float32(c.chunkPosY * CHUNK_SIZE), float32(c.chunkPosZ * CHUNK_SIZE)}
}

func (c *Chunk) AABBMax() mgl32.Vec3 {
	return mgl32.Vec3{float32(c.chunkPosX*CHUNK_SIZE + CHUNK_SIZE), float32(c.chunkPosY*CHUNK_SIZE + CHUNK_SIZE), float32(c.chunkPosZ*CHUNK_SIZE + CHUNK_SIZE)}
}