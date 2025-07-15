package effects

import (
	"fmt"
	"math/rand"
	"os/exec"
	"time"
)

type RGBDriftEffect struct {
	rng *rand.Rand
}

func NewRGBDriftEffect() *RGBDriftEffect {
	return &RGBDriftEffect{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *RGBDriftEffect) Apply(inputPath, outputPath string, intensity float64) error {
	// Generate random drift parameters for each channel
	redDriftX := r.rng.Intn(int(intensity*60)) - int(intensity*30)   // -30 to +30 at max intensity
	redDriftY := r.rng.Intn(int(intensity*40)) - int(intensity*20)   // -20 to +20 at max intensity
	greenDriftX := r.rng.Intn(int(intensity*60)) - int(intensity*30) // Independent drift for green
	greenDriftY := r.rng.Intn(int(intensity*40)) - int(intensity*20)
	blueDriftX := r.rng.Intn(int(intensity*60)) - int(intensity*30) // Independent drift for blue
	blueDriftY := r.rng.Intn(int(intensity*40)) - int(intensity*20)

	// Wave parameters for dynamic movement
	redWaveSpeed := 0.1 + (intensity * 0.3) // Wave speed multiplier
	greenWaveSpeed := 0.15 + (intensity * 0.25)
	blueWaveSpeed := 0.12 + (intensity * 0.35)

	waveAmplitude := int(intensity * 20) // How much the waves can move channels

	// Create complex filter for RGB channel separation with dynamic movement
	filterComplex := fmt.Sprintf(
		"[0:v]split=3[r_base][g_base][b_base];"+
			// Red channel with sine wave horizontal drift
			"[r_base]"+
			"lutrgb=g=0:b=0,"+ // Extract red channel only
			"pad=iw+%d:ih+%d:%d:%d,"+ // Pad for movement
			"geq="+
			"r='r(X-(%d+%d*sin(T*%f)),Y-(%d+%d*sin(T*%f+1)))':"+ // Dynamic red movement
			"g=0:b=0[r_layer];"+
			// Green channel with cosine wave vertical drift
			"[g_base]"+
			"lutrgb=r=0:b=0,"+ // Extract green channel only
			"pad=iw+%d:ih+%d:%d:%d,"+
			"geq="+
			"r=0:"+
			"g='g(X-(%d+%d*cos(T*%f)),Y-(%d+%d*cos(T*%f+2)))':"+ // Dynamic green movement
			"b=0[g_layer];"+
			// Blue channel with figure-8 pattern drift
			"[b_base]"+
			"lutrgb=r=0:g=0,"+ // Extract blue channel only
			"pad=iw+%d:ih+%d:%d:%d,"+
			"geq="+
			"r=0:g=0:"+
			"b='b(X-(%d+%d*sin(T*%f)*cos(T*%f)),Y-(%d+%d*cos(T*%f)*sin(T*%f+3)))'[b_layer];"+ // Figure-8 movement
			// Composite all channels back together
			"[r_layer][g_layer]overlay=0:0:format=auto,"+
			"[b_layer]overlay=0:0:format=auto",
		// Red channel parameters
		abs(redDriftX)+waveAmplitude*2, abs(redDriftY)+waveAmplitude*2,
		max(0, redDriftX)+waveAmplitude, max(0, redDriftY)+waveAmplitude,
		redDriftX, waveAmplitude, redWaveSpeed,
		redDriftY, waveAmplitude, redWaveSpeed,
		// Green channel parameters
		abs(greenDriftX)+waveAmplitude*2, abs(greenDriftY)+waveAmplitude*2,
		max(0, greenDriftX)+waveAmplitude, max(0, greenDriftY)+waveAmplitude,
		greenDriftX, waveAmplitude, greenWaveSpeed,
		greenDriftY, waveAmplitude, greenWaveSpeed,
		// Blue channel parameters
		abs(blueDriftX)+waveAmplitude*2, abs(blueDriftY)+waveAmplitude*2,
		max(0, blueDriftX)+waveAmplitude, max(0, blueDriftY)+waveAmplitude,
		blueDriftX, waveAmplitude, blueWaveSpeed, blueWaveSpeed,
		blueDriftY, waveAmplitude, blueWaveSpeed, blueWaveSpeed,
	)

	cmd := exec.Command("ffmpeg", "-i", inputPath, "-filter_complex", filterComplex, "-map", "0:a?", "-c:a", "copy", "-r", "30", "-y", outputPath)
	return cmd.Run()
}

func (r *RGBDriftEffect) CreatePresets() []RGBDriftParams {
	return []RGBDriftParams{
		{
			Intensity:      1.0,
			RedDriftX:      -15,
			RedDriftY:      10,
			GreenDriftX:    20,
			GreenDriftY:    -5,
			BlueDriftX:     -10,
			BlueDriftY:     15,
			WaveAmplitude:  10,
			RedWaveSpeed:   0.2,
			GreenWaveSpeed: 0.25,
			BlueWaveSpeed:  0.18,
		},
		{
			Intensity:      2.0,
			RedDriftX:      -30,
			RedDriftY:      20,
			GreenDriftX:    35,
			GreenDriftY:    -15,
			BlueDriftX:     -25,
			BlueDriftY:     30,
			WaveAmplitude:  25,
			RedWaveSpeed:   0.3,
			GreenWaveSpeed: 0.35,
			BlueWaveSpeed:  0.28,
		},
		{
			Intensity:      3.0,
			RedDriftX:      -50,
			RedDriftY:      35,
			GreenDriftX:    60,
			GreenDriftY:    -25,
			BlueDriftX:     -40,
			BlueDriftY:     50,
			WaveAmplitude:  40,
			RedWaveSpeed:   0.4,
			GreenWaveSpeed: 0.45,
			BlueWaveSpeed:  0.38,
		},
	}
}

type RGBDriftParams struct {
	Intensity      float64 `json:"intensity"`
	RedDriftX      int     `json:"red_drift_x"`
	RedDriftY      int     `json:"red_drift_y"`
	GreenDriftX    int     `json:"green_drift_x"`
	GreenDriftY    int     `json:"green_drift_y"`
	BlueDriftX     int     `json:"blue_drift_x"`
	BlueDriftY     int     `json:"blue_drift_y"`
	WaveAmplitude  int     `json:"wave_amplitude"`
	RedWaveSpeed   float64 `json:"red_wave_speed"`
	GreenWaveSpeed float64 `json:"green_wave_speed"`
	BlueWaveSpeed  float64 `json:"blue_wave_speed"`
}
