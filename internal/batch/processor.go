package batch

import (
	"fmt"
	"path/filepath"
	"sync"

	"moshr/internal/effects"
	"moshr/internal/video"
)

type Job struct {
	ID          string           `json:"id"`
	InputPath   string           `json:"input_path"`
	OutputDir   string           `json:"output_dir"`
	Effect      string           `json:"effect"`
	Params      video.MoshParams `json:"params"`
	Status      string           `json:"status"`
	Progress    float64          `json:"progress"`
	Error       string           `json:"error,omitempty"`
}

type WSHubInterface interface {
	BroadcastJobUpdate(jobID, status string, progress float64)
}

type ConverterInterface interface {
	GeneratePreview(inputPath, outputPath string, width, height int) error
}

type BatchProcessor struct {
	jobs      map[string]*Job
	jobsMu    sync.RWMutex
	workers   int
	queue     chan *Job
	wsHub     WSHubInterface
	converter ConverterInterface
}

func NewBatchProcessor(workers int, wsHub WSHubInterface, converter ConverterInterface) *BatchProcessor {
	return &BatchProcessor{
		jobs:      make(map[string]*Job),
		workers:   workers,
		queue:     make(chan *Job, 100),
		wsHub:     wsHub,
		converter: converter,
	}
}

func (bp *BatchProcessor) Start() {
	for i := 0; i < bp.workers; i++ {
		go bp.worker()
	}
}

func (bp *BatchProcessor) AddJob(job *Job) {
	bp.jobsMu.Lock()
	job.Status = "queued"
	bp.jobs[job.ID] = job
	bp.jobsMu.Unlock()
	
	bp.queue <- job
}

func (bp *BatchProcessor) GetJob(id string) (*Job, bool) {
	bp.jobsMu.RLock()
	defer bp.jobsMu.RUnlock()
	job, exists := bp.jobs[id]
	return job, exists
}

func (bp *BatchProcessor) GetAllJobs() []*Job {
	bp.jobsMu.RLock()
	defer bp.jobsMu.RUnlock()
	
	jobs := make([]*Job, 0, len(bp.jobs))
	for _, job := range bp.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

func (bp *BatchProcessor) worker() {
	for job := range bp.queue {
		fmt.Printf("Processing job: %s\n", job.ID)
		bp.processJob(job)
	}
}

func (bp *BatchProcessor) processJob(job *Job) {
	fmt.Printf("Starting to process job %s with input: %s\n", job.ID, job.InputPath)
	bp.updateJob(job.ID, "processing", 0.1, "")

	outputPath := filepath.Join(job.OutputDir, fmt.Sprintf("moshed_%s.avi", job.ID))
	fmt.Printf("Output path: %s\n", outputPath)

	var err error
	switch job.Effect {
	case "datamosh":
		fmt.Printf("Using datamosh effect\n")
		effect := effects.NewDatamoshEffect()
		err = effect.Apply(job.InputPath, outputPath, job.Params.Intensity)
	case "glitch":
		fmt.Printf("Using glitch effect\n")
		effect := effects.NewGlitchEffect()
		err = effect.Apply(job.InputPath, outputPath, job.Params.Intensity)
	case "corruption":
		fmt.Printf("Using corruption effect\n")
		effect := effects.NewCorruptionEffect()
		err = effect.Apply(job.InputPath, outputPath, job.Params.Intensity)
	case "byte_corruption":
		fmt.Printf("Using byte corruption effect\n")
		effect := effects.NewCorruptionEffect()
		err = effect.ApplyByteCorruption(job.InputPath, outputPath, job.Params.Intensity)
	case "channel_shift":
		fmt.Printf("Using channel shift effect\n")
		effect := effects.NewCorruptionEffect()
		err = effect.ApplyChannelShift(job.InputPath, outputPath, job.Params.Intensity)
	case "pixel_sort":
		fmt.Printf("Using pixel sort effect\n")
		effect := effects.NewCorruptionEffect()
		err = effect.ApplyPixelSort(job.InputPath, outputPath, job.Params.Intensity)
	case "scanline_displace":
		fmt.Printf("Using scanline displacement effect\n")
		effect := effects.NewCorruptionEffect()
		err = effect.ApplyScanlineDisplace(job.InputPath, outputPath, job.Params.Intensity)
	default:
		fmt.Printf("Using default mosher\n")
		mosher := video.NewMosher()
		err = mosher.MoshVideo(job.InputPath, outputPath, job.Params)
	}

	if err != nil {
		fmt.Printf("Job %s failed: %v\n", job.ID, err)
		bp.updateJob(job.ID, "failed", 0, err.Error())
	} else {
		fmt.Printf("Job %s completed successfully, generating preview\n", job.ID)
		bp.updateJob(job.ID, "processing", 0.9, "Generating preview")
		
		// Generate preview in the same directory as the mosh file
		previewPath := filepath.Join(job.OutputDir, fmt.Sprintf("preview_%s.jpg", job.ID))
		if bp.converter != nil {
			previewErr := bp.converter.GeneratePreview(outputPath, previewPath, 300, 200)
			if previewErr != nil {
				fmt.Printf("Failed to generate preview for job %s: %v\n", job.ID, previewErr)
			} else {
				fmt.Printf("Preview generated for job %s at %s\n", job.ID, previewPath)
			}
		}
		
		bp.updateJob(job.ID, "completed", 1.0, "")
	}
}

func (bp *BatchProcessor) updateJob(id, status string, progress float64, errorMsg string) {
	bp.jobsMu.Lock()
	defer bp.jobsMu.Unlock()
	
	if job, exists := bp.jobs[id]; exists {
		job.Status = status
		job.Progress = progress
		job.Error = errorMsg
		
		// Broadcast update via WebSocket if hub is available
		if bp.wsHub != nil {
			bp.wsHub.BroadcastJobUpdate(id, status, progress)
		}
	}
}

func (bp *BatchProcessor) CreateBatchFromPresets(inputPath, outputDir string, presets []video.MoshParams) []string {
	var jobIDs []string
	
	for i, params := range presets {
		jobID := fmt.Sprintf("batch_%d_%d", len(bp.jobs), i)
		job := &Job{
			ID:        jobID,
			InputPath: inputPath,
			OutputDir: outputDir,
			Effect:    "datamosh",
			Params:    params,
		}
		
		bp.AddJob(job)
		jobIDs = append(jobIDs, jobID)
	}
	
	return jobIDs
}