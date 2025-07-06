package video

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Scene struct {
	StartTime  float64 `json:"start_time"`
	EndTime    float64 `json:"end_time"`
	StartFrame int     `json:"start_frame"`
	EndFrame   int     `json:"end_frame"`
	Duration   float64 `json:"duration"`
	Type       string  `json:"type"`
}

type SceneDetector struct{}

func NewSceneDetector() *SceneDetector {
	return &SceneDetector{}
}

func (sd *SceneDetector) DetectScenes(inputPath string, threshold float64) ([]Scene, error) {
	if threshold == 0 {
		threshold = 0.3
	}

	cmd := exec.Command("ffprobe",
		"-f", "lavfi",
		"-i", fmt.Sprintf("movie=%s,select=gt(scene\\,%f)", inputPath, threshold),
		"-show_entries", "packet=pts_time",
		"-of", "csv=p=0",
		"-v", "quiet")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("scene detection failed: %v", err)
	}

	return sd.parseSceneOutput(string(output), inputPath)
}

func (sd *SceneDetector) parseSceneOutput(output, inputPath string) ([]Scene, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var scenes []Scene

	analyzer := NewAnalyzer()
	info, err := analyzer.AnalyzeVideo(inputPath)
	if err != nil {
		return nil, err
	}

	var lastTime float64 = 0
	sceneIndex := 0

	for _, line := range lines {
		if line == "" {
			continue
		}

		time, err := strconv.ParseFloat(strings.TrimSpace(line), 64)
		if err != nil {
			continue
		}

		if sceneIndex > 0 {
			scene := Scene{
				StartTime:  lastTime,
				EndTime:    time,
				StartFrame: int(lastTime * info.Framerate),
				EndFrame:   int(time * info.Framerate),
				Duration:   time - lastTime,
				Type:       fmt.Sprintf("scene_%d", sceneIndex),
			}
			scenes = append(scenes, scene)
		}

		lastTime = time
		sceneIndex++
	}

	if lastTime < info.Duration {
		scene := Scene{
			StartTime:  lastTime,
			EndTime:    info.Duration,
			StartFrame: int(lastTime * info.Framerate),
			EndFrame:   int(info.Duration * info.Framerate),
			Duration:   info.Duration - lastTime,
			Type:       fmt.Sprintf("scene_%d", sceneIndex),
		}
		scenes = append(scenes, scene)
	}

	return scenes, nil
}

func (sd *SceneDetector) DetectScenesAdvanced(inputPath string) ([]Scene, error) {
	cmd := exec.Command("ffprobe",
		"-i", inputPath,
		"-filter:v", "select='gt(scene,0.4)',showinfo",
		"-f", "null",
		"-",
		"-v", "info")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return sd.DetectScenes(inputPath, 0.3)
	}

	return sd.parseAdvancedSceneOutput(string(output), inputPath)
}

func (sd *SceneDetector) parseAdvancedSceneOutput(output, inputPath string) ([]Scene, error) {
	return sd.DetectScenes(inputPath, 0.3)
}

func (sd *SceneDetector) ClassifyScenes(inputPath string, scenes []Scene) ([]Scene, error) {
	for i := range scenes {
		sceneType, err := sd.classifyScene(inputPath, scenes[i])
		if err == nil {
			scenes[i].Type = sceneType
		}
	}
	return scenes, nil
}

func (sd *SceneDetector) classifyScene(inputPath string, scene Scene) (string, error) {
	thumbnailPath := filepath.Join("temp", fmt.Sprintf("scene_thumb_%d.jpg", scene.StartFrame))

	cmd := exec.Command("ffmpeg",
		"-i", inputPath,
		"-ss", fmt.Sprintf("%.2f", scene.StartTime),
		"-frames:v", "1",
		"-q:v", "2",
		thumbnailPath,
		"-y")

	err := cmd.Run()
	if err != nil {
		return "unknown", err
	}

	brightness := sd.analyzeBrightness(thumbnailPath)
	motion := sd.analyzeMotion(inputPath, scene)

	if brightness < 50 {
		return "dark_scene", nil
	} else if brightness > 200 {
		return "bright_scene", nil
	} else if motion > 0.8 {
		return "action_scene", nil
	} else if motion < 0.2 {
		return "static_scene", nil
	}

	return "normal_scene", nil
}

func (sd *SceneDetector) analyzeBrightness(imagePath string) float64 {
	cmd := exec.Command("ffprobe",
		"-f", "lavfi",
		"-i", fmt.Sprintf("movie=%s,signalstats", imagePath),
		"-show_entries", "frame=pkt_pts_time:frame_tags=lavfi.signalstats.YAVG",
		"-of", "csv=p=0",
		"-v", "quiet")

	output, err := cmd.Output()
	if err != nil {
		return 128
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 && len(strings.Split(lines[0], ",")) > 1 {
		if brightness, err := strconv.ParseFloat(strings.Split(lines[0], ",")[1], 64); err == nil {
			return brightness
		}
	}

	return 128
}

func (sd *SceneDetector) analyzeMotion(inputPath string, scene Scene) float64 {
	cmd := exec.Command("ffprobe",
		"-i", inputPath,
		"-ss", fmt.Sprintf("%.2f", scene.StartTime),
		"-t", fmt.Sprintf("%.2f", scene.Duration),
		"-filter:v", "select='gte(scene,0.1)',showinfo",
		"-f", "null",
		"-",
		"-v", "info")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0.5
	}

	motionLines := strings.Count(string(output), "scene_score")
	if scene.Duration > 0 {
		return float64(motionLines) / scene.Duration
	}

	return 0.5
}
