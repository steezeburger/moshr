class MoshrApp {
    constructor() {
        this.currentProjectData = null;
        this.currentFile = null;
        this.convertedPath = null;
        this.ws = null;
        this.jobsMap = new Map();
        this.currentFrames = [];
        this.selectedFrames = [];
        this.detectedScenes = [];
        this.createdClips = [];
        this.moshHistory = [];
        this.selectedClip = null;
        
        this.initializeElements();
        this.setupEventListeners();
        this.connectWebSocket();
        this.loadProjects();
    }

    initializeElements() {
        this.newProjectBtn = document.getElementById('newProjectBtn');
        this.loadProjectBtn = document.getElementById('loadProjectBtn');
        this.migrateBtn = document.getElementById('migrateBtn');
        this.projectSelector = document.getElementById('projectSelector');
        this.projectList = document.getElementById('projectList');
        this.selectProjectBtn = document.getElementById('selectProjectBtn');
        this.currentProjectElement = document.getElementById('currentProject');
        this.currentProjectName = document.getElementById('currentProjectName');
        this.uploadSection = document.getElementById('uploadSection');
        this.uploadArea = document.getElementById('uploadArea');
        this.fileInput = document.getElementById('fileInput');
        this.videoInfo = document.getElementById('videoInfo');
        this.videoDetails = document.getElementById('videoDetails');
        this.timelineSection = document.getElementById('timelineSection');
        this.generateTimelineBtn = document.getElementById('generateTimelineBtn');
        this.detectScenesBtn = document.getElementById('detectScenesBtn');
        this.keyFramesBtn = document.getElementById('keyFramesBtn');
        this.scenesList = document.getElementById('scenesList');
        this.scenesContainer = document.getElementById('scenesContainer');
        this.timelineContainer = document.getElementById('timelineContainer');
        this.timelineFrames = document.getElementById('timelineFrames');
        this.selectionStart = document.getElementById('selectionStart');
        this.selectionEnd = document.getElementById('selectionEnd');
        this.selectionDuration = document.getElementById('selectionDuration');
        this.clearSelectionBtn = document.getElementById('clearSelectionBtn');
        this.createClipBtn = document.getElementById('createClipBtn');
        this.clipsLibrary = document.getElementById('clipsLibrary');
        this.clipsGrid = document.getElementById('clipsGrid');
        this.moshHistory = document.getElementById('moshHistory');
        this.historyContainer = document.getElementById('historyContainer');
        this.controls = document.getElementById('controls');
        this.convertBtn = document.getElementById('convertBtn');
        this.moshBtn = document.getElementById('moshBtn');
        this.effectType = document.getElementById('effectType');
        this.intensity = document.getElementById('intensity');
        this.intensityValue = document.getElementById('intensityValue');
        this.batchMode = document.getElementById('batchMode');
        this.clipSource = document.getElementById('clipSource');
        this.progress = document.getElementById('progress');
        this.progressBar = document.getElementById('progressBar');
        this.progressText = document.getElementById('progressText');
        this.results = document.getElementById('results');
        this.previewGrid = document.getElementById('previewGrid');
        this.jobsSection = document.getElementById('jobs');
        this.jobsList = document.getElementById('jobsList');
    }

    setupEventListeners() {
        this.newProjectBtn.addEventListener('click', this.createNewProject.bind(this));
        this.loadProjectBtn.addEventListener('click', this.showProjectSelector.bind(this));
        this.migrateBtn.addEventListener('click', this.migrateOldFiles.bind(this));
        this.selectProjectBtn.addEventListener('click', this.selectProject.bind(this));
        
        this.uploadArea.addEventListener('click', () => this.fileInput.click());
        this.uploadArea.addEventListener('dragover', this.handleDragOver.bind(this));
        this.uploadArea.addEventListener('dragleave', this.handleDragLeave.bind(this));
        this.uploadArea.addEventListener('drop', this.handleDrop.bind(this));
        
        this.fileInput.addEventListener('change', this.handleFileSelect.bind(this));
        this.generateTimelineBtn.addEventListener('click', this.generateTimeline.bind(this));
        this.detectScenesBtn.addEventListener('click', this.detectScenes.bind(this));
        this.keyFramesBtn.addEventListener('click', this.generateKeyFrames.bind(this));
        this.clearSelectionBtn.addEventListener('click', this.clearFrameSelection.bind(this));
        this.createClipBtn.addEventListener('click', this.createClipFromSelection.bind(this));
        this.convertBtn.addEventListener('click', this.convertVideo.bind(this));
        this.moshBtn.addEventListener('click', this.generateMosh.bind(this));
        
        this.intensity.addEventListener('input', (e) => {
            this.intensityValue.textContent = e.target.value;
        });
    }

    handleDragOver(e) {
        e.preventDefault();
        this.uploadArea.classList.add('dragover');
    }

    handleDragLeave(e) {
        e.preventDefault();
        this.uploadArea.classList.remove('dragover');
    }

    handleDrop(e) {
        e.preventDefault();
        this.uploadArea.classList.remove('dragover');
        
        const files = Array.from(e.dataTransfer.files);
        if (files.length > 0) {
            this.processFile(files[0]);
        }
    }

    handleFileSelect(e) {
        const files = Array.from(e.target.files);
        if (files.length > 0) {
            this.processFile(files[0]);
        }
    }

    async processFile(file) {
        if (!this.currentProjectData) {
            alert('Please create or select a project first!');
            return;
        }

        const formData = new FormData();
        formData.append('video', file);

        try {
            this.updateProgress('Uploading file...', 20);
            
            const response = await fetch(`/api/projects/${this.currentProjectData.id}/upload`, {
                method: 'POST',
                body: formData
            });

            if (!response.ok) {
                throw new Error('Upload failed');
            }

            const result = await response.json();
            this.currentFile = result;
            
            this.displayVideoInfo(result.info);
            this.timelineSection.style.display = 'block';
            this.controls.style.display = 'block';
            
            // If uploaded file is already AVI, enable mosh button
            if (result.filename.toLowerCase().endsWith('.avi')) {
                this.convertedPath = result.path;
                this.moshBtn.disabled = false;
            }
            
            this.updateProgress('File uploaded successfully', 100);
            
            setTimeout(() => {
                this.progress.style.display = 'none';
            }, 2000);

        } catch (error) {
            console.error('Upload error:', error);
            this.updateProgress('Upload failed', 0);
        }
    }

    displayVideoInfo(info) {
        this.videoDetails.innerHTML = `
            <div class="info-grid">
                <div><strong>Duration:</strong> ${info.duration?.toFixed(2)} seconds</div>
                <div><strong>Resolution:</strong> ${info.width}x${info.height}</div>
                <div><strong>Framerate:</strong> ${info.framerate?.toFixed(2)} fps</div>
                <div><strong>Video Codec:</strong> ${info.video_codec}</div>
                <div><strong>Audio Codec:</strong> ${info.audio_codec}</div>
                <div><strong>Bitrate:</strong> ${(info.bitrate / 1000).toFixed(0)} kbps</div>
            </div>
        `;
        this.videoInfo.style.display = 'block';
    }

    async convertVideo() {
        if (!this.currentFile || !this.currentProjectData) return;

        try {
            this.updateProgress('Converting to AVI...', 30);
            this.convertBtn.disabled = true;

            const response = await fetch(`/api/projects/${this.currentProjectData.id}/convert`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) {
                throw new Error('Conversion failed');
            }

            const result = await response.json();
            this.convertedPath = result.output_path;
            
            this.moshBtn.disabled = false;
            this.updateProgress('Conversion completed', 100);
            
            setTimeout(() => {
                this.progress.style.display = 'none';
            }, 2000);

        } catch (error) {
            console.error('Conversion error:', error);
            this.updateProgress('Conversion failed', 0);
        } finally {
            this.convertBtn.disabled = false;
        }
    }

    async generateMosh() {
        const inputPath = this.getSelectedInputPath();
        if (!inputPath) return;

        try {
            console.log('Starting mosh generation with input:', inputPath);
            this.updateProgress('Generating mosh effects...', 10);
            this.moshBtn.disabled = true;

            const response = await fetch(`/api/projects/${this.currentProjectData.id}/mosh`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    input_path: inputPath,
                    effect: this.effectType.value,
                    intensity: parseFloat(this.intensity.value),
                    batch: this.batchMode.checked
                })
            });

            if (!response.ok) {
                throw new Error('Mosh generation failed');
            }

            const result = await response.json();
            
            if (result.job_ids) {
                this.monitorJobs(result.job_ids);
            } else if (result.job_id) {
                this.monitorJobs([result.job_id]);
            }

            this.jobsSection.style.display = 'block';
            this.updateProgress('Jobs queued', 100);

        } catch (error) {
            console.error('Mosh generation error:', error);
            this.updateProgress('Mosh generation failed', 0);
        } finally {
            this.moshBtn.disabled = false;
        }
    }

    getSelectedInputPath() {
        if (this.clipSource.value === 'clip' && this.selectedClip) {
            console.log('Using clip path:', this.selectedClip.file_path || this.selectedClip.path);
            return this.selectedClip.file_path || this.selectedClip.path;
        }
        console.log('Using converted path:', this.convertedPath);
        return this.convertedPath;
    }

    async generateTimeline() {
        if (!this.currentFile || !this.currentProjectData) return;

        try {
            this.updateProgress('Generating timeline...', 30);

            const response = await fetch(`/api/projects/${this.currentProjectData.id}/timeline`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    interval: 30
                })
            });

            if (!response.ok) {
                throw new Error('Timeline generation failed');
            }

            const result = await response.json();
            this.currentFrames = result.frames;
            this.displayTimeline(result.frames);
            this.updateProgress('Timeline generated', 100);

            setTimeout(() => {
                this.progress.style.display = 'none';
            }, 2000);

        } catch (error) {
            console.error('Timeline generation error:', error);
            this.updateProgress('Timeline generation failed', 0);
        }
    }

    async generateKeyFrames() {
        if (!this.currentFile || !this.currentProjectData) return;

        try {
            this.updateProgress('Extracting key frames...', 30);

            const response = await fetch(`/api/projects/${this.currentProjectData.id}/timeline`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    keyframes_only: true
                })
            });

            if (!response.ok) {
                throw new Error('Key frame extraction failed');
            }

            const result = await response.json();
            this.currentFrames = result.frames;
            this.displayTimeline(result.frames);
            this.updateProgress('Key frames extracted', 100);

            setTimeout(() => {
                this.progress.style.display = 'none';
            }, 2000);

        } catch (error) {
            console.error('Key frame extraction error:', error);
            this.updateProgress('Key frame extraction failed', 0);
        }
    }

    async detectScenes() {
        if (!this.currentFile || !this.currentProjectData) return;

        try {
            this.updateProgress('Detecting scenes...', 30);

            const response = await fetch(`/api/projects/${this.currentProjectData.id}/scenes`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    threshold: 0.3,
                    advanced: true
                })
            });

            if (!response.ok) {
                throw new Error('Scene detection failed');
            }

            const result = await response.json();
            this.detectedScenes = result.scenes;
            this.displayScenes(result.scenes);
            this.updateProgress('Scenes detected', 100);

            setTimeout(() => {
                this.progress.style.display = 'none';
            }, 2000);

        } catch (error) {
            console.error('Scene detection error:', error);
            this.updateProgress('Scene detection failed', 0);
        }
    }

    displayTimeline(frames) {
        this.timelineContainer.style.display = 'block';
        this.timelineFrames.innerHTML = '';

        frames.forEach((frame, index) => {
            const frameThumb = document.createElement('div');
            frameThumb.className = 'frame-thumb';
            frameThumb.dataset.frameNumber = frame.frame_number;
            frameThumb.dataset.timestamp = frame.timestamp;

            frameThumb.innerHTML = `
                <img src="/${frame.thumbnail_path}" alt="Frame ${frame.frame_number}" />
                <div class="frame-info">
                    Frame ${frame.frame_number}<br>
                    ${frame.timestamp.toFixed(2)}s
                </div>
            `;

            frameThumb.addEventListener('click', () => this.selectFrame(frameThumb, frame));
            this.timelineFrames.appendChild(frameThumb);
        });
    }

    displayScenes(scenes) {
        this.scenesList.style.display = 'block';
        this.scenesContainer.innerHTML = '';

        scenes.forEach((scene, index) => {
            const sceneItem = document.createElement('div');
            sceneItem.className = 'scene-item';
            sceneItem.dataset.sceneIndex = index;

            sceneItem.innerHTML = `
                <h5>Scene ${index + 1}</h5>
                <div class="scene-time">
                    ${scene.start_time.toFixed(2)}s - ${scene.end_time.toFixed(2)}s
                </div>
                <div class="scene-time">
                    Frames ${scene.start_frame} - ${scene.end_frame}
                </div>
                <div class="scene-type">${scene.type}</div>
            `;

            sceneItem.addEventListener('click', () => this.selectScene(sceneItem, scene));
            this.scenesContainer.appendChild(sceneItem);
        });
    }

    selectFrame(frameElement, frame) {
        if (this.selectedFrames.length === 0) {
            this.selectedFrames = [frame];
            frameElement.classList.add('selected');
            this.updateSelectionInfo();
        } else if (this.selectedFrames.length === 1) {
            const firstFrame = this.selectedFrames[0];
            const startFrame = Math.min(firstFrame.frame_number, frame.frame_number);
            const endFrame = Math.max(firstFrame.frame_number, frame.frame_number);
            
            this.selectedFrames = this.currentFrames.filter(f => 
                f.frame_number >= startFrame && f.frame_number <= endFrame
            );
            
            this.updateFrameSelection();
            this.updateSelectionInfo();
        } else {
            this.clearFrameSelection();
            this.selectedFrames = [frame];
            frameElement.classList.add('selected');
            this.updateSelectionInfo();
        }
    }

    selectScene(sceneElement, scene) {
        document.querySelectorAll('.scene-item').forEach(item => item.classList.remove('selected'));
        sceneElement.classList.add('selected');

        this.selectedFrames = this.currentFrames.filter(f => 
            f.frame_number >= scene.start_frame && f.frame_number <= scene.end_frame
        );
        
        this.updateFrameSelection();
        this.updateSelectionInfo();
    }

    updateFrameSelection() {
        document.querySelectorAll('.frame-thumb').forEach(thumb => {
            thumb.classList.remove('selected', 'in-selection');
            const frameNum = parseInt(thumb.dataset.frameNumber);
            
            if (this.selectedFrames.some(f => f.frame_number === frameNum)) {
                if (frameNum === this.selectedFrames[0].frame_number || 
                    frameNum === this.selectedFrames[this.selectedFrames.length - 1].frame_number) {
                    thumb.classList.add('selected');
                } else {
                    thumb.classList.add('in-selection');
                }
            }
        });
    }

    updateSelectionInfo() {
        if (this.selectedFrames.length === 0) {
            this.selectionStart.textContent = 'Start: --';
            this.selectionEnd.textContent = 'End: --';
            this.selectionDuration.textContent = 'Duration: --';
            this.createClipBtn.disabled = true;
        } else {
            const startFrame = this.selectedFrames[0];
            const endFrame = this.selectedFrames[this.selectedFrames.length - 1];
            const duration = endFrame.timestamp - startFrame.timestamp;

            this.selectionStart.textContent = `Start: Frame ${startFrame.frame_number} (${startFrame.timestamp.toFixed(2)}s)`;
            this.selectionEnd.textContent = `End: Frame ${endFrame.frame_number} (${endFrame.timestamp.toFixed(2)}s)`;
            this.selectionDuration.textContent = `Duration: ${duration.toFixed(2)}s`;
            this.createClipBtn.disabled = false;
        }
    }

    clearFrameSelection() {
        this.selectedFrames = [];
        document.querySelectorAll('.frame-thumb').forEach(thumb => {
            thumb.classList.remove('selected', 'in-selection');
        });
        document.querySelectorAll('.scene-item').forEach(item => {
            item.classList.remove('selected');
        });
        this.updateSelectionInfo();
    }

    async createClipFromSelection() {
        if (this.selectedFrames.length === 0) return;

        try {
            const startFrame = this.selectedFrames[0];
            const endFrame = this.selectedFrames[this.selectedFrames.length - 1];
            
            this.updateProgress('Creating clip...', 50);

            const response = await fetch(`/api/projects/${this.currentProjectData.id}/clip`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    frame_range: {
                        start_frame: startFrame.frame_number,
                        end_frame: endFrame.frame_number,
                        start_time: startFrame.timestamp,
                        end_time: endFrame.timestamp
                    },
                    output_name: `clip_${startFrame.frame_number}_${endFrame.frame_number}.avi`
                })
            });

            if (!response.ok) {
                throw new Error('Clip creation failed');
            }

            const result = await response.json();
            
            const clip = {
                name: result.clip_name,
                path: result.output_path,
                startFrame: startFrame.frame_number,
                endFrame: endFrame.frame_number,
                duration: endFrame.timestamp - startFrame.timestamp,
                timestamp: new Date().toISOString()
            };

            this.createdClips.push(clip);
            this.displayClips();
            this.updateProgress('Clip created successfully', 100);

            setTimeout(() => {
                this.progress.style.display = 'none';
            }, 2000);

        } catch (error) {
            console.error('Clip creation error:', error);
            this.updateProgress('Clip creation failed', 0);
        }
    }

    displayClips() {
        this.clipsLibrary.style.display = 'block';
        this.clipsGrid.innerHTML = '';

        this.createdClips.forEach((clip, index) => {
            const clipItem = document.createElement('div');
            clipItem.className = 'clip-item';
            clipItem.dataset.clipIndex = index;

            clipItem.innerHTML = `
                <h5>${clip.name}</h5>
                <div class="clip-info">
                    Frames ${clip.startFrame} - ${clip.endFrame}<br>
                    Duration: ${clip.duration.toFixed(2)}s
                </div>
                <div class="clip-actions">
                    <button class="use-clip-btn" onclick="app.selectClip(${index})">Use for Mosh</button>
                    <button class="delete-clip-btn" onclick="app.deleteClip(${index})">Delete</button>
                </div>
            `;

            this.clipsGrid.appendChild(clipItem);
        });
    }

    selectClip(index) {
        this.selectedClip = this.createdClips[index];
        this.clipSource.value = 'clip';
        
        console.log('Selected clip:', this.selectedClip);
        
        document.querySelectorAll('.clip-item').forEach((item, i) => {
            item.classList.toggle('selected', i === index);
        });

        this.moshBtn.disabled = false;
    }

    deleteClip(index) {
        this.createdClips.splice(index, 1);
        this.displayClips();
        
        if (this.selectedClip === this.createdClips[index]) {
            this.selectedClip = null;
            this.clipSource.value = 'full';
        }
    }

    async monitorJobs(jobIds) {
        for (const jobId of jobIds) {
            this.jobsMap.set(jobId, { status: 'queued', progress: 0 });
        }
        this.updateJobsDisplay();
    }

    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.handleWebSocketMessage(message);
        };

        this.ws.onclose = () => {
            setTimeout(() => this.connectWebSocket(), 3000);
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    handleWebSocketMessage(message) {
        switch (message.type) {
            case 'job_update':
                this.updateJobStatus(message.data.job_id, message.data.status, message.data.progress);
                break;
            case 'jobs_update':
                this.updateAllJobs(message.data);
                break;
        }
    }

    updateJobStatus(jobId, status, progress) {
        if (this.jobsMap.has(jobId)) {
            this.jobsMap.set(jobId, { status, progress });
            this.updateJobsDisplay();
            
            if (status === 'completed') {
                this.loadResults();
            }
        }
    }

    updateAllJobs(jobs) {
        jobs.forEach(job => {
            this.jobsMap.set(job.id, {
                status: job.status,
                progress: job.progress || 0,
                output_path: job.output_path
            });
        });
        this.updateJobsDisplay();
    }

    updateJobsDisplay() {
        let html = '';
        this.jobsMap.forEach((job, jobId) => {
            html += `
                <div class="job-item">
                    <h4>Job: ${jobId}</h4>
                    <div class="status ${job.status}">${job.status}</div>
                    <div class="job-progress">
                        <div class="job-progress-bar" style="width: ${job.progress * 100}%"></div>
                    </div>
                </div>
            `;
        });
        this.jobsList.innerHTML = html;
    }

    async loadResults() {
        if (!this.currentProjectData) return;
        
        try {
            const response = await fetch(`/api/projects/${this.currentProjectData.id}/jobs`);
            if (!response.ok) return;

            const data = await response.json();
            const completedJobs = data.jobs.filter(job => job.status === 'completed');
            
            if (completedJobs.length > 0) {
                this.displayResults(completedJobs);
            }

        } catch (error) {
            console.error('Error loading results:', error);
        }
    }

    displayResults(jobs) {
        if (!this.currentProjectData) return;
        
        let html = '';
        const newMoshes = [];
        
        jobs.forEach((job, index) => {
            const filename = `moshed_${job.id}.avi`;
            html += `
                <div class="preview-item">
                    <img src="/api/projects/${this.currentProjectData.id}/preview/${filename}?width=300&height=200" alt="Preview ${index + 1}" />
                    <h4>Variation ${index + 1}</h4>
                    <div class="status completed">Completed</div>
                    <p>Intensity: ${job.params.intensity}</p>
                </div>
            `;

            newMoshes.push({
                id: job.id,
                filename: filename,
                effect: job.effect,
                params: job.params,
                timestamp: new Date().toISOString(),
                source: this.clipSource.value === 'clip' ? this.selectedClip?.name : 'Full Video'
            });
        });
        
        this.previewGrid.innerHTML = html;
        this.results.style.display = 'block';
        
        this.addToMoshHistory(newMoshes);
    }

    addToMoshHistory(newMoshes) {
        if (newMoshes.length === 0) return;

        const sessionGroup = {
            timestamp: new Date().toISOString(),
            moshes: newMoshes,
            source: this.clipSource.value === 'clip' ? this.selectedClip?.name : 'Full Video'
        };

        this.moshHistory.unshift(sessionGroup);
        this.displayMoshHistory();
        this.moshHistory.style.display = 'block';
    }

    displayMoshHistory() {
        if (this.moshHistory.length === 0) return;

        const historyContainer = this.historyContainer;
        historyContainer.innerHTML = '';

        this.moshHistory.forEach((group, groupIndex) => {
            const groupElement = document.createElement('div');
            groupElement.className = 'history-group';

            const timestamp = new Date(group.timestamp).toLocaleString();
            
            groupElement.innerHTML = `
                <h4>
                    Session ${this.moshHistory.length - groupIndex}
                    <span class="history-timestamp">${timestamp}</span>
                </h4>
                <div class="history-source">Source: ${group.source}</div>
                <div class="history-moshes" id="historyMoshes${groupIndex}"></div>
            `;

            const moshesContainer = groupElement.querySelector(`#historyMoshes${groupIndex}`);
            
            group.moshes.forEach(mosh => {
                const moshElement = document.createElement('div');
                moshElement.className = 'history-mosh-item';

                moshElement.innerHTML = `
                    <img src="/api/preview/${mosh.filename}?width=200&height=150" alt="${mosh.effect}" />
                    <div class="mosh-info">
                        ${mosh.effect.charAt(0).toUpperCase() + mosh.effect.slice(1)} Effect
                    </div>
                    <div class="mosh-params">
                        Intensity: ${mosh.params.intensity}<br>
                        ${mosh.params.iframe_removal ? 'I-Frame Removal' : ''}<br>
                        ${mosh.params.pframe_duplication ? `P-Frame Dup: ${mosh.params.duplication_count}` : ''}
                    </div>
                `;

                moshesContainer.appendChild(moshElement);
            });

            historyContainer.appendChild(groupElement);
        });
    }

    async loadProjects() {
        try {
            const response = await fetch('/api/projects');
            if (response.ok) {
                const data = await response.json();
                this.populateProjectList(data.projects || []);
            }
        } catch (error) {
            console.error('Failed to load projects:', error);
        }
    }

    populateProjectList(projects) {
        this.projectList.innerHTML = '<option value="">Select a project...</option>';
        if (projects && Array.isArray(projects)) {
            projects.forEach(project => {
                const option = document.createElement('option');
                option.value = project.id;
                option.textContent = `${project.name} (${new Date(project.created_at).toLocaleDateString()})`;
                this.projectList.appendChild(option);
            });
        }
    }

    async createNewProject() {
        const name = prompt('Enter project name:');
        if (!name) return;

        try {
            const response = await fetch('/api/projects', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name })
            });

            if (response.ok) {
                const data = await response.json();
                console.log('Project created:', data);
                this.currentProjectData = data.project;
                this.showProjectSelected();
                this.loadProjects(); // Refresh project list
            } else {
                const errorText = await response.text();
                console.error('Project creation failed:', response.status, errorText);
                alert(`Failed to create project: ${response.status}`);
            }
        } catch (error) {
            console.error('Failed to create project:', error);
            alert('Failed to create project: ' + error.message);
        }
    }

    showProjectSelector() {
        this.projectSelector.style.display = 'block';
    }

    async selectProject() {
        const projectId = this.projectList.value;
        if (!projectId) return;

        try {
            const response = await fetch(`/api/projects/${projectId}`);
            if (response.ok) {
                const data = await response.json();
                this.currentProjectData = data.project;
                this.showProjectSelected();
                
                // Load project data
                if (data.clips && data.clips.length > 0) {
                    this.createdClips = data.clips;
                    this.displayClips();
                }
                if (data.sessions && data.sessions.length > 0) {
                    this.moshHistory = data.sessions;
                    this.displayMoshHistory();
                }
                if (data.scenes) this.detectedScenes = data.scenes;
                
                // If project already has a file, load it
                if (data.project.original_file) {
                    this.loadExistingProjectFile(data.project);
                }
            }
        } catch (error) {
            console.error('Failed to load project:', error);
            alert('Failed to load project');
        }
    }

    showProjectSelected() {
        this.projectSelector.style.display = 'none';
        this.currentProjectElement.style.display = 'block';
        this.currentProjectName.textContent = this.currentProjectData.name;
        this.uploadSection.style.display = 'block';
    }

    async loadExistingProjectFile(project) {
        console.log('Loading existing project file:', project.original_file);
        
        // Create a mock file object from the existing project file
        this.currentFile = {
            filename: project.original_file.split('/').pop(),
            path: project.original_file,
            info: null
        };

        // Show that file is loaded immediately
        this.videoInfo.style.display = 'block';
        this.videoDetails.innerHTML = `
            <div class="info-grid">
                <div><strong>File:</strong> ${this.currentFile.filename}</div>
                <div><strong>Status:</strong> Loaded from project</div>
                <div><strong>Path:</strong> ${project.original_file}</div>
                <div><em>Generate timeline to see full video info</em></div>
            </div>
        `;
        
        // Enable timeline and conversion
        this.timelineSection.style.display = 'block';
        this.controls.style.display = 'block';
        
        // Set converted file if it exists, OR if original is already AVI
        if (project.converted_file) {
            this.convertedPath = project.converted_file;
            this.moshBtn.disabled = false;
        } else if (project.original_file.toLowerCase().endsWith('.avi')) {
            // Original file is already AVI, can use it directly for moshing
            this.convertedPath = project.original_file;
            this.moshBtn.disabled = false;
        }
        
        console.log('Project file loaded successfully');
        
        // Auto-load timeline if it exists
        this.autoLoadTimeline();
    }

    async autoLoadTimeline() {
        if (!this.currentProjectData) return;
        
        try {
            const response = await fetch(`/api/projects/${this.currentProjectData.id}/timeline`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    interval: 30
                })
            });

            if (response.ok) {
                const result = await response.json();
                if (result.frames && result.frames.length > 0) {
                    this.currentFrames = result.frames;
                    this.displayTimeline(result.frames);
                    console.log('Auto-loaded existing timeline with', result.frames.length, 'frames');
                }
            }
        } catch (error) {
            console.log('No existing timeline found, user can generate one manually');
        }
    }

    async migrateOldFiles() {
        if (!confirm('This will migrate your old uploads and moshes to the new project system. Continue?')) {
            return;
        }

        try {
            this.updateProgress('Migrating old files...', 50);
            
            const response = await fetch('/api/migrate', {
                method: 'POST'
            });

            if (response.ok) {
                const data = await response.json();
                alert(`Migration completed! Created ${data.migrated_projects?.length || 0} projects.`);
                this.loadProjects(); // Refresh project list
                this.updateProgress('Migration completed', 100);
                
                setTimeout(() => {
                    this.progress.style.display = 'none';
                }, 2000);
            } else {
                throw new Error('Migration failed');
            }
        } catch (error) {
            console.error('Migration error:', error);
            alert('Migration failed: ' + error.message);
            this.updateProgress('Migration failed', 0);
        }
    }

    updateProgress(text, percentage) {
        this.progress.style.display = 'block';
        this.progressText.textContent = text;
        this.progressBar.style.width = percentage + '%';
    }
}

let app;

document.addEventListener('DOMContentLoaded', () => {
    app = new MoshrApp();
});