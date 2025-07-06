package video

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Converter struct{}

func NewConverter() *Converter {
	return &Converter{}
}

func (c *Converter) MP4ToAVI(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-c:v", "libxvid", "-c:a", "libmp3lame", outputPath, "-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg conversion failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

func (c *Converter) GetVideoInfo(inputPath string) (*VideoInfo, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", inputPath)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %v", err)
	}

	return parseVideoInfo(string(output))
}

func (c *Converter) GeneratePreview(inputPath, outputPath string, width, height int) error {
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-vf", fmt.Sprintf("scale=%d:%d", width, height), "-frames:v", "1", outputPath, "-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("preview generation failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

func GetOutputPath(inputPath, suffix string) string {
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, ext)
	return fmt.Sprintf("%s_%s%s", base, suffix, ext)
}

func (c *Converter) MoshedAVIToMP4(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "18", // High quality
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart", // Web-optimized
		"-pix_fmt", "yuv420p", // Broad compatibility
		"-progress", "pipe:1", // Enable progress reporting
		outputPath,
		"-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("moshed AVI to MP4 conversion failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

func (c *Converter) MoshedAVIToWebM(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-c:v", "libvpx-vp9",
		"-crf", "30", // Good quality for VP9
		"-b:v", "0", // Variable bitrate
		"-c:a", "libvorbis",
		"-b:a", "128k",
		"-progress", "pipe:1", // Enable progress reporting
		outputPath,
		"-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("moshed AVI to WebM conversion failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

func (c *Converter) MoshedAVIToMP4WithProgress(inputPath, outputPath string, progressCallback func(float64)) error {
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "18",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-pix_fmt", "yuv420p",
		"-progress", "pipe:1",
		outputPath,
		"-y")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %v", err)
	}

	scanner := bufio.NewScanner(stdout)
	var totalTime float64

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_ms=") {
			timeStr := strings.TrimPrefix(line, "out_time_ms=")
			if timeMs, err := strconv.ParseFloat(timeStr, 64); err == nil {
				currentTime := timeMs / 1000000.0
				if totalTime == 0 {
					totalTime = currentTime * 2
				}
				if totalTime > 0 {
					progress := currentTime / totalTime
					if progress > 1.0 {
						progress = 1.0
					}
					progressCallback(progress)
				}
			}
		}
	}

	stderrBytes, _ := io.ReadAll(stderr)
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("moshed AVI to MP4 conversion failed: %v\nOutput: %s", err, string(stderrBytes))
	}

	return nil
}

func (c *Converter) MoshedAVIToWebMWithProgress(inputPath, outputPath string, progressCallback func(float64)) error {
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-c:v", "libvpx-vp9",
		"-crf", "30",
		"-b:v", "0",
		"-c:a", "libvorbis",
		"-b:a", "128k",
		"-progress", "pipe:1",
		outputPath,
		"-y")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %v", err)
	}

	scanner := bufio.NewScanner(stdout)
	var totalTime float64

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "out_time_ms=") {
			timeStr := strings.TrimPrefix(line, "out_time_ms=")
			if timeMs, err := strconv.ParseFloat(timeStr, 64); err == nil {
				currentTime := timeMs / 1000000.0
				if totalTime == 0 {
					totalTime = currentTime * 2
				}
				if totalTime > 0 {
					progress := currentTime / totalTime
					if progress > 1.0 {
						progress = 1.0
					}
					progressCallback(progress)
				}
			}
		}
	}

	stderrBytes, _ := io.ReadAll(stderr)
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("moshed AVI to WebM conversion failed: %v\nOutput: %s", err, string(stderrBytes))
	}

	return nil
}
