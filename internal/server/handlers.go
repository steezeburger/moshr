package server

import (
	"encoding/json"
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
	projectpkg "moshr/internal/project"
	"moshr/internal/video"
)

type Server struct {
	processor      *batch.BatchProcessor
	converter      *video.Converter
	analyzer       *video.Analyzer
	sceneDetector  *video.SceneDetector
	frameExtractor *video.FrameExtractor
	projectManager *projectpkg.Manager
	wsHub          *WSHub
}

func NewServer() *Server {
	wsHub := NewWSHub()
	go wsHub.Run()

	converter := video.NewConverter()
	processor := batch.NewBatchProcessor(2, wsHub, converter)
	processor.Start()

	return &Server{
		processor:      processor,
		converter:      converter,
		analyzer:       video.NewAnalyzer(),
		sceneDetector:  video.NewSceneDetector(),
		frameExtractor: video.NewFrameExtractor(),
		projectManager: projectpkg.NewManager(),
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
		api.GET("/projects/:id/moshes", s.handleGetMoshes)
		api.GET("/projects/:id/moshes/:moshId", s.handleGetMosh)
		api.GET("/projects/:id/preview/:filename", s.handlePreview)
		api.POST("/projects/:id/scenes", s.handleDetectScenes)
		api.POST("/projects/:id/timeline", s.handleGenerateTimeline)
		api.POST("/projects/:id/clip", s.handleExtractClip)
		api.DELETE("/projects/:id/clips/:clipId", s.handleDeleteClip)
		api.DELETE("/projects/:id/sessions/:sessionId", s.handleDeleteSession)
		api.DELETE("/projects/:id/sessions/:sessionId/mosh/:moshId", s.handleDeleteMosh)
		api.GET("/projects/:id/converted-files/:sessionId/:moshId", s.handleGetConvertedFiles)
		api.GET("/projects/:id/play-converted/:moshId/:format", s.handlePlayConverted)
		api.GET("/projects/:id/frame/:filename/:timestamp", s.handleGetFrame)
		api.POST("/projects/:id/convert-mosh/:filename", s.handleConvertMosh)
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

	s.projectManager.RecoverOrphanedClips(projectID)

	clips, _ := s.projectManager.LoadClips(projectID)
	sessions, _ := s.projectManager.LoadMoshSessions(projectID)
	scenes, _ := s.projectManager.LoadScenes(projectID)

	c.JSON(http.StatusOK, gin.H{
		"project":  project,
		"clips":    clips,
		"sessions": sessions,
		"scenes":   scenes,
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
			presets = effect.CreatePresets()
		case "duallayer":
			effect := effects.NewDualLayerEffect()
			dualPresets := effect.CreatePresets()
			// Convert DualLayerParams to MoshParams for compatibility
			for _, preset := range dualPresets {
				presets = append(presets, video.MoshParams{
					Intensity:         preset.Intensity,
					IFrameRemoval:     false,
					PFrameDuplication: false,
					DuplicationCount:  0,
				})
			}
		case "rgbdrift", "echotrail", "glitchmosaic", "chromaticblur", "kaleidoscope":
			// These effects use intensity-based presets
			presets = []video.MoshParams{
				{Intensity: 1.0, IFrameRemoval: false, PFrameDuplication: false, DuplicationCount: 0},
				{Intensity: 2.0, IFrameRemoval: false, PFrameDuplication: false, DuplicationCount: 0},
				{Intensity: 3.0, IFrameRemoval: false, PFrameDuplication: false, DuplicationCount: 0},
			}
		default:
			// Default to datamosh-style presets
			presets = []video.MoshParams{
				{Intensity: 0.4, IFrameRemoval: true, PFrameDuplication: true, DuplicationCount: 44},
				{Intensity: 0.7, IFrameRemoval: true, PFrameDuplication: true, DuplicationCount: 62},
				{Intensity: 1.0, IFrameRemoval: true, PFrameDuplication: true, DuplicationCount: 80},
			}
		}

		moshIDs := s.processor.CreateBatchFromPresets(req.InputPath, sessionDir, req.Effect, presets)

		c.JSON(http.StatusOK, gin.H{"mosh_ids": moshIDs, "session_id": sessionID})
	} else {
		moshID := fmt.Sprintf("single_%d", time.Now().Unix())
		// Generate effect-specific parameters for single mosh
		var params video.MoshParams
		switch req.Effect {
		case "datamosh":
			effect := effects.NewDatamoshEffect()
			params = effect.GenerateParams(req.Intensity)
		case "glitch":
			effect := effects.NewGlitchEffect()
			params = effect.GenerateRandomParams(req.Intensity)
		case "duallayer":
			params = video.MoshParams{
				Intensity:         req.Intensity,
				IFrameRemoval:     false,
				PFrameDuplication: false,
				DuplicationCount:  0,
			}
		case "rgbdrift", "echotrail", "glitchmosaic", "chromaticblur", "kaleidoscope":
			params = video.MoshParams{
				Intensity:         req.Intensity,
				IFrameRemoval:     false,
				PFrameDuplication: false,
				DuplicationCount:  0,
			}
		default:
			// Default datamosh-style parameters
			params = video.MoshParams{
				Intensity:         req.Intensity,
				IFrameRemoval:     req.Intensity > 0.1,
				PFrameDuplication: req.Intensity > 0.05,
				DuplicationCount:  int(req.Intensity*60) + 20,
			}
		}

		mosh := &batch.Mosh{
			ID:        moshID,
			InputPath: req.InputPath,
			OutputDir: sessionDir,
			Effect:    req.Effect,
			Params:    params,
		}

		s.processor.AddMosh(mosh)

		c.JSON(http.StatusOK, gin.H{"mosh_id": moshID, "session_id": sessionID})
	}
}

func (s *Server) handleGetMoshes(c *gin.Context) {
	moshes := s.processor.GetAllMoshes()

	c.JSON(http.StatusOK, gin.H{"moshes": moshes})
}

func (s *Server) handleGetMosh(c *gin.Context) {
	moshID := c.Param("id")
	mosh, exists := s.processor.GetMosh(moshID)

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mosh not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"mosh": mosh})
}

func (s *Server) handlePreview(c *gin.Context) {
	projectID := c.Param("id")
	filename := c.Param("filename")

	// Extract mosh ID from filename (moshed_moshID.avi -> moshID)
	var moshID string
	if strings.HasPrefix(filename, "moshed_") && strings.HasSuffix(filename, ".avi") {
		moshID = strings.TrimSuffix(strings.TrimPrefix(filename, "moshed_"), ".avi")
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filename format"})
		return
	}

	// Look for the preview file in project moshes directory
	paths := s.projectManager.GetProjectPaths(projectID)
	moshesDir := paths["moshes"]

	var previewPath string
	entries, err := os.ReadDir(moshesDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				sessionPreviewPath := filepath.Join(moshesDir, entry.Name(), fmt.Sprintf("preview_%s.jpg", moshID))
				if _, err := os.Stat(sessionPreviewPath); err == nil {
					previewPath = sessionPreviewPath
					break
				}
			}
		}
	}

	if previewPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Preview not found"})
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
			"frames":       existingFrames,
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
		"frames":       frames,
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

	// Save clip metadata
	clipMetadata := projectpkg.ClipMetadata{
		ID:         fmt.Sprintf("clip_%d", time.Now().Unix()),
		Name:       req.OutputName,
		StartFrame: req.FrameRange.StartFrame,
		EndFrame:   req.FrameRange.EndFrame,
		StartTime:  req.FrameRange.StartTime,
		EndTime:    req.FrameRange.EndTime,
		Duration:   req.FrameRange.EndTime - req.FrameRange.StartTime,
		FilePath:   outputPath,
		CreatedAt:  time.Now(),
	}

	clips, err := s.projectManager.LoadClips(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load clips metadata"})
		return
	}

	clips = append(clips, clipMetadata)
	err = s.projectManager.SaveClips(projectID, clips)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save clips metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"output_path": outputPath,
		"clip_name":   req.OutputName,
	})
}

func (s *Server) handleDeleteClip(c *gin.Context) {
	projectID := c.Param("id")
	clipID := c.Param("clipId")

	clips, err := s.projectManager.LoadClips(projectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load clips"})
		return
	}

	// Find the clip to delete
	var clipToDelete *projectpkg.ClipMetadata
	var clipIndex int = -1
	for i, clip := range clips {
		if clip.ID == clipID {
			clipToDelete = &clip
			clipIndex = i
			break
		}
	}

	if clipToDelete == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Clip not found"})
		return
	}

	// Delete the clip file
	if err := os.Remove(clipToDelete.FilePath); err != nil && !os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete clip file"})
		return
	}

	// Remove clip from metadata
	clips = append(clips[:clipIndex], clips[clipIndex+1:]...)

	// Save updated clips metadata
	if err := s.projectManager.SaveClips(projectID, clips); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update clips metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Clip deleted successfully",
		"deleted_clip_id": clipID,
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
		"message":           "Migration completed",
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

func (s *Server) handleConvertMosh(c *gin.Context) {
	projectID := c.Param("id")
	filename := c.Param("filename")

	var req struct {
		Format string `json:"format"` // "mp4" or "webm"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Format != "mp4" && req.Format != "webm" {
		req.Format = "mp4" // Default to MP4
	}

	// Find the moshed file in project moshes directory
	paths := s.projectManager.GetProjectPaths(projectID)
	moshesDir := paths["moshes"]

	var inputPath string
	entries, err := os.ReadDir(moshesDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read moshes directory"})
		return
	}

	// Look for the file in session directories
	for _, entry := range entries {
		if entry.IsDir() {
			sessionPath := filepath.Join(moshesDir, entry.Name(), filename)
			if _, err := os.Stat(sessionPath); err == nil {
				inputPath = sessionPath
				break
			}
		}
	}

	if inputPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Moshed file not found"})
		return
	}

	// Generate output filename
	baseName := strings.TrimSuffix(filename, ".avi")
	outputFilename := fmt.Sprintf("%s_converted.%s", baseName, req.Format)
	outputPath := filepath.Join(filepath.Dir(inputPath), outputFilename)

	// Generate conversion ID for WebSocket updates
	conversionID := fmt.Sprintf("convert_%s_%s_%d", filename, req.Format, time.Now().Unix())

	// Create progress callback for WebSocket updates
	progressCallback := func(progress float64) {
		s.wsHub.BroadcastMoshUpdate(conversionID, "processing", progress)
	}

	// Convert the file with progress tracking
	var convertErr error
	if req.Format == "mp4" {
		convertErr = s.converter.MoshedAVIToMP4WithProgress(inputPath, outputPath, progressCallback)
	} else {
		convertErr = s.converter.MoshedAVIToWebMWithProgress(inputPath, outputPath, progressCallback)
	}

	if convertErr != nil {
		s.wsHub.BroadcastMoshUpdate(conversionID, "failed", 0)
		c.JSON(http.StatusInternalServerError, gin.H{"error": convertErr.Error()})
		return
	}

	s.wsHub.BroadcastMoshUpdate(conversionID, "completed", 1.0)
	c.JSON(http.StatusOK, gin.H{
		"message":       "Conversion completed",
		"output_file":   outputFilename,
		"output_path":   outputPath,
		"format":        req.Format,
		"conversion_id": conversionID,
	})
}

func (s *Server) handleGetConvertedFiles(c *gin.Context) {
	projectID := c.Param("id")
	sessionID := c.Param("sessionId")
	moshID := c.Param("moshId")

	paths := s.projectManager.GetProjectPaths(projectID)
	moshesDir := paths["moshes"]

	convertedFiles := map[string]bool{
		"mp4":  false,
		"webm": false,
	}

	// Check specific session directory for converted files
	sessionDir := filepath.Join(moshesDir, sessionID)

	// Check for MP4
	mp4Path := filepath.Join(sessionDir, fmt.Sprintf("moshed_%s_converted.mp4", moshID))
	fmt.Printf("DEBUG: Checking MP4 path: %s\n", mp4Path)
	if _, err := os.Stat(mp4Path); err == nil {
		fmt.Printf("DEBUG: Found MP4 file!\n")
		convertedFiles["mp4"] = true
	} else {
		fmt.Printf("DEBUG: MP4 file not found: %v\n", err)
	}

	// Check for WebM
	webmPath := filepath.Join(sessionDir, fmt.Sprintf("moshed_%s_converted.webm", moshID))
	fmt.Printf("DEBUG: Checking WebM path: %s\n", webmPath)
	if _, err := os.Stat(webmPath); err == nil {
		fmt.Printf("DEBUG: Found WebM file!\n")
		convertedFiles["webm"] = true
	} else {
		fmt.Printf("DEBUG: WebM file not found: %v\n", err)
	}

	// Also list what files actually exist in the session directory
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		fmt.Printf("DEBUG: Cannot read session dir %s: %v\n", sessionDir, err)
	} else {
		fmt.Printf("DEBUG: Files in session dir %s:\n", sessionDir)
		for _, entry := range entries {
			fmt.Printf("  - %s\n", entry.Name())
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"mosh_id":         moshID,
		"session_id":      sessionID,
		"converted_files": convertedFiles,
		"debug": map[string]string{
			"session_dir": sessionDir,
			"mp4_path":    mp4Path,
			"webm_path":   webmPath,
		},
	})
}

func (s *Server) checkMoshConvertedFiles(projectID, sessionID, moshID string) map[string]bool {
	paths := s.projectManager.GetProjectPaths(projectID)
	moshesDir := paths["moshes"]

	convertedFiles := map[string]bool{
		"mp4":  false,
		"webm": false,
	}

	// Check only the specific session directory
	sessionDir := filepath.Join(moshesDir, sessionID)

	// Check for MP4
	mp4Path := filepath.Join(sessionDir, fmt.Sprintf("moshed_%s_converted.mp4", moshID))
	if _, err := os.Stat(mp4Path); err == nil {
		convertedFiles["mp4"] = true
	}

	// Check for WebM
	webmPath := filepath.Join(sessionDir, fmt.Sprintf("moshed_%s_converted.webm", moshID))
	if _, err := os.Stat(webmPath); err == nil {
		convertedFiles["webm"] = true
	}

	return convertedFiles
}

func (s *Server) handlePlayConverted(c *gin.Context) {
	projectID := c.Param("id")
	moshID := c.Param("moshId")
	format := c.Param("format")

	paths := s.projectManager.GetProjectPaths(projectID)
	moshesDir := paths["moshes"]

	// Search all session directories for the converted file
	entries, err := os.ReadDir(moshesDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read moshes directory"})
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			sessionDir := filepath.Join(moshesDir, entry.Name())
			convertedFile := filepath.Join(sessionDir, fmt.Sprintf("moshed_%s_converted.%s", moshID, format))

			if _, err := os.Stat(convertedFile); err == nil {
				// Found the file, return the relative path for serving
				relativePath := fmt.Sprintf("/projects/%s/moshes/%s/moshed_%s_converted.%s", projectID, entry.Name(), moshID, format)
				c.JSON(http.StatusOK, gin.H{
					"file_path":   relativePath,
					"session_dir": entry.Name(),
				})
				return
			}
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "Converted file not found"})
}

func (s *Server) handleDeleteMosh(c *gin.Context) {
	projectID := c.Param("id")
	sessionID := c.Param("sessionId")
	moshID := c.Param("moshId")

	paths := s.projectManager.GetProjectPaths(projectID)
	moshesDir := paths["moshes"]
	sessionDir := filepath.Join(moshesDir, sessionID)

	// Delete all files related to this mosh
	filesToDelete := []string{
		fmt.Sprintf("moshed_%s.avi", moshID),
		fmt.Sprintf("moshed_%s_converted.mp4", moshID),
		fmt.Sprintf("moshed_%s_converted.webm", moshID),
		fmt.Sprintf("preview_%s.jpg", moshID),
	}

	deletedFiles := []string{}
	for _, filename := range filesToDelete {
		filePath := filepath.Join(sessionDir, filename)
		if err := os.Remove(filePath); err == nil {
			deletedFiles = append(deletedFiles, filename)
		}
	}

	// Update session metadata to remove the deleted mosh
	sessionFile := filepath.Join(sessionDir, "session.json")
	if sessionData, err := os.ReadFile(sessionFile); err == nil {
		var session projectpkg.MoshSession
		if json.Unmarshal(sessionData, &session) == nil {
			// Remove the mosh with matching mosh ID from the session
			updatedMoshes := []projectpkg.MoshMetadata{}
			for _, mosh := range session.Moshes {
				// Check both the stored ID and extract ID from filename
				storedMoshID := mosh.ID
				if mosh.FilePath != "" {
					filename := filepath.Base(mosh.FilePath)
					if strings.HasPrefix(filename, "moshed_") && strings.HasSuffix(filename, ".avi") {
						storedMoshID = strings.TrimSuffix(strings.TrimPrefix(filename, "moshed_"), ".avi")
					}
				}

				// Keep moshes that don't match the deleted mosh ID
				if storedMoshID != moshID {
					updatedMoshes = append(updatedMoshes, mosh)
				}
			}

			session.Moshes = updatedMoshes

			// Save updated session metadata
			if updatedData, err := json.MarshalIndent(session, "", "  "); err == nil {
				os.WriteFile(sessionFile, updatedData, 0644)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Mosh deleted successfully",
		"session_id":    sessionID,
		"mosh_id":       moshID,
		"deleted_files": deletedFiles,
	})
}

func (s *Server) handleDeleteSession(c *gin.Context) {
	projectID := c.Param("id")
	sessionID := c.Param("sessionId")

	paths := s.projectManager.GetProjectPaths(projectID)
	moshesDir := paths["moshes"]
	sessionDir := filepath.Join(moshesDir, sessionID)

	// Delete entire session directory
	err := os.RemoveAll(sessionDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete session: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Session deleted successfully",
		"session_id": sessionID,
	})
}

func (s *Server) extractJobIDFromFilename(filename string, fallbackIndex int) string {
	// Extract job ID from filename like "moshed_batch_0.avi" -> "batch_0" or "moshed_single_1749018199.avi" -> "single_1749018199"
	if strings.HasPrefix(filename, "moshed_") && strings.HasSuffix(filename, ".avi") {
		return strings.TrimSuffix(strings.TrimPrefix(filename, "moshed_"), ".avi")
	}
	// Fallback should match current job ID patterns
	return fmt.Sprintf("unknown_%d", fallbackIndex)
}
