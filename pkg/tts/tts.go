// pkg/tts/tts.go
package tts

import (
	"context"
	"fmt"
	"io"

	pb "github.com/flutter-webrtc/flutter-webrtc-server/google/cloud/texttospeech/v1beta1"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

// TTSService handles text-to-speech conversion using Google Cloud TTS
type TTSService struct {
	client     pb.TextToSpeechClient
	sampleRate int32
}

// AudioStream represents a stream of audio data
type AudioStream struct {
	AudioChan chan []byte
	ErrChan   chan error
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewTTSService creates a new TTS service instance
func NewTTSService(ctx context.Context, ts oauth2.TokenSource) (*TTSService, error) {
	// Create proper transport credentials using TLS
	creds := credentials.NewClientTLSFromCert(nil, "")

	// Create gRPC dial options with both TLS and OAuth2
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(oauth.TokenSource{TokenSource: ts}),
	}

	// Connect to the service
	conn, err := grpc.DialContext(ctx, "texttospeech.googleapis.com:443", opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %v", err)
	}

	return &TTSService{
		client:     pb.NewTextToSpeechClient(conn),
		sampleRate: 24000, // Google Cloud TTS uses 24kHz sample rate
	}, nil
}

// StreamSpeech converts text to speech and returns an audio stream
func (s *TTSService) StreamSpeech(ctx context.Context, text string, voice string, lang string) (*AudioStream, error) {
	// Create streaming client
	streamClient, err := s.client.StreamingSynthesize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create streaming client: %v", err)
	}

	// Send initial config
	configReq := &pb.StreamingSynthesizeRequest{
		StreamingRequest: &pb.StreamingSynthesizeRequest_StreamingConfig{
			StreamingConfig: &pb.StreamingSynthesizeConfig{
				Voice: &pb.VoiceSelectionParams{
					LanguageCode: lang,
					Name:         voice,
				},
			},
		},
	}

	if err := streamClient.Send(configReq); err != nil {
		return nil, fmt.Errorf("failed to send config: %v", err)
	}

	// Send text input
	inputReq := &pb.StreamingSynthesizeRequest{
		StreamingRequest: &pb.StreamingSynthesizeRequest_Input{
			Input: &pb.StreamingSynthesisInput{
				InputSource: &pb.StreamingSynthesisInput_Text{
					Text: text,
				},
			},
		},
	}

	if err := streamClient.Send(inputReq); err != nil {
		return nil, fmt.Errorf("failed to send input: %v", err)
	}

	// Close send direction
	if err := streamClient.CloseSend(); err != nil {
		return nil, fmt.Errorf("failed to close send: %v", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	audioStream := &AudioStream{
		AudioChan: make(chan []byte, 100), // Buffer size of 100
		ErrChan:   make(chan error, 1),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start receiving audio data in a goroutine
	go func() {
		defer close(audioStream.AudioChan)
		defer close(audioStream.ErrChan)

		for {
			resp, err := streamClient.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				audioStream.ErrChan <- fmt.Errorf("failed to receive: %v", err)
				return
			}

			select {
			case <-ctx.Done():
				return
			case audioStream.AudioChan <- resp.GetAudioContent():
			}
		}
	}()

	return audioStream, nil
}

// Close stops the audio stream
func (a *AudioStream) Close() {
	if a.cancel != nil {
		a.cancel()
	}
}

// Read implements io.Reader for the audio stream
func (a *AudioStream) Read(p []byte) (n int, err error) {
	select {
	case <-a.ctx.Done():
		return 0, io.EOF
	case err := <-a.ErrChan:
		return 0, err
	case data, ok := <-a.AudioChan:
		if !ok {
			return 0, io.EOF
		}
		n = copy(p, data)
		return n, nil
	}
}
