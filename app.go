package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"SpeedMap/pkg/cloud"
	"SpeedMap/pkg/history"
	"SpeedMap/pkg/notify"
	"SpeedMap/pkg/scanner"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct manages application-level state and Wails bridge integration
type App struct {
	ctx           context.Context
	activeScanner *scanner.Scanner
	scannerMu     sync.Mutex

	reportServerMu    sync.Mutex
	reportServerPort  int
	currentReportHTML string

	gdriveManager *cloud.GDriveManager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		gdriveManager: cloud.NewGDriveManager(),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	fmt.Println("[GO LOG] SpeedMap backend started successfully.")

	// Auto-prune old scan runs in background
	go func() {
		_ = history.AutoPruneHistory(20, 30)
	}()
}

// shutdown is called when the application closes to ensure all Chrome processes are killed
func (a *App) shutdown(ctx context.Context) {
	fmt.Println("[GO LOG] Application shutting down. Canceling active scans...")
	a.CancelScan()
}

// OpenURL opens the given URL in the user's default web browser
func (a *App) OpenURL(rawURL string) {
	fmt.Printf("[GO LOG] OpenURL called for: %s\n", rawURL)
	if a.ctx != nil && rawURL != "" {
		runtime.BrowserOpenURL(a.ctx, rawURL)
	}
}

// PlayNotificationSound plays embedded Pikachu MP3 via OS player.
// WebKitGTK often cannot decode MP3 in <audio>, so we play through Pulse/CoreAudio
// kind: "page" | "full"
func (a *App) PlayNotificationSound(kind string) error {
	file := "frontend/pika-page.mp3"
	if kind == "full" {
		file = "frontend/pika-full.mp3"
	}

	data, err := assets.ReadFile(file)
	if err != nil {
		fmt.Printf("[GO LOG] PlayNotificationSound read %s: %v\n", file, err)
		return fmt.Errorf("sound asset missing: %w", err)
	}

	tmp, err := os.CreateTemp("", "speedmap-*-"+filepath.Base(file))
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	tmp.Close()

	var cmd *exec.Cmd
	if _, err := exec.LookPath("afplay"); err == nil {
		cmd = exec.Command("afplay", tmpPath)
	} else if _, err := exec.LookPath("ffplay"); err == nil {
		cmd = exec.Command("ffplay", "-nodisp", "-autoexit", "-loglevel", "quiet", tmpPath)
	} else if _, err := exec.LookPath("mpg123"); err == nil {
		cmd = exec.Command("mpg123", "-q", tmpPath)
	} else {
		os.Remove(tmpPath)
		return fmt.Errorf("no audio player found (afplay/ffplay/mpg123)")
	}

	fmt.Printf("[GO LOG] PlayNotificationSound kind=%s via %s\n", kind, cmd.Path)
	go func() {
		defer os.Remove(tmpPath)
		if err := cmd.Run(); err != nil {
			fmt.Printf("[GO LOG] PlayNotificationSound error: %v\n", err)
		}
	}()

	return nil
}

// SendSystemNotification sends a native OS notification with the application icon and sound
func (a *App) SendSystemNotification(title, subtitle, message string) {
	fmt.Printf("[GO LOG] SendSystemNotification: title=%q subtitle=%q message=%q\n", title, subtitle, message)
	notify.Send(title, subtitle, message)
}
