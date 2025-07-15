package effects

import (
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"time"
)

type EchoTrailEffect struct {
	rng *rand.Rand
}

func NewEchoTrailEffect() *EchoTrailEffect {
	return &EchoTrailEffect{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (e *EchoTrailEffect) Apply(inputPath, outputPath string, intensity float64) error {
	// Calculate number of trails based on intensity
	numTrails := int(3 + intensity*5) // 3-8 trails at max intensity
	if numTrails > 8 {
		numTrails = 8 // Cap at 8 for performance
	}

	// Generate trail parameters
	maxDelay := int(intensity * 15) // Up to 15 frame delays
	if maxDelay < 2 {
		maxDelay = 2
	}

	var filterParts []string
	var overlayChain string

	// Split video into multiple streams
	filterParts = append(filterParts, fmt.Sprintf("[0:v]split=%d", numTrails+1))

	// Generate stream labels
	streamLabels := []string{"[orig]"}
	for i := 0; i < numTrails; i++ {
		streamLabels = append(streamLabels, fmt.Sprintf("[trail%d_base]", i))
	}
	filterParts[0] += strings.Join(streamLabels, "")

	// Create trail effects for each stream
	for i := 0; i < numTrails; i++ {
		trailDelay := (i + 1) * (maxDelay / numTrails) // Distribute delays evenly
		if trailDelay == 0 {
			trailDelay = i + 1
		}

		// Calculate transparency (further trails are more transparent)
		alpha := 1.0 - (float64(i+1) / float64(numTrails+1))
		alpha = alpha * (0.3 + intensity*0.4) // Scale based on intensity

		// Random color shift for each trail
		hueShift := e.rng.Intn(360)       // Random hue
		saturation := 0.8 + intensity*0.5 // Boost saturation

		// Random offset for each trail
		offsetX := e.rng.Intn(int(intensity*20)) - int(intensity*10) // -10 to +10
		offsetY := e.rng.Intn(int(intensity*20)) - int(intensity*10) // -10 to +10

		trailFilter := fmt.Sprintf(
			"[trail%d_base]"+
				"tblend=all_mode=average:all_opacity=%.2f,"+ // Time blend for motion blur
				"hue=h=%d:s=%.2f,"+ // Color shift
				"format=rgba,"+
				"geq=a='%.2f*alpha(X,Y)':r=r:g=g:b=b,"+ // Apply transparency
				"pad=iw+%d:ih+%d:%d:%d[trail%d]",
			i, 0.7, // Blend opacity for motion blur
			hueShift, saturation,
			alpha,
			abs(offsetX)+10, abs(offsetY)+10, max(0, offsetX)+5, max(0, offsetY)+5,
			i,
		)
		filterParts = append(filterParts, trailFilter)
	}

	// Pad the original for consistency
	origPadding := fmt.Sprintf(
		"[orig]pad=iw+%d:ih+%d:%d:%d[orig_padded]",
		int(intensity*20)+20, int(intensity*20)+20, int(intensity*10)+10, int(intensity*10)+10,
	)
	filterParts = append(filterParts, origPadding)

	// Build overlay chain - start with original, then overlay each trail
	overlayChain = "[orig_padded]"
	for i := 0; i < numTrails; i++ {
		if i == 0 {
			overlayChain += fmt.Sprintf("[trail%d]overlay=0:0:format=auto", i)
		} else {
			overlayChain += fmt.Sprintf(",[trail%d]overlay=0:0:format=auto", i)
		}

		if i < numTrails-1 {
			overlayChain += fmt.Sprintf("[tmp%d];[tmp%d]", i, i)
		}
	}

	// Combine all filter parts
	filterComplex := strings.Join(filterParts, ";") + ";" + overlayChain

	cmd := exec.Command("ffmpeg", "-i", inputPath, "-filter_complex", filterComplex, "-map", "0:a?", "-c:a", "copy", "-r", "30", "-y", outputPath)
	return cmd.Run()
}

func (e *EchoTrailEffect) CreatePresets() []EchoTrailParams {
	return []EchoTrailParams{
		{
			Intensity:  1.0,
			NumTrails:  4,
			MaxDelay:   8,
			BaseAlpha:  0.6,
			BlurAmount: 2.0,
			ColorShift: true,
		},
		{
			Intensity:  2.0,
			NumTrails:  6,
			MaxDelay:   12,
			BaseAlpha:  0.8,
			BlurAmount: 3.0,
			ColorShift: true,
		},
		{
			Intensity:  3.0,
			NumTrails:  8,
			MaxDelay:   18,
			BaseAlpha:  0.9,
			BlurAmount: 4.0,
			ColorShift: true,
		},
	}
}

type EchoTrailParams struct {
	Intensity  float64 `json:"intensity"`
	NumTrails  int     `json:"num_trails"`
	MaxDelay   int     `json:"max_delay"`
	BaseAlpha  float64 `json:"base_alpha"`
	BlurAmount float64 `json:"blur_amount"`
	ColorShift bool    `json:"color_shift"`
}
