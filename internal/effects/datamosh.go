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
	params := d.generateParams(intensity)
	return d.mosher.MoshVideo(inputPath, outputPath, params)
}

func (d *DatamoshEffect) generateParams(intensity float64) video.MoshParams {
	params := video.MoshParams{
		Intensity: intensity,
	}

	if intensity > 0.3 {
		params.IFrameRemoval = true
	}

	if intensity > 0.5 {
		params.PFrameDuplication = true
		params.DuplicationCount = int(intensity * 5)
	}

	return params
}

func (d *DatamoshEffect) CreatePresets() []video.MoshParams {
	return []video.MoshParams{
		{
			Intensity:         0.2,
			IFrameRemoval:     false,
			PFrameDuplication: true,
			DuplicationCount:  1,
		},
		{
			Intensity:         0.5,
			IFrameRemoval:     true,
			PFrameDuplication: true,
			DuplicationCount:  2,
		},
		{
			Intensity:         0.8,
			IFrameRemoval:     true,
			PFrameDuplication: true,
			DuplicationCount:  4,
		},
	}
}