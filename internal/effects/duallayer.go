package effects

import (
	"fmt"
	"math/rand"
	"os/exec"
	"time"
)

type DualLayerEffect struct {
	rng *rand.Rand
}

func NewDualLayerEffect() *DualLayerEffect {
	return &DualLayerEffect{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (d *DualLayerEffect) Apply(inputPath, outputPath string, intensity float64) error {
	// Generate much larger random offsets for both layers - now supports intensity up to 3.0
	maxOffset := int(intensity * 80) // Up to 240 pixels at max intensity

	greenOffsetX := d.rng.Intn(maxOffset*2) - maxOffset  // -240 to +240 at max intensity
	greenOffsetY := d.rng.Intn(maxOffset*2) - maxOffset  // -240 to +240 at max intensity
	purpleOffsetX := d.rng.Intn(maxOffset*2) - maxOffset // -240 to +240 at max intensity
	purpleOffsetY := d.rng.Intn(maxOffset*2) - maxOffset // -240 to +240 at max intensity

	// Ensure minimum offset at low intensities
	if greenOffsetX == 0 && greenOffsetY == 0 {
		greenOffsetX = d.rng.Intn(10) - 5 // -5 to +5
		greenOffsetY = d.rng.Intn(10) - 5 // -5 to +5
	}
	if purpleOffsetX == 0 && purpleOffsetY == 0 {
		purpleOffsetX = d.rng.Intn(10) - 5 // -5 to +5
		purpleOffsetY = d.rng.Intn(10) - 5 // -5 to +5
	}

	// Much more dramatic color shifts - can now go beyond 1.0 for extreme effects
	greenShift := 0.5 + (intensity * 1.5)  // 0.5 to 5.0 green boost
	purpleShift := 0.5 + (intensity * 1.5) // 0.5 to 5.0 purple boost

	// Create dramatic chromatic aberration effect with much stronger color separation
	// Calculate canvas size to accommodate all offsets
	canvasW := abs(greenOffsetX) + abs(purpleOffsetX) + 100 // Extra padding
	canvasH := abs(greenOffsetY) + abs(purpleOffsetY) + 100 // Extra padding

	// Calculate positions for each layer on the canvas
	origX := max(abs(greenOffsetX), abs(purpleOffsetX)) + 50
	origY := max(abs(greenOffsetY), abs(purpleOffsetY)) + 50
	greenX := origX + greenOffsetX
	greenY := origY + greenOffsetY
	purpleX := origX + purpleOffsetX
	purpleY := origY + purpleOffsetY

	filterComplex := fmt.Sprintf(
		"[0:v]split=3[orig][green_base][purple_base];"+
			// Green layer: subtle green tint with transparency
			"[green_base]"+
			"hue=h=120:s=%.1f,"+ // Green hue shift with saturation boost
			"colorbalance=gs=%.2f,"+ // Moderate green balance
			"format=rgba,colorkey=0x000000:0.3:0.1,"+ // Add transparency
			"pad=iw+%d:ih+%d:%d:%d[green_layer];"+
			// Purple layer: magenta/purple tint with transparency
			"[purple_base]"+
			"hue=h=300:s=%.1f,"+ // Purple/magenta hue shift
			"colorbalance=rs=%.2f:bs=%.2f,"+ // Red and blue balance for purple
			"format=rgba,colorkey=0x000000:0.3:0.1,"+ // Add transparency
			"pad=iw+%d:ih+%d:%d:%d[purple_layer];"+
			// Original layer stays mostly normal
			"[orig]"+
			"pad=iw+%d:ih+%d:%d:%d[orig_layer];"+
			// Composite with blend modes for better color mixing
			"[orig_layer][green_layer]overlay=%d:%d:format=auto,"+
			"[purple_layer]overlay=%d:%d:format=auto",
		// Green layer parameters
		1.0+intensity*0.5, // Moderate saturation boost
		greenShift*0.3,    // Gentle green balance
		canvasW, canvasH, greenX, greenY,
		// Purple layer parameters
		1.0+intensity*0.5,                // Moderate saturation boost
		purpleShift*0.3, purpleShift*0.4, // Red and blue for purple
		canvasW, canvasH, purpleX, purpleY,
		// Original layer
		canvasW, canvasH, origX, origY,
		// Overlay positions
		greenX, greenY,
		purpleX, purpleY,
	)

	cmd := exec.Command("ffmpeg", "-i", inputPath, "-filter_complex", filterComplex, "-map", "0:a?", "-c:a", "copy", "-r", "30", "-y", outputPath)
	return cmd.Run()
}

func (d *DualLayerEffect) CreatePresets() []DualLayerParams {
	return []DualLayerParams{
		{
			Intensity:     1.0,
			GreenOffsetX:  -30,
			GreenOffsetY:  20,
			PurpleOffsetX: 40,
			PurpleOffsetY: -15,
			GreenShift:    2.0,
			PurpleShift:   2.0,
		},
		{
			Intensity:     2.0,
			GreenOffsetX:  -80,
			GreenOffsetY:  60,
			PurpleOffsetX: 100,
			PurpleOffsetY: -40,
			GreenShift:    3.5,
			PurpleShift:   3.5,
		},
		{
			Intensity:     3.0,
			GreenOffsetX:  -150,
			GreenOffsetY:  120,
			PurpleOffsetX: 180,
			PurpleOffsetY: -90,
			GreenShift:    5.0,
			PurpleShift:   5.0,
		},
	}
}

func (d *DualLayerEffect) ApplyWithParams(inputPath, outputPath string, params DualLayerParams) error {
	// Apply effect with specific parameters instead of random generation
	filterComplex := fmt.Sprintf(
		"[0:v]split=3[orig][green_base][purple_base];"+
			"[green_base]"+
			"colorbalance=gs=%.2f:bs=-0.2,"+
			"pad=iw+%d:ih+%d:%d:%d[green_layer];"+
			"[purple_base]"+
			"colorbalance=rs=%.2f:bs=%.2f,"+
			"pad=iw+%d:ih+%d:%d:%d[purple_layer];"+
			"[orig]pad=iw+%d:ih+%d:%d:%d[orig_layer];"+
			"[orig_layer][green_layer]overlay=0:0:format=auto,"+
			"[purple_layer]overlay=0:0:format=auto",
		params.GreenShift,
		abs(params.GreenOffsetX)+abs(params.PurpleOffsetX), abs(params.GreenOffsetY)+abs(params.PurpleOffsetY),
		max(0, params.GreenOffsetX), max(0, params.GreenOffsetY),
		params.PurpleShift, params.PurpleShift,
		abs(params.GreenOffsetX)+abs(params.PurpleOffsetX), abs(params.GreenOffsetY)+abs(params.PurpleOffsetY),
		max(0, params.PurpleOffsetX), max(0, params.PurpleOffsetY),
		abs(params.GreenOffsetX)+abs(params.PurpleOffsetX), abs(params.GreenOffsetY)+abs(params.PurpleOffsetY),
		max(0, -params.GreenOffsetX-params.PurpleOffsetX), max(0, -params.GreenOffsetY-params.PurpleOffsetY),
	)

	cmd := exec.Command("ffmpeg", "-i", inputPath, "-filter_complex", filterComplex, "-map", "0:a?", "-c:a", "copy", "-r", "30", "-y", outputPath)
	return cmd.Run()
}

type DualLayerParams struct {
	Intensity     float64 `json:"intensity"`
	GreenOffsetX  int     `json:"green_offset_x"`
	GreenOffsetY  int     `json:"green_offset_y"`
	PurpleOffsetX int     `json:"purple_offset_x"`
	PurpleOffsetY int     `json:"purple_offset_y"`
	GreenShift    float64 `json:"green_shift"`
	PurpleShift   float64 `json:"purple_shift"`
}
