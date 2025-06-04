package video

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

type MoshParams struct {
	Intensity       float64 `json:"intensity"`
	IFrameRemoval   bool    `json:"iframe_removal"`
	PFrameDuplication bool  `json:"pframe_duplication"`
	DuplicationCount int   `json:"duplication_count"`
}

type Mosher struct{}

func NewMosher() *Mosher {
	return &Mosher{}
}

func (m *Mosher) MoshVideo(inputPath, outputPath string, params MoshParams) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %v", err)
	}

	moshedData, err := m.processAVIData(data, params)
	if err != nil {
		return fmt.Errorf("failed to process video data: %v", err)
	}

	if err := os.WriteFile(outputPath, moshedData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %v", err)
	}

	return nil
}

func (m *Mosher) processAVIData(data []byte, params MoshParams) ([]byte, error) {
	var result bytes.Buffer
	
	chunkPos := 0
	for chunkPos < len(data)-8 {
		if chunkPos+8 > len(data) {
			break
		}

		chunkID := string(data[chunkPos:chunkPos+4])
		chunkSize := int(data[chunkPos+4]) | int(data[chunkPos+5])<<8 | int(data[chunkPos+6])<<16 | int(data[chunkPos+7])<<24

		if chunkPos+8+chunkSize > len(data) {
			result.Write(data[chunkPos:])
			break
		}

		chunkData := data[chunkPos+8 : chunkPos+8+chunkSize]

		if m.isVideoChunk(chunkID) {
			if m.isIFrame(chunkData) && params.IFrameRemoval {
				chunkPos += 8 + chunkSize
				if chunkSize%2 == 1 {
					chunkPos++
				}
				continue
			}

			if m.isPFrame(chunkData) && params.PFrameDuplication {
				result.Write(data[chunkPos:chunkPos+8+chunkSize])
				
				for i := 0; i < params.DuplicationCount; i++ {
					result.Write(data[chunkPos:chunkPos+8+chunkSize])
				}
			} else {
				result.Write(data[chunkPos:chunkPos+8+chunkSize])
			}
		} else {
			result.Write(data[chunkPos:chunkPos+8+chunkSize])
		}

		chunkPos += 8 + chunkSize
		if chunkSize%2 == 1 {
			chunkPos++
		}
	}

	return result.Bytes(), nil
}

func (m *Mosher) isVideoChunk(chunkID string) bool {
	return chunkID == "00dc" || chunkID == "01dc" || chunkID == "00db" || chunkID == "01db"
}

func (m *Mosher) isIFrame(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	return data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x01 && (data[3]&0x1F) == 0x07
}

func (m *Mosher) isPFrame(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	return data[0] == 0x00 && data[1] == 0x00 && data[2] == 0x01 && (data[3]&0x1F) == 0x01
}

func (m *Mosher) CreateVariations(inputPath string, outputDir string, variations []MoshParams) ([]string, error) {
	var outputPaths []string
	
	for i, params := range variations {
		outputPath := filepath.Join(outputDir, fmt.Sprintf("mosh_variation_%d.avi", i+1))
		
		if err := m.MoshVideo(inputPath, outputPath, params); err != nil {
			return nil, fmt.Errorf("failed to create variation %d: %v", i+1, err)
		}
		
		outputPaths = append(outputPaths, outputPath)
	}
	
	return outputPaths, nil
}