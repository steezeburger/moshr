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

	if intensity > 0.2 {
		params.IFrameRemoval = true
	}

	if intensity > 0.1 {
		params.PFrameDuplication = true
		params.DuplicationCount = int(intensity * 25) + 5
	}

	return params
}

func (d *DatamoshEffect) CreatePresets() []video.MoshParams {
	return []video.MoshParams{
		{
			Intensity:         0.5,
			IFrameRemoval:     true,
			PFrameDuplication: true,
			DuplicationCount:  15,
		},
		{
			Intensity:         0.8,
			IFrameRemoval:     true,
			PFrameDuplication: true,
			DuplicationCount:  25,
		},
		{
			Intensity:         1.0,
			IFrameRemoval:     true,
			PFrameDuplication: true,
			DuplicationCount:  40,
		},
	}
}