package voxel

const (
	EMPTY              byte  = 0
	CHUNK_SIZE         int32 = 32
	CHUNK_SIZE_SQUARED int32 = CHUNK_SIZE * CHUNK_SIZE
	CHUNK_SIZE_CUBED   int32 = CHUNK_SIZE * CHUNK_SIZE * CHUNK_SIZE
)

type Constants struct {
	ChunkSizeSquared int
	ChunkXAmount     int
	ChunkYAmount     int
	ChunkZAmount     int
}

func ManhattanDistance3(a, b Int3) int32 {
	return Abs(a.X-b.X) + Abs(a.Y-b.Y) + Abs(a.Z-b.Z)
}

func ManhattanDistance2(a, b Int3) int32 {
	return Abs(a.X-b.X) + Abs(a.Z-b.Z)
}
func EuclideanDistance2(a, b Int3) float32 {
	return float32((a.X-b.X)*(a.X-b.X) + (a.Z-b.Z)*(a.Z-b.Z))
	// eg. for (0,0) and (1,1) this would be 2
}
func Abs(i int32) int32 {
	if i < 0 {
		return -i
	}
	return i
}
