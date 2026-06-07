package database

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/rehiy/web-modem/models"
)

func TestGetSmsListByDeviceKeyUsesMigratedSIMColumns(t *testing.T) {
	t.Setenv("DB_PATH", filepath.Join(t.TempDir(), "modem.db"))
	if err := InitDB(); err != nil {
		t.Fatalf("InitDB() error = %v", err)
	}
	t.Cleanup(func() {
		_ = Close()
	})

	sms := &models.Sms{
		Content:       "hello",
		SmsIDs:        "1",
		ReceiveTime:   time.Now(),
		ReceiveNumber: "+8613800138000",
		SendNumber:    "+8613800138001",
		Direction:     "in",
		ModemName:     "ttyUSB2",
		DeviceIMEI:    "000000000000000",
		SimICCID:      "89860000000000000000",
		SimIMSI:       "460001234567890",
		Operator:      "46000",
	}
	if err := CreateSms(sms); err != nil {
		t.Fatalf("CreateSms() error = %v", err)
	}

	got, total, err := GetSmsList(&models.SmsFilter{
		DeviceKey: "89860000000000000000",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("GetSmsList() error = %v", err)
	}
	if total != 1 || len(got) != 1 {
		t.Fatalf("GetSmsList() total = %d, len = %d; want 1, 1", total, len(got))
	}
}
