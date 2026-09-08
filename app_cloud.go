package main

import (
	"SpeedMap/pkg/cloud"
)

// StartGDriveAuth starts Google OAuth2 browser flow
func (a *App) StartGDriveAuth(clientID, clientSecret string) (string, error) {
	if a.gdriveManager == nil {
		a.gdriveManager = cloud.NewGDriveManager()
	}
	return a.gdriveManager.StartAuthFlow(clientID, clientSecret)
}

// GetGDriveStatus returns connection status and user email
func (a *App) GetGDriveStatus() map[string]interface{} {
	if a.gdriveManager == nil {
		a.gdriveManager = cloud.NewGDriveManager()
	}
	return map[string]interface{}{
		"connected": a.gdriveManager.IsConnected(),
		"email":     a.gdriveManager.GetUserEmail(),
	}
}

// DisconnectGDrive removes saved OAuth token
func (a *App) DisconnectGDrive() error {
	if a.gdriveManager == nil {
		return nil
	}
	return a.gdriveManager.Disconnect()
}

// UploadFileToGDrive uploads a given local file to Google Drive and returns public link
func (a *App) UploadFileToGDrive(filePath string, folderName string) (*cloud.DriveUploadResult, error) {
	if a.gdriveManager == nil {
		a.gdriveManager = cloud.NewGDriveManager()
	}
	return a.gdriveManager.UploadFile(filePath, folderName)
}

// SaveGDriveCredentials saves Client ID & Secret
func (a *App) SaveGDriveCredentials(clientID, clientSecret string) error {
	if a.gdriveManager == nil {
		a.gdriveManager = cloud.NewGDriveManager()
	}
	return a.gdriveManager.SaveCredentials(clientID, clientSecret)
}

// GetGDriveCredentials gets saved Client ID & Secret
func (a *App) GetGDriveCredentials() map[string]string {
	if a.gdriveManager == nil {
		a.gdriveManager = cloud.NewGDriveManager()
	}
	return a.gdriveManager.GetCredentials()
}
