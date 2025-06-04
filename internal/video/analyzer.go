package video

import (
	"encoding/json"
	"strconv"
	"strings"
)

type VideoInfo struct {
	Duration    float64 `json:"duration"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	Bitrate     int     `json:"bitrate"`
	Framerate   float64 `json:"framerate"`
	Format      string  `json:"format"`
	VideoCodec  string  `json:"video_codec"`
	AudioCodec  string  `json:"audio_codec"`
}

type FFProbeFormat struct {
	Duration string `json:"duration"`
	Bitrate  string `json:"bit_rate"`
}

type FFProbeStream struct {
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	RFrameRate string `json:"r_frame_rate"`
}

type FFProbeOutput struct {
	Format  FFProbeFormat   `json:"format"`
	Streams []FFProbeStream `json:"streams"`
}

func parseVideoInfo(jsonOutput string) (*VideoInfo, error) {
	var probe FFProbeOutput
	if err := json.Unmarshal([]byte(jsonOutput), &probe); err != nil {
		return nil, err
	}

	info := &VideoInfo{}

	if probe.Format.Duration != "" {
		if duration, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
			info.Duration = duration
		}
	}

	if probe.Format.Bitrate != "" {
		if bitrate, err := strconv.Atoi(probe.Format.Bitrate); err == nil {
			info.Bitrate = bitrate
		}
	}

	for _, stream := range probe.Streams {
		if stream.CodecType == "video" {
			info.Width = stream.Width
			info.Height = stream.Height
			info.VideoCodec = stream.CodecName
			
			if stream.RFrameRate != "" {
				info.Framerate = parseFramerate(stream.RFrameRate)
			}
		} else if stream.CodecType == "audio" {
			info.AudioCodec = stream.CodecName
		}
	}

	return info, nil
}

func parseFramerate(rFrameRate string) float64 {
	parts := strings.Split(rFrameRate, "/")
	if len(parts) != 2 {
		return 0
	}
	
	num, err1 := strconv.ParseFloat(parts[0], 64)
	den, err2 := strconv.ParseFloat(parts[1], 64)
	
	if err1 != nil || err2 != nil || den == 0 {
		return 0
	}
	
	return num / den
}

type Analyzer struct{}

func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

func (a *Analyzer) AnalyzeVideo(path string) (*VideoInfo, error) {
	converter := NewConverter()
	return converter.GetVideoInfo(path)
}

func (a *Analyzer) EstimateFrameCount(info *VideoInfo) int {
	if info.Duration > 0 && info.Framerate > 0 {
		return int(info.Duration * info.Framerate)
	}
	return 0
}