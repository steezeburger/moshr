# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Moshr is a video datamoshing tool built in Go with a web interface. It manipulates video data to create aggressive glitch effects by removing I-frames and duplicating P-frames in AVI files. The tool now produces dramatic, highly visible datamoshing effects.

## Core Architecture

- **cmd/moshr/main.go**: Entry point supporting CLI and web modes
- **internal/server/**: Gin-based web server with WebSocket support for real-time updates
- **internal/video/**: Core video processing engine
  - `mosher.go`: Main datamoshing logic with proper AVI structure parsing (RIFF → LIST movi → video frames)
  - `converter.go`: Video format conversion utilities including moshed AVI to MP4/WebM conversion
  - `analyzer.go`: Video analysis and frame detection
  - `scene.go`: Scene detection algorithms
  - `frames.go`: Frame extraction and clip creation with proper AVI encoding
- **internal/batch/**: Batch processing for multiple variations with debug logging
- **internal/effects/**: Effect implementations (datamosh, glitch) with aggressive parameters
- **internal/project/**: Project management and persistence
- **web/**: Static frontend assets (HTML, CSS, JS) with conversion UI
- **bin/**: Build output directory (gitignored)

## Development Commands

Use `just` for common development tasks:

```bash
# Build the application
just build

# Run in web mode (default port 8080)
just run

# Run with custom port
just run-port 3000

# Install dependencies
just deps

# Clean build artifacts and temp files
just clean

# Development server with live reload (requires air)
just dev

# Format code
just fmt

# Run tests
just test

# Lint code (requires golangci-lint)
just lint

# Build for all platforms
just build-all
```

Manual commands:
```bash
# Build manually
go build -o bin/moshr cmd/moshr/main.go

# Run with live reload during development
go run cmd/moshr/main.go -web
```

## Key Technical Details

- Uses Gin framework for HTTP routing and middleware
- WebSocket hub pattern for real-time progress updates
- **Proper AVI file manipulation**: Correctly parses RIFF → LIST movi → video frame structure
- **Aggressive moshing parameters**: 
  - Removes 50% of video frames when I-frame removal enabled
  - Duplicates frames 15-40+ times (vs original 1-5)
  - I-frame removal threshold: 0.2 intensity (vs 0.5)
  - P-frame duplication threshold: 0.1 intensity (vs 0.3)
- **Enhanced chunk detection**: Supports multiple AVI chunk types (`00dc`, `01dc`, etc.)
- Project-based file organization with uploads/, clips/, moshes/, timeline/ directories
- **Clip extraction with proper AVI encoding**: Forces libxvid codec for moshable clips
- **Moshed file conversion**: Convert broken AVI output to stable MP4/WebM formats
- Frontend communicates via REST API and WebSocket for progress tracking

## Video Processing Pipeline

1. **Upload**: Video files uploaded to project directory
2. **Convert**: Convert to proper AVI format if needed (internal/video/converter.go)
3. **Analyze**: Analyze video structure and detect scenes (internal/video/analyzer.go, scene.go)
4. **Extract**: Create clips with proper AVI encoding (internal/video/frames.go)
5. **Mosh**: Apply aggressive datamoshing effects by manipulating video frames in LIST movi chunks (internal/video/mosher.go)
6. **Convert Output**: Convert moshed AVI files to stable MP4/WebM formats for sharing
7. **Output**: Files saved in project moshes/ directory with session organization

## Moshing Algorithm

The core moshing process:

1. **Parse AVI Structure**: Properly navigate RIFF → LIST chunks → movi chunk → video frames
2. **Frame Detection**: Identify video chunks (`00dc`, `01dc`, etc.) vs audio chunks (`01wb`)
3. **Aggressive Removal**: Remove every 2nd video frame when I-frame removal enabled
4. **Massive Duplication**: Duplicate remaining frames 15-40+ times with optional corruption
5. **Byte Corruption**: Corrupt every 3rd duplicated frame for extra glitch artifacts
6. **Debug Logging**: Track chunks processed, removed, and duplicated

## Conversion Features

- **MP4 Output**: H.264 video + AAC audio, high quality (CRF 18), web-optimized
- **WebM Output**: VP9 video + Vorbis audio, good quality (CRF 30), smaller files
- **Preserved Effects**: All glitch artifacts maintained in stable format
- **Frontend Integration**: Convert buttons on all moshed video previews

## File Structure

```
projects/
├── project_id/
│   ├── project.json
│   ├── original_filename.ext
│   ├── converted.avi
│   ├── clips/
│   │   ├── clip_start_end.avi
│   │   └── clips.json
│   ├── moshes/
│   │   ├── session_timestamp/
│   │   │   ├── moshed_jobid.avi
│   │   │   ├── moshed_jobid_converted.mp4
│   │   │   ├── moshed_jobid_converted.webm
│   │   │   ├── preview_jobid.jpg
│   │   │   └── session.json
│   └── timeline/
│       └── frame_xxxxxx.jpg
```

## Current Moshing Parameters

- **Single Mosh**: Up to 30 duplications per frame (intensity * 25 + 5)
- **Batch Presets**: 15, 25, 40 duplications across three intensity levels
- **Datamosh Effect**: Up to 30 duplications with 0.2 I-frame threshold
- **Glitch Effect**: Up to 38 random duplications with 0.8 chance per frame
- **Frame Removal**: 50% removal rate for dramatic effects

## Important Notes

- **AVI Format Required**: Moshing only works on proper AVI files with libxvid encoding
- **Clip Conversion**: Clips are automatically converted to proper AVI format (no more `-c copy`)
- **Debug Output**: Moshing process logs chunks processed for troubleshooting
- **File Sizes**: Expect dramatic file size changes (removal = smaller, duplication = larger)
- **Conversion Recommended**: Use MP4/WebM conversion for sharing and compatibility
- **No Testing Framework**: Currently no automated tests implemented
- **Binary Location**: Built to `bin/` directory and gitignored

## Known Issues

Current UI/UX issues that need addressing:
- **Batch Generation UI**: Incorrect rendering - shows decreasing count instead of stable count
- **Refresh Required**: Must refresh page after MP4/WebM generation to show convert buttons
- **Session Management**: Must refresh after session deletion to update session list
- **Scene Detection**: Scene detection function is not working
- **Keyframes Feature**: Show keyframes feature purpose is unclear

## Future Enhancements

Features identified for future development:
- **Mosh Stitching**: Ability to combine multiple moshes into one sequence
- **Video Masks**: Apply masking to specific areas of video during moshing
- **UI Improvements**: Fix refresh requirements and batch generation display

## Troubleshooting

- **No Visible Effects**: Check debug logs for "Total video chunks: 0" - indicates AVI parsing issue
- **Conversion Errors**: Ensure ffmpeg is installed and accessible
- **File Playback Issues**: Use the conversion feature to create stable MP4/WebM files
- **Clip Moshing Fails**: Clips are now properly converted to AVI - old clips may need recreation