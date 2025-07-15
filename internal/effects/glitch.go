package effects

import (
	"fmt"
	"math/rand"
	"moshr/internal/video"
	"os"
	"time"
)

type GlitchEffect struct {
	mosher     *video.Mosher
	corruption *CorruptionEffect
	rng        *rand.Rand
}

func NewGlitchEffect() *GlitchEffect {
	return &GlitchEffect{
		mosher:     video.NewMosher(),
		corruption: NewCorruptionEffect(),
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (g *GlitchEffect) Apply(inputPath, outputPath string, intensity float64) error {
	fmt.Printf("GLITCH: Starting enhanced glitch effect on %s -> %s with intensity %.2f\n", inputPath, outputPath, intensity)

	// Read the input file
	data, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Printf("GLITCH: Failed to read input file: %v - falling back to moshing\n", err)
		params := g.GenerateRandomParams(intensity)
		return g.mosher.MoshVideo(inputPath, outputPath, params)
	}

	fmt.Printf("GLITCH: Read %d bytes from input file\n", len(data))

	// Apply byte-level corruption for immediate visual difference
	corruptedData := g.applyAdvancedByteCorruption(data, intensity)

	// Write corrupted data to temp file
	tempPath := outputPath + ".temp"
	err = os.WriteFile(tempPath, corruptedData, 0644)
	if err != nil {
		fmt.Printf("GLITCH: Failed to write corrupted temp file: %v - falling back to moshing\n", err)
		params := g.GenerateRandomParams(intensity)
		return g.mosher.MoshVideo(inputPath, outputPath, params)
	}

	fmt.Printf("GLITCH: Wrote %d bytes of corrupted data to temp file\n", len(corruptedData))

	// Apply minimal moshing to the corrupted result for very subtle compound effects
	params := g.GenerateRandomParams(intensity * 0.1) // Minimal moshing on corrupted data
	fmt.Printf("GLITCH: Applying moshing to corrupted data with params: %+v\n", params)
	err = g.mosher.MoshVideo(tempPath, outputPath, params)

	// Cleanup temp file
	fmt.Printf("GLITCH: Cleaning up temp file %s\n", tempPath)
	os.Remove(tempPath)

	if err != nil {
		fmt.Printf("GLITCH: Final moshing step failed: %v\n", err)
	} else {
		fmt.Printf("GLITCH: Enhanced glitch effect completed successfully\n")
	}

	return err
}

// Advanced byte corruption that creates more visible artifacts
func (g *GlitchEffect) applyAdvancedByteCorruption(data []byte, intensity float64) []byte {
	corruptedData := make([]byte, len(data))
	copy(corruptedData, data)

	// Skip AVI header (first 12 bytes: RIFF + size + AVI)
	startPos := 12
	if len(data) < startPos {
		return corruptedData
	}

	// Extremely conservative corruption rate to avoid breaking files
	corruptionRate := intensity * 0.00005 // Up to 0.005% corruption at max intensity

	// Apply different types of corruption
	for i := startPos; i < len(corruptedData); i++ {
		if g.rng.Float64() < corruptionRate {
			// Favor gentler corruption types
			corruptionType := g.rng.Intn(10)
			switch {
			case corruptionType < 6:
				// Bit flip (most common - 60% chance)
				bitPos := g.rng.Intn(8)
				corruptedData[i] ^= (1 << bitPos)
			case corruptionType < 8:
				// Byte swap with nearby byte (20% chance)
				if i+1 < len(corruptedData) {
					corruptedData[i], corruptedData[i+1] = corruptedData[i+1], corruptedData[i]
				}
			case corruptionType < 9:
				// Light random change (10% chance) - small values only
				corruptedData[i] = byte(g.rng.Intn(32)) // 0-31 only
			default:
				// Rare full random byte (10% chance)
				corruptedData[i] = byte(g.rng.Intn(256))
			}
		}
	}

	// Only add chunk-level corruption at maximum intensity
	if intensity >= 1.0 {
		g.corruptVideoChunks(corruptedData, intensity)
	}

	return corruptedData
}

// Corrupt video chunks specifically for more visible glitch effects
func (g *GlitchEffect) corruptVideoChunks(data []byte, intensity float64) {
	// Look for video chunk identifiers and corrupt them extremely rarely
	chunkCorruptionRate := intensity * 0.005 // Up to 0.5% chunk corruption

	for i := 12; i < len(data)-8; i++ {
		// Look for common AVI chunk identifiers
		if i+4 < len(data) {
			chunk := string(data[i : i+4])
			if chunk == "00dc" || chunk == "01dc" || chunk == "00db" || chunk == "01db" {
				if g.rng.Float64() < chunkCorruptionRate {
					// Only do very light corruption - single bit flips instead of full byte changes
					if g.rng.Float64() < 0.7 {
						// Light bit flip in chunk data area (skip identifier and size)
						if i+8 < len(data) {
							dataPos := i + 8 + g.rng.Intn(16) // Corrupt in first 16 bytes of chunk data
							if dataPos < len(data) {
								bitPos := g.rng.Intn(8)
								data[dataPos] ^= (1 << bitPos)
							}
						}
					} else {
						// Very rarely, do a single byte change in chunk data
						if i+8 < len(data) {
							dataPos := i + 8 + g.rng.Intn(8)
							if dataPos < len(data) {
								data[dataPos] = byte(g.rng.Intn(256))
							}
						}
					}
				}
			}
		}
	}
}

func (g *GlitchEffect) GenerateRandomParams(intensity float64) video.MoshParams {
	params := video.MoshParams{
		Intensity: intensity,
	}

	// Aggressive, chaotic approach - more likely to break things
	// Random I-frame removal for stuttering effect
	iframeChance := intensity * 1.2 // Higher chance than datamosh
	if g.rng.Float64() < iframeChance {
		params.IFrameRemoval = true
	}

	// Always duplicate P-frames for glitch effect
	params.PFrameDuplication = true

	// Highly variable, low duplication counts for stuttering/jittering
	if intensity < 0.3 {
		// Low intensity: micro-stutters (1-8 duplications)
		params.DuplicationCount = g.rng.Intn(8) + 1
	} else if intensity < 0.7 {
		// Medium intensity: choppy breaks (3-18 duplications)
		params.DuplicationCount = g.rng.Intn(16) + 3
	} else {
		// High intensity: extreme stutters (5-35 duplications)
		params.DuplicationCount = g.rng.Intn(31) + 5
	}

	return params
}

func (g *GlitchEffect) CreateRandomVariations(count int, intensity float64) []video.MoshParams {
	var variations []video.MoshParams

	for i := 0; i < count; i++ {
		// Each variation uses different random parameters for unpredictable results
		variations = append(variations, g.GenerateRandomParams(intensity))
	}

	return variations
}

// Create specific glitch presets that showcase different corruption+moshing combinations
func (g *GlitchEffect) CreatePresets() []video.MoshParams {
	return []video.MoshParams{
		{
			Intensity:         0.4, // Light corruption + minimal moshing
			IFrameRemoval:     false,
			PFrameDuplication: true,
			DuplicationCount:  3, // Subtle digital artifacts
		},
		{
			Intensity:         0.7, // Medium corruption + light moshing
			IFrameRemoval:     true,
			PFrameDuplication: true,
			DuplicationCount:  8, // Visible glitch effects
		},
		{
			Intensity:         1.0, // Heavy corruption + moderate moshing
			IFrameRemoval:     true,
			PFrameDuplication: true,
			DuplicationCount:  15, // Extreme digital chaos
		},
	}
}
