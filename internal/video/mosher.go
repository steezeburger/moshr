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
	fmt.Printf("MOSH: Starting mosh of %s -> %s with params: %+v\n", inputPath, outputPath, params)
	
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %v", err)
	}
	
	fmt.Printf("MOSH: Read %d bytes from input file\n", len(data))
	
	// Check if it's actually an AVI file
	if len(data) < 12 {
		return fmt.Errorf("file too small to be a valid AVI file")
	}
	
	if string(data[0:4]) != "RIFF" {
		return fmt.Errorf("not a RIFF file (first 4 bytes: %v)", data[0:4])
	}
	
	if string(data[8:12]) != "AVI " {
		return fmt.Errorf("not an AVI file (bytes 8-12: %v)", data[8:12])
	}
	
	fmt.Printf("MOSH: Confirmed AVI file format\n")

	moshedData, err := m.processAVIData(data, params)
	if err != nil {
		return fmt.Errorf("failed to process video data: %v", err)
	}

	fmt.Printf("MOSH: Processed data, output size: %d bytes\n", len(moshedData))

	if err := os.WriteFile(outputPath, moshedData, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %v", err)
	}

	fmt.Printf("MOSH: Successfully wrote moshed file to %s\n", outputPath)
	return nil
}

func (m *Mosher) processAVIData(data []byte, params MoshParams) ([]byte, error) {
	var result bytes.Buffer
	
	removedCount := 0
	duplicatedCount := 0
	totalVideoChunks := 0
	
	// Write RIFF header first
	result.Write(data[0:12])
	
	// Process the AVI file structure properly - skip RIFF header
	chunkPos := 12 // Skip RIFF header (4 bytes ID + 4 bytes size + 4 bytes AVI)
	for chunkPos < len(data)-8 {
		if chunkPos+8 > len(data) {
			result.Write(data[chunkPos:])
			break
		}

		chunkID := string(data[chunkPos:chunkPos+4])
		chunkSize := int(data[chunkPos+4]) | int(data[chunkPos+5])<<8 | int(data[chunkPos+6])<<16 | int(data[chunkPos+7])<<24

		if chunkPos+8+chunkSize > len(data) {
			result.Write(data[chunkPos:])
			break
		}

		// Check if this is a LIST movi chunk
		if chunkID == "LIST" && chunkSize > 4 {
			listType := string(data[chunkPos+8:chunkPos+12])
			fmt.Printf("MOSH: Found LIST chunk, type='%s'\n", listType)
			if listType == "movi" {
				fmt.Printf("MOSH: Processing movi LIST chunk\n")
				// Write LIST header
				result.Write(data[chunkPos:chunkPos+12])
				
				// Process contents of movi chunk
				moviStart := chunkPos + 12
				moviEnd := chunkPos + 8 + chunkSize
				framePos := moviStart
				
				for framePos < moviEnd-8 {
					if framePos+8 > len(data) {
						result.Write(data[framePos:])
						break
					}
					
					frameID := string(data[framePos:framePos+4])
					frameSize := int(data[framePos+4]) | int(data[framePos+5])<<8 | int(data[framePos+6])<<16 | int(data[framePos+7])<<24
					
					if framePos+8+frameSize > len(data) {
						result.Write(data[framePos:])
						break
					}
					
					if m.isVideoChunk(frameID) {
						totalVideoChunks++
						
						if params.IFrameRemoval && (totalVideoChunks%2 == 0) {
							removedCount++
							framePos += 8 + frameSize
							if frameSize%2 == 1 {
								framePos++
							}
							continue
						}

						if params.PFrameDuplication && (totalVideoChunks%2 == 0) {
							for i := 0; i < params.DuplicationCount; i++ {
								if i%3 == 0 && frameSize > 20 {
									corruptedChunk := make([]byte, 8+frameSize)
									copy(corruptedChunk, data[framePos:framePos+8+frameSize])
									for j := 16; j < len(corruptedChunk)-4 && j < 50; j += 4 {
										corruptedChunk[j] = byte((int(corruptedChunk[j]) + 127) % 255)
									}
									result.Write(corruptedChunk)
								} else {
									result.Write(data[framePos:framePos+8+frameSize])
								}
								duplicatedCount++
							}
						} else {
							result.Write(data[framePos:framePos+8+frameSize])
						}
					} else {
						result.Write(data[framePos:framePos+8+frameSize])
					}

					framePos += 8 + frameSize
					if frameSize%2 == 1 {
						framePos++
					}
				}
				
				chunkPos = moviEnd
				if chunkSize%2 == 1 {
					chunkPos++
				}
				continue
			}
		}
		
		// Regular chunk processing for non-movi chunks
		result.Write(data[chunkPos:chunkPos+8+chunkSize])
		chunkPos += 8 + chunkSize
		if chunkSize%2 == 1 {
			chunkPos++
		}
	}

	fmt.Printf("MOSH DEBUG: Total video chunks: %d, Removed: %d, Duplicated: %d\n", totalVideoChunks, removedCount, duplicatedCount)
	return result.Bytes(), nil
}

func (m *Mosher) isVideoChunk(chunkID string) bool {
	return chunkID == "00dc" || chunkID == "01dc" || chunkID == "00db" || chunkID == "01db" ||
		   chunkID == "02dc" || chunkID == "03dc" || chunkID == "02db" || chunkID == "03db" ||
		   chunkID == "vids" || chunkID == "DIB " || chunkID == "RGB " || chunkID == "MJPG"
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