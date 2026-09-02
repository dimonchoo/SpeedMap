package notify

import "testing"

func TestSendNotification(t *testing.T) {
	// Simple non-blocking test that Send does not crash
	Send("SpeedMap Test", "Тест сповіщення", "Перевірка роботи сповіщень")
}
