package effects

import (
	"moshr/internal/video"
)

type DatamoshEffect struct {
	mosher *video.Mosher
}

func NewDatamoshEffect() *DatamoshEffect {
	return &DatamoshEffect{
		mosher: video.NewMosher(),
	}
}

func (d *DatamoshEffect) Apply(inputPath, outputPath string, intensity float64) error {
	params := d.GenerateParams(intensity)
	return d.mosher.MoshVideo(inputPath, outputPath, params)
}

func (d *DatamoshEffect) GenerateParams(intensity float64) video.MoshParams {
	params := video.MoshParams{
		Intensity: intensity,
	}

	// Classic datamoshing: Heavy I-frame removal for smooth motion trails
	if intensity > 0.1 {
		params.IFrameRemoval = true
	}

	// Conservative P-frame duplication for fluid motion blur
	if intensity > 0.05 {
		params.PFrameDuplication = true
		// Much higher duplication count for smooth, flowing effects
		params.DuplicationCount = int(intensity*60) + 20 // 20-80 duplications
	}

	return params
}

func (d *DatamoshEffect) CreatePresets() []video.MoshParams {
	return []video.MoshParams{
		{
			Intensity:         0.4,
			IFrameRemoval:     true,
			PFrameDuplication: true,
			DuplicationCount:  44, // Smooth flow
		},
		{
			Intensity:         0.7,
			IFrameRemoval:     true,
			PFrameDuplication: true,
			DuplicationCount:  62, // Heavy motion blur
		},
		{
			Intensity:         1.0,
			IFrameRemoval:     true,
			PFrameDuplication: true,
			DuplicationCount:  80, // Maximum flow effect
		},
	}
}
