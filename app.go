package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx      context.Context
	settings AppSettings
	data     AppData
}

// AppSettings defines user-configurable settings
type AppSettings struct {
	AutoChangeEnabled   bool     `json:"auto_change_enabled"`
	ChangeIntervalHours int      `json:"change_interval_hours"`
	DownloadSources     []string `json:"download_sources"`
	MaxWallpapers       int      `json:"max_wallpapers"`
}

// WallpaperInfo holds metadata about a downloaded wallpaper
type WallpaperInfo struct {
	ID           string    `json:"id"`
	Filename     string    `json:"filename"`
	Filepath     string    `json:"filepath"`
	LocalURL     string    `json:"local_url"`
	DownloadDate time.Time `json:"download_date"`
	SourceURL    string    `json:"source_url"`
	FileSize     int64     `json:"file_size"`
}

// AppData holds the application's runtime data
type AppData struct {
	Wallpapers []WallpaperInfo `json:"wallpapers"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Load settings and wallpapers from disk on startup
	a.loadSettings()
	a.loadWallpapers()

	// Start the background wallpaper changer
	go a.startAutoChanger()
}

// --- Exposed Go Methods for Svelte ---

// GetWallpapers returns the list of saved wallpapers
func (a *App) GetWallpapers() []WallpaperInfo {
	// Update local URLs for webview access
	for i := range a.data.Wallpapers {
		a.data.Wallpapers[i].LocalURL = "file://" + a.data.Wallpapers[i].Filepath
	}
	return a.data.Wallpapers
}

// GetWallpaperAsBase64 returns wallpaper as base64 data URL for preview
func (a *App) GetWallpaperAsBase64(filepath string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", filepath)
	}

	// Read the file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	// Encode to base64 with proper data URI prefix
	encoded := base64.StdEncoding.EncodeToString(data)
	dataURI := "data:image/jpeg;base64," + encoded

	return dataURI, nil
}

// GetSettings returns the current application settings
func (a *App) GetSettings() AppSettings {
	return a.settings
}

// UpdateSettings saves new settings and restarts the auto-changer
func (a *App) UpdateSettings(newSettings AppSettings) error {
	a.settings = newSettings
	return a.saveSettings()
}

// DownloadAndSetWallpaper fetches a new wallpaper, sets it, and saves it
func (a *App) DownloadAndSetWallpaper() (*WallpaperInfo, error) {
	for _, url := range a.settings.DownloadSources {
		info, err := a.downloadFile(url)
		if err != nil {
			fmt.Printf("Failed to download from %s: %v\n", url, err)
			continue
		}

		err = a.SetWallpaper(info.Filepath)
		if err != nil {
			fmt.Printf("Failed to set wallpaper %s: %v\n", info.Filepath, err)
			continue
		}

		a.addWallpaper(*info)
		wailsruntime.EventsEmit(a.ctx, "wallpaperChanged", *info)
		return info, nil
	}
	return nil, fmt.Errorf("all download sources failed")
}

// SetWallpaper sets the desktop background from a given file path
func (a *App) SetWallpaper(filepath string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Use PowerShell for better Windows wallpaper setting
		psScript := fmt.Sprintf(`
			Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; 
			public class Wallpaper { 
				[DllImport("user32.dll", CharSet=CharSet.Auto)] 
				public static extern int SystemParametersInfo(int uAction, int uParam, string lpvParam, int fuWinIni); 
			}'; 
			[Wallpaper]::SystemParametersInfo(20, 0, '%s', 3)
		`, filepath)
		cmd = exec.Command("powershell", "-Command", psScript)
	case "darwin":
		cmd = exec.Command("osascript", "-e", fmt.Sprintf(`tell application "Finder" to set desktop picture to POSIX file "%s"`, filepath))
	case "linux":
		// Try multiple Linux desktop environments
		commands := [][]string{
			{"gsettings", "set", "org.gnome.desktop.background", "picture-uri", "file://" + filepath},
			{"feh", "--bg-scale", filepath},
			{"nitrogen", "--set-scaled", filepath},
		}

		for _, cmdArgs := range commands {
			cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)
			if cmd.Run() == nil {
				return nil
			}
		}
		return fmt.Errorf("no suitable wallpaper command found")
	}

	return cmd.Run()
}

// DeleteWallpaper removes a wallpaper file and its metadata
func (a *App) DeleteWallpaper(id string) error {
	var newWallpapers []WallpaperInfo
	var deletedFile string

	for _, wp := range a.data.Wallpapers {
		if wp.ID == id {
			deletedFile = wp.Filepath
		} else {
			newWallpapers = append(newWallpapers, wp)
		}
	}

	if deletedFile != "" {
		os.Remove(deletedFile)
		a.data.Wallpapers = newWallpapers
		a.saveWallpapers()
		wailsruntime.EventsEmit(a.ctx, "wallpapersUpdated", a.data.Wallpapers)
	}

	return nil
}

// GetWallpaperDirectory returns the directory where wallpapers are stored
func (a *App) GetWallpaperDirectory() string {
	return a.getWallpaperDir()
}

// OpenWallpaperDirectory opens the wallpaper directory in file explorer
func (a *App) OpenWallpaperDirectory() error {
	dir := a.getWallpaperDir()
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	case "linux":
		cmd = exec.Command("xdg-open", dir)
	}

	return cmd.Run()
}

// --- Internal Helper Functions ---

// getWallpaperDir gets the directory where wallpapers are stored
func (a *App) getWallpaperDir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, "Pictures", "WallpaperEngine")
	os.MkdirAll(dir, os.ModePerm)
	return dir
}

// downloadFile downloads a file from a URL to the wallpaper directory
func (a *App) downloadFile(url string) (*WallpaperInfo, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "WallpaperEngine/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Generate unique ID and filename
	id := generateID()
	filename := fmt.Sprintf("wallpaper_%d_%s.jpg", time.Now().Unix(), id[:8])
	filepath := filepath.Join(a.getWallpaperDir(), filename)

	out, err := os.Create(filepath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	size, err := io.Copy(out, resp.Body)
	if err != nil {
		return nil, err
	}

	// Validate minimum file size (50KB)
	if size < 50000 {
		os.Remove(filepath)
		return nil, fmt.Errorf("file too small: %d bytes", size)
	}

	return &WallpaperInfo{
		ID:           id,
		Filename:     filename,
		Filepath:     filepath,
		LocalURL:     "", // Will be set in GetWallpapers
		DownloadDate: time.Now(),
		SourceURL:    url,
		FileSize:     size,
	}, nil
}

// generateID creates a random ID
func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}

// addWallpaper adds wallpaper metadata and saves the list
func (a *App) addWallpaper(info WallpaperInfo) {
	a.data.Wallpapers = append(a.data.Wallpapers, info)

	// Sort wallpapers by date, newest first
	sort.Slice(a.data.Wallpapers, func(i, j int) bool {
		return a.data.Wallpapers[i].DownloadDate.After(a.data.Wallpapers[j].DownloadDate)
	})

	// Keep only max wallpapers
	if len(a.data.Wallpapers) > a.settings.MaxWallpapers {
		// Remove oldest wallpapers
		for i := a.settings.MaxWallpapers; i < len(a.data.Wallpapers); i++ {
			os.Remove(a.data.Wallpapers[i].Filepath)
		}
		a.data.Wallpapers = a.data.Wallpapers[:a.settings.MaxWallpapers]
	}

	a.saveWallpapers()
}

// --- Persistence ---

func (a *App) getConfigPath(filename string) string {
	configDir, _ := os.UserConfigDir()
	appDir := filepath.Join(configDir, "WallpaperEngine")
	os.MkdirAll(appDir, os.ModePerm)
	return filepath.Join(appDir, filename)
}

func (a *App) saveSettings() error {
	data, err := json.MarshalIndent(a.settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.getConfigPath("settings.json"), data, 0644)
}

func (a *App) loadSettings() {
	data, err := os.ReadFile(a.getConfigPath("settings.json"))
	if err == nil {
		json.Unmarshal(data, &a.settings)
	} else {
		// Default settings with high-quality wallpaper sources
		a.settings = AppSettings{
			AutoChangeEnabled:   true,
			ChangeIntervalHours: 1,
			MaxWallpapers:       20,
			DownloadSources: []string{
				// 4K Sources
				"https://source.unsplash.com/3840x2160/landscape",
				"https://source.unsplash.com/3840x2160/nature",
				"https://source.unsplash.com/3840x2160/mountain",
				"https://source.unsplash.com/3840x2160/forest",
				"https://source.unsplash.com/3840x2160/ocean",
				// 2K Sources
				"https://source.unsplash.com/2560x1440/architecture",
				"https://source.unsplash.com/2560x1440/city",
				"https://source.unsplash.com/2560x1440/space",
				// Picsum for variety
				"https://picsum.photos/3840/2160",
				"https://picsum.photos/2560/1440",
			},
		}
		a.saveSettings()
	}
}

func (a *App) saveWallpapers() {
	data, _ := json.MarshalIndent(a.data, "", "  ")
	os.WriteFile(a.getConfigPath("wallpapers.json"), data, 0644)
}

func (a *App) loadWallpapers() {
	data, err := os.ReadFile(a.getConfigPath("wallpapers.json"))
	if err == nil {
		json.Unmarshal(data, &a.data)
		// Clean up missing files
		var validWallpapers []WallpaperInfo
		for _, wp := range a.data.Wallpapers {
			if _, err := os.Stat(wp.Filepath); err == nil {
				validWallpapers = append(validWallpapers, wp)
			}
		}
		a.data.Wallpapers = validWallpapers
	}
}

// --- Background Service ---

func (a *App) startAutoChanger() {
	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	go func() {
		lastChange := time.Now()
		for range ticker.C {
			if a.settings.AutoChangeEnabled {
				interval := time.Duration(a.settings.ChangeIntervalHours) * time.Hour
				if time.Since(lastChange) >= interval {
					fmt.Printf("Auto-changing wallpaper at %s\n", time.Now().Format("15:04:05"))
					_, err := a.DownloadAndSetWallpaper()
					if err != nil {
						fmt.Printf("Auto-change failed: %v\n", err)
					}
					lastChange = time.Now()
				}
			}
		}
	}()
}
