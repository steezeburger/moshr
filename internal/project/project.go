package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	OriginalFile string   `json:"original_file"`
	ConvertedFile string  `json:"converted_file"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	BasePath    string    `json:"base_path"`
}

type ClipMetadata struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	StartFrame int     `json:"start_frame"`
	EndFrame   int     `json:"end_frame"`
	StartTime  float64 `json:"start_time"`
	EndTime    float64 `json:"end_time"`
	Duration   float64 `json:"duration"`
	FilePath   string  `json:"file_path"`
	CreatedAt  time.Time `json:"created_at"`
}

type MoshSession struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	CreatedAt time.Time      `json:"created_at"`
	Source    string         `json:"source"`
	Moshes    []MoshMetadata `json:"moshes"`
}

type MoshMetadata struct {
	ID       string            `json:"id"`
	Effect   string            `json:"effect"`
	FilePath string            `json:"file_path"`
	Params   map[string]interface{} `json:"params"`
	CreatedAt time.Time        `json:"created_at"`
}

type Manager struct {
	projectsDir string
}

func NewManager() *Manager {
	projectsDir := "projects"
	os.MkdirAll(projectsDir, 0755)
	
	return &Manager{
		projectsDir: projectsDir,
	}
}

func (m *Manager) CreateProject(originalFileName string) (*Project, error) {
	timestamp := time.Now().Format("20060102_150405")
	baseName := filepath.Base(originalFileName)
	ext := filepath.Ext(baseName)
	nameWithoutExt := baseName[:len(baseName)-len(ext)]
	
	projectID := fmt.Sprintf("%s_%s", nameWithoutExt, timestamp)
	projectPath := filepath.Join(m.projectsDir, projectID)
	
	err := os.MkdirAll(projectPath, 0755)
	if err != nil {
		return nil, err
	}

	subdirs := []string{"timeline", "clips", "moshes"}
	for _, dir := range subdirs {
		err := os.MkdirAll(filepath.Join(projectPath, dir), 0755)
		if err != nil {
			return nil, err
		}
	}

	project := &Project{
		ID:          projectID,
		Name:        nameWithoutExt,
		OriginalFile: "",
		ConvertedFile: "",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		BasePath:    projectPath,
	}

	err = m.SaveProject(project)
	if err != nil {
		return nil, err
	}

	return project, nil
}

func (m *Manager) SaveProject(project *Project) error {
	project.UpdatedAt = time.Now()
	
	projectFile := filepath.Join(project.BasePath, "project.json")
	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(projectFile, data, 0644)
}

func (m *Manager) LoadProject(projectID string) (*Project, error) {
	projectPath := filepath.Join(m.projectsDir, projectID)
	projectFile := filepath.Join(projectPath, "project.json")

	data, err := os.ReadFile(projectFile)
	if err != nil {
		return nil, err
	}

	var project Project
	err = json.Unmarshal(data, &project)
	if err != nil {
		return nil, err
	}

	return &project, nil
}

func (m *Manager) ListProjects() ([]*Project, error) {
	entries, err := os.ReadDir(m.projectsDir)
	if err != nil {
		return nil, err
	}

	var projects []*Project
	for _, entry := range entries {
		if entry.IsDir() {
			project, err := m.LoadProject(entry.Name())
			if err != nil {
				continue
			}
			projects = append(projects, project)
		}
	}

	return projects, nil
}

func (m *Manager) SaveClips(projectID string, clips []ClipMetadata) error {
	clipsFile := filepath.Join(m.projectsDir, projectID, "clips", "clips.json")
	data, err := json.MarshalIndent(clips, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(clipsFile, data, 0644)
}

func (m *Manager) LoadClips(projectID string) ([]ClipMetadata, error) {
	clipsFile := filepath.Join(m.projectsDir, projectID, "clips", "clips.json")
	
	if _, err := os.Stat(clipsFile); os.IsNotExist(err) {
		return []ClipMetadata{}, nil
	}

	data, err := os.ReadFile(clipsFile)
	if err != nil {
		return nil, err
	}

	var clips []ClipMetadata
	err = json.Unmarshal(data, &clips)
	if err != nil {
		return nil, err
	}

	return clips, nil
}

func (m *Manager) SaveMoshSession(projectID string, session MoshSession) error {
	sessionDir := filepath.Join(m.projectsDir, projectID, "moshes", session.ID)
	err := os.MkdirAll(sessionDir, 0755)
	if err != nil {
		return err
	}

	sessionFile := filepath.Join(sessionDir, "session.json")
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sessionFile, data, 0644)
}

func (m *Manager) LoadMoshSessions(projectID string) ([]MoshSession, error) {
	moshesDir := filepath.Join(m.projectsDir, projectID, "moshes")
	
	if _, err := os.Stat(moshesDir); os.IsNotExist(err) {
		return []MoshSession{}, nil
	}

	entries, err := os.ReadDir(moshesDir)
	if err != nil {
		return nil, err
	}

	var sessions []MoshSession
	for _, entry := range entries {
		if entry.IsDir() {
			sessionFile := filepath.Join(moshesDir, entry.Name(), "session.json")
			if data, err := os.ReadFile(sessionFile); err == nil {
				var session MoshSession
				if json.Unmarshal(data, &session) == nil {
					sessions = append(sessions, session)
				}
			}
		}
	}

	return sessions, nil
}

func (m *Manager) SaveScenes(projectID string, scenes interface{}) error {
	scenesFile := filepath.Join(m.projectsDir, projectID, "scenes.json")
	data, err := json.MarshalIndent(scenes, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(scenesFile, data, 0644)
}

func (m *Manager) LoadScenes(projectID string) (interface{}, error) {
	scenesFile := filepath.Join(m.projectsDir, projectID, "scenes.json")
	
	if _, err := os.Stat(scenesFile); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(scenesFile)
	if err != nil {
		return nil, err
	}

	var scenes interface{}
	err = json.Unmarshal(data, &scenes)
	if err != nil {
		return nil, err
	}

	return scenes, nil
}

func (m *Manager) GetProjectPaths(projectID string) map[string]string {
	basePath := filepath.Join(m.projectsDir, projectID)
	return map[string]string{
		"base":     basePath,
		"timeline": filepath.Join(basePath, "timeline"),
		"clips":    filepath.Join(basePath, "clips"),
		"moshes":   filepath.Join(basePath, "moshes"),
	}
}

func (m *Manager) ScanAndRecoverProject(projectID string) error {
	project, err := m.LoadProject(projectID)
	if err != nil {
		return err
	}

	paths := m.GetProjectPaths(projectID)
	
	if project.OriginalFile == "" {
		entries, err := os.ReadDir(paths["base"])
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					name := entry.Name()
					if filepath.Ext(name) == ".mp4" || filepath.Ext(name) == ".mov" {
						project.OriginalFile = filepath.Join(paths["base"], name)
						break
					}
				}
			}
		}
	}

	if project.ConvertedFile == "" {
		entries, err := os.ReadDir(paths["base"])
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					name := entry.Name()
					if filepath.Ext(name) == ".avi" {
						project.ConvertedFile = filepath.Join(paths["base"], name)
						break
					}
				}
			}
		}
	}

	return m.SaveProject(project)
}