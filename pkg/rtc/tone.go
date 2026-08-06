package rtc

import (
	"math"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

const (
	toneSampleRate = 48000
	tonePerFrame   = 960 // samples per 20 ms frame at 48 kHz
	toneFrequency  = 440.0
	toneAmplitude  = 2200
	toneFrameDur   = 20 * time.Millisecond
)

// toneLoop writes a soft 440 Hz sine wave to the bridged audio track at a fixed
// cadence so the agent hears something on the far end once the call connects.
// This is a stand-in for a real remote caller until a PSTN/SIP leg wires real
// RTP in; the pack-and-write pattern is identical.
type toneLoop struct {
	stopCh chan struct{}
	once   sync.Once
	wg     sync.WaitGroup
}

// startToneLoop begins a background writer; call stop() to end it.
func startToneLoop(track *webrtc.TrackLocalStaticSample) *toneLoop {
	t := &toneLoop{stopCh: make(chan struct{})}
	go t.run(track)
	return t
}

func (t *toneLoop) run(track *webrtc.TrackLocalStaticSample) {
	t.wg.Add(1)
	defer t.wg.Done()

	buf := make([]byte, tonePerFrame*2)
	phase := 0.0
	for {
		select {
		case <-t.stopCh:
			return
		default:
		}
		for i := 0; i < tonePerFrame; i++ {
			sample := int16(toneAmplitude * math.Sin(2*math.Pi*toneFrequency*phase/toneSampleRate))
			buf[i*2] = byte(sample)
			buf[i*2+1] = byte(sample >> 8)
			phase++
		}
		_ = track.WriteSample(media.Sample{Data: buf, Duration: toneFrameDur})
		select {
		case <-t.stopCh:
			return
		case <-time.After(toneFrameDur):
		}
	}
}

func (t *toneLoop) stop() {
	t.once.Do(func() { close(t.stopCh) })
	t.wg.Wait()
}
