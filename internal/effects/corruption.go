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
	corruptionRate := intensity * 0.001 // 0.1% corruption at max intensity
	
	for i := startPos; i < len(corruptedData); i++ {
		if c.rng.Float64() < corruptionRate {
			corruptedData[i] = byte(c.rng.Intn(256))
		}
	}

	return os.WriteFile(outputPath, corruptedData, 0644)
}

func (c *CorruptionEffect) applyChannelShift(inputPath, outputPath string, intensity float64) error {
	shiftAmount := int(intensity * 10) + 2 // 2-12 pixel shift
	
	filterComplex := fmt.Sprintf("rgbashift=rh=%d:gh=-%d", shiftAmount, shiftAmount/2)
	
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-vf", filterComplex, "-y", outputPath)
	return cmd.Run()
}

func (c *CorruptionEffect) applyPixelSort(inputPath, outputPath string, intensity float64) error {
	// Use FFmpeg's shuffleplanes and geq for actual pixel manipulation
	threshold := int(intensity * 128) + 32 // 32-160 brightness threshold
	
	// Create a scrambling effect that sorts pixels based on brightness
	filterComplex := fmt.Sprintf("format=rgb24,geq="+
		"r='if(gte((r(X,Y)+g(X,Y)+b(X,Y))/3,%d),r(X+%d,Y),r(X,Y))':"+
		"g='if(gte((r(X,Y)+g(X,Y)+b(X,Y))/3,%d),g(X-%d,Y),g(X,Y))':"+
		"b='if(gte((r(X,Y)+g(X,Y)+b(X,Y))/3,%d),b(X,Y+%d),b(X,Y))'", 
		threshold, c.rng.Intn(20)+5, threshold, c.rng.Intn(20)+5, threshold, c.rng.Intn(10)+2)
	
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-vf", filterComplex, "-y", outputPath)
	return cmd.Run()
}

func (c *CorruptionEffect) applyScanlineDisplace(inputPath, outputPath string, intensity float64) error {
	displacement := int(intensity * 50) + 10 // 10-60 pixel displacement
	frequency := intensity * 0.1 + 0.05 // Wave frequency
	
	// Create actual scanline displacement using geq
	filterComplex := fmt.Sprintf("geq="+
		"r='r(X+%d*sin(Y*%f),Y)':"+
		"g='g(X-%d*cos(Y*%f),Y)':"+
		"b='b(X+%d*sin(Y*%f+1),Y)'", 
		displacement, frequency, displacement/2, frequency, displacement/3, frequency)
	
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