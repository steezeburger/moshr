package effects

import (
	"fmt"
	"math/rand"
	"os/exec"
	"time"
)

type ChromaticBlurEffect struct {
	rng *rand.Rand
}

func NewChromaticBlurEffect() *ChromaticBlurEffect {
	return &ChromaticBlurEffect{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (c *ChromaticBlurEffect) Apply(inputPath, outputPath string, intensity float64) error {
	// Generate random blur parameters for each channel
	redBlurX := 1 + int(intensity*15)   // 1-15 horizontal blur for red
	redBlurY := 1 + int(intensity*8)    // 1-8 vertical blur for red
	greenBlurX := 1 + int(intensity*12) // Different blur amounts for each channel
	greenBlurY := 1 + int(intensity*10)
	blueBlurX := 1 + int(intensity*20) // Blue gets the most blur
	blueBlurY := 1 + int(intensity*6)

	// Random blur directions for variety
	blurDirections := []string{"horizontal", "vertical", "diagonal", "radial"}
	redDirection := blurDirections[c.rng.Intn(len(blurDirections))]
	greenDirection := blurDirections[c.rng.Intn(len(blurDirections))]
	blueDirection := blurDirections[c.rng.Intn(len(blurDirections))]

	// Color saturation and contrast adjustments
	redSaturation := 1.0 + intensity*0.5
	greenSaturation := 1.0 + intensity*0.3
	blueSaturation := 1.0 + intensity*0.7

	var redBlurFilter, greenBlurFilter, blueBlurFilter string

	// Create direction-specific blur filters for each channel
	switch redDirection {
	case "horizontal":
		redBlurFilter = fmt.Sprintf("boxblur=%d:1", redBlurX)
	case "vertical":
		redBlurFilter = fmt.Sprintf("boxblur=1:%d", redBlurY)
	case "diagonal":
		redBlurFilter = fmt.Sprintf("boxblur=%d:%d", redBlurX/2, redBlurY/2)
	case "radial":
		redBlurFilter = fmt.Sprintf("gblur=sigma=%d", redBlurX)
	}

	switch greenDirection {
	case "horizontal":
		greenBlurFilter = fmt.Sprintf("boxblur=%d:1", greenBlurX)
	case "vertical":
		greenBlurFilter = fmt.Sprintf("boxblur=1:%d", greenBlurY)
	case "diagonal":
		greenBlurFilter = fmt.Sprintf("boxblur=%d:%d", greenBlurX/2, greenBlurY/2)
	case "radial":
		greenBlurFilter = fmt.Sprintf("gblur=sigma=%d", greenBlurX)
	}

	switch blueDirection {
	case "horizontal":
		blueBlurFilter = fmt.Sprintf("boxblur=%d:1", blueBlurX)
	case "vertical":
		blueBlurFilter = fmt.Sprintf("boxblur=1:%d", blueBlurY)
	case "diagonal":
		blueBlurFilter = fmt.Sprintf("boxblur=%d:%d", blueBlurX/2, blueBlurY/2)
	case "radial":
		blueBlurFilter = fmt.Sprintf("gblur=sigma=%d", blueBlurX)
	}

	// Add motion blur effect for more dreamy look
	motionBlur := ""
	if intensity > 1.5 {
		motionBlur = fmt.Sprintf(",tblend=all_mode=average:all_opacity=%.2f", 0.3+intensity*0.1)
	}

	// Create complex filter for chromatic blur effect
	filterComplex := fmt.Sprintf(
		"[0:v]split=3[r_base][g_base][b_base];"+
			// Red channel processing
			"[r_base]"+
			"lutrgb=g=0:b=0,"+ // Extract red channel
			"eq=saturation=%.2f:contrast=%.2f,"+ // Adjust saturation and contrast
			"%s%s[r_layer];"+ // Apply blur and optional motion blur
			// Green channel processing
			"[g_base]"+
			"lutrgb=r=0:b=0,"+ // Extract green channel
			"eq=saturation=%.2f:contrast=%.2f,"+
			"%s%s[g_layer];"+
			// Blue channel processing
			"[b_base]"+
			"lutrgb=r=0:g=0,"+ // Extract blue channel
			"eq=saturation=%.2f:contrast=%.2f,"+
			"%s%s[b_layer];"+
			// Composite channels back together with screen blend
			"[r_layer][g_layer]overlay=0:0:format=auto,"+
			"[b_layer]overlay=0:0:format=auto,"+
			// Final adjustments
			"eq=brightness=%.2f:contrast=%.2f:saturation=%.2f",
		// Red channel parameters
		redSaturation, 1.0+intensity*0.2,
		redBlurFilter, motionBlur,
		// Green channel parameters
		greenSaturation, 1.0+intensity*0.15,
		greenBlurFilter, motionBlur,
		// Blue channel parameters
		blueSaturation, 1.0+intensity*0.25,
		blueBlurFilter, motionBlur,
		// Final adjustments
		-0.05*intensity,   // Slight darkening
		1.0+intensity*0.1, // Contrast boost
		1.0+intensity*0.3, // Overall saturation boost
	)

	cmd := exec.Command("ffmpeg", "-i", inputPath, "-filter_complex", filterComplex, "-map", "0:a?", "-c:a", "copy", "-r", "30", "-y", outputPath)
	return cmd.Run()
}

func (c *ChromaticBlurEffect) CreatePresets() []ChromaticBlurParams {
	return []ChromaticBlurParams{
		{
			Intensity:       1.0,
			RedBlurRadius:   5,
			GreenBlurRadius: 3,
			BlueBlurRadius:  8,
			RedDirection:    "horizontal",
			GreenDirection:  "vertical",
			BlueDirection:   "radial",
			Saturation:      1.3,
			MotionBlur:      false,
		},
		{
			Intensity:       2.0,
			RedBlurRadius:   10,
			GreenBlurRadius: 7,
			BlueBlurRadius:  15,
			RedDirection:    "diagonal",
			GreenDirection:  "horizontal",
			BlueDirection:   "vertical",
			Saturation:      1.6,
			MotionBlur:      true,
		},
		{
			Intensity:       3.0,
			RedBlurRadius:   18,
			GreenBlurRadius: 12,
			BlueBlurRadius:  25,
			RedDirection:    "radial",
			GreenDirection:  "diagonal",
			BlueDirection:   "horizontal",
			Saturation:      2.0,
			MotionBlur:      true,
		},
	}
}

type ChromaticBlurParams struct {
	Intensity       float64 `json:"intensity"`
	RedBlurRadius   int     `json:"red_blur_radius"`
	GreenBlurRadius int     `json:"green_blur_radius"`
	BlueBlurRadius  int     `json:"blue_blur_radius"`
	RedDirection    string  `json:"red_direction"`
	GreenDirection  string  `json:"green_direction"`
	BlueDirection   string  `json:"blue_direction"`
	Saturation      float64 `json:"saturation"`
	MotionBlur      bool    `json:"motion_blur"`
}
