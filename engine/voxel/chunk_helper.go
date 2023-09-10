package voxel

type ChunkHelper struct {
    visitXN []bool
    visitXP []bool
    visitYN []bool
    visitYP []bool
    visitZN []bool
    visitZP []bool
}

func (h *ChunkHelper) Reset(chunkSizeCube int32) {
    h.visitXN = make([]bool, chunkSizeCube)
    h.visitXP = make([]bool, chunkSizeCube)
    h.visitYN = make([]bool, chunkSizeCube)
    h.visitYP = make([]bool, chunkSizeCube)
    h.visitZN = make([]bool, chunkSizeCube)
    h.visitZP = make([]bool, chunkSizeCube)
}
