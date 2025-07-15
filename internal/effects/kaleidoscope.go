package effects

import (
	"fmt"
	"math"
	"math/rand"
	"os/exec"
	"strings"
	"time"
)

type KaleidoscopeEffect struct {
	rng *rand.Rand
}

func NewKaleidoscopeEffect() *KaleidoscopeEffect {
	return &KaleidoscopeEffect{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (k *KaleidoscopeEffect) Apply(inputPath, outputPath string, intensity float64) error {
	// Number of kaleidoscope segments
	numSegments := int(4 + intensity*4) // 4-8 segments
	if numSegments > 8 {
		numSegments = 8
	}

	// Calculate rotation angles for each segment
	angleStep := 360.0 / float64(numSegments)

	// Fragment parameters
	fragmentSize := 0.3 + intensity*0.4 // How much of the original to use as fragment
	zoom := 1.0 + intensity*1.5         // Zoom level for fragments
	rotation := intensity * 180         // Random rotation amount

	var filterParts []string
	var overlayChain []string

	// Create base video stream
	filterParts = append(filterParts, fmt.Sprintf("[0:v]split=%d[base]", numSegments+1))

	// Generate stream labels
	streamLabels := []string{}
	for i := 0; i < numSegments; i++ {
		streamLabels = append(streamLabels, fmt.Sprintf("[seg%d_base]", i))
	}
	filterParts[0] += strings.Join(streamLabels, "")

	// Create each kaleidoscope segment
	for i := 0; i < numSegments; i++ {
		_ = float64(i) * angleStep // angle for future use

		// Random parameters for each segment
		segmentRotation := rotation + k.rng.Float64()*60 - 30 // ±30 degrees variation
		segmentZoom := zoom + k.rng.Float64()*0.5 - 0.25      // ±0.25 zoom variation

		// Random crop position for fragment source
		cropX := k.rng.Float64() * (1.0 - fragmentSize)
		cropY := k.rng.Float64() * (1.0 - fragmentSize)

		// Color shift for each segment
		hueShift := float64(i * 45) // 45 degree hue shifts
		satBoost := 1.0 + intensity*0.5

		// Time delay for some segments (creates temporal kaleidoscope)
		timeDelay := ""
		if i%2 == 1 && intensity > 1.5 {
			timeDelay = "tblend=all_mode=average:all_opacity=0.6,"
		}

		// Create segment filter
		segmentFilter := fmt.Sprintf(
			"[seg%d_base]"+
				"%s"+ // Optional time delay
				"crop=iw*%.3f:ih*%.3f:iw*%.3f:ih*%.3f,"+ // Crop fragment
				"scale=iw*%.2f:ih*%.2f,"+ // Scale fragment
				"rotate=%.2f*PI/180:fillcolor=black:ow=rotw(%.2f*PI/180):oh=roth(%.2f*PI/180),"+ // Rotate
				"hue=h=%.1f:s=%.2f,"+ // Color shift
				"pad=iw*2:ih*2:iw*0.5:ih*0.5[seg%d]", // Pad for positioning
			i,
			timeDelay,
			fragmentSize, fragmentSize, cropX, cropY,
			segmentZoom, segmentZoom,
			segmentRotation, segmentRotation, segmentRotation,
			hueShift, satBoost,
			i,
		)
		filterParts = append(filterParts, segmentFilter)
	}

	// Create black canvas
	canvasSize := int(512 + intensity*256) // Larger canvas for higher intensities
	canvasFilter := fmt.Sprintf(
		"[base]scale=%d:%d,drawbox=color=black:width=%d:height=%d:t=fill[canvas]",
		canvasSize, canvasSize, canvasSize, canvasSize,
	)
	filterParts = append(filterParts, canvasFilter)

	// Calculate positions for each segment in a circular pattern
	centerX := canvasSize / 2
	centerY := canvasSize / 2
	radius := int(float64(canvasSize) * 0.3) // Distance from center

	// Build overlay chain
	currentLayer := "canvas"
	for i := 0; i < numSegments; i++ {
		angle := float64(i) * angleStep * math.Pi / 180
		posX := centerX + int(float64(radius)*math.Cos(angle))
		posY := centerY + int(float64(radius)*math.Sin(angle))

		nextLayer := fmt.Sprintf("comp%d", i)
		if i == numSegments-1 {
			nextLayer = "kaleidoscope"
		}

		// Add blend mode for more interesting composition (for future use)
		_ = "overlay" // blendMode for future implementation

		overlayFilter := fmt.Sprintf(
			"[%s][seg%d]overlay=%d:%d:format=auto[%s]",
			currentLayer, i, posX-canvasSize/4, posY-canvasSize/4, nextLayer,
		)
		overlayChain = append(overlayChain, overlayFilter)
		currentLayer = nextLayer
	}

	// Add final effects
	finalFilter := fmt.Sprintf(
		"[kaleidoscope]"+
			"eq=saturation=%.2f:contrast=%.2f:brightness=%.2f,"+ // Color adjustments
			"gblur=sigma=%.1f,"+ // Slight blur for smoothness
			"unsharp=luma_msize_x=5:luma_msize_y=5:luma_amount=%.2f", // Sharpening
		1.0+intensity*0.4, // Saturation
		1.0+intensity*0.2, // Contrast
		-0.1*intensity,    // Brightness
		intensity*0.5,     // Blur amount
		0.5+intensity*0.3, // Sharpening
	)
	overlayChain = append(overlayChain, finalFilter)

	// Combine all parts
	allParts := append(filterParts, overlayChain...)
	filterComplex := strings.Join(allParts, ";")

	cmd := exec.Command("ffmpeg", "-i", inputPath, "-filter_complex", filterComplex, "-map", "0:a?", "-c:a", "copy", "-r", "30", "-y", outputPath)
	return cmd.Run()
}

func (k *KaleidoscopeEffect) CreatePresets() []KaleidoscopeParams {
	return []KaleidoscopeParams{
		{
			Intensity:    1.0,
			NumSegments:  6,
			FragmentSize: 0.4,
			Zoom:         1.5,
			Rotation:     90,
			TimeDelay:    false,
			BlurAmount:   1.0,
		},
		{
			Intensity:    2.0,
			NumSegments:  8,
			FragmentSize: 0.5,
			Zoom:         2.0,
			Rotation:     180,
			TimeDelay:    true,
			BlurAmount:   2.0,
		},
		{
			Intensity:    3.0,
			NumSegments:  8,
			FragmentSize: 0.7,
			Zoom:         2.5,
			Rotation:     270,
			TimeDelay:    true,
			BlurAmount:   3.0,
		},
	}
}

type KaleidoscopeParams struct {
	Intensity    float64 `json:"intensity"`
	NumSegments  int     `json:"num_segments"`
	FragmentSize float64 `json:"fragment_size"`
	Zoom         float64 `json:"zoom"`
	Rotation     float64 `json:"rotation"`
	TimeDelay    bool    `json:"time_delay"`
	BlurAmount   float64 `json:"blur_amount"`
}
