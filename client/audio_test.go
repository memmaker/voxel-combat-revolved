package client

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// TestSoundsSynthesize checks every effect is audible and not clipped.
// Set BT_DUMP_SOUNDS=<dir> to write them as WAV files for listening.
func TestSoundsSynthesize(t *testing.T) {
	a := &Audio{}
	a.synthesizeAll()
	dumpDir := os.Getenv("BT_DUMP_SOUNDS")
	for id := SoundID(0); id < soundCount; id++ {
		pcm := a.samples[id]
		if len(pcm) == 0 {
			t.Fatalf("sound %d is empty", id)
		}
		peak, sumSquares := 0.0, 0.0
		for i := 0; i+1 < len(pcm); i += 2 {
			v := float64(int16(binary.LittleEndian.Uint16(pcm[i:]))) / 32768
			peak = math.Max(peak, math.Abs(v))
			sumSquares += v * v
		}
		rms := math.Sqrt(sumSquares / float64(len(pcm)/2))
		if peak < 0.1 || rms < 0.005 {
			t.Errorf("sound %d is nearly silent (peak %.3f, rms %.4f)", id, peak, rms)
		}
		if peak > 0.99 {
			t.Errorf("sound %d clips (peak %.3f)", id, peak)
		}
		if dumpDir != "" {
			writeWav(t, filepath.Join(dumpDir, fmt.Sprintf("sound_%02d.wav", id)), pcm)
		}
	}
}

func writeWav(t *testing.T, filename string, pcm []byte) {
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	header := []any{
		[4]byte{'R', 'I', 'F', 'F'}, uint32(36 + len(pcm)), [4]byte{'W', 'A', 'V', 'E'},
		[4]byte{'f', 'm', 't', ' '}, uint32(16), uint16(1), uint16(1), uint32(sampleRate), uint32(sampleRate * 2), uint16(2), uint16(16),
		[4]byte{'d', 'a', 't', 'a'}, uint32(len(pcm)),
	}
	for _, field := range header {
		_ = binary.Write(file, binary.LittleEndian, field)
	}
	_, _ = file.Write(pcm)
}
