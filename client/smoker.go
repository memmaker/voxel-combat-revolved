package client

import (
    "github.com/go-gl/mathgl/mgl32"
    "github.com/memmaker/battleground/engine/glhf"
    "github.com/memmaker/battleground/engine/util"
    "github.com/memmaker/battleground/engine/voxel"
)

type SmokeEffect int

const (
    SmokeEffectBlockLoS SmokeEffect = 1 << iota
    SmokeEffectDamage
)

type SmokeEffectInfo struct {
    Effect        SmokeEffect
    TurnsLeft     int
    Color         mgl32.Vec3
    BufferOffset  int
    Origin        voxel.Int3
    ParticleCount int
}

func (i SmokeEffectInfo) WithBufferOffset(offset int) SmokeEffectInfo {
    i.BufferOffset = offset
    return i
}

func (i SmokeEffectInfo) WithOrigin(origin voxel.Int3) SmokeEffectInfo {
    i.Origin = origin
    return i
}

func (i SmokeEffectInfo) WithParticleCount(count int) SmokeEffectInfo {
    i.ParticleCount = count
    return i
}
type Smoker struct {
    particles         *glhf.ParticleSystem
    voxelMap          *voxel.Map
    particlesPerBlock int
    getMap            func() *voxel.Map
    locationInfo      map[voxel.Int3]SmokeEffectInfo
    runningAnimations map[voxel.Int3]SmokeSpawnAnimation
}

func NewSmoker(getMap func() *voxel.Map, system *glhf.ParticleSystem) *Smoker {
    return &Smoker{
        particles:         system,
        particlesPerBlock: 2,
        getMap:            getMap,
        runningAnimations: make(map[voxel.Int3]SmokeSpawnAnimation),
        locationInfo:      make(map[voxel.Int3]SmokeEffectInfo),
    }
}
func (s *Smoker) getFireProps(origin mgl32.Vec3, color mgl32.Vec3) glhf.ParticleProperties {
    return glhf.ParticleProperties{
        Origin:               origin,
        PositionVariation:    mgl32.Vec3{0.5, 0.2, 0.5},
        VelocityFromPosition: func(origin, pos mgl32.Vec3) mgl32.Vec3 { return mgl32.Vec3{0.0, 0.6, 0.0} },
        VelocityVariation:    mgl32.Vec3{0.56, 0.84, 0.56},
        MaxDistance:          0.0,
        SizeBegin:            0.2,
        SizeVariation:        0.1,
        SizeEnd:              0.1,
        Lifetime:             -0.8, // loop
        SpreadLifetime:       0.15,
        ColorBegin:           color,
        ColorVariation:       0.1,
        ColorEnd:             mgl32.Vec3{0.1, 0.1, 0.1},
    }
}
func (s *Smoker) getSmokeProps(origin mgl32.Vec3, color mgl32.Vec3) glhf.ParticleProperties {
    return glhf.ParticleProperties{
        Origin:               origin,
        PositionVariation:    mgl32.Vec3{0.5, 0.5, 0.5},
        VelocityFromPosition: func(origin, pos mgl32.Vec3) mgl32.Vec3 { return mgl32.Vec3{0.0, 0.0, 0.0} },
        VelocityVariation:    mgl32.Vec3{0.16, 0.2, 0.16},
        MaxDistance:          0.5,
        SizeBegin:            0.2,
        SizeVariation:        0.1,
        SizeEnd:              0.1,
        Lifetime: -101, // infinite
        ColorBegin: color,
        ColorVariation:       0.1,
        ColorEnd:             mgl32.Vec3{0.2, 0.2, 0.2},
    }
}

func (s *Smoker) Draw(elapsed float64) {
    s.particles.Draw(elapsed)
}

func (s *Smoker) Update(elapsed float64) {
    for i := range s.runningAnimations {
        animation := s.runningAnimations[i]
        currentRadius, radiusChanged := animation.Update(elapsed)
        if radiusChanged {
            s.createSmokeVolume(animation.origin, currentRadius, animation.effect)
        }
        if !animation.Running() {
            delete(s.runningAnimations, i)
        } else {
            s.runningAnimations[i] = animation
        }
    }
}

func (s *Smoker) NextTurn() {
    for location, info := range s.locationInfo {
        info.TurnsLeft--
        if info.TurnsLeft == 0 {
            s.ClearSmokeAt(location)
            continue
        }
        s.locationInfo[location] = info
    }
}

func (s *Smoker) ClearSmokeAt(location voxel.Int3) {
    info, exists := s.locationInfo[location]
    if !exists {
        return
    }
    delete(s.locationInfo, location)
    s.particles.Clear(info.BufferOffset, info.ParticleCount)
    if _, running := s.runningAnimations[info.Origin]; running {
        delete(s.runningAnimations, info.Origin)
    }
}

type SmokeVolume struct {
    locations []voxel.Int3
    origin    voxel.Int3
    radius    float64
    effect    SmokeEffectInfo
    turnsLeft int
}

func (s *Smoker) createSmokeVolume(origin voxel.Int3, radius float64, effect SmokeEffectInfo) {
    withOrigin := effect.WithOrigin(origin)
    s.getMap().ForBlockInSphericFloodFill(origin, radius, func(origin voxel.Int3, steps int, radius float64, x int32, y int32, z int32) {
        if y < 1 {
            return
        }
        pos := voxel.Int3{X: int32(x), Y: int32(y), Z: int32(z)}
        s.addSmokeAt(pos, withOrigin)
    })
}

func (s *Smoker) addSmokeAt(spawnPos voxel.Int3, info SmokeEffectInfo) {
    if _, exists := s.locationInfo[spawnPos]; exists {
        return
    }
    if info.ParticleCount == 0 {
        info = info.WithParticleCount(s.particlesPerBlock)
    }
    particleProperties := s.getSmokeProps(spawnPos.ToBlockCenterVec3D(), info.Color)
    bufferVertexOffset := s.particles.Emit(particleProperties, info.ParticleCount)
    s.locationInfo[spawnPos] = info.WithBufferOffset(bufferVertexOffset)
}
func (s *Smoker) addFireAt(spawnPos voxel.Int3, info SmokeEffectInfo) {
    if _, exists := s.locationInfo[spawnPos]; exists {
        return
    }
    if info.ParticleCount == 0 {
        info = info.WithParticleCount(s.particlesPerBlock)
    }
    particleProperties := s.getFireProps(spawnPos.ToBlockCenterVec3().Add(mgl32.Vec3{0.1, 0.4, 0.1}), info.Color)
    bufferVertexOffset := s.particles.Emit(particleProperties, info.ParticleCount)
    s.locationInfo[spawnPos] = info.WithBufferOffset(bufferVertexOffset)
}
func (s *Smoker) AddSmokeCloud(location voxel.Int3, radius float64, turns int) {
    s.addAnimatedSmokeEffect(location, radius, SmokeEffectInfo{
        Effect:    SmokeEffectBlockLoS,
        TurnsLeft: turns,
        Color:     mgl32.Vec3{0.2, 0.2, 0.2},
    })
}
func (s *Smoker) AddPoisonCloud(location voxel.Int3, radius float64, turns int) {
    s.addAnimatedSmokeEffect(location, radius, SmokeEffectInfo{
        Effect:    SmokeEffectDamage,
        TurnsLeft: turns,
        Color:     mgl32.Vec3{0.1, 0.5, 0.1},
    })
}

func (s *Smoker) AddFire(location voxel.Int3, turns int) {
    s.addFireAt(location, SmokeEffectInfo{
        Effect:        SmokeEffectDamage,
        TurnsLeft:     turns,
        Color:         mgl32.Vec3{0.5, 0.1, 0.1},
        ParticleCount: 10,
    })
}
func (s *Smoker) addAnimatedSmokeEffect(location voxel.Int3, radius float64, effect SmokeEffectInfo) {
    s.runningAnimations[location] = SmokeSpawnAnimation{
        origin:     location,
        lifeTime:   5,
        radius:     radius,
        effect: effect,
    }
}
type SmokeSpawnAnimation struct {
    origin     voxel.Int3
    lived      float64
    lifeTime   float64
    radius     float64
    lastRadius float64
    effect     SmokeEffectInfo
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