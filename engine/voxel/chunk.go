package voxel

import (
    "fmt"
    "github.com/go-gl/mathgl/mgl32"
    "github.com/memmaker/battleground/engine/glhf"
    "math"
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

    meshChannel chan ChunkMesh
}

func NewChunk(voxelMap *Map, x, y, z int32) *Chunk {
    c := &Chunk{
        data: make([]*Block, voxelMap.ChunkSizeCube),
        m:           voxelMap,
        chunkPosX:   x,
        chunkPosY:   y,
        chunkPosZ:   z,
        chunkHelper: &ChunkHelper{
            visitXN: make([]bool, voxelMap.ChunkSizeCube),
            visitXP: make([]bool, voxelMap.ChunkSizeCube),
            visitYN: make([]bool, voxelMap.ChunkSizeCube),
            visitYP: make([]bool, voxelMap.ChunkSizeCube),
            visitZN: make([]bool, voxelMap.ChunkSizeCube),
            visitZP: make([]bool, voxelMap.ChunkSizeCube),
        },
        isDirty:    true,
        meshBuffer: NewMeshBuffer(),
        meshChannel: make(chan ChunkMesh, 20),
    }
    for i := int32(0); i < voxelMap.ChunkSizeCube; i++ {
        c.data[i] = NewAirBlock()
    }
    return c
}

func (c *Chunk) blockIndex(i, j, k int32) int32 {
    return i + j*c.m.ChunkSizeHorizontal + k*(c.m.ChunkSizeHorizontal*c.m.ChunkSizeHeight)
}
func (c *Chunk) Contains(x, y, z int32) bool {
    return x >= 0 && x < c.m.ChunkSizeHorizontal && y >= 0 && y < c.m.ChunkSizeHeight && z >= 0 && z < c.m.ChunkSizeHorizontal
}
func (c *Chunk) GetLocalBlock(i, j, k int32) *Block {
    if !c.Contains(i, j, k) {
        return NewAirBlock()
    }
    block := c.data[c.blockIndex(i, j, k)]
    if block == nil {
        return NewAirBlock()
    }
    return block
}

func (c *Chunk) SetBlock(x, y, z int32, block *Block) {
    c.data[c.blockIndex(x, y, z)] = block
    c.isDirty = true
}

type VoxelFace struct {
    inVisible bool
    side      FaceType
    textureIndex byte
}

func (v *VoxelFace) EqualForMerge(face *VoxelFace) bool {
    if face.inVisible {
        return face.inVisible == v.inVisible
    }
    return face.inVisible == v.inVisible && face.textureIndex == v.textureIndex
}

func (c *Chunk) InitNeighbors() {
    if c.chunkPosX > 0 && c.cXN == nil {
        c.cXN = c.m.GetChunk(c.chunkPosX-1, c.chunkPosY, c.chunkPosZ)
    }

    // Positive CollisionX side
    if c.chunkPosX < c.m.width-1 && c.cXP == nil {
        c.cXP = c.m.GetChunk(c.chunkPosX+1, c.chunkPosY, c.chunkPosZ)
    }

    // Negative CollisionY side
    if c.chunkPosY > 0 && c.cYN == nil {
        c.cYN = c.m.GetChunk(c.chunkPosX, c.chunkPosY-1, c.chunkPosZ)
    }

    // Positive CollisionY side
    if c.chunkPosY < c.m.height-1 && c.cYP == nil {
        c.cYP = c.m.GetChunk(c.chunkPosX, c.chunkPosY+1, c.chunkPosZ)
    }

    // Negative Z neighbour
    if c.chunkPosZ > 0 && c.cZN == nil {
        c.cZN = c.m.GetChunk(c.chunkPosX, c.chunkPosY, c.chunkPosZ-1)
    }

    // Positive Z side
    if c.chunkPosZ < c.m.depth-1 && c.cZP == nil {
        c.cZP = c.m.GetChunk(c.chunkPosX, c.chunkPosY, c.chunkPosZ+1)
    }
}

func (c *Chunk) GreedyMeshing(outChannel chan ChunkMesh) {
    // adapted from: https://github.com/roboleary/GreedyMesh/blob/master/src/mygame/Main.java
    c.InitNeighbors()
    //println(fmt.Sprintf("Greedy meshing chunk %d,%d,%d", c.chunkPosX, c.chunkPosY, c.chunkPosZ))
    c.chunkHelper.Reset(c.m.ChunkSizeCube)
    mesh := NewMeshBuffer()

    var (
        i, j, k, l, w, h, u, v, n int32
        side                      FaceType

        x  = [3]int32{}
        q  = [3]int32{}
        du = [3]int32{}
        dv = [3]int32{}

        mask       = make([]*VoxelFace, c.m.ChunkSizeHorizontal*c.m.ChunkSizeHorizontal)
        voxelFace  *VoxelFace
        voxelFace1 *VoxelFace

        axisSize = [3]int32{c.m.ChunkSizeHorizontal, c.m.ChunkSizeHeight, c.m.ChunkSizeHorizontal}
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
                side = map[bool]FaceType{true: West, false: East}[backFace]
            case d == 1:
                side = map[bool]FaceType{true: Bottom, false: Top}[backFace]
            case d == 2:
                side = map[bool]FaceType{true: North, false: South}[backFace]
            }

            for x[d] = -1; x[d] < axisSize[d]; {
                n = 0
                // this part will fill the mask with 2d slices for the current axis
                // size of mask should now be axisSize[u] * axisSize[v]
                for x[v] = 0; x[v] < axisSize[v]; x[v]++ {
                    for x[u] = 0; x[u] < axisSize[u]; x[u]++ {

                        if x[d] >= 0 { // not at the edge of the chunk
                            voxelFace = c.getVoxelFace(x[0], x[1], x[2], side)
                        } else {
                            voxelFace = nil
                        }

                        if x[d] < axisSize[d]-1 { // not at the edge of the chunk
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

                // step in the main direction
                x[d]++
                // assemble the mesh for the current plane, iterating through axisSize[u] * axisSize[v]
                n = 0
                for j = 0; j < axisSize[v]; j++ { // Dim 0 - slot 0
                    for i = 0; i < axisSize[u]; { // Dim 1 - slot 0
                        if mask[n] != nil {
                            w = 1
                            // Dim 1 - slot 1
                            for i+w < axisSize[u] && mask[n+w] != nil && mask[n+w].EqualForMerge(mask[n]) {
                                w++
                            }

                            done := false
                            h = 1
                            // Dim 0 - slot 1
                            for h+j < axisSize[v] {
                                for k = 0; k < w; k++ {
                                    // Dim 1 - slot 2 + 3
                                    if mask[n+k+h*axisSize[u]] == nil || !mask[n+k+h*axisSize[u]].EqualForMerge(mask[n]) {
                                        done = true
                                        break
                                    }
                                }
                                if done {
                                    break
                                }
                                h++
                            }

                            if !mask[n].inVisible {
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
                                mesh.AppendQuad(topRight, bottomRight, bottomLeft, topLeft, mask[n].side, mask[n].textureIndex, [4]uint8{})
                                // we also would have width (w) and height (h) here
                                // backface is a bool, true if we are rendering the backface of a block
                            }

                            for l = 0; l < h; l++ {
                                for k = 0; k < w; k++ {
                                    // Dim 1 - slot 4
                                    // NOT axisSize[d]
                                    // NOT axisSize[v]
                                    mask[n+k+l*axisSize[u]] = nil
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

    println(fmt.Sprintf("[Greedy] Chunk %d,%d,%d was meshed into %d triangles", c.chunkPosX, c.chunkPosY, c.chunkPosZ, c.meshBuffer.TriangleCount()))
    //return c.meshBuffer
    outChannel <- mesh
}

func (c *Chunk) getVoxelFace(x int32, y int32, z int32, side FaceType) *VoxelFace {
    block := c.GetLocalBlock(x, y, z)
    if block == nil || block.IsAir() {
        return &VoxelFace{inVisible: true}
    }
    var neighbor *Block
    neighborIsFromChunkAboveOrBelow := false
    switch side {
    case West:
        if x == 0 {
            if c.cXN != nil {
                neighbor = c.cXN.GetLocalBlock(c.m.ChunkSizeHorizontal-1, y, z)
            }
        } else {
            neighbor = c.GetLocalBlock(x-1, y, z)
        }
    case East:
        if x == c.m.ChunkSizeHorizontal-1 {
            if c.cXP != nil {
                neighbor = c.cXP.GetLocalBlock(0, y, z)
            }
        } else {
            neighbor = c.GetLocalBlock(x+1, y, z)
        }
    case Bottom:
        if y == 0 {
            if c.cYN != nil {
                neighbor = c.cYN.GetLocalBlock(x, c.m.ChunkSizeHeight-1, z)
                neighborIsFromChunkAboveOrBelow = true
            }
        } else {
            neighbor = c.GetLocalBlock(x, y-1, z)
        }
    case Top:
        if y == c.m.ChunkSizeHeight-1 {
            if c.cYP != nil {
                neighbor = c.cYP.GetLocalBlock(x, 0, z)
                neighborIsFromChunkAboveOrBelow = true
            }
        } else {
            neighbor = c.GetLocalBlock(x, y+1, z)
        }
    case North:
        if z == 0 {
            if c.cZN != nil {
                neighbor = c.cZN.GetLocalBlock(x, y, c.m.ChunkSizeHorizontal-1)
            }
        } else {
            neighbor = c.GetLocalBlock(x, y, z-1)
        }
    case South:
        if z == c.m.ChunkSizeHorizontal-1 {
            if c.cZP != nil {
                neighbor = c.cZP.GetLocalBlock(x, y, 0)
            }
        } else {
            neighbor = c.GetLocalBlock(x, y, z+1)
        }
    }

    face := &VoxelFace{side: side, textureIndex: c.m.getTextureIndexForSide(block, side)}
    if neighbor != nil && !neighbor.IsAir() && !neighborIsFromChunkAboveOrBelow {
        face.inVisible = true
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

func (i Int3) Div(factor int) Int3 {
    return Int3{i.X / int32(factor), i.Y / int32(factor), i.Z / int32(factor)}
}

func (i Int3) ToBlockCenterVec3() mgl32.Vec3 {
    return mgl32.Vec3{float32(i.X) + 0.5, float32(i.Y), float32(i.Z) + 0.5}
}

func (i Int3) ToBlockCenterVec3D() mgl32.Vec3 {
    return mgl32.Vec3{float32(i.X) + 0.5, float32(i.Y), float32(i.Z) + 0.5}
}

func (i Int3) ToString() string {
    return fmt.Sprintf("(%d,%d,%d)", i.X, i.Y, i.Z)
}
func (i Int3) ToCardinalDirection() Int3 {
    absX := int32(math.Abs(float64(i.X)))
    absZ := int32(math.Abs(float64(i.Z)))

    signX := int32(1)
    signZ := int32(1)

    if i.X < 0 {
        signX = -1
    }
    if i.Z < 0 {
        signZ = -1
    }

    if absX > absZ {
        return Int3{signX, 0, 0}
    } else {
        return Int3{0, 0, signZ}
    }
}
func (i Int3) ToDiagonalDirection() Int3 {
    // allowed values are
    // north: 0,0,-1
    // east: 1,0,0
    // south: 0,0,1
    // west: -1,0,0
    // north-east: 1,0,-1
    // south-east: 1,0,1
    // south-west: -1,0,1
    // north-west: -1,0,-1
    absX := int32(math.Abs(float64(i.X)))
    absZ := int32(math.Abs(float64(i.Z)))
    signX := int32(1)
    signZ := int32(1)
    if i.X < 0 {
        signX = -1
    }
    if i.Z < 0 {
        signZ = -1
    }

    ratio := float64(absX) / float64(absZ)

    if ratio > 0.5 && ratio < 1.5 { // diagonal
        return Int3{signX, 0, signZ}
    } else if ratio >= 1.5 { // vertical
        return Int3{signX, 0, 0}
    } else { // horizontal
        return Int3{0, 0, signZ}
    }
}

func (i Int3) ManhattanLength() int {
    return int(math.Abs(float64(i.X)) + math.Abs(float64(i.Y)) + math.Abs(float64(i.Z)))
}

func (i Int3) ManhattanLength2() int {
    return int(math.Abs(float64(i.X)) + math.Abs(float64(i.Z)))
}

func (i Int3) IsBelow(grid Int3) bool {
    return i.Y == grid.Y-1
}

func (i Int3) LengthInt() int32 {
    return int32(math.Sqrt(float64(i.X*i.X + i.Y*i.Y + i.Z*i.Z)))
}
func (i Int3) Length() float64 {
    return math.Sqrt(float64(i.X*i.X + i.Y*i.Y + i.Z*i.Z))
}

func (c *Chunk) Draw(shader *glhf.Shader, modelUniformIndex int) {
    if c.meshBuffer == nil || c.meshBuffer.TriangleCount() == 0 {
        return
    }
    shader.SetUniformAttr(modelUniformIndex, c.GetMatrix())
    c.meshBuffer.Draw()
}
func (c *Chunk) getDiscreteCamDir(camDir mgl32.Vec3) Int3 {
    intPos := Int3{0, 0, 0}
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
    return mgl32.Translate3D(float32(c.chunkPosX*c.m.ChunkSizeHorizontal), float32(c.chunkPosY*c.m.ChunkSizeHeight), float32(c.chunkPosZ*c.m.ChunkSizeHorizontal))
}

func (c *Chunk) Position() Int3 {
    return Int3{c.chunkPosX, c.chunkPosY, c.chunkPosZ}
}

func (c *Chunk) IsBlockAt(i int32, j int32, k int32) bool {
    block := c.data[c.blockIndex(i, j, k)]
    return block != nil && !block.IsAir()
}

func (c *Chunk) SetDirty() {
    c.isDirty = true
}

func (c *Chunk) ClearAllBlocks() {
    for i := int32(0); i < c.m.ChunkSizeHorizontal; i++ {
        for j := int32(0); j < c.m.ChunkSizeHeight; j++ {
            for k := int32(0); k < c.m.ChunkSizeHorizontal; k++ {
                c.SetBlock(i, j, k, NewAirBlock())
            }
        }
    }
}

func (c *Chunk) GenerateMesh() {
    if !c.isDirty {
        return
    }
    c.isDirty = false
    go c.GreedyMeshing(c.meshChannel) // will set isDirty to false
}

func (c *Chunk) CheckForNewMeshes() bool {
    select {
    case meshBuffer := <-c.meshChannel:
        if meshBuffer.TriangleCount() > 0 {
            meshBuffer.UploadMeshToGPU(c.m.chunkShader)
            c.meshBuffer = meshBuffer
            return true
        }
    default:
    }
    return false
}
