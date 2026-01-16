package services

import (
	"context"
	"fmt"

	speech "cloud.google.com/go/speech/apiv1"
	"cloud.google.com/go/speech/apiv1/speechpb"
)

type SpeechToTextService struct {
	client *speech.Client
}

func NewSpeechToTextService(ctx context.Context) (*SpeechToTextService, error) {
	client, err := speech.NewClient(ctx)
	if err != nil {
		// If Google Speech-to-Text is not configured, return service with nil client
		// It will use mock transcription
		return &SpeechToTextService{client: nil}, nil
	}
	
	return &SpeechToTextService{
		client: client,
	}, nil
}

// TranscribeAudio converts speech audio to text
func (s *SpeechToTextService) TranscribeAudio(ctx context.Context, audioData []byte) (text string, language string, confidence float64, err error) {
	if s.client == nil {
		// Mock transcription for development/testing
		return s.mockTranscription(audioData)
	}
	
	// Real Google Speech-to-Text API call
	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 16000,
			LanguageCode:    "en-US",
			EnableAutomaticPunctuation: true,
			Model: "default",
			EnableWordTimeOffsets: true,
			AlternativeLanguageCodes: []string{
				"es-ES", "fr-FR", "de-DE", "it-IT", 
				"pt-PT", "ja-JP", "ko-KR", "zh-CN",
			},
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: audioData,
			},
		},
	}
	
	resp, err := s.client.Recognize(ctx, req)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to recognize speech: %w", err)
	}
	
	if len(resp.Results) == 0 {
		return "", "", 0, fmt.Errorf("no speech detected")
	}
	
	result := resp.Results[0]
	if len(result.Alternatives) == 0 {
		return "", "", 0, fmt.Errorf("no alternatives found")
	}
	
	alternative := result.Alternatives[0]
	
	// Detect language from the result
	detectedLanguage := result.LanguageCode
	if detectedLanguage == "" {
		detectedLanguage = "en"
	}
	
	return alternative.Transcript, detectedLanguage, float64(alternative.Confidence), nil
}

// StreamTranscribe provides real-time streaming transcription
func (s *SpeechToTextService) StreamTranscribe(ctx context.Context) (*StreamTranscriber, error) {
	if s.client == nil {
		return nil, fmt.Errorf("speech client not initialized")
	}
	
	stream, err := s.client.StreamingRecognize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create streaming client: %w", err)
	}
	
	// Send the initial configuration
	if err := stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					Encoding:        speechpb.RecognitionConfig_LINEAR16,
					SampleRateHertz: 16000,
					LanguageCode:    "en-US",
					EnableAutomaticPunctuation: true,
					AlternativeLanguageCodes: []string{
						"es-ES", "fr-FR", "de-DE", "it-IT",
						"pt-PT", "ja-JP", "ko-KR", "zh-CN",
					},
				},
				InterimResults: true,
			},
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to send config: %w", err)
	}
	
	return &StreamTranscriber{
		stream: stream,
	}, nil
}

type StreamTranscriber struct {
	stream speechpb.Speech_StreamingRecognizeClient
}

// SendAudio sends audio chunk to the streaming transcription
func (st *StreamTranscriber) SendAudio(audioData []byte) error {
	return st.stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
			AudioContent: audioData,
		},
	})
}

// ReceiveTranscription receives transcription results
func (st *StreamTranscriber) ReceiveTranscription() (text string, isFinal bool, confidence float64, err error) {
	resp, err := st.stream.Recv()
	if err != nil {
		return "", false, 0, err
	}
	
	if len(resp.Results) == 0 {
		return "", false, 0, nil
	}
	
	result := resp.Results[0]
	if len(result.Alternatives) == 0 {
		return "", false, 0, nil
	}
	
	alternative := result.Alternatives[0]
	
	return alternative.Transcript, result.IsFinal, float64(alternative.Confidence), nil
}

// Close closes the streaming transcription
func (st *StreamTranscriber) Close() error {
	return st.stream.CloseSend()
}

// mockTranscription provides mock transcription for development
func (s *SpeechToTextService) mockTranscription(audioData []byte) (text string, language string, confidence float64, err error) {
	// Mock transcription based on audio data length
	mockTexts := []string{
		"Hello, how are you?",
		"I'm learning a new language.",
		"This is a test message.",
		"Can you help me with this?",
		"Thank you very much!",
	}
	
	// Use audio data length as pseudo-random index
	index := len(audioData) % len(mockTexts)
	
	return mockTexts[index], "en", 0.95, nil
}

// TranscribeWithTimestamps provides word-level timestamps
type TimestampedWord struct {
	Word      string  `json:"word"`
	StartTime float64 `json:"startTime"`
	EndTime   float64 `json:"endTime"`
	Confidence float64 `json:"confidence"`
}

func (s *SpeechToTextService) TranscribeWithTimestamps(ctx context.Context, audioData []byte) ([]TimestampedWord, error) {
	if s.client == nil {
		return nil, fmt.Errorf("speech client not initialized")
	}
	
	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 16000,
			LanguageCode:    "en-US",
			EnableWordTimeOffsets: true,
			EnableAutomaticPunctuation: true,
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: audioData,
			},
		},
	}
	
	resp, err := s.client.Recognize(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to recognize speech: %w", err)
	}
	
	if len(resp.Results) == 0 {
		return nil, fmt.Errorf("no speech detected")
	}
	
	words := []TimestampedWord{}
	
	for _, result := range resp.Results {
		if len(result.Alternatives) == 0 {
			continue
		}
		
		alternative := result.Alternatives[0]
		
		for _, wordInfo := range alternative.Words {
			words = append(words, TimestampedWord{
				Word:      wordInfo.Word,
				StartTime: float64(wordInfo.StartTime.Seconds) + float64(wordInfo.StartTime.Nanos)/1e9,
				EndTime:   float64(wordInfo.EndTime.Seconds) + float64(wordInfo.EndTime.Nanos)/1e9,
				Confidence: float64(alternative.Confidence),
			})
		}
	}
	
	return words, nil
}

// DetectLanguageFromAudio detects the language of spoken audio
func (s *SpeechToTextService) DetectLanguageFromAudio(ctx context.Context, audioData []byte) (string, float64, error) {
	if s.client == nil {
		return "en", 0.95, nil
	}
	
	// Use multiple language codes to detect
	req := &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 16000,
			LanguageCode:    "en-US",
			AlternativeLanguageCodes: []string{
				"es-ES", "fr-FR", "de-DE", "it-IT",
				"pt-PT", "ja-JP", "ko-KR", "zh-CN",
			},
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Content{
				Content: audioData,
			},
		},
	}
	
	resp, err := s.client.Recognize(ctx, req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to detect language: %w", err)
	}
	
	if len(resp.Results) == 0 {
		return "", 0, fmt.Errorf("no speech detected")
	}
	
	result := resp.Results[0]
	detectedLanguage := result.LanguageCode
	if detectedLanguage == "" {
		detectedLanguage = "en-US"
	}
	
	// Extract language code (e.g., "en" from "en-US")
	if len(detectedLanguage) >= 2 {
		detectedLanguage = detectedLanguage[:2]
	}
	
	confidence := 0.0
	if len(result.Alternatives) > 0 {
		confidence = float64(result.Alternatives[0].Confidence)
	}
	
	return detectedLanguage, confidence, nil
}

// Close closes the Speech-to-Text client
func (s *SpeechToTextService) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}
