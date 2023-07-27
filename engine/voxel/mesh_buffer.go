package voxel

import (
	"github.com/memmaker/battleground/engine/glhf"
	"math"
)

type DrawMode int

const (
	Flat DrawMode = iota
	Indexed
	Partial
)

type MeshBuffer struct {
	drawableVertexData *glhf.VertexSlice[glhf.GlInt]
	flatVertexData     []glhf.GlInt
	sortedVertexData   map[FaceType][]glhf.GlInt
	shader             *glhf.Shader
	vertexCount        int
	indexMap           map[int32]uint32
	indexBuffer        []uint32
	drawMode           DrawMode
	faceMap            map[Int3]MultiDrawIndex
}

func (m *MeshBuffer) GetShader() *glhf.Shader {
	return m.shader
}

type MultiDrawIndex struct {
	Start []int32
	Count []int32
}

func (m *MeshBuffer) TriangleCount() int {
	return m.vertexCount / 3
}

// Possible Performance gains..
// Further reduce the amount of data we need to send to the GPU
// Use indexed drawing?

func (m *MeshBuffer) AppendQuad(tr, br, bl, tl Int3, normal FaceType, textureIndex byte, extraBits [4]uint8) {
	//println(fmt.Sprintf("bl: %v, tl: %v, br: %v, tr: %v, normal: %v, textureIndex: %v", bl, tl, br, tr, normal, textureIndex))
	// min => bl, max => tr
	// 8+3 = 11 bits + 4*18 bits => 72 + 11 = 83 bits, one 64bit and one 32bit integer => 96 bits
	compressedVertexTR := m.Compress(tr, normal, textureIndex, extraBits[0])
	compressedVertexBR := m.Compress(br, normal, textureIndex, extraBits[1])
	compressedVertexBL := m.Compress(bl, normal, textureIndex, extraBits[2])
	compressedVertexTL := m.Compress(tl, normal, textureIndex, extraBits[3]) // we use 29 of 32 bits, only 18 are different between the vertices

	// quad info:
	// 4x Position (x,y,z) => 4x3x6 bits = 72 bits
	// 1x FaceType (normal) => 1x3 bits = 3 bits
	// 1x TextureIndex => 1x8 bits = 8 bits
	// -> 83 bits

	// we are using 6x 32 bits = 192 bits

	reverseOrder := normal%2 == 1
	if reverseOrder {
		m.addVertex(compressedVertexTL, normal)
		m.addVertex(compressedVertexBL, normal)
		m.addVertex(compressedVertexTR, normal)

		m.addVertex(compressedVertexBL, normal)
		m.addVertex(compressedVertexBR, normal)
		m.addVertex(compressedVertexTR, normal)

	} else {
		// tl,bl,tr clockwise
		m.addVertex(compressedVertexTR, normal)
		m.addVertex(compressedVertexBL, normal)
		m.addVertex(compressedVertexTL, normal)

		// tr,bl,br clockwise
		m.addVertex(compressedVertexTR, normal)
		m.addVertex(compressedVertexBR, normal)
		m.addVertex(compressedVertexBL, normal)
	}

	m.vertexCount += 6
}

func (m *MeshBuffer) addVertex(vertex int32, normal FaceType) {
	if m.drawMode == Indexed {
		m.addIndexedVertex(vertex)
	} else if m.drawMode == Partial {
		m.addVertexSorted(vertex, normal)
	} else {
		m.addFlatVertex(vertex)
	}
}

func (m *MeshBuffer) addFlatVertex(vertex int32) {
	m.flatVertexData = append(m.flatVertexData, glhf.GlInt(vertex))
}

func (m *MeshBuffer) addVertexSorted(vertex int32, normal FaceType) {
	m.sortedVertexData[normal] = append(m.sortedVertexData[normal], glhf.GlInt(vertex))
}

func (m *MeshBuffer) addIndexedVertex(vertex int32) {
	if vertexIndex, isCached := m.indexMap[vertex]; isCached {
		m.indexBuffer = append(m.indexBuffer, vertexIndex)
		return
	}
	vertexIndex := len(m.flatVertexData)
	m.flatVertexData = append(m.flatVertexData, glhf.GlInt(vertex))
	if vertexIndex > math.MaxUint32 {
		println("vertexIndex out of bounds: ", vertexIndex)
	}
	vertexIndexUnsigned := uint32(vertexIndex)
	m.indexMap[vertex] = vertexIndexUnsigned
	m.indexBuffer = append(m.indexBuffer, vertexIndexUnsigned)
}

func (m *MeshBuffer) Reset() {
	m.indexMap = make(map[int32]uint32)
	m.indexBuffer = m.indexBuffer[:0]
	m.flatVertexData = m.flatVertexData[:0]
	m.vertexCount = 0
}

func (m *MeshBuffer) FlushMesh() {
	if m.drawMode == Indexed {
		m.drawableVertexData = glhf.MakeIntVertexSlice(m.shader, m.vertexCount, m.vertexCount, m.indexBuffer)
	} else {
		m.drawableVertexData = glhf.MakeIntVertexSlice(m.shader, m.vertexCount, m.vertexCount, nil)
	}
	m.drawableVertexData.Begin()
	if m.drawMode == Partial {
		m.drawableVertexData.SetVertexData(m.preparePartialVertexData(m.sortedVertexData))
	} else {
		m.drawableVertexData.SetVertexData(m.flatVertexData)
	}

	m.drawableVertexData.End()
}

func (m *MeshBuffer) MergeBuffer(buffer ChunkMesh) {
	if _, ok := buffer.(*MeshBuffer); !ok || buffer == nil {
		return
	}
	otherBuffer := buffer.(*MeshBuffer)
	m.flatVertexData = append(m.flatVertexData, otherBuffer.flatVertexData...)
	m.vertexCount += otherBuffer.vertexCount
}

func (m *MeshBuffer) Draw() {
	if m.drawableVertexData == nil {
		return
	}
	m.drawableVertexData.Begin()
	m.drawableVertexData.Draw()
	m.drawableVertexData.End()
}

func (m *MeshBuffer) PartialDraw(camDirection Int3) {
	if m.drawableVertexData == nil {
		return
	}
	if faceIndices, ok := m.faceMap[camDirection]; ok {
		m.drawableVertexData.Begin()
		m.drawableVertexData.MultiDraw(faceIndices.Start, faceIndices.Count)
		m.drawableVertexData.End()
	}
}

// Compresses the position, normal direction and texture index into a 32 bit integer.
func (m *MeshBuffer) Compress(position Int3, normalDirection FaceType, textureIndex byte, extraBits uint8) int32 {
	// 6 bits for the x y z axis
	// max value for each axis is 2^6 - 1 = 63
	// we want to pack these into one 32 bit integer
	maxX := int32(63)
	maxY := int32(63)
	maxZ := int32(63)

	xAxis := position.X
	yAxis := position.Y << 6
	zAxis := position.Z << 12

	if position.X < 0 || position.X > maxX {
		println("x axis out of bounds: ", position.X)
	}
	if position.Y < 0 || position.Y > maxY {
		println("y axis out of bounds: ", position.Y)
	}
	if position.Z < 0 || position.Z > maxZ {
		println("z axis out of bounds: ", position.Z)
	}
	compressedPosition := xAxis | yAxis | zAxis

	// 3 bits for the normal direction (0..5)
	attributes := int32(normalDirection) << 18
	// 8 bits for the texture index (0..255)
	attributes |= int32(textureIndex) << 21

	// add the first three extra bits
	attributes |= int32(extraBits) << 29

	compressedVertex := compressedPosition | attributes
	// total: 29 bits, 3 bits left
	return compressedVertex
}

func (m *MeshBuffer) preparePartialVertexData(data map[FaceType][]glhf.GlInt) []glhf.GlInt {
	mergedData := make([]glhf.GlInt, 0, 0)
	faceVertices := make(map[FaceType][2]int32, 0)
	for normalDir, vertexData := range data {
		startIndex, count := len(mergedData), len(vertexData)
		mergedData = append(mergedData, vertexData...)
		faceVertices[normalDir] = [2]int32{int32(startIndex), int32(count)}
	}
	m.faceMap[Int3{0, 0, 1}] = MultiDrawIndex{Start: []int32{faceVertices[ZP][0]}, Count: []int32{faceVertices[ZP][1]}}
	m.faceMap[Int3{0, 0, -1}] = MultiDrawIndex{Start: []int32{faceVertices[ZN][0]}, Count: []int32{faceVertices[ZN][1]}}
	m.faceMap[Int3{0, 1, 0}] = MultiDrawIndex{Start: []int32{faceVertices[YP][0]}, Count: []int32{faceVertices[YP][1]}}
	m.faceMap[Int3{0, -1, 0}] = MultiDrawIndex{Start: []int32{faceVertices[YN][0]}, Count: []int32{faceVertices[YN][1]}}
	m.faceMap[Int3{1, 0, 0}] = MultiDrawIndex{Start: []int32{faceVertices[XP][0]}, Count: []int32{faceVertices[XP][1]}}
	m.faceMap[Int3{-1, 0, 0}] = MultiDrawIndex{Start: []int32{faceVertices[XN][0]}, Count: []int32{faceVertices[XN][1]}}

	// two axis
	m.faceMap[Int3{0, 1, 1}] = MultiDrawIndex{Start: []int32{faceVertices[ZP][0], faceVertices[YP][0]}, Count: []int32{faceVertices[ZP][1], faceVertices[YP][1]}}
	m.faceMap[Int3{0, 1, -1}] = MultiDrawIndex{Start: []int32{faceVertices[ZN][0], faceVertices[YP][0]}, Count: []int32{faceVertices[ZN][1], faceVertices[YP][1]}}
	m.faceMap[Int3{0, -1, 1}] = MultiDrawIndex{Start: []int32{faceVertices[ZP][0], faceVertices[YN][0]}, Count: []int32{faceVertices[ZP][1], faceVertices[YN][1]}}
	m.faceMap[Int3{0, -1, -1}] = MultiDrawIndex{Start: []int32{faceVertices[ZN][0], faceVertices[YN][0]}, Count: []int32{faceVertices[ZN][1], faceVertices[YN][1]}}

	m.faceMap[Int3{1, 0, 1}] = MultiDrawIndex{Start: []int32{faceVertices[ZP][0], faceVertices[XP][0]}, Count: []int32{faceVertices[ZP][1], faceVertices[XP][1]}}
	m.faceMap[Int3{1, 0, -1}] = MultiDrawIndex{Start: []int32{faceVertices[ZN][0], faceVertices[XP][0]}, Count: []int32{faceVertices[ZN][1], faceVertices[XP][1]}}
	m.faceMap[Int3{-1, 0, 1}] = MultiDrawIndex{Start: []int32{faceVertices[ZP][0], faceVertices[XN][0]}, Count: []int32{faceVertices[ZP][1], faceVertices[XN][1]}}
	m.faceMap[Int3{-1, 0, -1}] = MultiDrawIndex{Start: []int32{faceVertices[ZN][0], faceVertices[XN][0]}, Count: []int32{faceVertices[ZN][1], faceVertices[XN][1]}}

	m.faceMap[Int3{1, 1, 0}] = MultiDrawIndex{Start: []int32{faceVertices[YP][0], faceVertices[XP][0]}, Count: []int32{faceVertices[YP][1], faceVertices[XP][1]}}
	m.faceMap[Int3{1, -1, 0}] = MultiDrawIndex{Start: []int32{faceVertices[YN][0], faceVertices[XP][0]}, Count: []int32{faceVertices[YN][1], faceVertices[XP][1]}}
	m.faceMap[Int3{-1, 1, 0}] = MultiDrawIndex{Start: []int32{faceVertices[YP][0], faceVertices[XN][0]}, Count: []int32{faceVertices[YP][1], faceVertices[XN][1]}}
	m.faceMap[Int3{-1, -1, 0}] = MultiDrawIndex{Start: []int32{faceVertices[YN][0], faceVertices[XN][0]}, Count: []int32{faceVertices[YN][1], faceVertices[XN][1]}}

	// three axis
	m.faceMap[Int3{1, 1, 1}] = MultiDrawIndex{Start: []int32{faceVertices[ZP][0], faceVertices[YP][0], faceVertices[XP][0]}, Count: []int32{faceVertices[ZP][1], faceVertices[YP][1], faceVertices[XP][1]}}
	m.faceMap[Int3{1, 1, -1}] = MultiDrawIndex{Start: []int32{faceVertices[ZN][0], faceVertices[YP][0], faceVertices[XP][0]}, Count: []int32{faceVertices[ZN][1], faceVertices[YP][1], faceVertices[XP][1]}}
	m.faceMap[Int3{1, -1, 1}] = MultiDrawIndex{Start: []int32{faceVertices[ZP][0], faceVertices[YN][0], faceVertices[XP][0]}, Count: []int32{faceVertices[ZP][1], faceVertices[YN][1], faceVertices[XP][1]}}
	m.faceMap[Int3{1, -1, -1}] = MultiDrawIndex{Start: []int32{faceVertices[ZN][0], faceVertices[YN][0], faceVertices[XP][0]}, Count: []int32{faceVertices[ZN][1], faceVertices[YN][1], faceVertices[XP][1]}}

	m.faceMap[Int3{-1, 1, 1}] = MultiDrawIndex{Start: []int32{faceVertices[ZP][0], faceVertices[YP][0], faceVertices[XN][0]}, Count: []int32{faceVertices[ZP][1], faceVertices[YP][1], faceVertices[XN][1]}}
	m.faceMap[Int3{-1, 1, -1}] = MultiDrawIndex{Start: []int32{faceVertices[ZN][0], faceVertices[YP][0], faceVertices[XN][0]}, Count: []int32{faceVertices[ZN][1], faceVertices[YP][1], faceVertices[XN][1]}}
	m.faceMap[Int3{-1, -1, 1}] = MultiDrawIndex{Start: []int32{faceVertices[ZP][0], faceVertices[YN][0], faceVertices[XN][0]}, Count: []int32{faceVertices[ZP][1], faceVertices[YN][1], faceVertices[XN][1]}}
	m.faceMap[Int3{-1, -1, -1}] = MultiDrawIndex{Start: []int32{faceVertices[ZN][0], faceVertices[YN][0], faceVertices[XN][0]}, Count: []int32{faceVertices[ZN][1], faceVertices[YN][1], faceVertices[XN][1]}}

	return mergedData
}

func NewMeshBuffer(shader *glhf.Shader) ChunkMesh {
	faceMap := make(map[FaceType][]glhf.GlInt)
	for i := 0; i < 6; i++ {
		faceMap[FaceType(i)] = make([]glhf.GlInt, 0, 0)
	}
	return &MeshBuffer{
		shader:           shader,
		indexMap:         make(map[int32]uint32),
		sortedVertexData: faceMap,
		drawMode:         Flat,
		faceMap:          make(map[Int3]MultiDrawIndex),
	}
}
