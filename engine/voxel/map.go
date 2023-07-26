package voxel

import (
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"math"
	"math/rand"
	"os"
)

type Map struct {
	chunks         []*Chunk
	width          int32
	height         int32
	depth          int32
	chunkShader    *glhf.Shader
	terrainTexture *glhf.Texture
}

func NewMap(width, height, depth int32) *Map {
	m := &Map{
		chunks: make([]*Chunk, width*height*depth),
		width:  width,
		height: height,
		depth:  depth,
	}
	//m.culler = occlusion.NewOcclusionCuller(512, m)
	return m
}

func (m *Map) SaveToDisk() {
	// serialize this map manually to a byte array
	outfile, err := os.Create("assets/maps/map.bin")
	if err != nil {
		panic(err)
	}
	// use a gzip writer to compress the byte array
	// then write the compressed byte array to the file

	gzipWriter := gzip.NewWriter(outfile)
	// write 3xint32
	binary.Write(gzipWriter, binary.LittleEndian, m.width)
	binary.Write(gzipWriter, binary.LittleEndian, m.height)
	binary.Write(gzipWriter, binary.LittleEndian, m.depth)
	// write the map dimensions
	println(fmt.Sprintf("[Map] Saving map with dimensions %d %d %d", m.width, m.height, m.depth))

	// write the number of chunks
	chunkCount := int16(len(m.chunks))
	binary.Write(gzipWriter, binary.LittleEndian, chunkCount)
	println(fmt.Sprintf("[Map] Saving %d chunks", chunkCount))

	// write the chunks
	for _, chunk := range m.chunks {
		binary.Write(gzipWriter, binary.LittleEndian, chunk.chunkPosX)
		binary.Write(gzipWriter, binary.LittleEndian, chunk.chunkPosY)
		binary.Write(gzipWriter, binary.LittleEndian, chunk.chunkPosZ)
		println(fmt.Sprintf("[Map] Saving chunk %d %d %d", chunk.chunkPosX, chunk.chunkPosY, chunk.chunkPosZ))
		for _, block := range chunk.data {
			binary.Write(gzipWriter, binary.LittleEndian, block.ID)
		}
		println(fmt.Sprintf("[Map] Saved %d blocks", len(chunk.data)))
	}

	gzipWriter.Close()
	outfile.Close()
}

func (m *Map) LoadFromDisk() {
	filename := "assets/maps/map.bin"
	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		panic(err)
	}

	// read the map dimensions
	binary.Read(gzipReader, binary.LittleEndian, &m.width)
	binary.Read(gzipReader, binary.LittleEndian, &m.height)
	binary.Read(gzipReader, binary.LittleEndian, &m.depth)
	println(fmt.Sprintf("[Map] Loading map with dimensions %d %d %d", m.width, m.height, m.depth))

	// read the number of chunks

	chunkCount := int16(0)
	binary.Read(gzipReader, binary.LittleEndian, &chunkCount)
	println(fmt.Sprintf("[Map] Loading %d chunks", chunkCount))

	m.chunks = make([]*Chunk, chunkCount)

	// read the chunks
	for i := int16(0); i < chunkCount; i++ {
		var chunkPos [3]int32
		binary.Read(gzipReader, binary.LittleEndian, &chunkPos[0])
		binary.Read(gzipReader, binary.LittleEndian, &chunkPos[1])
		binary.Read(gzipReader, binary.LittleEndian, &chunkPos[2])
		println(fmt.Sprintf("[Map] Loading chunk %d %d %d", chunkPos[0], chunkPos[1], chunkPos[2]))
		chunk := NewChunk(m.chunkShader, m, chunkPos[0], chunkPos[1], chunkPos[2])
		m.chunks[i] = chunk
		for j := int32(0); j < CHUNK_SIZE_CUBED; j++ {
			blockID := byte(0)
			binary.Read(gzipReader, binary.LittleEndian, &blockID)
			chunk.data[j] = NewBlock(blockID)
		}
		chunk.SetDirty()
	}
	m.GenerateAllMeshes()
}

func (m *Map) GetChunk(x, y, z int32) *Chunk {
	i := x + y*m.width + z*m.width*m.height
	if i < 0 || i >= int32(len(m.chunks)) {
		return nil
	} else {
		return m.chunks[i]
	}
}

func (m *Map) SetChunk(x, y, z int32, c *Chunk) {
	m.chunks[x+y*m.width+z*m.width*m.height] = c
}

func (m *Map) Draw(camDirection mgl32.Vec3, frustum []mgl32.Vec4) {
	m.terrainTexture.Begin()
	for _, chunk := range m.chunks {
		if chunk == nil || !isChunkVisibleInFrustum(frustum, chunk.Position()) {
			continue
		}
		chunk.Draw(m.chunkShader, camDirection)
	}
	m.terrainTexture.End()
}
func isChunkVisibleInFrustum(planes []mgl32.Vec4, chunkPos Int3) bool {
	p := mgl32.Vec3{float32(chunkPos.X * CHUNK_SIZE), float32(chunkPos.Y * CHUNK_SIZE), float32(chunkPos.Z * CHUNK_SIZE)}
	const m = float32(CHUNK_SIZE)

	points := []mgl32.Vec3{
		mgl32.Vec3{p.X(), p.Y(), p.Z()},
		mgl32.Vec3{p.X() + m, p.Y(), p.Z()},
		mgl32.Vec3{p.X() + m, p.Y(), p.Z() + m},
		mgl32.Vec3{p.X(), p.Y(), p.Z() + m},

		mgl32.Vec3{p.X(), p.Y() + 256, p.Z()},
		mgl32.Vec3{p.X() + m, p.Y() + 256, p.Z()},
		mgl32.Vec3{p.X() + m, p.Y() + 256, p.Z() + m},
		mgl32.Vec3{p.X(), p.Y() + 256, p.Z() + m},
	}
	for _, plane := range planes {
		var in, out int
		for _, point := range points {
			if plane.Dot(point.Vec4(1)) < 0 {
				out++
			} else {
				in++
			}
			if in != 0 && out != 0 {
				break
			}
		}
		if in == 0 {
			return false
		}
	}
	return true
}

func (m *Map) GenerateAllMeshes() {
	totalTriangles := 0
	for _, chunk := range m.chunks {
		if chunk != nil {
			meshBuffer := chunk.GreedyMeshing()
			totalTriangles += meshBuffer.TriangleCount()
			if meshBuffer.TriangleCount() > 0 {
				meshBuffer.FlushMesh()
			}
		}
	}
	println(fmt.Sprintf("[Greedy] Total triangles: %d", totalTriangles))
}

func (m *Map) GetChunkFromPosition(pos mgl32.Vec3) *Chunk {
	x := math.Floor(float64(pos.X()))
	y := math.Floor(float64(pos.Y()))
	z := math.Floor(float64(pos.Z()))
	return m.GetChunkFromBlock(int32(x), int32(y), int32(z))
}

func (m *Map) GetBlockFromPosition(position mgl32.Vec3) *Block {
	x := math.Floor(float64(position.X()))
	y := math.Floor(float64(position.Y()))
	z := math.Floor(float64(position.Z()))
	return m.GetGlobalBlock(int32(x), int32(y), int32(z))
}

func (m *Map) GetChunkFromBlock(x, y, z int32) *Chunk {
	return m.GetChunk(x/CHUNK_SIZE, y/CHUNK_SIZE, z/CHUNK_SIZE)
}
func (m *Map) ChunkExists(x, y, z int32) bool {
	return m.GetChunk(x, y, z) != nil
}
func (m *Map) SetBlock(x int32, y int32, z int32, block *Block) {
	chunkX := x / CHUNK_SIZE
	chunkY := y / CHUNK_SIZE
	chunkZ := z / CHUNK_SIZE
	chunk := m.GetChunk(chunkX, chunkY, chunkZ)
	if chunk != nil {
		chunk.SetBlock(x%CHUNK_SIZE, y%CHUNK_SIZE, z%CHUNK_SIZE, block)
	}
}

func (m *Map) ContainsVec(pos mgl32.Vec3) bool {
	x, y, z := pos.X(), pos.Y(), pos.Z()
	return m.Contains(int32(x), int32(y), int32(z))
}

func (m *Map) Contains(x int32, y int32, z int32) bool {
	return x >= 0 && x < m.width*CHUNK_SIZE && y >= 0 && y < m.height*CHUNK_SIZE && z >= 0 && z < m.depth*CHUNK_SIZE
}

func (m *Map) GetBlockFromVec(pos Int3) *Block {
	return m.GetGlobalBlock(int32(pos.X), int32(pos.Y), int32(pos.Z))
}
func (m *Map) GetGlobalBlock(x int32, y int32, z int32) *Block {
	chunkX := x / CHUNK_SIZE
	chunkY := y / CHUNK_SIZE
	chunkZ := z / CHUNK_SIZE
	chunk := m.GetChunk(chunkX, chunkY, chunkZ)
	if chunk != nil {
		blockX := x % CHUNK_SIZE
		blockY := y % CHUNK_SIZE
		blockZ := z % CHUNK_SIZE
		return chunk.GetLocalBlock(blockX, blockY, blockZ)
	} else {
		return nil
	}
}

func (m *Map) IsSolidBlockAt(x int32, y int32, z int32) bool {
	block := m.GetGlobalBlock(x, y, z)
	return block != nil && !block.IsAir()
}

func (m *Map) ContainsGrid(position Int3) bool {
	return m.Contains(position.X, position.Y, position.Z)
}

func (m *Map) SetChunkShader(shader *glhf.Shader) {
	m.chunkShader = shader
}

func (m *Map) SetTerrainTexture(texture *glhf.Texture) {
	m.terrainTexture = texture
}

func (m *Map) NewChunk(cX int32, cY int32, cZ int32) *Chunk {
	chunk := NewChunk(m.chunkShader, m, cX, cY, cZ)
	m.SetChunk(cX, cY, cZ, chunk)
	return chunk
}

func (m *Map) SetFloorAtHeight(yLevel int, block *Block) {
	for _, chunk := range m.chunks {
		for x := int32(0); x < CHUNK_SIZE; x++ {
			for z := int32(0); z < CHUNK_SIZE; z++ {
				chunk.SetBlock(x, int32(yLevel), z, block)
			}
		}
	}
}

func (m *Map) SetSetRandomStuff(block *Block) {
	for _, chunk := range m.chunks {
		if chunk.chunkPosX == 0 && chunk.chunkPosZ == 0 && chunk.chunkPosY == 0 {
			continue
		}
		for x := int32(0); x < CHUNK_SIZE; x++ {
			for z := int32(0); z < CHUNK_SIZE; z++ {
				randomHeight := rand.Intn(10)
				chunk.SetBlock(x, int32(randomHeight), z, block)
			}
		}
	}
}

type MapObject interface {
	GetOccupiedBlockOffsets() []Int3
	GetPosition() mgl32.Vec3
}

func (m *Map) MoveUnitTo(unit MapObject, pos mgl32.Vec3) bool {
	if m.IsUnitPlaceable(unit, pos) {
		m.RemoveUnit(unit, unit.GetPosition())
		m.AddUnit(unit, pos)
		return true
	}
	return false
}
func ToGridInt3(pos mgl32.Vec3) Int3 {
	return Int3{int32(math.Floor(float64(pos.X()))), int32(math.Floor(float64(pos.Y()))), int32(math.Floor(float64(pos.Z())))}
}
func (m *Map) IsUnitPlaceable(unit MapObject, pos mgl32.Vec3) bool {
	blockPos := ToGridInt3(pos)
	offsets := unit.GetOccupiedBlockOffsets()
	for _, offset := range offsets {
		occupiedBlockPos := blockPos.Add(offset)
		if !m.ContainsGrid(occupiedBlockPos) || m.IsSolidBlockAt(occupiedBlockPos.X, occupiedBlockPos.Y, occupiedBlockPos.Z) {
			return false
		}
	}
	return true
}

func (m *Map) RemoveUnit(unit MapObject, position mgl32.Vec3) {
	blockPos := ToGridInt3(position)
	offsets := unit.GetOccupiedBlockOffsets()
	for _, offset := range offsets {
		occupiedBlockPos := blockPos.Add(offset)
		block := m.GetGlobalBlock(occupiedBlockPos.X, occupiedBlockPos.Y, occupiedBlockPos.Z)
		block.RemoveUnit(unit)
	}
}

func (m *Map) AddUnit(unit MapObject, pos mgl32.Vec3) {
	blockPos := ToGridInt3(pos)
	offsets := unit.GetOccupiedBlockOffsets()
	for _, offset := range offsets {
		occupiedBlockPos := blockPos.Add(offset)
		block := m.GetGlobalBlock(occupiedBlockPos.X, occupiedBlockPos.Y, occupiedBlockPos.Z)
		block.AddUnit(unit)
	}
}

func IsObstacle(vec *Block) bool {
	return vec != nil && !vec.IsAir()
}

func GetBlocksNeededByConstruction(construction *Construction) []string {
	blocks := make(map[string]bool)
	for _, section := range construction.Sections {
		for _, block := range section.Blocks {
			if block == nil {
				continue
			}
			blocks[block.Name] = true
		}
	}
	var blockNames []string
	println("[Map] Blocks needed by construction:")
	for blockName := range blocks {
		blockNames = append(blockNames, blockName)
		println("[Map] -", blockName)
	}
	return blockNames
}
func GetBlockEntitiesNeededByConstruction(construction *Construction) []string {
	blockEntities := make(map[string]bool)
	for _, section := range construction.Sections {
		for _, blockEntity := range section.BlockEntities {
			blockEntities[blockEntity.Name] = true
		}
	}
	var blockEntityNames []string
	println("[Map] BlockEntities needed by construction:")
	for name := range blockEntities {
		blockEntityNames = append(blockEntityNames, name)
		println("[Map] -", name)
	}
	return blockEntityNames
}
func NewMapFromConstruction(bf *BlockFactory, shader *glhf.Shader, construction *Construction) *Map {
	minX, minY, minZ := int32(math.MaxInt32), int32(math.MaxInt32), int32(math.MaxInt32)
	maxX, maxY, maxZ := int32(math.MinInt32), int32(math.MinInt32), int32(math.MinInt32)
	for _, section := range construction.Sections {
		if section.MinBlockX < minX {
			minX = section.MinBlockX
		}
		if section.MinBlockY < minY {
			minY = section.MinBlockY
		}
		if section.MinBlockZ < minZ {
			minZ = section.MinBlockZ
		}
		if section.MinBlockX+int32(section.ShapeX) > maxX {
			maxX = section.MinBlockX + int32(section.ShapeX)
		}
		if section.MinBlockY+int32(section.ShapeY) > maxY {
			maxY = section.MinBlockY + int32(section.ShapeY)
		}
		if section.MinBlockZ+int32(section.ShapeZ) > maxZ {
			maxZ = section.MinBlockZ + int32(section.ShapeZ)
		}
	}
	println(fmt.Sprintf("[Map] Construction bounds: %d %d %d - %d %d %d", minX, minY, minZ, maxX, maxY, maxZ))
	chunkCountX := int32(math.Ceil(float64(maxX-minX) / float64(CHUNK_SIZE)))
	chunkCountY := int32(math.Ceil(float64(maxY-minY) / float64(CHUNK_SIZE)))
	chunkCountZ := int32(math.Ceil(float64(maxZ-minZ) / float64(CHUNK_SIZE)))
	println(fmt.Sprintf("[Map] Chunk count: %d %d %d", chunkCountX, chunkCountY, chunkCountZ))
	voxelMap := NewMap(chunkCountX, chunkCountY, chunkCountZ)
	voxelMap.SetChunkShader(shader)
	chunkCounter := 0
	for cX := int32(0); cX < chunkCountX; cX++ {
		for cY := int32(0); cY < chunkCountY; cY++ {
			for cZ := int32(0); cZ < chunkCountZ; cZ++ {
				voxelMap.NewChunk(cX, cY, cZ)
				println(fmt.Sprintf("[Map] Created chunk %d %d %d", cX, cY, cZ))
				chunkCounter++
			}
		}
	}
	// offsets for allignment at 0,0,0
	xChunkOffset := int32(0)
	yChunkOffset := int32(0)
	zChunkOffset := int32(0)
	if minX != 0 {
		xChunkOffset = -minX
	}
	if minY != 0 {
		yChunkOffset = -minY
	}
	if minZ != 0 {
		zChunkOffset = -minZ
	}
	blockCounter := 0
	println(fmt.Sprintf("[Map] Chunk offset for blocks: %d %d %d", xChunkOffset, yChunkOffset, zChunkOffset))
	for _, section := range construction.Sections {
		blockIndex := 0
		for x := section.MinBlockX; x < section.MinBlockX+int32(section.ShapeX); x++ {
			for y := section.MinBlockY; y < section.MinBlockY+int32(section.ShapeY); y++ {
				for z := section.MinBlockZ; z < section.MinBlockZ+int32(section.ShapeZ); z++ {
					sourceBlockDef := section.Blocks[blockIndex]
					if sourceBlockDef == nil {
						sourceBlockDef = &BlockDefinition{
							Name:      "air",
							NameSpace: "minecraft",
						}
					}
					alignedX := x + xChunkOffset
					alignedY := y + yChunkOffset
					alignedZ := z + zChunkOffset
					voxelMap.SetBlock(alignedX, alignedY, alignedZ, bf.GetBlockByName(sourceBlockDef.Name))
					blockCounter++
					blockIndex++
				}
			}
		}

		for _, blockEntityDef := range section.BlockEntities {
			alignedX := blockEntityDef.X + xChunkOffset
			alignedY := blockEntityDef.Y + yChunkOffset
			alignedZ := blockEntityDef.Z + zChunkOffset
			voxelMap.SetBlock(alignedX, alignedY, alignedZ, bf.GetBlockByName(blockEntityDef.Name))
		}
	}

	// debug
	for blockname := range bf.UnknownBlocks {
		println("[Map] Unknown block:", blockname)
	}

	println(fmt.Sprintf("[Map] Loaded map with %d blocks in %d chunks", blockCounter, chunkCounter))

	return voxelMap
}
