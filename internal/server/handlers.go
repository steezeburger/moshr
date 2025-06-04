package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"moshr/internal/batch"
	"moshr/internal/effects"
	"moshr/internal/project"
	"moshr/internal/video"
)

type Server struct {
	processor      *batch.BatchProcessor
	converter      *video.Converter
	analyzer       *video.Analyzer
	sceneDetector  *video.SceneDetector
	frameExtractor *video.FrameExtractor
	projectManager *project.Manager
	wsHub          *WSHub
}

func NewServer() *Server {
	wsHub := NewWSHub()
	go wsHub.Run()
	
	processor := batch.NewBatchProcessor(2, wsHub)
	processor.Start()
	
	return &Server{
		processor:      processor,
		converter:      video.NewConverter(),
		analyzer:       video.NewAnalyzer(),
		sceneDetector:  video.NewSceneDetector(),
		frameExtractor: video.NewFrameExtractor(),
		projectManager: project.NewManager(),
		wsHub:          wsHub,
	}
}

func (s *Server) SetupRoutes() *gin.Engine {
	r := gin.Default()
	
	r.Static("/static", "web")
	r.Static("/timeline", "timeline")
	r.Static("/projects", "projects")
	r.StaticFile("/", "web/index.html")
	
	api := r.Group("/api")
	{
		api.GET("/projects", s.handleListProjects)
		api.POST("/projects", s.handleCreateProject)
		api.GET("/projects/:id", s.handleGetProject)
		api.POST("/projects/:id/scan", s.handleScanProject)
		
		api.POST("/projects/:id/upload", s.handleUpload)
		api.POST("/projects/:id/convert", s.handleConvert)
		api.POST("/projects/:id/mosh", s.handleMosh)
		api.GET("/projects/:id/jobs", s.handleGetJobs)
		api.GET("/projects/:id/jobs/:jobId", s.handleGetJob)
		api.GET("/projects/:id/preview/:filename", s.handlePreview)
		api.POST("/projects/:id/scenes", s.handleDetectScenes)
		api.POST("/projects/:id/timeline", s.handleGenerateTimeline)
		api.POST("/projects/:id/clip", s.handleExtractClip)
		api.GET("/projects/:id/frame/:filename/:timestamp", s.handleGetFrame)
		api.POST("/migrate", s.handleMigrateOldFiles)
	}
	
	return r
}

func (s *Server) handleListProjects(c *gin.Context) {
	projects, err := s.projectManager.ListProjects()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

func (s *Server) handleCreateProject(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	project, err := s.projectManager.CreateProject(req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"project": project})
}

func (s *Server) handleGetProject(c *gin.Context) {
	projectID := c.Param("id")
	
	project, err := s.projectManager.LoadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Scan for existing clips and moshes first
	s.scanExistingClips(projectID)
	s.scanExistingMoshes(projectID)
	
	clips, _ := s.projectManager.LoadClips(projectID)
	sessions, _ := s.projectManager.LoadMoshSessions(projectID)
	scenes, _ := s.projectManager.LoadScenes(projectID)

	c.JSON(http.StatusOK, gin.H{
		"project": project,
		"clips":   clips,
		"sessions": sessions,
		"scenes":  scenes,
	})
}

func (s *Server) handleScanProject(c *gin.Context) {
	projectID := c.Param("id")
	
	err := s.projectManager.ScanAndRecoverProject(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	project, err := s.projectManager.LoadProject(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"project": project,
		"message": "Project scanned and recovered successfully",
	})
}

func (s *Server) handleUpload(c *gin.Context) {
	projectID := c.Param("id")
	
	project, err := s.projectManager.LoadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	file, header, err := c.Request.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	filename := "original_" + header.Filename
	filePath := filepath.Join(project.BasePath, filename)

	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	project.OriginalFile = filePath
	err = s.projectManager.SaveProject(project)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	info, err := s.analyzer.AnalyzeVideo(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to analyze video"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"filename": filename,
		"path":     filePath,
		"info":     info,
		"project":  project,
	})
}

func (s *Server) handleConvert(c *gin.Context) {
	projectID := c.Param("id")
	
	project, err := s.projectManager.LoadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	if project.OriginalFile == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No original file to convert"})
		return
	}

	outputPath := filepath.Join(project.BasePath, "converted.avi")

	err = s.converter.MP4ToAVI(project.OriginalFile, outputPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	project.ConvertedFile = outputPath
	err = s.projectManager.SaveProject(project)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"output_path": outputPath,
		"project":     project,
	})
}

func (s *Server) handleMosh(c *gin.Context) {
	projectID := c.Param("id")
	
	_, err := s.projectManager.LoadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	var req struct {
		InputPath string  `json:"input_path"`
		Effect    string  `json:"effect"`
		Intensity float64 `json:"intensity"`
		Batch     bool    `json:"batch"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Create session directory in project's moshes folder
	sessionID := fmt.Sprintf("session_%d", time.Now().Unix())
	paths := s.projectManager.GetProjectPaths(projectID)
	sessionDir := filepath.Join(paths["moshes"], sessionID)
	err = os.MkdirAll(sessionDir, 0755)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session directory"})
		return
	}

	if req.Batch {
		var presets []video.MoshParams
		switch req.Effect {
		case "datamosh":
			effect := effects.NewDatamoshEffect()
			presets = effect.CreatePresets()
		case "glitch":
			effect := effects.NewGlitchEffect()
			presets = effect.CreateRandomVariations(3, req.Intensity)
		default:
			presets = []video.MoshParams{
				{Intensity: 0.3, IFrameRemoval: false, PFrameDuplication: true, DuplicationCount: 1},
				{Intensity: 0.6, IFrameRemoval: true, PFrameDuplication: true, DuplicationCount: 2},
				{Intensity: 0.9, IFrameRemoval: true, PFrameDuplication: true, DuplicationCount: 4},
			}
		}

		jobIDs := s.processor.CreateBatchFromPresets(req.InputPath, sessionDir, presets)
		c.JSON(http.StatusOK, gin.H{"job_ids": jobIDs, "session_id": sessionID})
	} else {
		jobID := fmt.Sprintf("single_%d", time.Now().Unix())
		job := &batch.Job{
			ID:        jobID,
			InputPath: req.InputPath,
			OutputDir: sessionDir,
			Effect:    req.Effect,
			Params: video.MoshParams{
				Intensity:         req.Intensity,
				IFrameRemoval:     req.Intensity > 0.5,
				PFrameDuplication: req.Intensity > 0.3,
				DuplicationCount:  int(req.Intensity * 5),
			},
		}

		s.processor.AddJob(job)
		c.JSON(http.StatusOK, gin.H{"job_id": jobID, "session_id": sessionID})
	}
}

func (s *Server) handleGetJobs(c *gin.Context) {
	jobs := s.processor.GetAllJobs()
	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

func (s *Server) handleGetJob(c *gin.Context) {
	jobID := c.Param("id")
	job, exists := s.processor.GetJob(jobID)
	
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"job": job})
}

func (s *Server) handlePreview(c *gin.Context) {
	projectID := c.Param("id")
	filename := c.Param("filename")
	width, _ := strconv.Atoi(c.DefaultQuery("width", "320"))
	height, _ := strconv.Atoi(c.DefaultQuery("height", "240"))

	// Look for the file in project moshes directory first
	paths := s.projectManager.GetProjectPaths(projectID)
	
	// Search for the file in all session directories
	var inputPath string
	moshesDir := paths["moshes"]
	
	entries, err := os.ReadDir(moshesDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				sessionPath := filepath.Join(moshesDir, entry.Name(), filename)
				if _, err := os.Stat(sessionPath); err == nil {
					inputPath = sessionPath
					break
				}
			}
		}
	}
	
	// Fallback to old output directory if not found in project
	if inputPath == "" {
		inputPath = filepath.Join("output", filename)
	}

	previewPath := filepath.Join("output", "preview_"+filename+".jpg")

	err = s.converter.GeneratePreview(inputPath, previewPath, width, height)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.File(previewPath)
}

func (s *Server) handleDetectScenes(c *gin.Context) {
	var req struct {
		InputPath string  `json:"input_path"`
		Threshold float64 `json:"threshold"`
		Advanced  bool    `json:"advanced"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var scenes []video.Scene
	var err error

	if req.Advanced {
		scenes, err = s.sceneDetector.DetectScenesAdvanced(req.InputPath)
	} else {
		scenes, err = s.sceneDetector.DetectScenes(req.InputPath, req.Threshold)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	scenes, _ = s.sceneDetector.ClassifyScenes(req.InputPath, scenes)

	c.JSON(http.StatusOK, gin.H{"scenes": scenes})
}

func (s *Server) handleGenerateTimeline(c *gin.Context) {
	projectID := c.Param("id")
	
	project, err := s.projectManager.LoadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	var req struct {
		Interval  int  `json:"interval"`
		KeyFrames bool `json:"keyframes_only"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Interval == 0 {
		req.Interval = 30
	}

	inputPath := project.OriginalFile
	if inputPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No video file in project"})
		return
	}

	paths := s.projectManager.GetProjectPaths(projectID)
	timelineDir := paths["timeline"]

	// First check if timeline already exists
	existingFrames, err := s.scanExistingTimeline(timelineDir, projectID)
	if err == nil && len(existingFrames) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"frames": existingFrames,
			"timeline_dir": timelineDir,
		})
		return
	}

	// Generate new timeline if none exists
	var frames []video.FrameInfo

	if req.KeyFrames {
		frames, err = s.frameExtractor.GenerateKeyFrameThumbnails(inputPath, timelineDir)
	} else {
		frames, err = s.frameExtractor.GenerateTimeline(inputPath, timelineDir, req.Interval)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"frames": frames,
		"timeline_dir": timelineDir,
	})
}

func (s *Server) handleExtractClip(c *gin.Context) {
	projectID := c.Param("id")
	
	project, err := s.projectManager.LoadProject(projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	var req struct {
		FrameRange video.FrameRange `json:"frame_range"`
		OutputName string           `json:"output_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	inputPath := project.OriginalFile
	if inputPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No video file in project"})
		return
	}

	info, err := s.analyzer.AnalyzeVideo(inputPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to analyze video"})
		return
	}

	if req.OutputName == "" {
		req.OutputName = fmt.Sprintf("clip_%d_%d.avi", req.FrameRange.StartFrame, req.FrameRange.EndFrame)
	}

	paths := s.projectManager.GetProjectPaths(projectID)
	outputPath := filepath.Join(paths["clips"], req.OutputName)
	
	err = s.frameExtractor.ExtractClip(inputPath, outputPath, req.FrameRange, info.Framerate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// TODO: Fix ClipMetadata type issue
	// clipMetadata := project.ClipMetadata{...}
	// clips, _ := s.projectManager.LoadClips(projectID)
	// clips = append(clips, clipMetadata)
	// s.projectManager.SaveClips(projectID, clips)

	c.JSON(http.StatusOK, gin.H{
		"output_path": outputPath,
		"clip_name":   req.OutputName,
	})
}

func (s *Server) handleGetFrame(c *gin.Context) {
	filename := c.Param("filename")
	timestampStr := c.Param("timestamp")
	
	timestamp, err := strconv.ParseFloat(timestampStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timestamp"})
		return
	}

	inputPath := filepath.Join("uploads", filename)
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		inputPath = filepath.Join("output", filename)
	}

	framePath, err := s.frameExtractor.GetFrameAtTime(inputPath, timestamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.File(framePath)
	
	go func() {
		time.Sleep(10 * time.Second)
		os.Remove(framePath)
	}()
}

func (s *Server) handleMigrateOldFiles(c *gin.Context) {
	uploadsDir := "uploads"
	outputDir := "output"
	
	// Check if old directories exist
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{"message": "No uploads directory to migrate"})
		return
	}

	// Find files in uploads
	uploadEntries, err := os.ReadDir(uploadsDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read uploads directory"})
		return
	}

	var migratedProjects []string

	for _, entry := range uploadEntries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		// Create project based on filename
		baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
		project, err := s.projectManager.CreateProject(baseName)
		if err != nil {
			continue
		}

		// Move original file
		oldPath := filepath.Join(uploadsDir, filename)
		newPath := filepath.Join(project.BasePath, "original_"+filename)
		err = os.Rename(oldPath, newPath)
		if err != nil {
			// Try copy if rename fails
			if err := s.copyFile(oldPath, newPath); err == nil {
				os.Remove(oldPath)
			}
		}

		project.OriginalFile = newPath

		// Look for converted file in output
		convertedName := strings.TrimSuffix(filename, filepath.Ext(filename)) + "_converted.avi"
		convertedOldPath := filepath.Join(outputDir, convertedName)
		if _, err := os.Stat(convertedOldPath); err == nil {
			convertedNewPath := filepath.Join(project.BasePath, "converted.avi")
			if err := s.copyFile(convertedOldPath, convertedNewPath); err == nil {
				project.ConvertedFile = convertedNewPath
				os.Remove(convertedOldPath)
			}
		}

		s.projectManager.SaveProject(project)
		migratedProjects = append(migratedProjects, project.ID)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Migration completed",
		"migrated_projects": migratedProjects,
	})
}

func (s *Server) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (s *Server) scanExistingTimeline(timelineDir, projectID string) ([]video.FrameInfo, error) {
	entries, err := os.ReadDir(timelineDir)
	if err != nil {
		return nil, err
	}

	var frames []video.FrameInfo
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), "frame_") && strings.HasSuffix(entry.Name(), ".jpg") {
			// Extract frame number from filename like "frame_000030.jpg"
			frameName := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "frame_"), ".jpg")
			frameNum, err := strconv.Atoi(frameName)
			if err != nil {
				continue
			}

			// Assume 30fps for timestamp calculation (could be improved by reading from project)
			timestamp := float64(frameNum) / 30.0

			frames = append(frames, video.FrameInfo{
				FrameNumber:   frameNum,
				Timestamp:     timestamp,
				ThumbnailPath: filepath.Join("projects", projectID, "timeline", entry.Name()),
			})
		}
	}

	// Sort by frame number
	for i := 0; i < len(frames)-1; i++ {
		for j := i + 1; j < len(frames); j++ {
			if frames[i].FrameNumber > frames[j].FrameNumber {
				frames[i], frames[j] = frames[j], frames[i]
			}
		}
	}

	return frames, nil
}

func (s *Server) scanExistingClips(projectID string) error {
	paths := s.projectManager.GetProjectPaths(projectID)
	clipsDir := paths["clips"]
	
	entries, err := os.ReadDir(clipsDir)
	if err != nil {
		return err
	}

	var clips []project.ClipMetadata
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".avi") {
			// Parse clip filename like "clip_0_180.avi" for frame numbers
			clipPath := filepath.Join(clipsDir, entry.Name())
			
			startFrame := 0
			endFrame := 0
			duration := 0.0
			
			// Try to parse frame numbers from filename
			baseName := strings.TrimSuffix(entry.Name(), ".avi")
			if strings.HasPrefix(baseName, "clip_") {
				parts := strings.Split(baseName, "_")
				if len(parts) >= 3 {
					if start, err := strconv.Atoi(parts[1]); err == nil {
						startFrame = start
					}
					if end, err := strconv.Atoi(parts[2]); err == nil {
						endFrame = end
						// Assume 30fps for duration calculation
						duration = float64(endFrame-startFrame) / 30.0
					}
				}
			}
			
			clip := project.ClipMetadata{
				ID:         fmt.Sprintf("clip_%d", len(clips)),
				Name:       entry.Name(),
				StartFrame: startFrame,
				EndFrame:   endFrame,
				StartTime:  float64(startFrame) / 30.0,
				EndTime:    float64(endFrame) / 30.0,
				Duration:   duration,
				FilePath:   clipPath,
				CreatedAt:  time.Now(),
			}
			clips = append(clips, clip)
		}
	}

	if len(clips) > 0 {
		return s.projectManager.SaveClips(projectID, clips)
	}
	
	return nil
}

func (s *Server) scanExistingMoshes(projectID string) error {
	paths := s.projectManager.GetProjectPaths(projectID)
	moshesDir := paths["moshes"]
	
	// Check if moshes directory exists
	if _, err := os.Stat(moshesDir); os.IsNotExist(err) {
		return nil
	}

	// Scan for session directories
	sessionEntries, err := os.ReadDir(moshesDir)
	if err != nil {
		return err
	}

	for _, sessionEntry := range sessionEntries {
		if sessionEntry.IsDir() {
			sessionDir := filepath.Join(moshesDir, sessionEntry.Name())
			
			// Check if session.json already exists
			sessionFile := filepath.Join(sessionDir, "session.json")
			if _, err := os.Stat(sessionFile); err == nil {
				continue // Session metadata already exists
			}

			// Scan for mosh files in this session
			moshEntries, err := os.ReadDir(sessionDir)
			if err != nil {
				continue
			}

			var moshes []project.MoshMetadata
			for _, moshEntry := range moshEntries {
				if !moshEntry.IsDir() && strings.HasSuffix(moshEntry.Name(), ".avi") {
					moshPath := filepath.Join(sessionDir, moshEntry.Name())
					
					// Try to parse effect and intensity from filename
					effect := "unknown"
					intensity := 0.5
					
					if strings.Contains(moshEntry.Name(), "moshed_") {
						effect = "datamosh"
					} else if strings.Contains(moshEntry.Name(), "glitch_") {
						effect = "glitch"
					}

					mosh := project.MoshMetadata{
						ID:       fmt.Sprintf("mosh_%d", len(moshes)),
						Effect:   effect,
						FilePath: moshPath,
						Params: map[string]interface{}{
							"intensity": intensity,
							"source_file": moshEntry.Name(),
						},
						CreatedAt: time.Now(),
					}
					moshes = append(moshes, mosh)
				}
			}

			// Create session metadata if moshes found
			if len(moshes) > 0 {
				session := project.MoshSession{
					ID:        sessionEntry.Name(),
					Name:      fmt.Sprintf("Session: %s", sessionEntry.Name()),
					CreatedAt: time.Now(),
					Source:    "Scanned from existing files",
					Moshes:    moshes,
				}
				s.projectManager.SaveMoshSession(projectID, session)
			}
		}
	}

	return nil
}