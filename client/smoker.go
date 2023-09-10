package client

import (
    "github.com/go-gl/mathgl/mgl32"
    "github.com/memmaker/battleground/engine/glhf"
    "github.com/memmaker/battleground/engine/voxel"
)

type Smoker struct {
    particles         *glhf.ParticleSystem
    voxelMap          *voxel.Map
    radius            int
    particlesPerBlock int
    getMap            func() *voxel.Map
    locationToBuffer  map[voxel.Int3]int
}

func NewSmoker(getMap func() *voxel.Map, system *glhf.ParticleSystem) *Smoker {
    return &Smoker{
        particles:         system,
        radius:            5,
        particlesPerBlock: 5,
        getMap:            getMap,
        locationToBuffer:  make(map[voxel.Int3]int),
    }
}
func (s *Smoker) getSmokeProps(origin mgl32.Vec3) glhf.ParticleProperties {
    return glhf.ParticleProperties{
        Origin:               origin,
        PositionVariation:    mgl32.Vec3{0.5, 0.5, 0.5},
        VelocityFromPosition: func(origin, pos mgl32.Vec3) mgl32.Vec3 { return mgl32.Vec3{0.0, 0.0, 0.0} },
        VelocityVariation:    mgl32.Vec3{0.16, 0.2, 0.16},
        MaxDistance:          0.5,
        SizeBegin:            0.2,
        SizeVariation:        0.1,
        SizeEnd:              0.1,
        Lifetime:             -1, // infinite
        ColorBegin:           mgl32.Vec3{0.3, 0.3, 0.3},
        ColorVariation:       0.1,
        ColorEnd:             mgl32.Vec3{0.2, 0.2, 0.2},
    }
}

func (s *Smoker) Draw(elapsed float64) {
    s.particles.Draw(elapsed)
}

func (s *Smoker) ClearParticlesAt(location voxel.Int3) {
    vertexBufferOffset, exists := s.locationToBuffer[location]
    if !exists {
        return
    }
    s.particles.Clear(vertexBufferOffset, s.particlesPerBlock)
}
func (s *Smoker) ClearAllAbove() {
    for location, _ := range s.locationToBuffer {
        if location.Y > 2 {
            s.ClearParticlesAt(location)
        }
    }
}
func (s *Smoker) CreateSmokeAt(location voxel.Int3) {
    s.getMap().ForBlockInHalfSphere(location, s.radius, func(origin voxel.Int3, radius int, x int32, y int32, z int32) {
        if y < 1 {
            return
        }
        pos := voxel.Int3{X: int32(x), Y: int32(y), Z: int32(z)}
        particleProperties := s.getSmokeProps(pos.ToBlockCenterVec3D())
        bufferVertexOffset := s.particles.Emit(particleProperties, s.particlesPerBlock)
        s.locationToBuffer[pos] = bufferVertexOffset
    })
}
