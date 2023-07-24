package voxel

type ChunkHelper struct {
    visitXN []bool
    visitXP []bool
    visitYN []bool
    visitYP []bool
    visitZN []bool
    visitZP []bool
}

func (h *ChunkHelper) Reset() {
    h.visitXN = make([]bool, CHUNK_SIZE_CUBED)
    h.visitXP = make([]bool, CHUNK_SIZE_CUBED)
    h.visitYN = make([]bool, CHUNK_SIZE_CUBED)
    h.visitYP = make([]bool, CHUNK_SIZE_CUBED)
    h.visitZN = make([]bool, CHUNK_SIZE_CUBED)
    h.visitZP = make([]bool, CHUNK_SIZE_CUBED)
}
