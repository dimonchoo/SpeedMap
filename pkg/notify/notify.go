package notify

// Send sends a native OS notification with the application's icon and sound
func Send(title, subtitle, message string) {
	sendOSNotification(title, subtitle, message)
}
