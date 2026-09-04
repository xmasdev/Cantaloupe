package main

import (
	"context"
	"fmt"
	"math/bits"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	clientengine "github.com/xmasdev/Cantaloupe/engine"
	"github.com/xmasdev/Cantaloupe/engine/metainfo"
	"github.com/xmasdev/Cantaloupe/engine/types"
)

// App struct
type App struct {
	ctx          context.Context
	mu           sync.RWMutex
	jobs         map[string]*downloadJob
	engine       *clientengine.Engine
	torrentPath  string
	outputDir    string
	status       DownloadStatus
	verifyPieces bool
}

type downloadJob struct {
	engine        *clientengine.Engine
	status        DownloadStatus
	stopRequested bool
	lastBytes     int64
	lastUploaded  int64
	lastAt        time.Time
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
	DownloadSpeed   int64  `json:"downloadSpeed"`
	UploadSpeed     int64  `json:"uploadSpeed"`
	UpdatedAt       string `json:"updatedAt"`
}

type TorrentFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

type TorrentPeer struct {
	Address string `json:"address"`
	Pieces  int    `json:"pieces"`
	State   string `json:"state"`
}

type TorrentTracker struct {
	URL    string `json:"url"`
	Status string `json:"status"`
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
	a.verifyPieces = true
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

func (a *App) GetVerifyPieces() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.verifyPieces
}

func (a *App) SetVerifyPieces(enabled bool) {
	a.mu.Lock()
	a.verifyPieces = enabled
	jobs := make([]*downloadJob, 0, len(a.jobs))
	for _, job := range a.jobs {
		jobs = append(jobs, job)
	}
	a.mu.Unlock()
	for _, job := range jobs {
		if job.engine != nil && job.engine.Storage != nil {
			job.engine.Storage.SetVerifyPieces(enabled)
		}
	}
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
	phase, phaseError := engine.Status()
	if phase != "" {
		status.State = phase
		status.Error = phaseError
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
		a.mu.Lock()
		status, engine := job.status, job.engine
		status = downloadStatus(engine, status)
		now := time.Now()
		if !job.lastAt.IsZero() {
			seconds := now.Sub(job.lastAt).Seconds()
			if seconds > 0 {
				if delta := status.DownloadedBytes - job.lastBytes; delta > 0 {
					status.DownloadSpeed = int64(float64(delta) / seconds)
				}
				if engine != nil {
					uploaded := engine.UploadedBytes.Load()
					if delta := uploaded - job.lastUploaded; delta > 0 {
						status.UploadSpeed = int64(float64(delta) / seconds)
					}
				}
			}
		}
		if engine != nil {
			job.lastUploaded = engine.UploadedBytes.Load()
		}
		job.lastBytes, job.lastAt, job.status = status.DownloadedBytes, now, status
		a.mu.Unlock()
		statuses = append(statuses, status)
	}
	return statuses
}

func (a *App) torrentMetadata(torrentPath string) (*types.TorrentMetadata, *clientengine.Engine, error) {
	a.mu.RLock()
	job := a.jobs[torrentPath]
	a.mu.RUnlock()
	if job != nil && job.engine != nil {
		return job.engine.Metadata, job.engine, nil
	}
	data, err := os.ReadFile(torrentPath)
	if err != nil {
		return nil, nil, err
	}
	metadata, err := metainfo.Parse(data)
	return metadata, nil, err
}

func (a *App) GetTorrentFiles(torrentPath string) ([]TorrentFile, error) {
	metadata, _, err := a.torrentMetadata(torrentPath)
	if err != nil {
		return nil, err
	}
	if len(metadata.Info.Files) == 0 {
		return []TorrentFile{{Path: metadata.Info.Name, Size: metadata.Info.Length}}, nil
	}
	files := make([]TorrentFile, 0, len(metadata.Info.Files))
	for _, file := range metadata.Info.Files {
		pathParts := append([]string{metadata.Info.Name}, file.Path...)
		files = append(files, TorrentFile{Path: filepath.Join(pathParts...), Size: file.Length})
	}
	return files, nil
}

func (a *App) GetTorrentPeers(torrentPath string) ([]TorrentPeer, error) {
	_, engine, err := a.torrentMetadata(torrentPath)
	if err != nil {
		return nil, err
	}
	if engine == nil {
		return []TorrentPeer{}, nil
	}
	peers := make([]TorrentPeer, 0, len(engine.Peers))
	for _, session := range engine.Peers {
		if session == nil {
			continue
		}
		pieceCount := 0
		for _, value := range session.RemoteBitfield {
			pieceCount += bits.OnesCount8(value)
		}
		state := "Ready"
		if session.Choked {
			state = "Choked"
		}
		peers = append(peers, TorrentPeer{Address: session.RemoteAddress(), Pieces: pieceCount, State: state})
	}
	return peers, nil
}

func (a *App) GetTorrentTrackers(torrentPath string) ([]TorrentTracker, error) {
	metadata, _, err := a.torrentMetadata(torrentPath)
	if err != nil {
		return nil, err
	}
	urls := make([]string, 0, 1)
	if metadata.Announce != "" {
		urls = append(urls, metadata.Announce)
	}
	for _, tier := range metadata.AnnounceList {
		urls = append(urls, tier...)
	}
	seen := make(map[string]bool)
	trackers := make([]TorrentTracker, 0, len(urls))
	for _, trackerURL := range urls {
		trackerURL = strings.TrimSpace(trackerURL)
		if trackerURL == "" || seen[trackerURL] {
			continue
		}
		seen[trackerURL] = true
		trackers = append(trackers, TorrentTracker{URL: trackerURL, Status: "Configured"})
	}
	return trackers, nil
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
		if existing.status.State == "downloading" || existing.status.State == "discovering" || existing.status.State == "connecting" || existing.status.State == "negotiating" {
			return fmt.Errorf("torrent is already downloading")
		}
		if err := existing.engine.Stop(); err != nil {
			return err
		}
		a.mu.Lock()
		existing.status.State = "starting"
		existing.status.Error = ""
		existing.status.OutputDir = outputDir
		existing.stopRequested = false
		a.engine, a.status = existing.engine, existing.status
		a.mu.Unlock()
		a.runTorrent(existing)
		return nil
	}

	engine, err := clientengine.NewEngine(torrentPath, outputDir)
	if err != nil {
		a.setError(err)
		return err
	}
	a.mu.RLock()
	verifyPieces := a.verifyPieces
	a.mu.RUnlock()
	engine.Storage.SetVerifyPieces(verifyPieces)

	status := DownloadStatus{State: "downloading", TorrentPath: torrentPath, OutputDir: outputDir, Name: engine.Metadata.Info.Name, TotalBytes: engine.Download.TotalLength(), PieceCount: len(engine.Download.Pieces), Port: engine.Port, UpdatedAt: time.Now().Format(time.RFC3339)}
	job := &downloadJob{engine: engine, status: status}
	a.mu.Lock()
	if a.jobs == nil {
		a.jobs = make(map[string]*downloadJob)
	}
	a.jobs[torrentPath] = job
	a.engine, a.status = engine, status
	a.mu.Unlock()
	a.runTorrent(job)
	return nil
}

func (a *App) runTorrent(job *downloadJob) {
	go func() {
		err := job.engine.Start()
		latest := downloadStatus(job.engine, job.status)
		a.mu.Lock()
		defer a.mu.Unlock()
		status := latest
		// StopTorrent marks the job before stopping the engine. Engine.Stop
		// intentionally returns a normal stop error so a paused job can be
		// resumed later; never expose that internal stop as a download error.
		wasPaused := job.stopRequested || job.status.State == "paused" || job.status.State == "stopping"
		if wasPaused {
			status.State = "paused"
		} else if err != nil {
			status.State = "error"
			status.Error = err.Error()
		} else if status.State == "complete" {
			// Keep the terminal state reported by the engine.
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
		if a.engine == job.engine {
			a.status = status
		}
	}()
}

func (a *App) StopDownload() {
	a.mu.Lock()
	engine := a.engine
	if engine != nil {
		a.status.State = "stopping"
	}
	a.mu.Unlock()
	if engine != nil {
		_ = engine.Stop()
	}
}

func (a *App) StopTorrent(torrentPath string) {
	a.mu.Lock()
	job := a.jobs[torrentPath]
	if job != nil {
		job.stopRequested = true
		job.status.State = "paused"
	}
	a.mu.Unlock()
	if job != nil {
		_ = job.engine.Stop()
	}
}

func (a *App) RemoveTorrent(torrentPath string) {
	a.mu.Lock()
	job := a.jobs[torrentPath]
	delete(a.jobs, torrentPath)
	if job != nil && a.engine == job.engine {
		a.engine = nil
		a.status = DownloadStatus{State: "idle", UpdatedAt: time.Now().Format(time.RFC3339)}
	}
	a.mu.Unlock()
	if job != nil {
		_ = job.engine.Stop()
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
