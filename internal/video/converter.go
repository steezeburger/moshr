package video

import (
	"fmt"
	"os/exec"
	"path/filepath"
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