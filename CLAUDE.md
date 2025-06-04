# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Moshr is a video datamoshing tool built in Go with a web interface. It manipulates video data to create glitch effects by removing I-frames and duplicating P-frames in AVI files.

## Core Architecture

- **cmd/moshr/main.go**: Entry point supporting CLI and web modes
- **internal/server/**: Gin-based web server with WebSocket support for real-time updates
- **internal/video/**: Core video processing engine
  - `mosher.go`: Main datamoshing logic that manipulates AVI chunks
  - `converter.go`: Video format conversion utilities
  - `analyzer.go`: Video analysis and frame detection
  - `scene.go`: Scene detection algorithms
- **internal/batch/**: Batch processing for multiple variations
- **internal/effects/**: Effect implementations (datamosh, glitch)
- **internal/project/**: Project management and persistence
- **web/**: Static frontend assets (HTML, CSS, JS)

## Development Commands

```bash
# Build the application
go build -o moshr cmd/moshr/main.go

# Run in web mode (default port 8080)
./moshr -web

# Run with custom port
./moshr -web -port=3000

# Install dependencies
go mod tidy

# Run with live reload during development
go run cmd/moshr/main.go -web
```

## Key Technical Details

- Uses Gin framework for HTTP routing and middleware
- WebSocket hub pattern for real-time progress updates
- Direct AVI file manipulation at byte level for datamoshing effects
- Project-based file organization with uploads/, output/, and projects/ directories
- No testing framework currently implemented
- Frontend communicates via REST API and WebSocket for progress tracking

## Video Processing Pipeline

1. Upload video files to project
2. Convert to AVI format if needed (internal/video/converter.go)
3. Analyze video structure and detect scenes (internal/video/analyzer.go, scene.go)
4. Apply datamoshing effects by manipulating I-frames and P-frames (internal/video/mosher.go)
5. Generate output files in output/ directory
