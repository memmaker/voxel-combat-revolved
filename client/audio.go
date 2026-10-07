package client

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/rand"
	"sync"

	"github.com/ebitengine/oto/v3"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/memmaker/battleground/engine/util"
	"github.com/memmaker/battleground/game"
)

type SoundID int

const (
	SoundRifle SoundID = iota
	SoundPistol
	SoundShotgun
	SoundSniper
	SoundRocket
	SoundExplosion
	SoundImpactWall
	SoundImpactFlesh
	SoundStep
	SoundUIClick
	SoundTurnStart
	SoundDeath
	SoundThrow
	SoundReload
	SoundOverwatch
	SoundVictory
	SoundDefeat
	soundCount
)

const sampleRate = 44100

// Audio plays synthesized sound effects. All sounds are generated at startup, so the game needs
// no audio files. It is safe to use when no audio device is available: everything becomes a no-op.
type Audio struct {
	context  *oto.Context
	samples  [soundCount][]byte
	volume   float64
	listener func() mgl32.Vec3

	lock    sync.Mutex
	playing []*oto.Player
}

var (
	audioOnce     sync.Once
	sharedAudio   *Audio
	audioDisabled = &Audio{}
)

// GetAudio returns the process-wide audio engine (oto allows only one context per process).
func GetAudio() *Audio {
	audioOnce.Do(func() {
		context, ready, err := oto.NewContext(&oto.NewContextOptions{
			SampleRate:   sampleRate,
			ChannelCount: 1,
			Format:       oto.FormatSignedInt16LE,
		})
		if err != nil {
			util.LogGlobalUnitDebug("[Audio] no audio device: " + err.Error())
			sharedAudio = audioDisabled
			return
		}
		a := &Audio{context: context, volume: 0.7}
		a.synthesizeAll()
		go func() { <-ready }()
		sharedAudio = a
	})
	return sharedAudio
}

func (a *Audio) SetVolume(volume float32) {
	a.volume = float64(volume)
}

// SetListener sets the function returning the listener (camera) position for distance attenuation.
func (a *Audio) SetListener(listener func() mgl32.Vec3) {
	a.listener = listener
}

// Play plays a non-positional sound (UI, announcements).
func (a *Audio) Play(id SoundID) {
	a.play(id, 1.0)
}

// PlayAt plays a sound emitted at a world position; it gets quieter with distance to the listener.
func (a *Audio) PlayAt(id SoundID, position mgl32.Vec3) {
	attenuation := 1.0
	if a.listener != nil {
		distance := float64(a.listener().Sub(position).Len())
		attenuation = 1.0 / (1.0 + math.Max(0, distance-8)*0.06)
	}
	a.play(id, attenuation)
}

func (a *Audio) play(id SoundID, gain float64) {
	if a.context == nil || a.volume <= 0 || int(id) >= len(a.samples) {
		return
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	// players must stay referenced while playing; drop the finished ones
	alive := a.playing[:0]
	for _, player := range a.playing {
		if player.IsPlaying() {
			alive = append(alive, player)
		} else {
			_ = player.Close()
		}
	}
	a.playing = alive
	if len(a.playing) > 24 { // never let effects pile up into noise
		return
	}
	player := a.context.NewPlayer(bytes.NewReader(a.samples[id]))
	player.SetVolume(math.Min(1, a.volume*gain))
	player.Play()
	a.playing = append(a.playing, player)
}

// PlayWeapon plays the firing sound for a weapon type.
func (a *Audio) PlayWeapon(weaponType game.WeaponType, position mgl32.Vec3) {
	switch weaponType {
	case game.WeaponPistol:
		a.PlayAt(SoundPistol, position)
	case game.WeaponShotgun:
		a.PlayAt(SoundShotgun, position)
	case game.WeaponSniper:
		a.PlayAt(SoundSniper, position)
	case game.WeaponRocketLauncher:
		a.PlayAt(SoundRocket, position)
	default:
		a.PlayAt(SoundRifle, position)
	}
}

// ---- synthesis ----

type synth struct {
	samples []float64
	rng     *rand.Rand
}

func newSynth(seconds float64, seed int64) *synth {
	return &synth{samples: make([]float64, int(seconds*sampleRate)), rng: rand.New(rand.NewSource(seed))}
}

func (s *synth) duration() float64 { return float64(len(s.samples)) / sampleRate }

// noise adds filtered noise with an exponential decay. lowpass in 0..1 (smaller is darker).
func (s *synth) noise(start, decay, amplitude, lowpass float64) {
	previous := 0.0
	for i := int(start * sampleRate); i < len(s.samples); i++ {
		t := float64(i)/sampleRate - start
		white := s.rng.Float64()*2 - 1
		previous += lowpass * (white - previous)
		s.samples[i] += previous * amplitude * math.Exp(-t/decay)
	}
}

// tone adds a sine sweeping from startFreq to endFreq with an exponential decay.
func (s *synth) tone(start, decay, amplitude, startFreq, endFreq float64) {
	phase := 0.0
	end := s.duration()
	for i := int(start * sampleRate); i < len(s.samples); i++ {
		t := float64(i)/sampleRate - start
		progress := t / math.Max(0.0001, end-start)
		frequency := startFreq + (endFreq-startFreq)*progress
		phase += 2 * math.Pi * frequency / sampleRate
		s.samples[i] += math.Sin(phase) * amplitude * math.Exp(-t/decay)
	}
}

// square adds a short square wave blip (UI sounds).
func (s *synth) square(start, length, amplitude, frequency float64) {
	for i := int(start * sampleRate); i < len(s.samples) && float64(i)/sampleRate < start+length; i++ {
		t := float64(i)/sampleRate - start
		value := 1.0
		if math.Mod(t*frequency, 1) > 0.5 {
			value = -1
		}
		fade := 1 - t/length
		s.samples[i] += value * amplitude * fade
	}
}

// echo adds a delayed copy, which makes shots sound like they happen outdoors.
func (s *synth) echo(delay, gain float64) {
	offset := int(delay * sampleRate)
	for i := len(s.samples) - 1; i >= offset; i-- {
		s.samples[i] += s.samples[i-offset] * gain
	}
}

func (s *synth) pcm() []byte {
	peak := 0.0
	for _, v := range s.samples {
		peak = math.Max(peak, math.Abs(v))
	}
	scale := 1.0
	if peak > 0.95 {
		scale = 0.95 / peak
	}
	buffer := new(bytes.Buffer)
	fadeOut := int(0.01 * sampleRate)
	for i, v := range s.samples {
		if remaining := len(s.samples) - i; remaining < fadeOut {
			v *= float64(remaining) / float64(fadeOut)
		}
		_ = binary.Write(buffer, binary.LittleEndian, int16(v*scale*32767))
	}
	return buffer.Bytes()
}

func (a *Audio) synthesizeAll() {
	build := func(id SoundID, seconds float64, compose func(s *synth)) {
		s := newSynth(seconds, int64(id)+1)
		compose(s)
		a.samples[id] = s.pcm()
	}
	build(SoundRifle, 0.35, func(s *synth) {
		s.noise(0, 0.04, 0.9, 0.55)
		s.tone(0, 0.05, 0.6, 140, 60)
		s.echo(0.09, 0.25)
	})
	build(SoundPistol, 0.3, func(s *synth) {
		s.noise(0, 0.025, 0.8, 0.7)
		s.tone(0, 0.03, 0.4, 220, 90)
		s.echo(0.07, 0.2)
	})
	build(SoundShotgun, 0.6, func(s *synth) {
		s.noise(0, 0.09, 1.0, 0.35)
		s.tone(0, 0.12, 0.8, 90, 40)
		s.echo(0.12, 0.3)
	})
	build(SoundSniper, 1.0, func(s *synth) {
		s.noise(0, 0.02, 1.0, 0.9)
		s.noise(0.005, 0.15, 0.5, 0.25)
		s.tone(0, 0.1, 0.7, 110, 45)
		s.echo(0.18, 0.35)
		s.echo(0.36, 0.2)
	})
	build(SoundRocket, 0.9, func(s *synth) {
		s.noise(0, 0.35, 0.7, 0.2)
		s.tone(0, 0.3, 0.4, 300, 600)
	})
	build(SoundExplosion, 1.8, func(s *synth) {
		s.noise(0, 0.5, 1.0, 0.12)
		s.noise(0, 0.08, 0.6, 0.6)
		s.tone(0, 0.6, 0.9, 70, 25)
		s.echo(0.25, 0.25)
	})
	build(SoundImpactWall, 0.12, func(s *synth) {
		s.noise(0, 0.015, 0.7, 0.8)
		s.tone(0, 0.02, 0.2, 900, 400)
	})
	build(SoundImpactFlesh, 0.2, func(s *synth) {
		s.noise(0, 0.04, 0.6, 0.15)
		s.tone(0, 0.05, 0.5, 160, 70)
	})
	build(SoundStep, 0.09, func(s *synth) {
		s.noise(0, 0.018, 0.35, 0.12)
		s.tone(0, 0.02, 0.2, 90, 60)
	})
	build(SoundUIClick, 0.05, func(s *synth) {
		s.square(0, 0.03, 0.18, 1400)
	})
	build(SoundTurnStart, 0.6, func(s *synth) {
		s.tone(0, 0.2, 0.35, 660, 660)
		s.tone(0.12, 0.25, 0.35, 880, 880)
	})
	build(SoundDeath, 0.7, func(s *synth) {
		s.tone(0, 0.3, 0.4, 260, 70)
		s.noise(0, 0.1, 0.3, 0.1)
	})
	build(SoundThrow, 0.4, func(s *synth) {
		s.noise(0, 0.12, 0.35, 0.3)
		s.tone(0, 0.15, 0.1, 300, 500)
	})
	build(SoundReload, 0.45, func(s *synth) {
		s.noise(0, 0.01, 0.6, 0.9)
		s.tone(0, 0.012, 0.3, 2200, 1800)
		s.noise(0.22, 0.012, 0.7, 0.9)
		s.tone(0.22, 0.015, 0.3, 1600, 1400)
	})
	build(SoundOverwatch, 0.4, func(s *synth) {
		s.tone(0, 0.2, 0.3, 500, 1000)
	})
	build(SoundVictory, 1.4, func(s *synth) {
		for i, frequency := range []float64{523, 659, 784, 1047} {
			s.tone(float64(i)*0.16, 0.35, 0.3, frequency, frequency)
		}
	})
	build(SoundDefeat, 1.4, func(s *synth) {
		for i, frequency := range []float64{440, 392, 349, 262} {
			s.tone(float64(i)*0.22, 0.4, 0.3, frequency, frequency)
		}
	})
}
