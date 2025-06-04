package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type FrameExtractor struct{}

type FrameRange struct {
	StartFrame int     `json:"start_frame"`
	EndFrame   int     `json:"end_frame"`
	StartTime  float64 `json:"start_time"`
	EndTime    float64 `json:"end_time"`
}

type FrameInfo struct {
	FrameNumber int     `json:"frame_number"`
	Timestamp   float64 `json:"timestamp"`
	ThumbnailPath string `json:"thumbnail_path"`
}

func NewFrameExtractor() *FrameExtractor {
	return &FrameExtractor{}
}

func (fe *FrameExtractor) ExtractClip(inputPath, outputPath string, frameRange FrameRange, framerate float64) error {
	startTime := float64(frameRange.StartFrame) / framerate
	endTime := float64(frameRange.EndFrame) / framerate
	duration := endTime - startTime

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-ss", fmt.Sprintf("%.3f", startTime),
		"-t", fmt.Sprintf("%.3f", duration),
		"-c:v", "libxvid",
		"-c:a", "pcm_s16le",
		"-f", "avi",
		outputPath,
		"-y")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("clip extraction failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}

func (fe *FrameExtractor) GenerateTimeline(inputPath, outputDir string, interval int) ([]FrameInfo, error) {
	os.MkdirAll(outputDir, 0755)

	analyzer := NewAnalyzer()
	info, err := analyzer.AnalyzeVideo(inputPath)
	if err != nil {
		return nil, err
	}

	totalFrames := int(info.Duration * info.Framerate)
	var frames []FrameInfo

	for frameNum := 0; frameNum < totalFrames; frameNum += interval {
		timestamp := float64(frameNum) / info.Framerate
		thumbnailPath := filepath.Join(outputDir, fmt.Sprintf("frame_%06d.jpg", frameNum))

		err := fe.extractFrame(inputPath, thumbnailPath, timestamp)
		if err != nil {
			continue
		}

		frames = append(frames, FrameInfo{
			FrameNumber:   frameNum,
			Timestamp:     timestamp,
			ThumbnailPath: thumbnailPath,
		})
	}

	return frames, nil
}

func (fe *FrameExtractor) extractFrame(inputPath, outputPath string, timestamp float64) error {
	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-ss", fmt.Sprintf("%.3f", timestamp),
		"-frames:v", "1",
		"-q:v", "5",
		"-vf", "scale=160:90",
		outputPath,
		"-y")

	return cmd.Run()
}

func (fe *FrameExtractor) GenerateKeyFrameThumbnails(inputPath, outputDir string) ([]FrameInfo, error) {
	os.MkdirAll(outputDir, 0755)

	keyFrames, err := fe.detectKeyFrames(inputPath)
	if err != nil {
		return nil, err
	}

	analyzer := NewAnalyzer()
	info, err := analyzer.AnalyzeVideo(inputPath)
	if err != nil {
		return nil, err
	}

	var frames []FrameInfo
	for i, frameNum := range keyFrames {
		timestamp := float64(frameNum) / info.Framerate
		thumbnailPath := filepath.Join(outputDir, fmt.Sprintf("keyframe_%06d.jpg", i))

		err := fe.extractFrame(inputPath, thumbnailPath, timestamp)
		if err != nil {
			continue
		}

		frames = append(frames, FrameInfo{
			FrameNumber:   frameNum,
			Timestamp:     timestamp,
			ThumbnailPath: thumbnailPath,
		})
	}

	return frames, nil
}

func (fe *FrameExtractor) detectKeyFrames(inputPath string) ([]int, error) {
	cmd := exec.Command("ffprobe",
		"-select_streams", "v:0",
		"-show_entries", "packet=pos,flags",
		"-of", "csv=p=0",
		"-v", "quiet",
		inputPath)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return fe.parseKeyFrames(string(output))
}

func (fe *FrameExtractor) parseKeyFrames(output string) ([]int, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var keyFrames []int
	frameNum := 0

	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) >= 2 && strings.Contains(parts[1], "K") {
			keyFrames = append(keyFrames, frameNum)
		}
		frameNum++
	}

	return keyFrames, nil
}

func (fe *FrameExtractor) CreatePreviewMontage(inputPath, outputPath string, cols, rows int) error {
	analyzer := NewAnalyzer()
	info, err := analyzer.AnalyzeVideo(inputPath)
	if err != nil {
		return err
	}

	totalFrames := cols * rows
	interval := info.Duration / float64(totalFrames)

	tempDir := "temp_montage"
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	var inputPaths []string
	for i := 0; i < totalFrames; i++ {
		timestamp := float64(i) * interval
		framePath := filepath.Join(tempDir, fmt.Sprintf("frame_%03d.jpg", i))
		
		err := fe.extractFrame(inputPath, framePath, timestamp)
		if err == nil {
			inputPaths = append(inputPaths, framePath)
		}
	}

	args := []string{"-i", inputPath}
	filterComplex := fmt.Sprintf("tile=%dx%d", cols, rows)
	
	for i, path := range inputPaths {
		args = append(args, "-i", path)
		if i > 0 {
			filterComplex = fmt.Sprintf("[%d:v]", i+1) + filterComplex
		}
	}

	args = append(args, "-filter_complex", filterComplex, "-q:v", "3", outputPath, "-y")
	
	cmd := exec.Command("ffmpeg", args...)
	return cmd.Run()
}

func (fe *FrameExtractor) GetFrameAtTime(inputPath string, timestamp float64) (string, error) {
	tempPath := fmt.Sprintf("temp_frame_%.3f.jpg", timestamp)
	
	err := fe.extractFrame(inputPath, tempPath, timestamp)
	if err != nil {
		return "", err
	}

	return tempPath, nil
}