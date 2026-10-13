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
	indexMap           map[vertex]uint32
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

// vertex is one packed vertex: position, normal, texture and light, plus the ambient extras
type vertex struct {
	compressed uint32
	extra      uint32 // bits 0..1: ambient occlusion (0..3), bits 2..5: sky openness (0..15)
}

func (m *MeshBuffer) AppendQuad(tr, br, bl, tl Int3, normal FaceType, textureIndex byte, corners [4]VertexLight) {
	//println(fmt.Sprintf("bl: %v, tl: %v, br: %v, tr: %v, normal: %v, textureIndex: %v", bl, tl, br, tr, normal, textureIndex))
	pack := func(position Int3, corner VertexLight) vertex {
		return vertex{
			compressed: m.Compress(position, normal, textureIndex, corner.Light),
			extra:      uint32(corner.AO) | uint32(corner.Sky)<<2,
		}
	}
	vertexBL, vertexTL, vertexBR, vertexTR := pack(bl, corners[0]), pack(tl, corners[1]), pack(br, corners[2]), pack(tr, corners[3])

	// a quad is split along one diagonal; with uneven corners the split shows as a crease,
	// so it runs along the diagonal whose corners are darker (as in 0fps' voxel AO)
	brightness := func(corner VertexLight) int { return int(corner.AO)*maxLightLevel + int(corner.Light)*3 }
	flip := brightness(corners[0])+brightness(corners[3]) > brightness(corners[1])+brightness(corners[2])

	var triangles [6]vertex
	if flip {
		triangles = [6]vertex{vertexTL, vertexBR, vertexBL, vertexTL, vertexTR, vertexBR}
	} else {
		triangles = [6]vertex{vertexTR, vertexBL, vertexTL, vertexTR, vertexBR, vertexBL}
	}
	reverseOrder := normal%2 == 1
	if reverseOrder {
		for i := 0; i < 3; i++ {
			triangles[i], triangles[5-i] = triangles[5-i], triangles[i]
		}
	}
	for _, v := range triangles {
		m.addVertex(v, normal)
	}

	m.vertexCount += 6
}

func (m *MeshBuffer) addVertex(v vertex, normal FaceType) {
	if m.drawMode == Indexed {
		m.addIndexedVertex(v)
	} else if m.drawMode == Partial {
		m.addVertexSorted(v, normal)
	} else {
		m.addFlatVertex(v)
	}
}

func (m *MeshBuffer) addFlatVertex(v vertex) {
	m.flatVertexData = append(m.flatVertexData, glhf.GlUInt(v.compressed), glhf.GlUInt(v.extra))
}

func (m *MeshBuffer) addVertexSorted(v vertex, normal FaceType) {
	m.sortedVertexData[normal] = append(m.sortedVertexData[normal], glhf.GlUInt(v.compressed), glhf.GlUInt(v.extra))
}

func (m *MeshBuffer) addIndexedVertex(v vertex) {
	if vertexIndex, isCached := m.indexMap[v]; isCached {
		m.indexBuffer = append(m.indexBuffer, vertexIndex)
		return
	}
	vertexIndex := len(m.flatVertexData) / 2
	m.flatVertexData = append(m.flatVertexData, glhf.GlUInt(v.compressed), glhf.GlUInt(v.extra))
	if vertexIndex > math.MaxUint32 {
		println("vertexIndex out of bounds: ", vertexIndex)
	}
	vertexIndexUnsigned := uint32(vertexIndex)
	m.indexMap[v] = vertexIndexUnsigned
	m.indexBuffer = append(m.indexBuffer, vertexIndexUnsigned)
}

func (m *MeshBuffer) Reset() {
	m.indexMap = make(map[vertex]uint32)
	m.indexBuffer = m.indexBuffer[:0]
	m.flatVertexData = m.flatVertexData[:0]
	m.vertexCount = 0
}

func (m *MeshBuffer) UploadMeshToGPU(shader *glhf.Shader) {
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
func (m *MeshBuffer) Compress(position Int3, normalDirection FaceType, textureIndex byte, lightLevel byte) uint32 {
	// 6 bits for the x y z axis
	// max value for each axis is 2^6 - 1 = 63
	// we want to pack these into one 32 bit integer
	maxX := uint32(63)
	maxY := uint32(31)
	maxZ := uint32(63)

	xAxis := uint32(position.X)       // 6 bits
	yAxis := uint32(position.Y) << 6  // 5 bits
	zAxis := uint32(position.Z) << 11 // 6 bits

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
	attributes := uint32(normalDirection) << 17
	// 8 bits for the texture index (0..255)
	attributes |= uint32(textureIndex) << 20

	// add the first four extra bits
	attributes |= uint32(lightLevel) << 28

	compressedVertex := compressedPosition | attributes
	// total: 32 bits
	return compressedVertex
}

func (m *MeshBuffer) preparePartialVertexData(data map[FaceType][]glhf.GlUInt) []glhf.GlUInt {
	mergedData := make([]glhf.GlUInt, 0, 0)
	faceVertices := make(map[FaceType][2]int32, 0)
	for normalDir, vertexData := range data {
		startIndex, count := len(mergedData)/2, len(vertexData)/2 // two values per vertex
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
		indexMap:         make(map[vertex]uint32),
		sortedVertexData: faceMap,
		drawMode:         Flat,
		faceMap:          make(map[Int3]MultiDrawIndex),
	}
}
