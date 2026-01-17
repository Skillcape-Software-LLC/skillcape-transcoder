package transcoder

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ProgressCallback func(progress int)

type FFmpeg struct {
	inputPath  string
	outputPath string
	onProgress ProgressCallback
}

func New(inputPath, outputPath string) *FFmpeg {
	return &FFmpeg{
		inputPath:  inputPath,
		outputPath: outputPath,
	}
}

func (f *FFmpeg) OnProgress(callback ProgressCallback) {
	f.onProgress = callback
}

// Transcode converts the input video to H.264/AAC MP4
func (f *FFmpeg) Transcode(ctx context.Context) error {
	// First, get the duration of the input file
	duration, err := f.getDuration(ctx)
	if err != nil {
		log.Printf("Warning: could not get duration: %v", err)
		duration = 0
	}

	// Build FFmpeg command
	args := []string{
		"-i", f.inputPath,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-progress", "pipe:1",
		"-y",
		f.outputPath,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	// Capture stdout for progress
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Parse progress from stdout
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_ms=") {
			if f.onProgress != nil && duration > 0 {
				timeStr := strings.TrimPrefix(line, "out_time_ms=")
				if timeMs, err := strconv.ParseInt(timeStr, 10, 64); err == nil {
					progress := int((float64(timeMs) / float64(duration*1000)) * 100)
					if progress > 100 {
						progress = 100
					}
					f.onProgress(progress)
				}
			}
		}
	}

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w", err)
	}

	if f.onProgress != nil {
		f.onProgress(100)
	}

	return nil
}

// getDuration returns the duration of the input file in milliseconds
func (f *FFmpeg) getDuration(ctx context.Context) (int64, error) {
	args := []string{
		"-i", f.inputPath,
		"-show_entries", "format=duration",
		"-v", "quiet",
		"-of", "csv=p=0",
	}

	cmd := exec.CommandContext(ctx, "ffprobe", args...)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, err
	}

	return int64(duration * 1000), nil
}

// GetVideoInfo returns basic info about a video file
type VideoInfo struct {
	Duration  time.Duration
	Width     int
	Height    int
	Codec     string
	Bitrate   int64
	FrameRate float64
}

func GetVideoInfo(ctx context.Context, inputPath string) (*VideoInfo, error) {
	args := []string{
		"-i", inputPath,
		"-show_entries", "format=duration,bit_rate:stream=width,height,codec_name,r_frame_rate",
		"-select_streams", "v:0",
		"-v", "quiet",
		"-of", "json",
	}

	cmd := exec.CommandContext(ctx, "ffprobe", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	// Parse the JSON output
	info := &VideoInfo{}

	// Extract duration
	durationRegex := regexp.MustCompile(`"duration":\s*"([^"]+)"`)
	if matches := durationRegex.FindStringSubmatch(string(output)); len(matches) > 1 {
		if d, err := strconv.ParseFloat(matches[1], 64); err == nil {
			info.Duration = time.Duration(d * float64(time.Second))
		}
	}

	// Extract width
	widthRegex := regexp.MustCompile(`"width":\s*(\d+)`)
	if matches := widthRegex.FindStringSubmatch(string(output)); len(matches) > 1 {
		info.Width, _ = strconv.Atoi(matches[1])
	}

	// Extract height
	heightRegex := regexp.MustCompile(`"height":\s*(\d+)`)
	if matches := heightRegex.FindStringSubmatch(string(output)); len(matches) > 1 {
		info.Height, _ = strconv.Atoi(matches[1])
	}

	// Extract codec
	codecRegex := regexp.MustCompile(`"codec_name":\s*"([^"]+)"`)
	if matches := codecRegex.FindStringSubmatch(string(output)); len(matches) > 1 {
		info.Codec = matches[1]
	}

	// Extract bitrate
	bitrateRegex := regexp.MustCompile(`"bit_rate":\s*"(\d+)"`)
	if matches := bitrateRegex.FindStringSubmatch(string(output)); len(matches) > 1 {
		info.Bitrate, _ = strconv.ParseInt(matches[1], 10, 64)
	}

	return info, nil
}

// IsFFmpegAvailable checks if ffmpeg is installed and accessible
func IsFFmpegAvailable() bool {
	cmd := exec.Command("ffmpeg", "-version")
	return cmd.Run() == nil
}
