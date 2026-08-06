package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)



// Default SpeedMap Public OAuth Client Credentials for Google Drive
const (
	DefaultClientID     = "982345678901-speedmapgdriveclient.apps.googleusercontent.com"
	DefaultClientSecret = "GOCSPX-speedmapdefaultsecretkey123"
	RedirectURI         = "http://127.0.0.1:8585/callback"
	OAuthAuthURL        = "https://accounts.google.com/o/oauth2/v2/auth"
	OAuthTokenURL       = "https://oauth2.googleapis.com/token"
	DriveAPIBase        = "https://www.googleapis.com/drive/v3"
	DriveUploadBase     = "https://www.googleapis.com/upload/drive/v3"
	ScopeDriveFile      = "https://www.googleapis.com/auth/drive.file"
	ScopeUserInfo       = "https://www.googleapis.com/auth/userinfo.email"
)

// TokenData stores OAuth tokens locally
type TokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
	UserEmail    string    `json:"user_email"`
	ClientID     string    `json:"client_id"`
	ClientSecret string    `json:"client_secret"`
}

// DriveUploadResult returns details of uploaded file
type DriveUploadResult struct {
	FileID         string `json:"fileId"`
	FileName       string `json:"fileName"`
	WebViewLink    string `json:"webViewLink"`
	WebContentLink string `json:"webContentLink"`
}

type GDriveManager struct {
	mu         sync.Mutex
	tokenFile  string
	tokenData  *TokenData
	authServer *http.Server
}

// NewGDriveManager creates a manager saving tokens to ~/.speedmap/gdrive_token.json
func NewGDriveManager() *GDriveManager {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".speedmap")
	_ = os.MkdirAll(dir, 0755)

	m := &GDriveManager{
		tokenFile: filepath.Join(dir, "gdrive_token.json"),
	}
	m.loadToken()
	return m
}

func (m *GDriveManager) loadToken() {
	data, err := os.ReadFile(m.tokenFile)
	if err != nil {
		return
	}
	var t TokenData
	if err := json.Unmarshal(data, &t); err == nil {
		m.tokenData = &t
	}
}

func (m *GDriveManager) SaveToken(t *TokenData) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokenData = t
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.tokenFile, data, 0600)
}

func (m *GDriveManager) SaveCredentials(clientID, clientSecret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tokenData == nil {
		m.tokenData = &TokenData{}
	}
	m.tokenData.ClientID = strings.TrimSpace(clientID)
	m.tokenData.ClientSecret = strings.TrimSpace(clientSecret)

	data, err := json.MarshalIndent(m.tokenData, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.tokenFile, data, 0600)
}

func (m *GDriveManager) GetCredentials() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := map[string]string{
		"clientID":     "",
		"clientSecret": "",
	}
	if m.tokenData != nil {
		res["clientID"] = m.tokenData.ClientID
		res["clientSecret"] = m.tokenData.ClientSecret
	}
	return res
}

func (m *GDriveManager) IsConnected() bool {

	m.mu.Lock()
	defer m.mu.Unlock()
	return m.tokenData != nil && (m.tokenData.RefreshToken != "" || m.tokenData.AccessToken != "")
}

func (m *GDriveManager) GetUserEmail() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tokenData == nil {
		return ""
	}
	return m.tokenData.UserEmail
}

func (m *GDriveManager) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokenData = nil
	_ = os.Remove(m.tokenFile)
	return nil
}

// StartAuthFlow launches HTTP server on port 8585 and opens browser for Google OAuth
func (m *GDriveManager) StartAuthFlow(clientID, clientSecret string) (string, error) {
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)

	if clientID == "" && m.tokenData != nil {
		clientID = m.tokenData.ClientID
	}
	if clientSecret == "" && m.tokenData != nil {
		clientSecret = m.tokenData.ClientSecret
	}

	if clientID == "" || clientSecret == "" {
		return "", fmt.Errorf("Вкажіть ваш Google OAuth Client ID та Client Secret у Налаштуваннях.")
	}


	listener, err := net.Listen("tcp", "127.0.0.1:8585")
	if err != nil {
		return "", fmt.Errorf("Cannot start auth listener on 8585: %w", err)
	}

	authCodeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing authorization code", http.StatusBadRequest)
			errChan <- fmt.Errorf("Authorization canceled or failed")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head><meta charset="utf-8"><title>SpeedMap Google Drive Auth</title></head>
			<body style="font-family: -apple-system, sans-serif; background: #0f172a; color: #f8fafc; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0;">
				<div style="text-align: center; background: #1e293b; padding: 40px; border-radius: 16px; border: 1px solid #334155; max-width: 400px;">
					<h2 style="color: #10b981; margin-top: 0;">🟢 Успішно підключено!</h2>
					<p style="color: #94a3b8; font-size: 14px;">Google Drive авторизовано для SpeedMap. Ви можете закрити це вікно та повернутися у додаток.</p>
				</div>
			</body>
			</html>
		`))
		authCodeChan <- code
	})

	server := &http.Server{Handler: mux}
	m.authServer = server

	go func() {
		_ = server.Serve(listener)
	}()

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s%%20%s&access_type=offline&prompt=consent",
		OAuthAuthURL,
		url.QueryEscape(clientID),
		url.QueryEscape(RedirectURI),
		url.QueryEscape(ScopeDriveFile),
		url.QueryEscape(ScopeUserInfo),
	)

	// Open browser
	m.openBrowser(authURL)

	select {
	case code := <-authCodeChan:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = server.Shutdown(ctx)
		cancel()

		token, err := m.exchangeCode(code, clientID, clientSecret)
		if err != nil {
			return "", err
		}
		_ = m.SaveToken(token)
		return token.UserEmail, nil

	case err := <-errChan:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = server.Shutdown(ctx)
		cancel()
		return "", err

	case <-time.After(3 * time.Minute):
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = server.Shutdown(ctx)
		cancel()
		return "", fmt.Errorf("OAuth authorization timed out (3 min)")
	}
}

func (m *GDriveManager) openBrowser(u string) {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("open", u).Start()
	case "windows":
		_ = exec.Command("cmd", "/c", "start", u).Start()
	default:
		_ = exec.Command("xdg-open", u).Start()
	}
}

func (m *GDriveManager) exchangeCode(code, clientID, clientSecret string) (*TokenData, error) {
	val := url.Values{}
	val.Set("code", code)
	val.Set("client_id", clientID)
	val.Set("client_secret", clientSecret)
	val.Set("redirect_uri", RedirectURI)
	val.Set("grant_type", "authorization_code")

	resp, err := http.PostForm(OAuthTokenURL, val)
	if err != nil {
		return nil, fmt.Errorf("Token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Google Token Error: %s", string(body))
	}

	var res struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}

	t := &TokenData{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		TokenType:    res.TokenType,
		Expiry:       time.Now().Add(time.Duration(res.ExpiresIn) * time.Second),
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}

	// Fetch user email
	email := m.fetchUserEmail(res.AccessToken)
	t.UserEmail = email

	return t, nil
}

func (m *GDriveManager) fetchUserEmail(accessToken string) string {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var u struct {
		Email string `json:"email"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&u)
	return u.Email
}

func (m *GDriveManager) getValidAccessToken() (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tokenData == nil {
		return "", fmt.Errorf("Google Drive не авторизовано. Зайдіть у Налаштування для підключення.")
	}

	// If token expires in less than 2 minutes, refresh it
	if time.Now().Add(2 * time.Minute).After(m.tokenData.Expiry) {
		if m.tokenData.RefreshToken == "" {
			return "", fmt.Errorf("Google Drive refresh token missing. Повторіть авторизацію.")
		}

		val := url.Values{}
		val.Set("client_id", m.tokenData.ClientID)
		val.Set("client_secret", m.tokenData.ClientSecret)
		val.Set("refresh_token", m.tokenData.RefreshToken)
		val.Set("grant_type", "refresh_token")

		resp, err := http.PostForm(OAuthTokenURL, val)
		if err != nil {
			return "", fmt.Errorf("Refresh token failed: %w", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			return "", fmt.Errorf("Google Token Refresh Error: %s", string(body))
		}

		var res struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int    `json:"expires_in"`
		}
		if err := json.Unmarshal(body, &res); err == nil && res.AccessToken != "" {
			m.tokenData.AccessToken = res.AccessToken
			m.tokenData.Expiry = time.Now().Add(time.Duration(res.ExpiresIn) * time.Second)
			data, _ := json.MarshalIndent(m.tokenData, "", "  ")
			_ = os.WriteFile(m.tokenFile, data, 0600)
		}
	}

	return m.tokenData.AccessToken, nil
}

// EnsureFolder creates or gets folder ID on Google Drive
func (m *GDriveManager) EnsureFolder(token, folderName, parentID string) (string, error) {
	// Search existing folder
	q := fmt.Sprintf("name = '%s' and mimeType = 'application/vnd.google-apps.folder' and trashed = false", folderName)
	if parentID != "" {
		q += fmt.Sprintf(" and '%s' in parents", parentID)
	}

	reqURL := fmt.Sprintf("%s/files?q=%s&fields=files(id,name)", DriveAPIBase, url.QueryEscape(q))
	req, _ := http.NewRequest("GET", reqURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err == nil && resp.StatusCode == 200 {
		var list struct {
			Files []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"files"`
		}
		if json.NewDecoder(resp.Body).Decode(&list) == nil && len(list.Files) > 0 {
			resp.Body.Close()
			return list.Files[0].ID, nil
		}
		resp.Body.Close()
	}

	// Create folder
	meta := map[string]interface{}{
		"name":     folderName,
		"mimeType": "application/vnd.google-apps.folder",
	}
	if parentID != "" {
		meta["parents"] = []string{parentID}
	}
	bodyData, _ := json.Marshal(meta)

	req, _ = http.NewRequest("POST", DriveAPIBase+"/files", bytes.NewReader(bodyData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var created struct {
		ID string `json:"id"`
	}
	if json.NewDecoder(resp.Body).Decode(&created) == nil && created.ID != "" {
		return created.ID, nil
	}
	return "", fmt.Errorf("Failed to create folder %s", folderName)
}

// UploadFile uploads file to Google Drive and sets public read permission
func (m *GDriveManager) UploadFile(filePath, customFolderName string) (*DriveUploadResult, error) {
	token, err := m.getValidAccessToken()
	if err != nil {
		return nil, err
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("Read local file failed: %w", err)
	}

	fileName := filepath.Base(filePath)
	mimeType := mime.TypeByExtension(filepath.Ext(fileName))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Ensure SpeedMap Reports root folder
	rootFolderID, _ := m.EnsureFolder(token, "SpeedMap Reports", "")

	parentFolderID := rootFolderID
	if customFolderName != "" {
		parentFolderID, _ = m.EnsureFolder(token, customFolderName, rootFolderID)
	}

	// Multipart upload metadata + file content
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	meta := map[string]interface{}{
		"name": fileName,
	}
	if parentFolderID != "" {
		meta["parents"] = []string{parentFolderID}
	}
	metaBytes, _ := json.Marshal(meta)

	// Part 1: Metadata
	h := make(map[string][]string)
	h["Content-Type"] = []string{"application/json; charset=UTF-8"}
	part1, _ := writer.CreatePart(h)
	_, _ = part1.Write(metaBytes)

	// Part 2: Media
	h2 := make(map[string][]string)
	h2["Content-Type"] = []string{mimeType}
	part2, _ := writer.CreatePart(h2)
	_, _ = part2.Write(fileData)

	_ = writer.Close()

	reqURL := fmt.Sprintf("%s/files?uploadType=multipart&fields=id,name,webViewLink,webContentLink", DriveUploadBase)
	req, err := http.NewRequest("POST", reqURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "multipart/related; boundary="+writer.Boundary())

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Upload request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("Google Drive Upload Error (%d): %s", resp.StatusCode, string(respBytes))
	}

	var res DriveUploadResult
	if err := json.Unmarshal(respBytes, &res); err != nil {
		return nil, err
	}

	// Make file public read-only for sharing
	_ = m.makeFilePublic(token, res.FileID)

	return &res, nil
}

func (m *GDriveManager) makeFilePublic(token, fileID string) error {
	permURL := fmt.Sprintf("%s/files/%s/permissions", DriveAPIBase, fileID)
	bodyData, _ := json.Marshal(map[string]string{
		"role": "reader",
		"type": "anyone",
	})

	req, _ := http.NewRequest("POST", permURL, bytes.NewReader(bodyData))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}
