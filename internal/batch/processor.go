package batch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"moshr/internal/effects"
	"moshr/internal/video"
)

type Mosh struct {
	ID             string           `json:"id"`
	InputPath      string           `json:"input_path"`
	OutputDir      string           `json:"output_dir"`
	Effect         string           `json:"effect"`
	Params         video.MoshParams `json:"params"`
	Status         string           `json:"status"`
	Progress       float64          `json:"progress"`
	Error          string           `json:"error,omitempty"`
	ConvertedFiles map[string]bool  `json:"converted_files,omitempty"`
}

type WSHubInterface interface {
	BroadcastMoshUpdate(moshID, status string, progress float64)
}

type ConverterInterface interface {
	GeneratePreview(inputPath, outputPath string, width, height int) error
}

type BatchProcessor struct {
	moshes    map[string]*Mosh
	moshesMu  sync.RWMutex
	workers   int
	queue     chan *Mosh
	wsHub     WSHubInterface
	converter ConverterInterface
}

func NewBatchProcessor(workers int, wsHub WSHubInterface, converter ConverterInterface) *BatchProcessor {
	return &BatchProcessor{
		moshes:    make(map[string]*Mosh),
		workers:   workers,
		queue:     make(chan *Mosh, 100),
		wsHub:     wsHub,
		converter: converter,
	}
}

func (bp *BatchProcessor) Start() {
	for i := 0; i < bp.workers; i++ {
		go bp.worker()
	}
}

func (bp *BatchProcessor) AddMosh(mosh *Mosh) {
	bp.moshesMu.Lock()
	mosh.Status = "queued"
	bp.moshes[mosh.ID] = mosh
	bp.moshesMu.Unlock()

	bp.queue <- mosh
}

func (bp *BatchProcessor) GetMosh(id string) (*Mosh, bool) {
	bp.moshesMu.RLock()
	defer bp.moshesMu.RUnlock()
	mosh, exists := bp.moshes[id]
	return mosh, exists
}

func (bp *BatchProcessor) GetAllMoshes() []*Mosh {
	bp.moshesMu.RLock()
	defer bp.moshesMu.RUnlock()

	moshes := make([]*Mosh, 0, len(bp.moshes))
	for _, mosh := range bp.moshes {
		moshes = append(moshes, mosh)
	}
	return moshes
}

func (bp *BatchProcessor) worker() {
	for mosh := range bp.queue {
		fmt.Printf("Processing mosh: %s\n", mosh.ID)
		bp.processMosh(mosh)
	}
}

func (bp *BatchProcessor) processMosh(mosh *Mosh) {
	fmt.Printf("Starting to process mosh %s with input: %s\n", mosh.ID, mosh.InputPath)
	bp.updateMosh(mosh.ID, "processing", 0.1, "")

	outputPath := filepath.Join(mosh.OutputDir, fmt.Sprintf("moshed_%s.avi", mosh.ID))
	fmt.Printf("Output path: %s\n", outputPath)

	// Start file size monitoring
	done := make(chan bool)
	go bp.monitorFileSize(mosh.ID, outputPath, done)
	defer func() {
		done <- true
		close(done)
	}()

	var err error
	switch mosh.Effect {
	case "datamosh":
		fmt.Printf("Using datamosh effect\n")
		effect := effects.NewDatamoshEffect()
		err = effect.Apply(mosh.InputPath, outputPath, mosh.Params.Intensity)
	case "glitch":
		fmt.Printf("Using glitch effect\n")
		effect := effects.NewGlitchEffect()
		err = effect.Apply(mosh.InputPath, outputPath, mosh.Params.Intensity)
	case "corruption":
		fmt.Printf("Using corruption effect\n")
		effect := effects.NewCorruptionEffect()
		err = effect.Apply(mosh.InputPath, outputPath, mosh.Params.Intensity)
	case "byte_corruption":
		fmt.Printf("Using byte corruption effect\n")
		effect := effects.NewCorruptionEffect()
		err = effect.ApplyByteCorruption(mosh.InputPath, outputPath, mosh.Params.Intensity)
	case "channel_shift":
		fmt.Printf("Using channel shift effect\n")
		effect := effects.NewCorruptionEffect()
		err = effect.ApplyChannelShift(mosh.InputPath, outputPath, mosh.Params.Intensity)
	case "pixel_sort":
		fmt.Printf("Using pixel sort effect\n")
		effect := effects.NewCorruptionEffect()
		err = effect.ApplyPixelSort(mosh.InputPath, outputPath, mosh.Params.Intensity)
	case "scanline_displace":
		fmt.Printf("Using scanline displacement effect\n")
		effect := effects.NewCorruptionEffect()
		err = effect.ApplyScanlineDisplace(mosh.InputPath, outputPath, mosh.Params.Intensity)
	case "duallayer":
		fmt.Printf("Using dual layer effect\n")
		effect := effects.NewDualLayerEffect()
		err = effect.Apply(mosh.InputPath, outputPath, mosh.Params.Intensity)
	case "rgbdrift":
		fmt.Printf("Using RGB drift effect\n")
		effect := effects.NewRGBDriftEffect()
		err = effect.Apply(mosh.InputPath, outputPath, mosh.Params.Intensity)
	case "echotrail":
		fmt.Printf("Using echo trail effect\n")
		effect := effects.NewEchoTrailEffect()
		err = effect.Apply(mosh.InputPath, outputPath, mosh.Params.Intensity)
	case "glitchmosaic":
		fmt.Printf("Using glitch mosaic effect\n")
		effect := effects.NewGlitchMosaicEffect()
		err = effect.Apply(mosh.InputPath, outputPath, mosh.Params.Intensity)
	case "chromaticblur":
		fmt.Printf("Using chromatic blur effect\n")
		effect := effects.NewChromaticBlurEffect()
		err = effect.Apply(mosh.InputPath, outputPath, mosh.Params.Intensity)
	case "kaleidoscope":
		fmt.Printf("Using kaleidoscope effect\n")
		effect := effects.NewKaleidoscopeEffect()
		err = effect.Apply(mosh.InputPath, outputPath, mosh.Params.Intensity)
	default:
		fmt.Printf("Using default mosher\n")
		mosher := video.NewMosher()
		err = mosher.MoshVideo(mosh.InputPath, outputPath, mosh.Params)
	}

	if err != nil {
		fmt.Printf("Mosh %s failed: %v\n", mosh.ID, err)
		bp.updateMosh(mosh.ID, "failed", 0, err.Error())
	} else {
		fmt.Printf("Mosh %s completed successfully, generating preview\n", mosh.ID)
		bp.updateMosh(mosh.ID, "processing", 0.9, "Generating preview")

		// Generate preview in the same directory as the mosh file
		previewPath := filepath.Join(mosh.OutputDir, fmt.Sprintf("preview_%s.jpg", mosh.ID))
		if bp.converter != nil {
			previewErr := bp.converter.GeneratePreview(outputPath, previewPath, 300, 200)
			if previewErr != nil {
				fmt.Printf("Failed to generate preview for mosh %s: %v\n", mosh.ID, previewErr)
			} else {
				fmt.Printf("Preview generated for mosh %s at %s\n", mosh.ID, previewPath)
			}
		}

		bp.updateMosh(mosh.ID, "completed", 1.0, "")

		// Update session metadata with the correct effect
		bp.updateSessionMetadata(mosh)
	}
}

func (bp *BatchProcessor) updateMosh(id, status string, progress float64, errorMsg string) {
	bp.moshesMu.Lock()
	defer bp.moshesMu.Unlock()

	if mosh, exists := bp.moshes[id]; exists {
		mosh.Status = status
		mosh.Progress = progress
		mosh.Error = errorMsg

		// Broadcast update via WebSocket if hub is available
		if bp.wsHub != nil {
			bp.wsHub.BroadcastMoshUpdate(id, status, progress)
		}
	}
}

func (bp *BatchProcessor) monitorFileSize(moshID, outputPath string, done chan bool) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	lastSize := int64(0)

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			stat, err := os.Stat(outputPath)
			if err != nil {
				continue
			}

			currentSize := stat.Size()
			elapsed := time.Since(startTime).Seconds()

			// Calculate progress based on file size growth
			var progress float64
			if currentSize > lastSize {
				// File is growing - estimate progress based on time and size growth
				sizeIncrease := currentSize - lastSize
				if sizeIncrease > 0 {
					// Rough progress estimate: 0.2 + (0.6 * elapsed/estimated_total_time)
					progress = 0.2 + (0.6 * elapsed / 30.0) // Assume 30 second average processing time
					if progress > 0.8 {
						progress = 0.8 // Cap at 80% until actual completion
					}
				}
			} else if elapsed > 5 {
				// If no size change after 5 seconds, show some progress anyway
				progress = 0.3 + (elapsed/60.0)*0.4 // Slow progress over 60 seconds
				if progress > 0.7 {
					progress = 0.7
				}
			}

			if progress > 0.1 {
				bp.updateMosh(moshID, "processing", progress, fmt.Sprintf("Processing... (%.1f MB)", float64(currentSize)/1024/1024))
			}

			lastSize = currentSize
		}
	}
}

func (bp *BatchProcessor) CreateBatchFromPresets(inputPath, outputDir, effect string, presets []video.MoshParams) []string {
	var moshIDs []string

	for i, params := range presets {
		moshID := fmt.Sprintf("batch_%d", i)
		mosh := &Mosh{
			ID:        moshID,
			InputPath: inputPath,
			OutputDir: outputDir,
			Effect:    effect,
			Params:    params,
		}

		bp.AddMosh(mosh)
		moshIDs = append(moshIDs, moshID)
	}

	return moshIDs
}

func (bp *BatchProcessor) updateSessionMetadata(mosh *Mosh) {
	// Extract session ID from output directory
	sessionDir := mosh.OutputDir
	sessionFile := filepath.Join(sessionDir, "session.json")

	fmt.Printf("Updating session metadata for mosh %s with effect %s\n", mosh.ID, mosh.Effect)

	// Read existing session.json or create new one
	var session map[string]interface{}

	if data, err := os.ReadFile(sessionFile); err == nil {
		json.Unmarshal(data, &session)
	} else {
		// Create new session
		sessionID := filepath.Base(sessionDir)
		session = map[string]interface{}{
			"id":         sessionID,
			"name":       fmt.Sprintf("Session: %s", sessionID),
			"created_at": time.Now(),
			"source":     fmt.Sprintf("Created with %s effect", mosh.Effect),
			"moshes":     []interface{}{},
		}
	}

	// Add or update this mosh's metadata
	moshes, _ := session["moshes"].([]interface{})

	// Check if this mosh already exists in moshes
	found := false
	for i, moshInterface := range moshes {
		if existingMosh, ok := moshInterface.(map[string]interface{}); ok {
			if existingMosh["id"] == mosh.ID {
				// Update existing mosh with correct effect
				existingMosh["effect"] = mosh.Effect
				existingMosh["file_path"] = filepath.Join(sessionDir, fmt.Sprintf("moshed_%s.avi", mosh.ID))
				moshes[i] = existingMosh
				found = true
				break
			}
		}
	}

	if !found {
		// Add new mosh metadata
		newMosh := map[string]interface{}{
			"id":        mosh.ID,
			"effect":    mosh.Effect, // THE CORRECT FUCKING EFFECT
			"file_path": filepath.Join(sessionDir, fmt.Sprintf("moshed_%s.avi", mosh.ID)),
			"params": map[string]interface{}{
				"intensity": mosh.Params.Intensity,
			},
			"created_at": time.Now(),
		}
		moshes = append(moshes, newMosh)
	}

	session["moshes"] = moshes

	// Save updated session.json
	if data, err := json.MarshalIndent(session, "", "  "); err == nil {
		os.WriteFile(sessionFile, data, 0644)
		fmt.Printf("Session metadata updated with effect: %s\n", mosh.Effect)
	}
}
