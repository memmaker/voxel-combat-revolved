package voxel

const (
	EMPTY              int32 = 0
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
