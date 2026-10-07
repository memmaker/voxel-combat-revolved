package client

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/rand"
	"sync"
)

// The soundtrack is generated at startup: a slow minor-key loop (Am - F - C - G) with a soft pad,
// a pulsing bass, a plucked arpeggio and quiet hi-hats. It loops seamlessly.

const (
	musicBPM      = 80.0
	musicBars     = 16
	beatsPerBar   = 4
	musicSections = 4 // the arpeggio comes and goes every few bars
)

var (
	musicOnce sync.Once
	musicPCM  []byte
)

func noteFrequency(midi float64) float64 {
	return 440 * math.Pow(2, (midi-69)/12)
}

func synthesizeMusic() []byte {
	beat := 60.0 / musicBPM
	bar := beat * beatsPerBar
	total := int(bar * musicBars * sampleRate)
	samples := make([]float64, total)
	rng := rand.New(rand.NewSource(99))
	chords := [][]float64{{57, 60, 64}, {53, 57, 60}, {48, 52, 55}, {55, 59, 62}} // Am, F, C, G

	addTone := func(start, length, frequency, amplitude, attack float64, shape func(phase float64) float64) {
		first := int(start * sampleRate)
		count := int(length * sampleRate)
		for i := 0; i < count && first+i < total; i++ {
			t := float64(i) / sampleRate
			envelope := math.Min(1, t/attack) * math.Min(1, (length-t)/math.Max(attack, 0.05))
			samples[first+i] += shape(t*frequency) * amplitude * envelope
		}
	}
	sine := func(p float64) float64 { return math.Sin(2 * math.Pi * p) }
	soft := func(p float64) float64 { // a few harmonics: warmer than a sine
		return math.Sin(2*math.Pi*p) + 0.3*math.Sin(4*math.Pi*p) + 0.12*math.Sin(6*math.Pi*p)
	}
	pluck := func(p float64) float64 { return 2*math.Abs(2*(p-math.Floor(p+0.5))) - 1 } // triangle

	for b := 0; b < musicBars; b++ {
		chord := chords[(b/2)%len(chords)] // each chord lasts two bars
		start := float64(b) * bar
		// pad: detuned chord tones with slow swells
		for _, note := range chord {
			for _, detune := range []float64{-0.08, 0.08} {
				addTone(start, bar, noteFrequency(note+detune), 0.045, 1.2, soft)
			}
		}
		// bass on beats 1 and 3
		for _, beatIndex := range []int{0, 2} {
			bassStart := start + float64(beatIndex)*beat
			first := int(bassStart * sampleRate)
			for i := 0; i < int(beat*1.6*sampleRate) && first+i < total; i++ {
				t := float64(i) / sampleRate
				samples[first+i] += sine(t*noteFrequency(chord[0]-24)) * 0.22 * math.Exp(-t/0.45)
			}
		}
		// arpeggio in eighth notes during the second half of each section
		if (b/(musicBars/musicSections))%2 == 1 || b%4 >= 2 {
			for step := 0; step < beatsPerBar*2; step++ {
				note := chord[(step*2+b)%len(chord)] + 12
				if step%4 == 3 {
					note += 12
				}
				noteStart := start + float64(step)*beat/2
				first := int(noteStart * sampleRate)
				for i := 0; i < int(beat*0.9*sampleRate) && first+i < total; i++ {
					t := float64(i) / sampleRate
					samples[first+i] += pluck(t*noteFrequency(note)) * 0.035 * math.Exp(-t/0.18)
				}
			}
		}
		// hi-hat ticks on the off-beats
		for step := 0; step < beatsPerBar; step++ {
			first := int((start + (float64(step)+0.5)*beat) * sampleRate)
			previous := 0.0
			for i := 0; i < int(0.05*sampleRate) && first+i < total; i++ {
				white := rng.Float64()*2 - 1
				high := white - previous // crude high-pass
				previous = white
				samples[first+i] += high * 0.025 * math.Exp(-float64(i)/sampleRate/0.012)
			}
		}
	}

	// a gentle low-pass keeps it in the background; crossfade the tail into the head for a seamless loop
	smoothed := 0.0
	for i := range samples {
		smoothed += 0.35 * (samples[i] - smoothed)
		samples[i] = smoothed
	}
	fade := int(0.5 * sampleRate)
	for i := 0; i < fade; i++ {
		mix := float64(i) / float64(fade)
		samples[i] = samples[i]*mix + samples[total-fade+i]*(1-mix)
	}
	samples = samples[:total-fade]

	peak := 0.0
	for _, v := range samples {
		peak = math.Max(peak, math.Abs(v))
	}
	buffer := new(bytes.Buffer)
	for _, v := range samples {
		_ = binary.Write(buffer, binary.LittleEndian, int16(v/peak*0.8*32767))
	}
	return buffer.Bytes()
}

// loopReader repeats its data forever, so the music never stops.
type loopReader struct {
	data []byte
	pos  int
}

func (l *loopReader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		copied := copy(p[n:], l.data[l.pos:])
		n += copied
		l.pos = (l.pos + copied) % len(l.data)
	}
	return n, nil
}

// PlayMusic starts the looping soundtrack (once; later calls only adjust the volume).
func (a *Audio) PlayMusic(volume float32) {
	if a.context == nil {
		return
	}
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.music == nil {
		musicOnce.Do(func() { musicPCM = synthesizeMusic() })
		a.music = a.context.NewPlayer(&loopReader{data: musicPCM})
		a.music.Play()
	}
	a.music.SetVolume(float64(volume) * 0.5)
}
