// pkg/webrtc/audio_stream.go
package audio_stream

import (
	"context"
	"io"
	"time"

	"github.com/flutter-webrtc/flutter-webrtc-server/pkg/tts"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

const (
	// Audio parameters matching Google TTS output
	sampleRate = 24000
	channels   = 1
)

// AudioStreamManager handles TTS audio streaming over WebRTC
type AudioStreamManager struct {
	ttsService     *tts.TTSService
	peerConnection *webrtc.PeerConnection
	audioTrack     *webrtc.TrackLocalStaticSample
}

func NewAudioStreamManager(ttsService *tts.TTSService) *AudioStreamManager {
	return &AudioStreamManager{
		ttsService: ttsService,
	}
}

// InitializePeerConnection sets up WebRTC peer connection with audio track
func (m *AudioStreamManager) InitializePeerConnection(config webrtc.Configuration) error {
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return err
	}

	// Create audio track
	track, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			ClockRate: sampleRate,
			Channels:  channels,
		},
		"audio",
		"tts-stream",
	)
	if err != nil {
		return err
	}

	// Add track to peer connection
	if _, err = pc.AddTrack(track); err != nil {
		return err
	}

	m.peerConnection = pc
	m.audioTrack = track
	return nil
}

// StreamTTSAudio streams synthesized audio from Google TTS through WebRTC
func (m *AudioStreamManager) StreamTTSAudio(ctx context.Context, text, voice, lang string) error {
	// Get audio stream from TTS service
	audioStream, err := m.ttsService.StreamSpeech(ctx, text, voice, lang)
	if err != nil {
		return err
	}
	defer audioStream.Close()

	// Read audio data and write to WebRTC track
	buf := make([]byte, 960) // 20ms of audio at 48kHz
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			n, err := audioStream.Read(buf)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}

			// Write audio samples to WebRTC track
			if err := m.audioTrack.WriteSample(media.Sample{
				Data:     buf[:n],
				Duration: time.Duration(n) * time.Second / time.Duration(sampleRate),
			}); err != nil {
				return err
			}
		}
	}
}

// Close closes the peer connection
func (m *AudioStreamManager) Close() error {
	if m.peerConnection != nil {
		return m.peerConnection.Close()
	}
	return nil
}
