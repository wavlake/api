package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/wavlake/api/internal/models"
)

// AudioProcessor handles audio file processing and compression
type AudioProcessor struct {
	tempDir string
}

// NewAudioProcessor creates a new audio processor
func NewAudioProcessor(tempDir string) *AudioProcessor {
	return &AudioProcessor{
		tempDir: tempDir,
	}
}

// AudioInfo contains metadata about an audio file
type AudioInfo struct {
	Duration   int   // Duration in seconds
	Size       int64 // File size in bytes
	Bitrate    int   // Bitrate in kbps
	SampleRate int   // Sample rate in Hz
	Channels   int   // Number of channels
}

// GetAudioInfo extracts metadata from an audio file using ffprobe
func (ap *AudioProcessor) GetAudioInfo(ctx context.Context, inputPath string) (*AudioInfo, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-show_entries", "format=duration,size,bit_rate",
		"-show_entries", "stream=sample_rate,channels",
		"-of", "csv=p=0",
		inputPath)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get audio info: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("unexpected ffprobe output format")
	}

	// Parse format info (duration, size, bit_rate)
	formatParts := strings.Split(lines[0], ",")
	if len(formatParts) < 3 {
		return nil, fmt.Errorf("unexpected format info format")
	}

	duration, err := strconv.ParseFloat(formatParts[0], 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse duration: %w", err)
	}

	size, err := strconv.ParseInt(formatParts[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse size: %w", err)
	}

	bitrate, err := strconv.ParseInt(formatParts[2], 10, 64)
	if err != nil {
		// Bitrate might be N/A, calculate from size and duration
		if duration > 0 && size > 0 {
			bitrate = int64((float64(size) * 8) / (duration * 1000))
		}
	}

	// Parse stream info (sample_rate, channels)
	streamParts := strings.Split(lines[1], ",")
	if len(streamParts) < 2 {
		return nil, fmt.Errorf("unexpected stream info format")
	}

	sampleRate, err := strconv.Atoi(streamParts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse sample rate: %w", err)
	}

	channels, err := strconv.Atoi(streamParts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse channels: %w", err)
	}

	return &AudioInfo{
		Duration:   int(duration),
		Size:       size,
		Bitrate:    int(bitrate),
		SampleRate: sampleRate,
		Channels:   channels,
	}, nil
}

// CompressAudio compresses an audio file to a reasonable streaming quality
// Target: 128kbps MP3, 44.1kHz sample rate
func (ap *AudioProcessor) CompressAudio(ctx context.Context, inputPath, outputPath string) error {
	// Create output directory if it doesn't exist
	// #nosec G301
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Use ffmpeg to compress the audio
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-i", inputPath,
		"-codec:a", "libmp3lame", // Use LAME MP3 encoder
		"-b:a", "128k", // 128 kbps bitrate
		"-ar", "44100", // 44.1 kHz sample rate
		"-ac", "2", // Stereo (2 channels)
		"-f", "mp3", // Output format
		"-y", // Overwrite output file
		outputPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to compress audio: %w, output: %s", err, string(output))
	}

	log.Printf("Successfully compressed audio: %s -> %s", inputPath, outputPath)
	return nil
}

// DownloadAndCompress downloads an audio file from a URL and compresses it
func (ap *AudioProcessor) DownloadAndCompress(ctx context.Context, sourceURL, outputPath string) (*AudioInfo, error) {
	// Create temporary file for download
	tempFile, err := os.CreateTemp(ap.tempDir, "audio_download_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Download the file using curl (more reliable than Go's http client for large files)
	cmd := exec.CommandContext(ctx, "curl", // #nosec G204 -- Curl execution with controlled args for file download
		"-L",                  // Follow redirects
		"-o", tempFile.Name(), // Output to temp file
		sourceURL)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to download audio file: %w", err)
	}

	// Get info about the original file
	audioInfo, err := ap.GetAudioInfo(ctx, tempFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to get audio info: %w", err)
	}

	// Compress the audio
	if err := ap.CompressAudio(ctx, tempFile.Name(), outputPath); err != nil {
		return nil, fmt.Errorf("failed to compress audio: %w", err)
	}

	return audioInfo, nil
}

// ValidateAudioFile checks if a file is a valid audio file
func (ap *AudioProcessor) ValidateAudioFile(ctx context.Context, filePath string) error {
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "stream=codec_type",
		"-of", "csv=p=0",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("file is not a valid audio file: %w", err)
	}

	if strings.TrimSpace(string(output)) != "audio" {
		return fmt.Errorf("file does not contain audio stream")
	}

	return nil
}

// GetSupportedFormats returns a list of supported audio formats
func (ap *AudioProcessor) GetSupportedFormats() []string {
	return []string{
		"mp3", "wav", "flac", "aac", "ogg", "m4a", "wma", "aiff", "au",
	}
}

// IsFormatSupported checks if an audio format is supported
func (ap *AudioProcessor) IsFormatSupported(extension string) bool {
	extension = strings.ToLower(strings.TrimPrefix(extension, "."))
	for _, format := range ap.GetSupportedFormats() {
		if format == extension {
			return true
		}
	}
	return false
}

// CompressAudioWithOptions compresses audio with specific user-defined options
func (ap *AudioProcessor) CompressAudioWithOptions(ctx context.Context, inputPath, outputPath string, options models.CompressionOption) error {
	log.Printf("Compressing audio with options: %+v", options)

	// Build ffmpeg command based on format and options
	args := []string{
		"-i", inputPath,
		"-y", // Overwrite output file
	}

	// Add format-specific encoding options
	switch options.Format {
	case "mp3":
		args = append(args, "-c:a", "libmp3lame")
		args = append(args, "-b:a", fmt.Sprintf("%dk", options.Bitrate))
	case "aac":
		args = append(args, "-c:a", "aac")
		args = append(args, "-b:a", fmt.Sprintf("%dk", options.Bitrate))
	case "ogg":
		args = append(args, "-c:a", "libvorbis")
		args = append(args, "-b:a", fmt.Sprintf("%dk", options.Bitrate))
	default:
		return fmt.Errorf("unsupported format: %s", options.Format)
	}

	// Add sample rate if specified
	if options.SampleRate > 0 {
		args = append(args, "-ar", fmt.Sprintf("%d", options.SampleRate))
	}

	// Add quality settings based on quality level
	switch options.Quality {
	case "low":
		args = append(args, "-q:a", "9") // Lower quality, smaller file
	case "medium":
		args = append(args, "-q:a", "5") // Balanced
	case "high":
		args = append(args, "-q:a", "1") // Higher quality, larger file
	}

	// Add output path
	args = append(args, outputPath)

	// Execute ffmpeg
	cmd := exec.CommandContext(ctx, "ffmpeg", args...) // #nosec G204 -- FFmpeg execution with controlled args for audio processing
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to compress audio with options %+v: %w, output: %s", options, err, string(output))
	}

	log.Printf("Successfully compressed audio with options: %s -> %s", inputPath, outputPath)
	return nil
}
