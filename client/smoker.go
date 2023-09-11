package client

import (
    "github.com/go-gl/mathgl/mgl32"
    "github.com/memmaker/battleground/engine/glhf"
    "github.com/memmaker/battleground/engine/util"
    "github.com/memmaker/battleground/engine/voxel"
)

type Smoker struct {
    particles         *glhf.ParticleSystem
    voxelMap          *voxel.Map
    particlesPerBlock int
    getMap            func() *voxel.Map
    locationToBuffer  map[voxel.Int3]int
    runningAnimations []SmokeSpawnAnimation
}

func NewSmoker(getMap func() *voxel.Map, system *glhf.ParticleSystem) *Smoker {
    return &Smoker{
        particles:         system,
        particlesPerBlock: 2,
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

func (s *Smoker) Update(elapsed float64) {
    for i := len(s.runningAnimations) - 1; i >= 0; i-- {
        currentRadius, radiusChanged := s.runningAnimations[i].Update(elapsed)
        if radiusChanged {
            s.createSmokeVolume(s.runningAnimations[i].origin, currentRadius)
        }
        if !s.runningAnimations[i].Running() {
            s.runningAnimations = append(s.runningAnimations[:i], s.runningAnimations[i+1:]...)
        }
    }
}

func (s *Smoker) ClearSmokeAt(location voxel.Int3) {
    vertexBufferOffset, exists := s.locationToBuffer[location]
    if !exists {
        return
    }
    delete(s.locationToBuffer, location)
    s.particles.Clear(vertexBufferOffset, s.particlesPerBlock)
}

func (s *Smoker) AddSmokeAt(location voxel.Int3) {
    if _, exists := s.locationToBuffer[location]; exists {
        return
    }
    particleProperties := s.getSmokeProps(location.ToBlockCenterVec3D())
    bufferVertexOffset := s.particles.Emit(particleProperties, s.particlesPerBlock)
    s.locationToBuffer[location] = bufferVertexOffset
}

func (s *Smoker) AddAnimatedSmokeAt(location voxel.Int3, radius float64) {
    s.runningAnimations = append(s.runningAnimations, SmokeSpawnAnimation{
        origin:     location,
        lifeTime:   5,
        radius:     radius,
        lastRadius: 0,
    })
}
func (s *Smoker) createSmokeVolume(location voxel.Int3, radius float64) {
    s.getMap().ForBlockInSphericFloodFill(location, radius, func(origin voxel.Int3, steps int, radius float64, x int32, y int32, z int32) {
        if y < 1 {
            return
        }
        pos := voxel.Int3{X: int32(x), Y: int32(y), Z: int32(z)}
        s.AddSmokeAt(pos)
    })
}

type SmokeSpawnAnimation struct {
    origin     voxel.Int3
    lived      float64
    lifeTime   float64
    radius     float64
    lastRadius float64
}

func (a *SmokeSpawnAnimation) Update(elapsed float64) (float64, bool) {
    a.lived += elapsed
    //percent := 1.0 - (a.lifeTimeLeft / a.lifeTime)
    easeValue := util.EaseSlowEnd(a.lived)

    radius := a.radius * easeValue
    if radius != a.lastRadius {
        a.lastRadius = radius
        return radius, true
    }
    return radius, false
}

func (a *SmokeSpawnAnimation) Running() bool {
    return a.lived < a.lifeTime
}