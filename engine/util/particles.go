package util

import (
    "github.com/go-gl/mathgl/mgl32"
    "github.com/memmaker/battleground/engine/glhf"
    "math"
    "math/rand"
)

type ParticleProperties struct {
    Position                          mgl32.Vec3
    Velocity, VelocityVariation       mgl32.Vec3
    RotationVariation                 float32
    ColorBegin, ColorEnd              mgl32.Vec4
    SizeBegin, SizeEnd, SizeVariation float32
    Lifetime                          float32
}

type Particle struct {
    id                     uint64
    Position, Velocity     mgl32.Vec3
    ColorBegin, ColorEnd   mgl32.Vec4
    Rotation               float32
    SizeBegin, SizeEnd     float32
    Lifetime, LifetimeLeft float32
    IsActive               bool
}

func (p Particle) GetID() uint64 {
    return p.id
}

type ParticleSystem struct {
    pool      []*Particle
    shader    *glhf.Shader
    poolIndex int

    //verti
}

func NewParticleSystem(shader *glhf.Shader, capacity int) *ParticleSystem {
    initialPool := make([]*Particle, capacity)
    for i := uint64(0); i < uint64(capacity); i++ {
        particle := &Particle{id: i}
        initialPool[i] = particle
    }
    return &ParticleSystem{
        shader:    shader,
        pool:      initialPool,
        poolIndex: len(initialPool) - 1,
    }
}

func (p *ParticleSystem) Emit(props ParticleProperties) {
    particle := p.pool[p.poolIndex]

    particle.IsActive = true
    particle.Position = props.Position
    particle.Rotation = props.RotationVariation * (rand.Float32() * 2.0 * math.Pi)
    particle.Velocity = mgl32.Vec3{
        props.Velocity.X() + props.VelocityVariation.X()*(rand.Float32()-0.5),
        props.Velocity.Y() + props.VelocityVariation.Y()*(rand.Float32()-0.5),
        props.Velocity.Z() + props.VelocityVariation.Z()*(rand.Float32()-0.5),
    }

    particle.ColorBegin = props.ColorBegin
    particle.ColorEnd = props.ColorEnd

    particle.SizeBegin = props.SizeBegin + props.SizeVariation*(rand.Float32()-0.5)
    particle.SizeEnd = props.SizeEnd

    particle.Lifetime = props.Lifetime
    particle.LifetimeLeft = props.Lifetime

    p.poolIndex--
    if p.poolIndex < 0 {
        p.poolIndex = len(p.pool) - 1
    }
}

func (p *ParticleSystem) Update(deltaTime float64) {
    for _, particle := range p.pool {
        if !particle.IsActive {
            continue
        }
        if particle.LifetimeLeft <= 0 {
            particle.IsActive = false
            continue
        }

        particle.LifetimeLeft -= float32(deltaTime)
        particle.Position = particle.Position.Add(particle.Velocity.Mul(float32(deltaTime)))
        particle.Rotation += 0.01 * float32(deltaTime)
    }
}

func (p *ParticleSystem) Draw() {

}
