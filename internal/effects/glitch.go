package effects

import (
	"math/rand"
	"moshr/internal/video"
	"time"
)

type GlitchEffect struct {
	mosher *video.Mosher
	rng    *rand.Rand
}

func NewGlitchEffect() *GlitchEffect {
	return &GlitchEffect{
		mosher: video.NewMosher(),
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (g *GlitchEffect) Apply(inputPath, outputPath string, intensity float64) error {
	params := g.generateRandomParams(intensity)
	return g.mosher.MoshVideo(inputPath, outputPath, params)
}

func (g *GlitchEffect) generateRandomParams(intensity float64) video.MoshParams {
	params := video.MoshParams{
		Intensity: intensity,
	}

	iframeChance := intensity * 0.9
	if g.rng.Float64() < iframeChance {
		params.IFrameRemoval = true
	}

	params.PFrameDuplication = true
	params.DuplicationCount = g.rng.Intn(int(intensity*15)) + 5

	return params
}

func (g *GlitchEffect) CreateRandomVariations(count int, intensity float64) []video.MoshParams {
	var variations []video.MoshParams
	
	for i := 0; i < count; i++ {
		variations = append(variations, g.generateRandomParams(intensity))
	}
	
	return variations
}