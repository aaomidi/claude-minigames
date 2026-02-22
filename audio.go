package main

import (
	"bytes"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// Shared audio context — created once, used by all games
var gameAudioCtx *audio.Context

func audioCtx() *audio.Context {
	if gameAudioCtx == nil {
		gameAudioCtx = audio.NewContext(44100)
	}
	return gameAudioCtx
}

const sampleRate = 44100

// --- PCM generation helpers ---

// writeSample writes a stereo int16LE sample to buf at offset
func writeSample(buf []byte, offset int, val float64) {
	if val > 1 {
		val = 1
	}
	if val < -1 {
		val = -1
	}
	s := int16(val * 32000)
	buf[offset] = byte(s)
	buf[offset+1] = byte(s >> 8)
	buf[offset+2] = byte(s)
	buf[offset+3] = byte(s >> 8)
}

// pcmStereo allocates a buffer for the given duration
func pcmStereo(seconds float64) []byte {
	n := int(float64(sampleRate) * seconds)
	return make([]byte, n*4) // 2 channels * 2 bytes
}

// --- Waveforms ---

func sineWave(t, freq float64) float64 {
	return math.Sin(2 * math.Pi * freq * t)
}

func squareWave(t, freq float64) float64 {
	if math.Mod(t*freq, 1.0) < 0.5 {
		return 1
	}
	return -1
}

func triangleWave(t, freq float64) float64 {
	m := math.Mod(t*freq, 1.0)
	if m < 0.5 {
		return 4*m - 1
	}
	return 3 - 4*m
}

func sawWave(t, freq float64) float64 {
	return 2*math.Mod(t*freq, 1.0) - 1
}

func noise() float64 {
	return rand.Float64()*2 - 1
}

// --- 8-bit style note helpers ---

// noteFreq returns frequency for MIDI-style note number (60 = middle C)
func noteFreq(note int) float64 {
	return 440.0 * math.Pow(2, float64(note-69)/12.0)
}

// --- Player helpers ---

// loopPlayer creates a looping audio player from PCM bytes
func loopPlayer(pcm []byte, vol float64) *audio.Player {
	r := bytes.NewReader(pcm)
	loop := audio.NewInfiniteLoop(r, int64(len(pcm)))
	p, _ := audioCtx().NewPlayer(loop)
	p.SetVolume(vol)
	return p
}

// playSFX plays a one-shot sound effect
func playSFX(pcm []byte, vol float64) {
	p := audioCtx().NewPlayerFromBytes(pcm)
	p.SetVolume(vol)
	p.Play()
}

// --- Envelope ---

// envelope applies attack-sustain-release to a 0-1 time fraction
func envelope(tFrac, attack, sustain, release float64) float64 {
	if tFrac < attack {
		return tFrac / attack
	}
	if tFrac < attack+sustain {
		return 1.0
	}
	rem := tFrac - attack - sustain
	if rem < release {
		return 1.0 - rem/release
	}
	return 0
}

// --- Common music patterns ---

// generateTrack creates a looping track from a note pattern
// Each note is {midiNote, durationInBeats} where 0 = rest
// waveFn picks the waveform, bpm sets tempo, vol sets volume
func generateTrack(notes [][2]int, waveFn func(float64, float64) float64, bpm float64, vol float64) []byte {
	beatSec := 60.0 / bpm
	totalBeats := 0.0
	for _, n := range notes {
		totalBeats += float64(n[1])
	}
	totalSec := totalBeats * beatSec
	buf := pcmStereo(totalSec)
	samples := len(buf) / 4

	sampleIdx := 0
	for _, n := range notes {
		noteSamples := int(float64(n[1]) * beatSec * float64(sampleRate))
		freq := noteFreq(n[0])
		for i := 0; i < noteSamples && sampleIdx < samples; i++ {
			t := float64(sampleIdx) / float64(sampleRate)
			tFrac := float64(i) / float64(noteSamples)
			val := 0.0
			if n[0] > 0 { // not a rest
				env := envelope(tFrac, 0.02, 0.6, 0.38)
				val = waveFn(t, freq) * env * vol
			}
			writeSample(buf, sampleIdx*4, val)
			sampleIdx++
		}
	}
	return buf
}

// generateDrumPattern creates a simple percussion loop
// pattern is a string like "X..x..X." where X=kick, x=hat, .=silence
func generateDrumPattern(pattern string, bpm float64, vol float64) []byte {
	beatSec := 60.0 / bpm / 4 // 16th notes
	totalSec := float64(len(pattern)) * beatSec
	buf := pcmStereo(totalSec)
	samples := len(buf) / 4

	for ci, ch := range pattern {
		startSample := int(float64(ci) * beatSec * float64(sampleRate))
		hitSamples := int(beatSec * float64(sampleRate))
		for i := 0; i < hitSamples && startSample+i < samples; i++ {
			t := float64(i) / float64(sampleRate)
			tFrac := float64(i) / float64(hitSamples)
			val := 0.0
			switch ch {
			case 'X', 'K': // kick
				freq := 80.0 * math.Exp(-t*20)
				val = sineWave(t, freq) * (1 - tFrac) * vol
			case 'x', 'h': // hi-hat
				val = noise() * math.Exp(-t*40) * vol * 0.4
			case 's', 'S': // snare
				val = (noise()*0.6 + sineWave(t, 200)*0.4) * math.Exp(-t*15) * vol * 0.7
			}
			idx := (startSample + i) * 4
			if idx+3 < len(buf) {
				// mix with existing
				existing := float64(int16(buf[idx])|int16(buf[idx+1])<<8) / 32000
				writeSample(buf, idx, existing+val)
			}
		}
	}
	return buf
}

// mixBuffers mixes two PCM buffers together (same length or uses shorter)
func mixBuffers(a, b []byte) []byte {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	out := make([]byte, n)
	for i := 0; i < n-3; i += 4 {
		sa := float64(int16(a[i])|int16(a[i+1])<<8) / 32000
		sb := float64(int16(b[i])|int16(b[i+1])<<8) / 32000
		mixed := sa + sb
		if mixed > 1 {
			mixed = 1
		}
		if mixed < -1 {
			mixed = -1
		}
		writeSample(out, i, mixed)
	}
	return out
}
