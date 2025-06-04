package effects

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type CorruptionEffect struct {
	rng *rand.Rand
}

func NewCorruptionEffect() *CorruptionEffect {
	return &CorruptionEffect{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (c *CorruptionEffect) Apply(inputPath, outputPath string, intensity float64) error {
	effect := c.rng.Intn(4)
	
	switch effect {
	case 0:
		return c.applyByteCorruption(inputPath, outputPath, intensity)
	case 1:
		return c.applyChannelShift(inputPath, outputPath, intensity)
	case 2:
		return c.applyPixelSort(inputPath, outputPath, intensity)
	case 3:
		return c.applyScanlineDisplace(inputPath, outputPath, intensity)
	default:
		return c.applyByteCorruption(inputPath, outputPath, intensity)
	}
}

func (c *CorruptionEffect) applyByteCorruption(inputPath, outputPath string, intensity float64) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %v", err)
	}

	corruptedData := make([]byte, len(data))
	copy(corruptedData, data)

	// Skip first 12 bytes (RIFF header) and corrupt random bytes
	startPos := 12
	corruptionRate := intensity * 0.001 // 0.1% corruption at max intensity
	
	for i := startPos; i < len(corruptedData); i++ {
		if c.rng.Float64() < corruptionRate {
			corruptedData[i] = byte(c.rng.Intn(256))
		}
	}

	return os.WriteFile(outputPath, corruptedData, 0644)
}

func (c *CorruptionEffect) applyChannelShift(inputPath, outputPath string, intensity float64) error {
	tempDir := filepath.Dir(outputPath)
	tempBase := filepath.Join(tempDir, "temp_channel_")
	
	// Extract frames
	extractCmd := exec.Command("ffmpeg", "-i", inputPath, "-y", tempBase+"%04d.png")
	if err := extractCmd.Run(); err != nil {
		return fmt.Errorf("failed to extract frames: %v", err)
	}
	
	// Process frames with channel shifting
	shiftAmount := int(intensity * 20) // Max 20 pixel shift
	
	// Use ffmpeg filter to create RGB channel separation effect
	filterComplex := fmt.Sprintf("[0:v]split=3[r][g][b];"+
		"[r]lutrgb=g=0:b=0,crop=iw-%d:ih:0:0[r_shifted];"+
		"[g]lutrgb=r=0:b=0,crop=iw-%d:ih:%d:0[g_shifted];"+
		"[b]lutrgb=r=0:g=0,crop=iw-%d:ih:%d:0[b_shifted];"+
		"[r_shifted][g_shifted]blend=all_mode=addition[rg];"+
		"[rg][b_shifted]blend=all_mode=addition", 
		shiftAmount, shiftAmount, shiftAmount/2, shiftAmount, shiftAmount)
	
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-vf", filterComplex, "-y", outputPath)
	return cmd.Run()
}

func (c *CorruptionEffect) applyPixelSort(inputPath, outputPath string, intensity float64) error {
	// Use ffmpeg with custom pixel sorting effect
	threshold := int(intensity * 100) // Sorting threshold
	
	filterComplex := fmt.Sprintf("format=rgb24,"+
		"geq=r='if(gte(r(X\\,Y)\\,%d)\\,r(X\\,Y)\\,r(X\\,mod(Y+%d\\,H)))':"+
		"g='if(gte(g(X\\,Y)\\,%d)\\,g(X\\,Y)\\,g(X\\,mod(Y+%d\\,H)))':"+
		"b='if(gte(b(X\\,Y)\\,%d)\\,b(X\\,Y)\\,b(X\\,mod(Y+%d\\,H)))'",
		threshold, c.rng.Intn(10)+1, threshold, c.rng.Intn(10)+1, threshold, c.rng.Intn(10)+1)
	
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-vf", filterComplex, "-y", outputPath)
	return cmd.Run()
}

func (c *CorruptionEffect) applyScanlineDisplace(inputPath, outputPath string, intensity float64) error {
	displacement := int(intensity * 50) // Max 50 pixel displacement
	
	// Create displacement map
	filterComplex := fmt.Sprintf("split[main][displace];"+
		"[displace]geq=r='sin(Y*0.1)*%d':g='cos(Y*0.1)*%d':b=0,"+
		"scale=iw:ih[dispmap];"+
		"[main][dispmap]displace", displacement, displacement)
	
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-vf", filterComplex, "-y", outputPath)
	return cmd.Run()
}

// Specific effect methods for direct access
func (c *CorruptionEffect) ApplyByteCorruption(inputPath, outputPath string, intensity float64) error {
	return c.applyByteCorruption(inputPath, outputPath, intensity)
}

func (c *CorruptionEffect) ApplyChannelShift(inputPath, outputPath string, intensity float64) error {
	return c.applyChannelShift(inputPath, outputPath, intensity)
}

func (c *CorruptionEffect) ApplyPixelSort(inputPath, outputPath string, intensity float64) error {
	return c.applyPixelSort(inputPath, outputPath, intensity)
}

func (c *CorruptionEffect) ApplyScanlineDisplace(inputPath, outputPath string, intensity float64) error {
	return c.applyScanlineDisplace(inputPath, outputPath, intensity)
}