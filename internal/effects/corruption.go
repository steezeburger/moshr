package effects

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
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
	corruptionRate := intensity * 0.0001 // 0.01% corruption at max intensity
	
	for i := startPos; i < len(corruptedData); i++ {
		if c.rng.Float64() < corruptionRate {
			corruptedData[i] = byte(c.rng.Intn(256))
		}
	}

	return os.WriteFile(outputPath, corruptedData, 0644)
}

func (c *CorruptionEffect) applyChannelShift(inputPath, outputPath string, intensity float64) error {
	shiftAmount := int(intensity * 50) + 10 // 10-60 pixel shift
	
	// Much more aggressive channel separation
	filterComplex := fmt.Sprintf("split=3[r][g][b];"+
		"[r]lutrgb=g=0:b=0,translate=%d:0[r_shifted];"+
		"[g]lutrgb=r=0:b=0,translate=-%d:0[g_shifted];"+
		"[b]lutrgb=r=0:g=0,translate=0:%d[b_shifted];"+
		"[r_shifted][g_shifted]blend=all_mode=addition[rg];"+
		"[rg][b_shifted]blend=all_mode=addition", 
		shiftAmount, shiftAmount, shiftAmount/3)
	
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-filter_complex", filterComplex, "-y", outputPath)
	return cmd.Run()
}

func (c *CorruptionEffect) applyPixelSort(inputPath, outputPath string, intensity float64) error {
	// Simple but very visible distortion effect
	blockSize := int(intensity * 8) + 2 // 2-10 pixel blocks
	noise := intensity * 0.3 // 0-0.3 noise level
	
	filterComplex := fmt.Sprintf("noise=alls=%f:allf=t,"+
		"scale=iw/%d:ih/%d:flags=neighbor,"+
		"scale=iw*%d:ih*%d:flags=neighbor,"+
		"hue=s=%.1f", 
		noise, blockSize, blockSize, blockSize, blockSize, 1.0+intensity)
	
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-vf", filterComplex, "-y", outputPath)
	return cmd.Run()
}

func (c *CorruptionEffect) applyScanlineDisplace(inputPath, outputPath string, intensity float64) error {
	// Use more visible distortion effects
	strength := int(intensity * 20) + 5 // 5-25 strength
	
	filterComplex := fmt.Sprintf("split[a][b];"+
		"[a]crop=iw:ih/2:0:0,scale=iw+%d:ih,crop=iw-%d:ih:0:0[top];"+
		"[b]crop=iw:ih/2:0:ih/2,scale=iw-%d:ih,pad=iw+%d:ih:0:0[bottom];"+
		"[top][bottom]vstack", 
		strength, strength, strength, strength)
	
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-filter_complex", filterComplex, "-y", outputPath)
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