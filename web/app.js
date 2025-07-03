class MoshrApp {
    constructor() {
        this.currentProjectData = null;
        this.currentFile = null;
        this.convertedPath = null;
        this.ws = null;
        this.moshesMap = new Map();
        this.currentFrames = [];
        this.selectedFrames = [];
        this.detectedScenes = [];
        this.createdClips = [];
        this.moshSessions = [];
        this.selectedClip = null;
        
        this.initializeElements();
        this.setupEventListeners();
        this.connectWebSocket();
        this.loadProjects();
    }

    initializeElements() {
        this.newProjectBtn = document.getElementById('newProjectBtn');
        this.projectManagement = document.getElementById('projectManagement');
        this.projectWorkspace = document.getElementById('projectWorkspace');
        this.backToProjectsBtn = document.getElementById('backToProjectsBtn');
        this.fileControls = document.getElementById('fileControls');
        this.addFileBtn = document.getElementById('addFileBtn');
        this.projectsGrid = document.getElementById('projectsGrid');
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
        this.moshesSection = document.getElementById('moshes');
        this.moshesList = document.getElementById('moshesList');
    }

    setupEventListeners() {
        this.newProjectBtn.addEventListener('click', this.createNewProject.bind(this));
        this.backToProjectsBtn.addEventListener('click', this.showProjectManagement.bind(this));
        this.addFileBtn.addEventListener('click', this.toggleUploadSection.bind(this));
        
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

            this.moshesSection.style.display = 'block';
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

    getClipThumbnailPath(frameNumber) {
        if (!this.currentProjectData || frameNumber === undefined || frameNumber === null) return null;
        
        return `/projects/${this.currentProjectData.id}/timeline/frame_${String(frameNumber).padStart(6, '0')}.jpg`;
    }

    displayClips() {
        this.clipsLibrary.style.display = 'block';
        this.clipsGrid.innerHTML = '';

        this.createdClips.forEach((clip, index) => {
            const clipItem = document.createElement('div');
            clipItem.className = 'clip-item';
            clipItem.dataset.clipIndex = index;

            const startFrame = clip.startFrame || clip.start_frame;
            const endFrame = clip.endFrame || clip.end_frame;
            const startThumbnailPath = this.getClipThumbnailPath(startFrame);
            const endThumbnailPath = this.getClipThumbnailPath(endFrame);

            const thumbnailsHtml = (startThumbnailPath || endThumbnailPath) ? `
                <div class="clip-thumbnails">
                    ${startThumbnailPath ? `<img src="${startThumbnailPath}" alt="First frame" class="clip-thumbnail" title="Frame ${startFrame}" />` : ''}
                    ${endThumbnailPath && endFrame !== startFrame ? `<img src="${endThumbnailPath}" alt="Last frame" class="clip-thumbnail" title="Frame ${endFrame}" />` : ''}
                </div>
            ` : '';

            clipItem.innerHTML = `
                ${thumbnailsHtml}
                <h5>${clip.name}</h5>
                <div class="clip-info">
                    Frames ${startFrame} - ${endFrame}<br>
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

    async deleteClip(index) {
        if (!this.currentProjectData) {
            alert('No project selected');
            return;
        }

        const clip = this.createdClips[index];
        if (!clip || !clip.id) {
            alert('Invalid clip selected');
            return;
        }

        if (!confirm(`Are you sure you want to delete the clip "${clip.name}"?`)) {
            return;
        }

        try {
            const response = await fetch(`/api/projects/${this.currentProjectData.id}/clips/${clip.id}`, {
                method: 'DELETE'
            });

            if (response.ok) {
                this.createdClips.splice(index, 1);
                this.displayClips();
                
                if (this.selectedClip === clip) {
                    this.selectedClip = null;
                    this.clipSource.value = 'full';
                }
                
                console.log('Clip deleted successfully');
            } else {
                const error = await response.text();
                throw new Error(error || 'Failed to delete clip');
            }
        } catch (error) {
            console.error('Error deleting clip:', error);
            alert('Failed to delete clip: ' + error.message);
        }
    }

    async monitorJobs(jobIds) {
        // Clear previous session jobs
        this.jobsMap.clear();
        
        for (const jobId of jobIds) {
            this.jobsMap.set(jobId, { status: 'queued', progress: 0 });
        }
        this.updateMoshesDisplay();
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
            case 'mosh_update':
                this.updateMoshStatus(message.data.mosh_id, message.data.status, message.data.progress);
                break;
            case 'moshes_update':
                this.updateAllMoshes(message.data);
                break;
        }
    }

    updateMoshStatus(moshId, status, progress) {
        console.log('WebSocket mosh update:', { moshId, status, progress });
        console.log('Moshes in map:', Array.from(this.moshesMap.keys()));
        
        if (this.moshesMap.has(moshId)) {
            const mosh = this.moshesMap.get(moshId);
            console.log('Found mosh:', mosh);
            mosh.status = status;
            mosh.progress = progress;
            this.moshesMap.set(moshId, mosh);
            this.updateMoshesDisplay();
            
            // Update local progress bar for conversions
            if (mosh.isConversion && mosh.moshId) {
                console.log('Updating local progress for moshId:', mosh.moshId);
                if (status === 'processing') {
                    this.showLocalProgress(mosh.moshId, `Converting to ${mosh.format.toUpperCase()}... (${Math.round(progress * 100)}%)`, progress * 100);
                } else if (status === 'completed') {
                    this.showLocalProgress(mosh.moshId, `Conversion to ${mosh.format.toUpperCase()} completed`, 100);
                    this.showConvertedFile(moshId, mosh.format);
                } else if (status === 'failed') {
                    this.showLocalProgress(mosh.moshId, `Conversion to ${mosh.format.toUpperCase()} failed`, 0);
                }
            } else {
                console.log('Mosh is not conversion or missing moshId:', mosh);
            }
            
            if (status === 'completed' && !mosh.isConversion) {
                this.loadResults();
                
                // Check if all moshes are completed and hide main progress
                const allCompleted = Array.from(this.moshesMap.values()).every(mosh => 
                    mosh.status === 'completed' || mosh.status === 'failed'
                );
                
                if (allCompleted) {
                    setTimeout(() => {
                        this.progress.style.display = 'none';
                    }, 2000);
                }
            }
        } else {
            console.log('Mosh not found in map for moshId:', moshId);
        }
    }

    updateAllMoshes(moshes) {
        moshes.forEach(mosh => {
            this.moshesMap.set(mosh.id, {
                status: mosh.status,
                progress: mosh.progress || 0,
                output_path: mosh.output_path
            });
        });
        this.updateMoshesDisplay();
    }

    updateMoshesDisplay() {
        let html = '';
        this.moshesMap.forEach((mosh, moshId) => {
            html += `
                <div class="mosh-item">
                    <h4>Mosh: ${moshId}</h4>
                    <div class="status ${mosh.status}">${mosh.status}</div>
                    <div class="mosh-progress">
                        <div class="mosh-progress-bar" style="width: ${mosh.progress * 100}%"></div>
                    </div>
                </div>
            `;
        });
        this.moshesList.innerHTML = html;
    }

    async loadResults() {
        if (!this.currentProjectData) return;
        
        try {
            const response = await fetch(`/api/projects/${this.currentProjectData.id}/moshes`);
            if (!response.ok) return;

            const data = await response.json();
            // Only get completed moshes from the current session
            const currentSessionMoshIds = Array.from(this.moshesMap.keys());
            const completedMoshes = data.moshes.filter(mosh => 
                mosh.status === 'completed' && currentSessionMoshIds.includes(mosh.id)
            );
            
            if (completedMoshes.length > 0) {
                this.displayResults(completedMoshes);
            }

        } catch (error) {
            console.error('Error loading results:', error);
        }
    }

    async displayResults(moshes) {
        if (!this.currentProjectData) return;
        
        let html = '';
        const newMoshes = [];
        
        // First, render the basic preview items
        moshes.forEach((mosh, index) => {
            const filename = `moshed_${mosh.id}.avi`;
            
            html += `
                <div class="preview-item" id="preview-${mosh.id}">
                    <img src="/api/projects/${this.currentProjectData.id}/preview/${filename}?width=300&height=200" alt="Preview ${index + 1}" />
                    <h4>Variation ${index + 1}</h4>
                    <div class="status completed">Completed</div>
                    <p>Intensity: ${mosh.params.intensity}</p>
                    <div class="conversion-progress" id="conversion-progress-${mosh.id}" style="display: none;">
                        <div class="progress-label" id="progress-label-${mosh.id}">Converting...</div>
                        <div class="progress-bar-container">
                            <div class="progress-bar" id="progress-bar-${mosh.id}"></div>
                        </div>
                    </div>
                    <div class="convert-actions">
                        <button onclick="app.handleMp4Action('${filename}', '${mosh.id}')" class="convert-btn" id="mp4-btn-${mosh.id}">
                            Convert to MP4
                        </button>
                        <button onclick="app.handleWebmAction('${filename}', '${mosh.id}')" class="convert-btn" id="webm-btn-${mosh.id}">
                            Convert to WebM
                        </button>
                        <button onclick="app.deleteMosh('${mosh.id}')" class="delete-btn" title="Delete this mosh">
                            🗑️
                        </button>
                    </div>
                </div>
            `;

            newMoshes.push({
                id: mosh.id,
                filename: filename,
                effect: mosh.effect,
                params: mosh.params,
                timestamp: new Date().toISOString(),
                source: this.clipSource.value === 'clip' ? this.selectedClip?.name : 'Full Video'
            });
        });
        
        this.previewGrid.innerHTML = html;
        this.results.style.display = 'block';
        
        // Now check for existing converted files for each mosh
        moshes.forEach(mosh => {
            if (mosh.converted_files) {
                this.updateConvertedFileUI(mosh.id, mosh.converted_files);
            }
        });
        
        this.addToMoshHistory(newMoshes);
    }

    addToMoshHistory(newMoshes) {
        if (newMoshes.length === 0) return;

        let sourceInfo = 'Full Video';
        if (this.clipSource.value === 'clip' && this.selectedClip) {
            const startFrame = this.selectedClip.startFrame || this.selectedClip.start_frame;
            const endFrame = this.selectedClip.endFrame || this.selectedClip.end_frame;
            sourceInfo = `${this.selectedClip.name} (Frames ${startFrame}-${endFrame})`;
        }

        const sessionGroup = {
            timestamp: new Date().toISOString(),
            moshes: newMoshes.map(mosh => ({
                ...mosh,
                source: sourceInfo
            })),
            source: sourceInfo
        };

        this.moshSessions.unshift(sessionGroup);
        this.displayMoshHistory();
        this.moshHistory.style.display = 'block';
    }

    displayMoshHistory() {
        if (this.moshSessions.length === 0) return;

        const historyContainer = this.historyContainer;
        historyContainer.innerHTML = '';

        this.moshSessions.forEach((group, groupIndex) => {
            const groupElement = document.createElement('div');
            groupElement.className = 'history-group';

            // Handle both API format (created_at) and frontend format (timestamp)
            const timestamp = new Date(group.created_at || group.timestamp).toLocaleString();
            
            // Extract session ID from group data or generate from timestamp
            const sessionId = group.id || group.session_id || this.getSessionIdFromTimestamp(group.created_at || group.timestamp);
            
            // Get source information
            const sourceInfo = group.source || (group.moshes && group.moshes[0] ? group.moshes[0].source : null) || 'Unknown Source';
            const isClipSource = sourceInfo !== 'Full Video' && sourceInfo !== 'Unknown Source';
            
            // Generate clip thumbnails if this session used a clip
            let clipThumbnailsHtml = '';
            if (isClipSource && sourceInfo.includes('Frames ')) {
                const frameMatch = sourceInfo.match(/Frames (\d+)-(\d+)/);
                if (frameMatch) {
                    const startFrame = parseInt(frameMatch[1]);
                    const endFrame = parseInt(frameMatch[2]);
                    const startThumbnail = this.getClipThumbnailPath(startFrame);
                    const endThumbnail = this.getClipThumbnailPath(endFrame);
                    
                    clipThumbnailsHtml = `
                        <div class="session-clip-thumbnails">
                            ${startThumbnail ? `<img src="${startThumbnail}" alt="Start frame" class="session-thumbnail" title="Frame ${startFrame}" />` : ''}
                            ${endThumbnail && endFrame !== startFrame ? `<img src="${endThumbnail}" alt="End frame" class="session-thumbnail" title="Frame ${endFrame}" />` : ''}
                        </div>
                    `;
                }
            }
            
            groupElement.innerHTML = `
                <h4>
                    ${group.name || `Session ${this.moshSessions.length - groupIndex}`}
                    <span class="history-timestamp">${timestamp}</span>
                    <button onclick="app.deleteSession('${sessionId}')" class="delete-session-btn" title="Delete entire session">
                        🗑️ Delete Session
                    </button>
                </h4>
                <div class="session-source">
                    <strong>Source:</strong> 
                    <span class="source-indicator ${isClipSource ? 'clip-source' : 'full-source'}">
                        ${isClipSource ? '📄 ' : '🎬 '}${sourceInfo}
                    </span>
                    ${clipThumbnailsHtml}
                </div>
                <div class="history-moshes" id="historyMoshes${groupIndex}"></div>
            `;

            const moshesContainer = groupElement.querySelector(`#historyMoshes${groupIndex}`);
            
            group.moshes.forEach(mosh => {
                const moshElement = document.createElement('div');
                moshElement.className = 'history-mosh-item';

                // Extract filename from file_path for API compatibility
                const filename = mosh.filename || (mosh.file_path ? mosh.file_path.split('/').pop() : 'unknown.avi');
                // Always extract mosh ID from filename for history items since stored IDs are often wrong
                let moshId;
                if (filename.startsWith('moshed_')) {
                    // Extract mosh ID from filename like "moshed_single_1749018199.avi" -> "single_1749018199"
                    moshId = filename.substring(7).replace('.avi', ''); // Remove "moshed_" and ".avi"
                } else {
                    // Fallback to stored ID if filename extraction fails
                    moshId = mosh.id || 'unknown';
                }

                // Get effect name, fallback to 'mosh' if not available
                const effectName = mosh.effect || 'mosh';

                moshElement.innerHTML = `
                    <img src="/api/projects/${this.currentProjectData.id}/preview/${filename}?width=200&height=150" alt="${effectName}" />
                    <div class="mosh-info">
                        ${effectName.charAt(0).toUpperCase() + effectName.slice(1)} Effect
                    </div>
                    <div class="mosh-params">
                        Intensity: ${mosh.params.intensity}<br>
                        ${mosh.params.iframe_removal ? 'I-Frame Removal' : ''}<br>
                        ${mosh.params.pframe_duplication ? `P-Frame Dup: ${mosh.params.duplication_count}` : ''}
                    </div>
                    <div class="conversion-progress" id="history-conversion-progress-${moshId}" style="display: none;">
                        <div class="progress-label" id="history-progress-label-${moshId}">Converting...</div>
                        <div class="progress-bar-container">
                            <div class="progress-bar" id="history-progress-bar-${moshId}"></div>
                        </div>
                    </div>
                    <div class="convert-actions">
                        <button onclick="app.handleMp4Action('${filename}', '${moshId}')" class="convert-btn small" id="history-mp4-btn-${moshId}">MP4</button>
                        <button onclick="app.handleWebmAction('${filename}', '${moshId}')" class="convert-btn small" id="history-webm-btn-${moshId}">WebM</button>
                        <button onclick="app.deleteMoshFromHistory('${sessionId}', '${moshId}')" class="delete-btn small" title="Delete this mosh">
                            🗑️
                        </button>
                    </div>
                `;
                
                // Check for existing converted files for this mosh (only if we have a valid mosh ID)
                if (moshId && moshId !== 'unknown') {
                    this.checkExistingConvertedFilesForHistory(sessionId, moshId);
                } else {
                    console.log('Skipping converted file check for mosh with invalid mosh ID:', moshId, 'filename:', filename);
                }

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
        this.projectsGrid.innerHTML = '';
        if (projects && Array.isArray(projects)) {
            projects.forEach(project => {
                const projectCard = document.createElement('div');
                projectCard.className = 'project-card';
                projectCard.dataset.projectId = project.id;
                
                const originalFile = project.original_file ? project.original_file.split('/').pop() : 'No file uploaded';
                const hasConvertedFile = project.converted_file ? 'Yes' : 'No';
                
                projectCard.innerHTML = `
                    <h4>${project.name}</h4>
                    <div class="project-date">Created: ${new Date(project.created_at).toLocaleDateString()}</div>
                    <div class="project-file">File: ${originalFile}</div>
                    <div class="project-stats">
                        <span>Converted: ${hasConvertedFile}</span>
                        <span>ID: ${project.id.split('_').pop()}</span>
                    </div>
                `;
                
                projectCard.addEventListener('click', () => this.selectProjectCard(project.id, projectCard));
                this.projectsGrid.appendChild(projectCard);
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

    async selectProjectCard(projectId, cardElement) {
        // Update UI to show selected project
        document.querySelectorAll('.project-card').forEach(card => card.classList.remove('selected'));
        cardElement.classList.add('selected');

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
                    this.moshSessions = data.sessions;
                    this.displayMoshHistory();
                    this.moshHistory.style.display = 'block';
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
        this.projectManagement.style.display = 'none';
        this.projectWorkspace.style.display = 'block';
        this.currentProjectName.textContent = this.currentProjectData.name;
        this.fileControls.style.display = 'block';
        this.uploadSection.style.display = 'none';
    }

    showProjectManagement() {
        this.projectManagement.style.display = 'block';
        this.projectWorkspace.style.display = 'none';
        this.fileControls.style.display = 'none';
        this.uploadSection.style.display = 'none';
        this.videoInfo.style.display = 'none';
        this.timelineSection.style.display = 'none';
        this.controls.style.display = 'none';
        this.progress.style.display = 'none';
        this.clipsLibrary.style.display = 'none';
        this.moshHistory.style.display = 'none';
        this.results.style.display = 'none';
        this.jobsSection.style.display = 'none';
        this.currentProjectData = null;
    }

    toggleUploadSection() {
        if (this.uploadSection.style.display === 'none') {
            this.uploadSection.style.display = 'block';
            this.addFileBtn.textContent = 'Hide Upload';
        } else {
            this.uploadSection.style.display = 'none';
            this.addFileBtn.textContent = 'Add New File';
        }
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


    updateProgress(text, percentage) {
        this.progress.style.display = 'block';
        this.progressText.textContent = text;
        this.progressBar.style.width = percentage + '%';
    }

    async convertMosh(filename, format) {
        if (!this.currentProjectData) return;

        try {
            // Extract mosh ID from filename (moshed_moshId.avi -> moshId)
            const moshId = filename.replace('moshed_', '').replace('.avi', '');
            
            // Show local progress bar
            this.showLocalProgress(moshId, `Starting ${format.toUpperCase()} conversion...`, 10);

            const response = await fetch(`/api/projects/${this.currentProjectData.id}/convert-mosh/${filename}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    format: format
                })
            });

            if (!response.ok) {
                throw new Error('Conversion failed');
            }

            const result = await response.json();
            
            // Track conversion progress via WebSocket
            if (result.conversion_id) {
                this.trackConversionProgress(result.conversion_id, format, moshId);
            }

        } catch (error) {
            console.error('Conversion error:', error);
            // Extract mosh ID for error display
            const moshId = filename.replace('moshed_', '').replace('.avi', '');
            this.showLocalProgress(moshId, 'Conversion failed', 0);
            alert('Conversion failed: ' + error.message);
        }
    }
    
    trackConversionProgress(conversionId, format, moshId) {
        // Add to moshes map so WebSocket updates are handled
        this.moshesMap.set(conversionId, { 
            status: 'processing', 
            progress: 0.1,
            isConversion: true,
            format: format,
            moshId: moshId
        });
        this.updateMoshesDisplay();
    }
    
    showLocalProgress(moshId, text, percentage) {
        console.log('showLocalProgress called:', { moshId, text, percentage });
        
        // Try both current results and history progress elements
        const progressElements = [
            document.getElementById(`conversion-progress-${moshId}`),
            document.getElementById(`history-conversion-progress-${moshId}`)
        ];
        
        const labelElements = [
            document.getElementById(`progress-label-${moshId}`),
            document.getElementById(`history-progress-label-${moshId}`)
        ];
        
        const barElements = [
            document.getElementById(`progress-bar-${moshId}`),
            document.getElementById(`history-progress-bar-${moshId}`)
        ];
        
        console.log('Found elements:', {
            progressElements: progressElements.filter(el => el !== null),
            labelElements: labelElements.filter(el => el !== null),
            barElements: barElements.filter(el => el !== null)
        });
        
        progressElements.forEach(el => {
            if (el) {
                el.style.display = percentage > 0 ? 'block' : 'none';
            }
        });
        
        labelElements.forEach(el => {
            if (el) {
                el.textContent = text;
            }
        });
        
        barElements.forEach(el => {
            if (el) {
                el.style.width = percentage + '%';
            }
        });
        
        // Hide progress after completion
        if (percentage >= 100) {
            setTimeout(() => {
                progressElements.forEach(el => {
                    if (el) el.style.display = 'none';
                });
            }, 2000);
        }
    }
    
    showConvertedFile(conversionId, format) {
        // Extract mosh ID from conversion ID (format: convert_moshed_moshid.avi_timestamp)
        const match = conversionId.match(/convert_moshed_(.+?)\.avi_/);
        if (!match) return;
        
        const moshId = match[1];
        
        // Update all convert buttons for this mosh ID and format
        const buttons = [
            document.getElementById(`${format}-btn-${moshId}`),
            document.getElementById(`history-${format}-btn-${moshId}`)
        ];
        
        buttons.forEach(button => {
            if (button) {
                button.classList.add('converted');
                button.innerHTML = `▶ Play ${format.toUpperCase()}`;
            }
        });
        
        // Also reload the current project to refresh all converted file statuses
        this.reloadCurrentProject();
    }
    
    updateConvertedFileUI(moshId, convertedFiles) {
        // Update MP4 button if file exists
        if (convertedFiles.mp4) {
            const mp4Btn = document.getElementById(`mp4-btn-${moshId}`);
            if (mp4Btn) {
                mp4Btn.classList.add('converted');
                mp4Btn.innerHTML = '▶ Play MP4';
            }
        }
        
        // Update WebM button if file exists
        if (convertedFiles.webm) {
            const webmBtn = document.getElementById(`webm-btn-${moshId}`);
            if (webmBtn) {
                webmBtn.classList.add('converted');
                webmBtn.innerHTML = '▶ Play WebM';
            }
        }
    }
    
    async checkExistingConvertedFilesForHistory(sessionId, moshId) {
        try {
            const response = await fetch(`/api/projects/${this.currentProjectData.id}/converted-files/${sessionId}/${moshId}`);
            if (!response.ok) {
                console.log(`API response not ok for mosh ${moshId}:`, response.status);
                return;
            }
            
            const data = await response.json();
            const convertedFiles = data.converted_files;

            // Update MP4 button if file exists
            if (convertedFiles.mp4) {
                const mp4Btn = document.getElementById(`history-mp4-btn-${moshId}`);
                if (mp4Btn) {
                    mp4Btn.classList.add('converted');
                    mp4Btn.innerHTML = '▶ MP4';
                }
            }
            
            // Update WebM button if file exists
            if (convertedFiles.webm) {
                const webmBtn = document.getElementById(`history-webm-btn-${moshId}`);
                if (webmBtn) {
                    webmBtn.classList.add('converted');
                    webmBtn.innerHTML = '▶ WebM';
                }
            }
            
        } catch (error) {
            console.error('Error checking converted files for history:', error);
        }
    }
    
    
    getSessionIdFromTimestamp(timestamp) {
        // Convert timestamp to session directory format (session_UNIX_TIMESTAMP)
        const date = new Date(timestamp);
        const unixTimestamp = Math.floor(date.getTime() / 1000);
        return `session_${unixTimestamp}`;
    }
    
    async handleMp4Action(filename, moshId) {
        // Check if MP4 already exists
        const mp4Btn = document.getElementById(`mp4-btn-${moshId}`) || document.getElementById(`history-mp4-btn-${moshId}`);
        if (mp4Btn && mp4Btn.classList.contains('converted')) {
            // MP4 exists, find it in session directories
            this.playConvertedFile(moshId, 'mp4');
        } else {
            // MP4 doesn't exist, convert it
            this.convertMosh(filename, 'mp4');
        }
    }
    
    async handleWebmAction(filename, moshId) {
        // Check if WebM already exists
        const webmBtn = document.getElementById(`webm-btn-${moshId}`) || document.getElementById(`history-webm-btn-${moshId}`);
        if (webmBtn && webmBtn.classList.contains('converted')) {
            // WebM exists, find it in session directories
            this.playConvertedFile(moshId, 'webm');
        } else {
            // WebM doesn't exist, convert it
            this.convertMosh(filename, 'webm');
        }
    }
    
    async playConvertedFile(moshId, format) {
        try {
            // Use the backend to find the file in the correct session directory
            const response = await fetch(`/api/projects/${this.currentProjectData.id}/play-converted/${moshId}/${format}`);
            if (response.ok) {
                const data = await response.json();
                if (data.file_path) {
                    window.open(data.file_path, '_blank');
                }
            }
        } catch (error) {
            console.error('Error playing converted file:', error);
        }
    }
    
    async deleteMosh(moshId) {
        if (!confirm('Are you sure you want to delete this mosh? This will remove all files (AVI, MP4, WebM, preview).')) {
            return;
        }
        
        try {
            // Find the session ID for this mosh (from current results)
            const currentMoshIds = Array.from(this.moshesMap.keys());
            if (!currentMoshIds.includes(moshId)) {
                alert('Cannot delete: This mosh is not from the current session.');
                return;
            }
            
            // Generate session ID from current time (approximation for current session)
            const sessionId = this.getCurrentSessionId();
            const response = await fetch(`/api/projects/${this.currentProjectData.id}/sessions/${sessionId}/mosh/${moshId}`, {
                method: 'DELETE'
            });
            
            if (response.ok) {
                // Remove from UI
                const previewItem = document.getElementById(`preview-${moshId}`);
                if (previewItem) {
                    previewItem.remove();
                }
                alert('Mosh deleted successfully!');
            } else {
                alert('Failed to delete mosh.');
            }
        } catch (error) {
            console.error('Error deleting mosh:', error);
            alert('Error deleting mosh: ' + error.message);
        }
    }
    
    async deleteMoshFromHistory(sessionId, moshId) {
        if (!confirm('Are you sure you want to delete this mosh? This will remove all files (AVI, MP4, WebM, preview).')) {
            return;
        }
        
        try {
            const response = await fetch(`/api/projects/${this.currentProjectData.id}/sessions/${sessionId}/mosh/${moshId}`, {
                method: 'DELETE'
            });
            
            if (response.ok) {
                // Reload the project to refresh history
                await this.reloadCurrentProject();
            } else {
                alert('Failed to delete mosh.');
            }
        } catch (error) {
            console.error('Error deleting mosh:', error);
            alert('Error deleting mosh: ' + error.message);
        }
    }
    
    async deleteSession(sessionId) {
        if (!confirm('Are you sure you want to delete this entire session? This will remove ALL moshes in this session permanently.')) {
            return;
        }
        
        try {
            const response = await fetch(`/api/projects/${this.currentProjectData.id}/sessions/${sessionId}`, {
                method: 'DELETE'
            });
            
            if (response.ok) {
                // Reload the project to refresh history
                await this.reloadCurrentProject();
            } else {
                alert('Failed to delete session.');
            }
        } catch (error) {
            console.error('Error deleting session:', error);
            alert('Error deleting session: ' + error.message);
        }
    }
    
    getCurrentSessionId() {
        // Generate session ID for current time
        const now = new Date();
        const unixTimestamp = Math.floor(now.getTime() / 1000);
        return `session_${unixTimestamp}`;
    }
    
    async reloadCurrentProject() {
        if (!this.currentProjectData) return;
        
        try {
            const response = await fetch(`/api/projects/${this.currentProjectData.id}`);
            if (response.ok) {
                const data = await response.json();
                this.currentProjectData = data.project;
                
                // Reload project data
                if (data.clips && data.clips.length > 0) {
                    this.createdClips = data.clips;
                    this.displayClips();
                }
                if (data.sessions && data.sessions.length > 0) {
                    this.moshSessions = data.sessions;
                    this.displayMoshHistory();
                    this.moshHistory.style.display = 'block';
                }
                if (data.scenes) this.detectedScenes = data.scenes;
            }
        } catch (error) {
            console.error('Failed to reload project:', error);
        }
    }
}

let app;

document.addEventListener('DOMContentLoaded', () => {
    app = new MoshrApp();
});