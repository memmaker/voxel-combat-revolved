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
	drawableVertexData *glhf.VertexSlice[glhf.GlUInt]
	flatVertexData     []glhf.GlUInt
	sortedVertexData   map[FaceType][]glhf.GlUInt
	vertexCount        int
	indexMap           map[uint32]uint32
	indexBuffer        []uint32
	drawMode           DrawMode
	faceMap            map[Int3]MultiDrawIndex
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

func (m *MeshBuffer) addVertex(vertex uint32, normal FaceType) {
	if m.drawMode == Indexed {
		m.addIndexedVertex(vertex)
	} else if m.drawMode == Partial {
		m.addVertexSorted(vertex, normal)
	} else {
		m.addFlatVertex(vertex)
	}
}

func (m *MeshBuffer) addFlatVertex(vertex uint32) {
	m.flatVertexData = append(m.flatVertexData, glhf.GlUInt(vertex))
}

func (m *MeshBuffer) addVertexSorted(vertex uint32, normal FaceType) {
	m.sortedVertexData[normal] = append(m.sortedVertexData[normal], glhf.GlUInt(vertex))
}

func (m *MeshBuffer) addIndexedVertex(vertex uint32) {
	if vertexIndex, isCached := m.indexMap[vertex]; isCached {
		m.indexBuffer = append(m.indexBuffer, vertexIndex)
		return
	}
	vertexIndex := len(m.flatVertexData)
	m.flatVertexData = append(m.flatVertexData, glhf.GlUInt(vertex))
	if vertexIndex > math.MaxUint32 {
		println("vertexIndex out of bounds: ", vertexIndex)
	}
	vertexIndexUnsigned := uint32(vertexIndex)
	m.indexMap[vertex] = vertexIndexUnsigned
	m.indexBuffer = append(m.indexBuffer, vertexIndexUnsigned)
}

func (m *MeshBuffer) Reset() {
	m.indexMap = make(map[uint32]uint32)
	m.indexBuffer = m.indexBuffer[:0]
	m.flatVertexData = m.flatVertexData[:0]
	m.vertexCount = 0
}

func (m *MeshBuffer) FlushMesh(shader *glhf.Shader) {
	if shader == nil {
		return
	}
	if m.drawMode == Indexed {
		m.drawableVertexData = glhf.MakeUIntVertexSlice(shader, m.vertexCount, m.vertexCount, m.indexBuffer)
	} else {
		m.drawableVertexData = glhf.MakeUIntVertexSlice(shader, m.vertexCount, m.vertexCount, nil)
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
func (m *MeshBuffer) Compress(position Int3, normalDirection FaceType, textureIndex byte, extraBits uint8) uint32 {
	// 6 bits for the x y z axis
	// max value for each axis is 2^6 - 1 = 63
	// we want to pack these into one 32 bit integer
	maxX := uint32(63)
	maxY := uint32(63)
	maxZ := uint32(63)

	xAxis := uint32(position.X)
	yAxis := uint32(position.Y) << 6
	zAxis := uint32(position.Z) << 12

	if position.X < 0 || uint32(position.X) > maxX {
		println("x axis out of bounds: ", position.X)
	}
	if position.Y < 0 || uint32(position.Y) > maxY {
		println("y axis out of bounds: ", position.Y)
	}
	if position.Z < 0 || uint32(position.Z) > maxZ {
		println("z axis out of bounds: ", position.Z)
	}
	compressedPosition := xAxis | yAxis | zAxis

	// 3 bits for the normal direction (0..5)
	attributes := uint32(normalDirection) << 18
	// 8 bits for the texture index (0..255)
	attributes |= uint32(textureIndex) << 21

	// add the first three extra bits
	attributes |= uint32(extraBits) << 29

	compressedVertex := compressedPosition | attributes
	// total: 29 bits, 3 bits left
	return compressedVertex
}

func (m *MeshBuffer) preparePartialVertexData(data map[FaceType][]glhf.GlUInt) []glhf.GlUInt {
	mergedData := make([]glhf.GlUInt, 0, 0)
	faceVertices := make(map[FaceType][2]int32, 0)
	for normalDir, vertexData := range data {
		startIndex, count := len(mergedData), len(vertexData)
		mergedData = append(mergedData, vertexData...)
		faceVertices[normalDir] = [2]int32{int32(startIndex), int32(count)}
	}
	m.faceMap[Int3{0, 0, 1}] = MultiDrawIndex{Start: []int32{faceVertices[South][0]}, Count: []int32{faceVertices[South][1]}}
	m.faceMap[Int3{0, 0, -1}] = MultiDrawIndex{Start: []int32{faceVertices[North][0]}, Count: []int32{faceVertices[North][1]}}
	m.faceMap[Int3{0, 1, 0}] = MultiDrawIndex{Start: []int32{faceVertices[Top][0]}, Count: []int32{faceVertices[Top][1]}}
	m.faceMap[Int3{0, -1, 0}] = MultiDrawIndex{Start: []int32{faceVertices[Bottom][0]}, Count: []int32{faceVertices[Bottom][1]}}
	m.faceMap[Int3{1, 0, 0}] = MultiDrawIndex{Start: []int32{faceVertices[East][0]}, Count: []int32{faceVertices[East][1]}}
	m.faceMap[Int3{-1, 0, 0}] = MultiDrawIndex{Start: []int32{faceVertices[West][0]}, Count: []int32{faceVertices[West][1]}}

	// two axis
	m.faceMap[Int3{0, 1, 1}] = MultiDrawIndex{Start: []int32{faceVertices[South][0], faceVertices[Top][0]}, Count: []int32{faceVertices[South][1], faceVertices[Top][1]}}
	m.faceMap[Int3{0, 1, -1}] = MultiDrawIndex{Start: []int32{faceVertices[North][0], faceVertices[Top][0]}, Count: []int32{faceVertices[North][1], faceVertices[Top][1]}}
	m.faceMap[Int3{0, -1, 1}] = MultiDrawIndex{Start: []int32{faceVertices[South][0], faceVertices[Bottom][0]}, Count: []int32{faceVertices[South][1], faceVertices[Bottom][1]}}
	m.faceMap[Int3{0, -1, -1}] = MultiDrawIndex{Start: []int32{faceVertices[North][0], faceVertices[Bottom][0]}, Count: []int32{faceVertices[North][1], faceVertices[Bottom][1]}}

	m.faceMap[Int3{1, 0, 1}] = MultiDrawIndex{Start: []int32{faceVertices[South][0], faceVertices[East][0]}, Count: []int32{faceVertices[South][1], faceVertices[East][1]}}
	m.faceMap[Int3{1, 0, -1}] = MultiDrawIndex{Start: []int32{faceVertices[North][0], faceVertices[East][0]}, Count: []int32{faceVertices[North][1], faceVertices[East][1]}}
	m.faceMap[Int3{-1, 0, 1}] = MultiDrawIndex{Start: []int32{faceVertices[South][0], faceVertices[West][0]}, Count: []int32{faceVertices[South][1], faceVertices[West][1]}}
	m.faceMap[Int3{-1, 0, -1}] = MultiDrawIndex{Start: []int32{faceVertices[North][0], faceVertices[West][0]}, Count: []int32{faceVertices[North][1], faceVertices[West][1]}}

	m.faceMap[Int3{1, 1, 0}] = MultiDrawIndex{Start: []int32{faceVertices[Top][0], faceVertices[East][0]}, Count: []int32{faceVertices[Top][1], faceVertices[East][1]}}
	m.faceMap[Int3{1, -1, 0}] = MultiDrawIndex{Start: []int32{faceVertices[Bottom][0], faceVertices[East][0]}, Count: []int32{faceVertices[Bottom][1], faceVertices[East][1]}}
	m.faceMap[Int3{-1, 1, 0}] = MultiDrawIndex{Start: []int32{faceVertices[Top][0], faceVertices[West][0]}, Count: []int32{faceVertices[Top][1], faceVertices[West][1]}}
	m.faceMap[Int3{-1, -1, 0}] = MultiDrawIndex{Start: []int32{faceVertices[Bottom][0], faceVertices[West][0]}, Count: []int32{faceVertices[Bottom][1], faceVertices[West][1]}}

	// three axis
	m.faceMap[Int3{1, 1, 1}] = MultiDrawIndex{Start: []int32{faceVertices[South][0], faceVertices[Top][0], faceVertices[East][0]}, Count: []int32{faceVertices[South][1], faceVertices[Top][1], faceVertices[East][1]}}
	m.faceMap[Int3{1, 1, -1}] = MultiDrawIndex{Start: []int32{faceVertices[North][0], faceVertices[Top][0], faceVertices[East][0]}, Count: []int32{faceVertices[North][1], faceVertices[Top][1], faceVertices[East][1]}}
	m.faceMap[Int3{1, -1, 1}] = MultiDrawIndex{Start: []int32{faceVertices[South][0], faceVertices[Bottom][0], faceVertices[East][0]}, Count: []int32{faceVertices[South][1], faceVertices[Bottom][1], faceVertices[East][1]}}
	m.faceMap[Int3{1, -1, -1}] = MultiDrawIndex{Start: []int32{faceVertices[North][0], faceVertices[Bottom][0], faceVertices[East][0]}, Count: []int32{faceVertices[North][1], faceVertices[Bottom][1], faceVertices[East][1]}}

	m.faceMap[Int3{-1, 1, 1}] = MultiDrawIndex{Start: []int32{faceVertices[South][0], faceVertices[Top][0], faceVertices[West][0]}, Count: []int32{faceVertices[South][1], faceVertices[Top][1], faceVertices[West][1]}}
	m.faceMap[Int3{-1, 1, -1}] = MultiDrawIndex{Start: []int32{faceVertices[North][0], faceVertices[Top][0], faceVertices[West][0]}, Count: []int32{faceVertices[North][1], faceVertices[Top][1], faceVertices[West][1]}}
	m.faceMap[Int3{-1, -1, 1}] = MultiDrawIndex{Start: []int32{faceVertices[South][0], faceVertices[Bottom][0], faceVertices[West][0]}, Count: []int32{faceVertices[South][1], faceVertices[Bottom][1], faceVertices[West][1]}}
	m.faceMap[Int3{-1, -1, -1}] = MultiDrawIndex{Start: []int32{faceVertices[North][0], faceVertices[Bottom][0], faceVertices[West][0]}, Count: []int32{faceVertices[North][1], faceVertices[Bottom][1], faceVertices[West][1]}}

	return mergedData
}

func NewMeshBuffer() *MeshBuffer {
	faceMap := make(map[FaceType][]glhf.GlUInt)
	for i := 0; i < 6; i++ {
		faceMap[FaceType(i)] = make([]glhf.GlUInt, 0, 0)
	}
	return &MeshBuffer{
		indexMap: make(map[uint32]uint32),
		sortedVertexData: faceMap,
		drawMode:         Flat,
		faceMap:          make(map[Int3]MultiDrawIndex),
	}
}
