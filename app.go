package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	clientengine "github.com/xmasdev/Cantaloupe/engine"
)

// App struct
type App struct {
	ctx         context.Context
	mu          sync.RWMutex
	jobs        map[string]*downloadJob
	engine      *clientengine.Engine
	torrentPath string
	outputDir   string
	status      DownloadStatus
}

type downloadJob struct {
	engine *clientengine.Engine
	status DownloadStatus
}

type DownloadStatus struct {
	State           string `json:"state"`
	Name            string `json:"name"`
	TorrentPath     string `json:"torrentPath"`
	OutputDir       string `json:"outputDir"`
	TotalBytes      int64  `json:"totalBytes"`
	DownloadedBytes int64  `json:"downloadedBytes"`
	PieceCount      int    `json:"pieceCount"`
	CompletedPieces int    `json:"completedPieces"`
	PeerCount       int    `json:"peerCount"`
	Port            int    `json:"port"`
	Error           string `json:"error"`
	UpdatedAt       string `json:"updatedAt"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.jobs = make(map[string]*downloadJob)
	a.status = DownloadStatus{State: "idle", UpdatedAt: time.Now().Format(time.RFC3339)}
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) SelectTorrent() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Choose a .torrent file",
		Filters: []runtime.FileFilter{{DisplayName: "Torrent files (*.torrent)", Pattern: "*.torrent"}},
	})
	if err != nil || path == "" {
		return "", err
	}

	a.mu.Lock()
	a.torrentPath = path
	a.mu.Unlock()
	return path, nil
}

func (a *App) SelectOutputDirectory() (string, error) {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                "Choose download location",
		CanCreateDirectories: true,
	})
	if err != nil || path == "" {
		return "", err
	}

	a.mu.Lock()
	a.outputDir = path
	a.mu.Unlock()
	return path, nil
}

func (a *App) SetTorrentPath(path string) {
	a.mu.Lock()
	a.torrentPath = path
	a.mu.Unlock()
}

func (a *App) SetOutputDirectory(path string) {
	a.mu.Lock()
	a.outputDir = path
	a.mu.Unlock()
}

func (a *App) GetDownloadStatus() DownloadStatus {
	a.mu.RLock()
	engine := a.engine
	status := a.status
	a.mu.RUnlock()

	if engine == nil {
		return status
	}
	return downloadStatus(engine, status)
}

func downloadStatus(engine *clientengine.Engine, status DownloadStatus) DownloadStatus {
	if engine == nil || engine.Download == nil || engine.Metadata == nil {
		return status
	}

	var downloaded int64
	var completed int
	for i := range engine.Download.Pieces {
		piece := &engine.Download.Pieces[i]
		downloaded += piece.DownloadedBytes()
		if piece.Complete() {
			completed++
		}
	}

	status.Name = engine.Metadata.Info.Name
	status.TotalBytes = engine.Download.TotalLength()
	status.DownloadedBytes = downloaded
	status.PieceCount = len(engine.Download.Pieces)
	status.CompletedPieces = completed
	status.PeerCount = len(engine.Peers)
	status.Port = engine.Port
	status.UpdatedAt = time.Now().Format(time.RFC3339)
	return status
}

func (a *App) GetDownloadStatuses() []DownloadStatus {
	a.mu.RLock()
	jobs := make([]*downloadJob, 0, len(a.jobs))
	for _, job := range a.jobs {
		jobs = append(jobs, job)
	}
	a.mu.RUnlock()

	statuses := make([]DownloadStatus, 0, len(jobs))
	for _, job := range jobs {
		a.mu.RLock()
		status, engine := job.status, job.engine
		a.mu.RUnlock()
		statuses = append(statuses, downloadStatus(engine, status))
	}
	return statuses
}

func (a *App) StartDownload() error {
	a.mu.Lock()
	torrentPath, outputDir := a.torrentPath, a.outputDir
	a.mu.Unlock()
	return a.StartTorrent(torrentPath, outputDir)
}

func (a *App) StartTorrent(torrentPath string, outputDir string) error {
	if torrentPath == "" || outputDir == "" {
		return fmt.Errorf("choose a torrent file and download location first")
	}
	a.mu.RLock()
	existing := a.jobs[torrentPath]
	a.mu.RUnlock()
	if existing != nil {
		return fmt.Errorf("torrent is already in the library")
	}

	engine, err := clientengine.NewEngine(torrentPath, outputDir)
	if err != nil {
		a.setError(err)
		return err
	}

	status := DownloadStatus{State: "downloading", TorrentPath: torrentPath, OutputDir: outputDir, Name: engine.Metadata.Info.Name, TotalBytes: engine.Download.TotalLength(), PieceCount: len(engine.Download.Pieces), Port: engine.Port, UpdatedAt: time.Now().Format(time.RFC3339)}
	job := &downloadJob{engine: engine, status: status}
	a.mu.Lock()
	if a.jobs == nil {
		a.jobs = make(map[string]*downloadJob)
	}
	a.jobs[torrentPath] = job
	a.engine, a.status = engine, status
	a.mu.Unlock()

	go func() {
		err := engine.Start()
		latest := downloadStatus(engine, status)
		a.mu.Lock()
		defer a.mu.Unlock()
		status := latest
		wasStopping := job.status.State == "stopping"
		if status.State == "stopping" {
			status.State = "idle"
		} else if wasStopping {
			status.State = "idle"
		} else if err != nil {
			status.State = "error"
			status.Error = err.Error()
		} else {
			if status.Error == "" && status.DownloadedBytes >= status.TotalBytes {
				status.State = "complete"
			} else if status.State != "error" {
				status.State = "error"
				if status.Error == "" {
					status.Error = "download stopped unexpectedly"
				}
			}
		}
		job.status = status
		if a.engine == engine {
			a.status = status
		}
	}()
	return nil
}

func (a *App) StopDownload() {
	a.mu.Lock()
	engine := a.engine
	if engine != nil {
		a.status.State = "stopping"
	}
	a.mu.Unlock()
	if engine != nil {
		_ = engine.Close()
	}
}

func (a *App) StopTorrent(torrentPath string) {
	a.mu.Lock()
	job := a.jobs[torrentPath]
	if job != nil {
		job.status.State = "stopping"
	}
	a.mu.Unlock()
	if job != nil {
		_ = job.engine.Close()
	}
}

func (a *App) setError(err error) {
	a.mu.Lock()
	a.status.State = "error"
	a.status.Error = err.Error()
	a.status.UpdatedAt = time.Now().Format(time.RFC3339)
	a.mu.Unlock()
}

func (a *App) DefaultDownloadDirectory() string {
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "Downloads")
	}
	return ""
}
