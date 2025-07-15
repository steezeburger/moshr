package effects

import (
	"fmt"
	"math/rand"
	"os/exec"
	"time"
)

type GlitchMosaicEffect struct {
	rng *rand.Rand
}

func NewGlitchMosaicEffect() *GlitchMosaicEffect {
	return &GlitchMosaicEffect{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (g *GlitchMosaicEffect) Apply(inputPath, outputPath string, intensity float64) error {
	// Simplified mosaic effect that's more reliable
	gridSize := int(4 + intensity*6) // 4x4 to 10x10 grid (reduced complexity)
	if gridSize > 10 {
		gridSize = 10
	}

	// Simpler approach: use scale and noise filters for mosaic effect
	blockSize := int(8 / (1 + intensity*2)) // Larger blocks at low intensity
	if blockSize < 1 {
		blockSize = 1
	}

	// Random parameters
	noiseAmount := intensity * 0.3
	scrambleIntensity := intensity * 50

	// Simple but effective mosaic filter
	filterComplex := fmt.Sprintf(
		"[0:v]"+
			"scale=iw/%d:ih/%d:flags=neighbor,"+        // Downscale to create blocks
			"noise=alls=%f:allf=t+u,"+                  // Add temporal noise
			"scale=iw*%d:ih*%d:flags=neighbor,"+        // Scale back up
			"hue=h=%d:s=%.2f,"+                         // Random color shift
			"crop=iw-mod(iw\\,%d):ih-mod(ih\\,%d),"+    // Clean up dimensions
			"unsharp=luma_msize_x=3:luma_msize_y=3:luma_amount=%.2f", // Sharpen for digital look
		blockSize, blockSize,
		noiseAmount,
		blockSize, blockSize,
		g.rng.Intn(60)-30, 1.0+intensity*0.5,
		gridSize, gridSize,
		scrambleIntensity,
	)

	cmd := exec.Command("ffmpeg", "-i", inputPath, "-filter_complex", filterComplex, "-map", "0:a?", "-c:a", "copy", "-r", "30", "-y", outputPath)
	return cmd.Run()
}

func (g *GlitchMosaicEffect) CreatePresets() []GlitchMosaicParams {
	return []GlitchMosaicParams{
		{
			Intensity:    1.0,
			GridSize:     6,
			TimeOffset:   10,
			ScrambleRate: 0.4,
			ColorShift:   true,
		},
		{
			Intensity:    2.0,
			GridSize:     10,
			TimeOffset:   20,
			ScrambleRate: 0.6,
			ColorShift:   true,
		},
		{
			Intensity:    3.0,
			GridSize:     16,
			TimeOffset:   30,
			ScrambleRate: 0.8,
			ColorShift:   true,
		},
	}
}

type GlitchMosaicParams struct {
	Intensity    float64 `json:"intensity"`
	GridSize     int     `json:"grid_size"`
	TimeOffset   int     `json:"time_offset"`
	ScrambleRate float64 `json:"scramble_rate"`
	ColorShift   bool    `json:"color_shift"`
}
