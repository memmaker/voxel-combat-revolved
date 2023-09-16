package voxel

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/glhf"
	"math"
	"os"
)

type Map struct {
	ChunkSizeHorizontal int32
	ChunkSizeHeight     int32
	ChunkSizeCube       int32

	chunks             []*Chunk
	width              int32
	height             int32
	depth              int32
	chunkShader        *glhf.Shader
	terrainTexture     *glhf.Texture
	knownUnitPositions map[uint64][]Int3
	mapInfoLogger      func(string)
	gameErrorLogger    func(string)

	spawnCounter          int
	textureCallback       func(block *Block, side FaceType) byte
	maxChunkHeightForDraw int32
}

func NewDefaultMap(width, height, depth int32) *Map {
	return NewMap(width, height, depth, 32, 32)
}

func NewMap(width, height, depth, chunkSizeHorizontal, chunkSizeHeight int32) *Map {
	m := &Map{
		ChunkSizeHorizontal:   chunkSizeHorizontal,
		ChunkSizeHeight:       chunkSizeHeight,
		chunks:                make([]*Chunk, width*height*depth),
		width:                 width,
		height:                height,
		depth:                 depth,
		knownUnitPositions:    make(map[uint64][]Int3),
		maxChunkHeightForDraw: height - 1,
	}
	m.ChunkSizeCube = m.ChunkSizeHorizontal * m.ChunkSizeHeight * m.ChunkSizeHorizontal
	//m.culler = occlusion.NewOcclusionCuller(512, m)
	return m
}

func NewMapFromSource(source []byte, shader *glhf.Shader, texture *glhf.Texture) *Map {
	m := &Map{
		chunks:              make([]*Chunk, 0),
		knownUnitPositions:  make(map[uint64][]Int3),
		chunkShader:         shader,
		terrainTexture:      texture,
	}
	m.LoadFromSource(source)
	return m
}

func (m *Map) Update(delta float64) {
	for _, chunk := range m.chunks {
		if chunk != nil {
			chunk.GenerateMesh()
			if chunk.CheckForNewMeshes() {
				return
			}
		}
	}
}
func (m *Map) SetLogger(mapLogger func(string), gameErrorLogger func(string)) {
	m.mapInfoLogger = mapLogger
	m.gameErrorLogger = gameErrorLogger
}

func (m *Map) logVoxelInfo(msg string) {
	if m.mapInfoLogger != nil {
		m.mapInfoLogger(msg)
	}
}

func (m *Map) logGameError(msg string) {
	if m.gameErrorLogger != nil {
		m.gameErrorLogger(msg)
	}
}

func (m *Map) SetSize(width, height, depth int32) {
	m.width = width
	m.height = height
	m.depth = depth
	m.chunks = make([]*Chunk, width*height*depth)
	for i := range m.chunks {
		x := i % int(width)
		y := (i / int(width)) % int(height)
		z := i / (int(width) * int(height))
		m.chunks[i] = NewChunk(m, int32(x), int32(y), int32(z))
		println("Created empty chunk at", x, y, z)
	}
}

func (m *Map) SaveToDisk(filename string) error {
	// serialize this map manually to a byte array
	outfile, err := os.Create(filename)
	if err != nil {
		return err
	}
	// use a gzip writer to compress the byte array
	// then write the compressed byte array to the file

	gzipWriter := gzip.NewWriter(outfile)
	// write 2xint32
	binary.Write(gzipWriter, binary.LittleEndian, m.ChunkSizeHorizontal)
	binary.Write(gzipWriter, binary.LittleEndian, m.ChunkSizeHeight)

	// write 3xint32
	binary.Write(gzipWriter, binary.LittleEndian, m.width)
	binary.Write(gzipWriter, binary.LittleEndian, m.height)
	binary.Write(gzipWriter, binary.LittleEndian, m.depth)
	// write the map dimensions
	m.logVoxelInfo(fmt.Sprintf("[Map] Saving map with dimensions %d %d %d", m.width, m.height, m.depth))

	// write the number of chunks
	chunkCount := int16(len(m.chunks))
	binary.Write(gzipWriter, binary.LittleEndian, chunkCount)
	m.logVoxelInfo(fmt.Sprintf("[Map] Saving %d chunks", chunkCount))

	// write the chunks
	for _, chunk := range m.chunks {
		binary.Write(gzipWriter, binary.LittleEndian, chunk.chunkPosX)
		binary.Write(gzipWriter, binary.LittleEndian, chunk.chunkPosY)
		binary.Write(gzipWriter, binary.LittleEndian, chunk.chunkPosZ)
		m.logVoxelInfo(fmt.Sprintf("[Map] Saving chunk %d %d %d", chunk.chunkPosX, chunk.chunkPosY, chunk.chunkPosZ))
		for _, block := range chunk.data {
			id := byte(0)
			if block != nil {
				id = block.ID
			}
			err = binary.Write(gzipWriter, binary.LittleEndian, id)
			if err != nil {
				return err
			}
		}
		m.logVoxelInfo(fmt.Sprintf("[Map] Saved %d blocks", len(chunk.data)))
	}

	err = gzipWriter.Close()
	if err != nil {
		return err
	}
	err = outfile.Close()
	if err != nil {
		return err
	}

	return nil
}

func (m *Map) LoadFromSource(source []byte) {
    reader := bytes.NewReader(source)
    gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		panic(err)
	}
	// read chunk size
	binary.Read(gzipReader, binary.LittleEndian, &m.ChunkSizeHorizontal)
	binary.Read(gzipReader, binary.LittleEndian, &m.ChunkSizeHeight)

	m.ChunkSizeCube = m.ChunkSizeHorizontal * m.ChunkSizeHeight * m.ChunkSizeHorizontal

	// read the map dimensions
	binary.Read(gzipReader, binary.LittleEndian, &m.width)
	binary.Read(gzipReader, binary.LittleEndian, &m.height)
	binary.Read(gzipReader, binary.LittleEndian, &m.depth)
	m.logVoxelInfo(fmt.Sprintf("[Map] Loading map with dimensions %d %d %d", m.width, m.height, m.depth))

	// read the number of chunks
	m.maxChunkHeightForDraw = m.height - 1

	chunkCount := int16(0)
	binary.Read(gzipReader, binary.LittleEndian, &chunkCount)
	m.logVoxelInfo(fmt.Sprintf("[Map] Loading %d chunks", chunkCount))

	m.chunks = make([]*Chunk, chunkCount)

	// read the chunks
	for i := int16(0); i < chunkCount; i++ {
		var chunkPos [3]int32
		binary.Read(gzipReader, binary.LittleEndian, &chunkPos[0])
		binary.Read(gzipReader, binary.LittleEndian, &chunkPos[1])
		binary.Read(gzipReader, binary.LittleEndian, &chunkPos[2])
		m.logVoxelInfo(fmt.Sprintf("[Map] Loading chunk %d %d %d", chunkPos[0], chunkPos[1], chunkPos[2]))
		chunk := NewChunk(m, chunkPos[0], chunkPos[1], chunkPos[2])
		m.chunks[i] = chunk
		for j := int32(0); j < m.ChunkSizeCube; j++ {
			blockID := byte(0)
			binary.Read(gzipReader, binary.LittleEndian, &blockID)
			chunk.data[j] = NewBlock(blockID)
		}
		chunk.SetDirty()
	}
}

func (m *Map) SetChunk(x, y, z int32, c *Chunk) {
	m.chunks[x+y*m.width+z*m.width*m.height] = c
}
func (m *Map) ChangeMaxChunkHeightForDraw(change int32) int32 {
	newValue := m.maxChunkHeightForDraw + change
	if newValue < 0 {
		newValue = 0
	} else if newValue > m.height-1 {
		newValue = m.height - 1
	}
	m.maxChunkHeightForDraw = newValue
	return newValue
}
func (m *Map) Draw(modelUniformIndex int, frustum []mgl32.Vec4) {
	m.terrainTexture.Begin()
	for _, chunk := range m.chunks {
		if chunk == nil || !m.isChunkVisibleInFrustum(frustum, chunk.Position()) || chunk.chunkPosY > m.maxChunkHeightForDraw {
			continue
		}
		chunk.Draw(m.chunkShader, modelUniformIndex)
	}
	m.terrainTexture.End()
}
func (m *Map) isChunkVisibleInFrustum(planes []mgl32.Vec4, chunkPos Int3) bool {
	p := mgl32.Vec3{float32(chunkPos.X * m.ChunkSizeHorizontal), float32(chunkPos.Y * m.ChunkSizeHeight), float32(chunkPos.Z * m.ChunkSizeHorizontal)}
	horizOffset := float32(m.ChunkSizeHorizontal)

	points := []mgl32.Vec3{
		mgl32.Vec3{p.X(), p.Y(), p.Z()},
		mgl32.Vec3{p.X() + horizOffset, p.Y(), p.Z()},
		mgl32.Vec3{p.X() + horizOffset, p.Y(), p.Z() + horizOffset},
		mgl32.Vec3{p.X(), p.Y(), p.Z() + horizOffset},

		mgl32.Vec3{p.X(), p.Y() + 256, p.Z()},
		mgl32.Vec3{p.X() + horizOffset, p.Y() + 256, p.Z()},
		mgl32.Vec3{p.X() + horizOffset, p.Y() + 256, p.Z() + horizOffset},
		mgl32.Vec3{p.X(), p.Y() + 256, p.Z() + horizOffset},
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
	return m.GetChunk(x/m.ChunkSizeHorizontal, y/m.ChunkSizeHeight, z/m.ChunkSizeHorizontal)
}
func (m *Map) GetChunk(x, y, z int32) *Chunk {
	i := x + y*m.width + z*m.width*m.height
	if i < 0 || i >= int32(len(m.chunks)) {
		return nil
	} else {
		return m.chunks[i]
	}
}
func (m *Map) ChunkExists(x, y, z int32) bool {
	return m.GetChunk(x, y, z) != nil
}
func (m *Map) SetBlock(x int32, y int32, z int32, block *Block) {
	chunkX := x / m.ChunkSizeHorizontal
	chunkY := y / m.ChunkSizeHeight
	chunkZ := z / m.ChunkSizeHorizontal
	chunk := m.GetChunk(chunkX, chunkY, chunkZ)
	if chunk != nil {
		chunk.SetBlock(x%m.ChunkSizeHorizontal, y%m.ChunkSizeHeight, z%m.ChunkSizeHorizontal, block)
	}
}

func (m *Map) SetAir(blockPos Int3) {
	m.SetBlock(blockPos.X, blockPos.Y, blockPos.Z, NewAirBlock())
}


func (m *Map) ContainsVec(pos mgl32.Vec3) bool {
	x, y, z := pos.X(), pos.Y(), pos.Z()
	return m.Contains(int32(x), int32(y), int32(z))
}

func (m *Map) Contains(x int32, y int32, z int32) bool {
	return x >= 0 && x < m.width*m.ChunkSizeHorizontal && y >= 0 && y < m.height*m.ChunkSizeHeight && z >= 0 && z < m.depth*m.ChunkSizeHorizontal
}

func (m *Map) GetBlockFromVec(pos Int3) *Block {
	return m.GetGlobalBlock(int32(pos.X), int32(pos.Y), int32(pos.Z))
}
func (m *Map) GetGlobalBlock(x int32, y int32, z int32) *Block {
	chunkX := x / m.ChunkSizeHorizontal
	chunkY := y / m.ChunkSizeHeight
	chunkZ := z / m.ChunkSizeHorizontal
	chunk := m.GetChunk(chunkX, chunkY, chunkZ)
	if chunk != nil {
		blockX := x % m.ChunkSizeHorizontal
		blockY := y % m.ChunkSizeHeight
		blockZ := z % m.ChunkSizeHorizontal
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

func (m *Map) SetShader(chunkShader *glhf.Shader) {
	m.chunkShader = chunkShader
}

func (m *Map) SetTerrainTexture(texture *glhf.Texture) {
	m.terrainTexture = texture
}

func (m *Map) NewChunk(cX int32, cY int32, cZ int32) *Chunk {
	chunk := NewChunk(m, cX, cY, cZ)
	m.SetChunk(cX, cY, cZ, chunk)
	return chunk
}

func (m *Map) SetFloorAtHeight(yLevel int, block *Block) {
	for _, chunk := range m.chunks {
		if chunk.chunkPosY != 0 {
			continue
		}
		for x := int32(0); x < m.ChunkSizeHorizontal; x++ {
			for z := int32(0); z < m.ChunkSizeHorizontal; z++ {
				chunk.SetBlock(x, int32(yLevel), z, block)
			}
		}
	}
}

type MapObject interface {
	GetOccupiedBlockOffsets(Int3) []Int3
	UnitID() uint64
	GetName() string
}

// PositionToGridInt3 converts a Vec3 to a Int3 by flooring the values.
// DO NOT USE THIS FOR NEGATIVE VALUES, eg. direction vectors
func PositionToGridInt3(pos mgl32.Vec3) Int3 {
	return Int3{int32(math.Floor(float64(pos.X()))), int32(math.Floor(float64(pos.Y()))), int32(math.Floor(float64(pos.Z())))}
}

func DirectionToGridInt3(dir mgl32.Vec3) Int3 {
	return Int3{int32(math.Round(float64(dir.X()))), int32(math.Round(float64(dir.Y()))), int32(math.Round(float64(dir.Z())))}
}
func (m *Map) IsUnitPlaceable(unit MapObject, blockPos Int3) (bool, string) {
	offsets := unit.GetOccupiedBlockOffsets(blockPos)
	for _, offset := range offsets {
		occupiedBlockPos := blockPos.Add(offset)
		outsideOfWorld := !m.ContainsGrid(occupiedBlockPos)
		if outsideOfWorld {
			return false, "Outside of world"
		}
		isWall := m.IsSolidBlockAt(occupiedBlockPos.X, occupiedBlockPos.Y, occupiedBlockPos.Z)
		if isWall {
			return false, "Wall"
		}
		unitIsBlocking := m.IsOccupiedExcept(occupiedBlockPos, unit.UnitID())
		if unitIsBlocking {
			blockingUnit := m.GetBlockFromVec(occupiedBlockPos).GetOccupant()
			return false, fmt.Sprintf("Unit %s(%d) is blocking", blockingUnit.GetName(), blockingUnit.UnitID())
		}
	}
	return true, ""
}

func (m *Map) IsHumanoidPlaceable(blockPos Int3) (bool, string) {
	offsets := []Int3{
		{0, 0, 0},
		{0, 1, 0},
	}
	for _, offset := range offsets {
		occupiedBlockPos := blockPos.Add(offset)
		outsideOfWorld := !m.ContainsGrid(occupiedBlockPos)
		if outsideOfWorld {
			return false, "Outside of world"
		}
		isWall := m.IsSolidBlockAt(occupiedBlockPos.X, occupiedBlockPos.Y, occupiedBlockPos.Z)
		if isWall {
			return false, "Wall"
		}
	}
	below := blockPos.Add(Int3{0, -1, 0})
	return m.IsSolidBlockAt(below.X, below.Y, below.Z), "No floor"
}

func (m *Map) RemoveUnit(unit MapObject) {
	currentPos, isOnMap := m.knownUnitPositions[unit.UnitID()]
	if !isOnMap {
		return
	}
	for _, occupiedBlockPos := range currentPos {
		block := m.GetGlobalBlock(occupiedBlockPos.X, occupiedBlockPos.Y, occupiedBlockPos.Z)
		if block != nil && block.IsOccupied() {
			block.RemoveUnit(unit)
		}
	}
	delete(m.knownUnitPositions, unit.UnitID())
}
func (m *Map) SetUnit(unit MapObject, blockPos Int3) bool {
	return m.SetUnitWithOffsets(unit, blockPos, unit.GetOccupiedBlockOffsets(blockPos))
}
func (m *Map) SetUnitWithOffsets(unit MapObject, blockPos Int3, offsets []Int3) bool {
	_, isOnMap := m.knownUnitPositions[unit.UnitID()]
	if isOnMap { // problem, we always remove the unit, but there are cases where we don't place it back
		m.RemoveUnit(unit)
	} // reason:
	ok, reason := m.IsUnitPlaceable(unit, blockPos)
	if ok {
		occupiedBlocks := make([]Int3, len(offsets))
        //println(fmt.Sprintf("[Map] Placed %s(%d) at %s occupying %d blocks:", unit.GetName(), unit.Attacker(), blockPos.ToString(), len(offsets)))
		for index, offset := range offsets {
			occupiedBlockPos := blockPos.Add(offset)
			block := m.GetGlobalBlock(occupiedBlockPos.X, occupiedBlockPos.Y, occupiedBlockPos.Z)
			block.AddUnit(unit)
			//println(fmt.Sprintf("[Map] - %s", occupiedBlockPos.ToString()))
			occupiedBlocks[index] = occupiedBlockPos
		}
		m.knownUnitPositions[unit.UnitID()] = occupiedBlocks
		return true
	}
	m.logGameError(fmt.Sprintf("[Map] ERR - Failed to place %s(%d): %s (%s)", unit.GetName(), unit.UnitID(), blockPos.ToString(), reason))
	return false
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
	for blockName := range blocks {
		blockNames = append(blockNames, blockName)
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
	for name := range blockEntities {
		blockEntityNames = append(blockEntityNames, name)
	}
	return blockEntityNames
}
func NewMapFromConstruction(bf *BlockFactory, chunkShader *glhf.Shader, construction *Construction, chunkSize Int3) *Map {
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
	chunkCountX := int32(math.Ceil(float64(maxX-minX) / float64(chunkSize.X)))
	chunkCountY := int32(math.Ceil(float64(maxY-minY) / float64(chunkSize.Y)))
	chunkCountZ := int32(math.Ceil(float64(maxZ-minZ) / float64(chunkSize.Z)))
	voxelMap := NewDefaultMap(chunkCountX, chunkCountY, chunkCountZ)
	voxelMap.SetShader(chunkShader)
	chunkCounter := 0
	for cX := int32(0); cX < chunkCountX; cX++ {
		for cY := int32(0); cY < chunkCountY; cY++ {
			for cZ := int32(0); cZ < chunkCountZ; cZ++ {
				voxelMap.NewChunk(cX, cY, cZ)
				chunkCounter++
			}
		}
	}
	// offsets for alignment at 0,0,0
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
		println("[Map] Unknown block: " + blockname)
	}

	return voxelMap
}

var cardinalNeighbors = []Int3{
	{1, 0, 0},
	{-1, 0, 0},
	{0, 0, 1},
	{0, 0, -1},
}

var cardinalNeighborsUpDown = []Int3{
	{0, 1, 0},
	{0, -1, 0},
}
var cardinalNeighbors3D = append(cardinalNeighbors, cardinalNeighborsUpDown...)
var diagonalNeighbors = []Int3{
	{1, 0, 1},
	{-1, 0, 1},
	{1, 0, -1},
	{-1, 0, -1},
}

var allNeighbors = append(cardinalNeighbors, diagonalNeighbors...)

func (m *Map) GetBlockedCardinalNeighborsUpToHeight(unitID uint64, position Int3, height int32) []Int3 {
	neighbors := make([]Int3, 0, 4)
	for _, offset := range cardinalNeighbors {
		neighbor := position.Add(offset)
		for y := int32(0); y < height; y++ {
			if m.IsSolidBlockAt(neighbor.X, neighbor.Y+y, neighbor.Z) || m.IsOccupiedExcept(neighbor.Add(Int3{0, y, 0}), unitID) {
				neighbors = append(neighbors, neighbor)
				break
			}
		}
	}
	return neighbors
}

func (m *Map) GetNeighborsForGroundMovement(block Int3, keepPredicate func(neighbor Int3) bool) []Int3 {
	neighbors := make([]Int3, 0, 8)

	for _, offset := range cardinalNeighbors {
		neighbor := block.Add(offset)
		if m.ContainsGrid(neighbor) {
			if m.IsSolidBlockAt(neighbor.X, neighbor.Y, neighbor.Z) {
				// check one block above (climbing one block is allowed)
				neighbor = neighbor.Add(Int3{Y: 1})
			} else {
				// get the lowest solid block below and test the block above that
				neighbor = m.GetGroundPosition(neighbor)
			}
			if m.ContainsGrid(neighbor) && keepPredicate(neighbor) {
				neighbors = append(neighbors, neighbor)
			}
		}
	}

	for _, offset := range diagonalNeighbors {
		neighbor := block.Add(offset)
		neighbor = m.GetGroundPosition(neighbor)
		if m.ContainsGrid(neighbor) && keepPredicate(neighbor) {
			neighbors = append(neighbors, neighbor)
		}
	}

	return neighbors
}

func (m *Map) GetGroundPosition(startBlock Int3) Int3 {
	// iterate down until we hit a solid block
	for y := startBlock.Y; y >= 1; y-- {
		if m.IsSolidBlockAt(startBlock.X, y-1, startBlock.Z) || !m.ContainsGrid(Int3{startBlock.X, y - 1, startBlock.Z}) {
			return Int3{startBlock.X, y, startBlock.Z}
		}
	}
	return startBlock
}
func (m *Map) IsOccupied(blockPos Int3) bool {
	block := m.GetBlockFromVec(blockPos)
	return block != nil && block.IsOccupied()
}

func (m *Map) IsOccupiedExcept(blockPos Int3, unitID uint64) bool {
	block := m.GetBlockFromVec(blockPos)
	occupied := block.IsOccupied()
	return block != nil && occupied && block.GetOccupant().UnitID() != unitID
}

func (m *Map) GetMapObjectAt(target Int3) MapObject {
	block := m.GetBlockFromVec(target)
	if block != nil {
		return block.GetOccupant()
	}
	return nil
}

func (m *Map) getTextureIndexForSide(block *Block, side FaceType) byte {
	if m.textureCallback != nil {
		return m.textureCallback(block, side)
	}
	return block.ID - 1
}

func (m *Map) SetTextureIndexCallback(callback func(block *Block, side FaceType) byte) {
	m.textureCallback = callback
}

func (m *Map) ClearAllChunks() {
	for _, chunk := range m.chunks {
		if chunk != nil {
			chunk.ClearAllBlocks()
		}
	}
}

func (m *Map) GetTerrainTexture() *glhf.Texture {
	return m.terrainTexture
}

func (m *Map) IsUnitOnMap(u MapObject) bool {
	_, ok := m.knownUnitPositions[u.UnitID()]
	return ok
}

func (m *Map) DebugGetAllOccupiedBlocks() []Int3 {
	result := make([]Int3, 0)
	for _, block := range m.knownUnitPositions {
		result = append(result, block...)
	}
	return result
}

func (m *Map) DebugGetOccupiedBlocks(id uint64) []Int3 {
	blocks, exists := m.knownUnitPositions[id]
	if !exists {
		return []Int3{}
	}
	return blocks
}

func (m *Map) GetSize() Int3 {
	return Int3{m.width, m.height, m.depth}
}

func (m *Map) CurrentlyDraws(x int32, y int32, z int32) bool {
	chunk := m.GetChunkFromBlock(x, y, z)
	return chunk != nil && chunk.chunkPosY <= m.maxChunkHeightForDraw
}

func (m *Map) ForBlockInHalfSphere(origin Int3, radius float64, applyToBlock func(origin Int3, radius float64, x int32, y int32, z int32)) {
	for x := int32(-radius); x <= int32(radius); x++ {
		for y := int32(0); y <= int32(radius); y++ {
			for z := int32(-radius); z <= int32(radius); z++ {
				if float64(x*x+y*y+z*z) <= radius*radius {
					if !m.Contains(origin.X+x, origin.Y+y, origin.Z+z) {
						continue
					}
					applyToBlock(origin, radius, origin.X+x, origin.Y+y, origin.Z+z)
				}
			}
		}
	}
}
func (m *Map) ForBlockInSphere(origin Int3, radius float64, applyToBlock func(origin Int3, radius float64, x int32, y int32, z int32)) {
	for x := int32(-radius); x <= int32(radius); x++ {
		for y := int32(-radius); y <= int32(radius); y++ {
			for z := int32(-radius); z <= int32(radius); z++ {
				if float64(x*x+y*y+z*z) <= radius*radius {
					if !m.Contains(origin.X+x, origin.Y+y, origin.Z+z) {
						continue
					}
					applyToBlock(origin, radius, origin.X+x, origin.Y+y, origin.Z+z)
				}
			}
		}
	}
}

type DistLocPair struct {
	Steps    int
	Distance float64
	Location Int3
}

func (m *Map) ForBlockInSphericFloodFill(origin Int3, radius float64, applyToBlock func(origin Int3, steps int, distance float64, x int32, y int32, z int32)) {
	visited := make(map[Int3]bool)
	visited[origin] = true
	queue := []DistLocPair{{0, 0, origin}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		applyToBlock(origin, current.Steps, current.Distance, current.Location.X, current.Location.Y, current.Location.Z)
		for _, offset := range cardinalNeighbors3D {
			neighbor := current.Location.Add(offset)
			if !m.ContainsGrid(neighbor) {
				continue
			}
			_, alreadyVisited := visited[neighbor]
			if alreadyVisited {
				continue
			}
			if m.IsSolidBlockAt(neighbor.X, neighbor.Y, neighbor.Z) {
				continue
			}
			neighborDistance := origin.Sub(neighbor).Length()
			neighborSteps := current.Steps + 1
			if neighborDistance > radius {
				continue
			}
			if float64(neighborSteps) > 2*radius {
				continue
			}
			visited[neighbor] = true
			queue = append(queue, DistLocPair{neighborSteps, neighborDistance, neighbor})
		}
	}
}

