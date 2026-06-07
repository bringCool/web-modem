package models

import (
	"time"
)

// Sms 短信模型
type Sms struct {
	ID            int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Content       string    `json:"content" gorm:"not null;type:text"`
	SmsIDs        string    `json:"sms_ids" gorm:"not null;type:text"`
	ReceiveTime   time.Time `json:"receive_time" gorm:"not null;index:idx_sms_receive_time"`
	ReceiveNumber string    `json:"receive_number" gorm:"type:text;index:idx_sms_receive_number"`
	SendNumber    string    `json:"send_number" gorm:"type:text;index:idx_sms_send_number"`
	Direction     string    `json:"direction" gorm:"not null;type:text;check:direction IN ('in', 'out');index:idx_sms_direction"` // "in" 或 "out"
	ModemName     string    `json:"modem_name" gorm:"type:text;index:idx_sms_modem_name"`
	DeviceIMEI    string    `json:"device_imei" gorm:"type:text;index:idx_sms_device_imei"`
	SimICCID      string    `json:"sim_iccid" gorm:"column:sim_icc_id;type:text;index:idx_sms_sim_iccid"`
	SimIMSI       string    `json:"sim_imsi" gorm:"type:text;index:idx_sms_sim_imsi"`
	Operator      string    `json:"operator" gorm:"type:text;index:idx_sms_operator"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// SmsFilter 短信查询过滤器
type SmsFilter struct {
	Direction     string    `json:"direction,omitempty"`
	SendNumber    string    `json:"send_number,omitempty"`
	ReceiveNumber string    `json:"receive_number,omitempty"`
	ModemName     string    `json:"modem_name,omitempty"`
	DeviceIMEI    string    `json:"device_imei,omitempty"`
	SimICCID      string    `json:"sim_iccid,omitempty"`
	SimIMSI       string    `json:"sim_imsi,omitempty"`
	Operator      string    `json:"operator,omitempty"`
	DeviceKey     string    `json:"device_key,omitempty"`
	DeviceKeys    []string  `json:"device_keys,omitempty"`
	StartTime     time.Time `json:"start_time,omitempty"`
	EndTime       time.Time `json:"end_time,omitempty"`
	Limit         int       `json:"limit,omitempty"`
	Offset        int       `json:"offset,omitempty"`
}

// Webhook Webhook配置模型
type Webhook struct {
	ID                   int       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name                 string    `json:"name" gorm:"not null;unique;type:text"`
	URL                  string    `json:"url" gorm:"not null;type:text"`
	Template             string    `json:"template" gorm:"type:text;default:'{}'"`
	Enabled              bool      `json:"enabled"`
	EventType            string    `json:"event_type" gorm:"type:text;default:'sms_received';index:idx_webhook_event_type"`
	MatchType            string    `json:"match_type" gorm:"type:text;default:'all';index:idx_webhook_match_type"`
	MatchValue           string    `json:"match_value" gorm:"type:text;index:idx_webhook_match_value"`
	RetryCount           int       `json:"retry_count"`
	RetryIntervalSeconds int       `json:"retry_interval_seconds"`
	RetryBackoff         bool      `json:"retry_backoff"`
	CreatedAt            time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt            time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// WebhookDelivery Webhook发送记录模型
type WebhookDelivery struct {
	ID             int       `json:"id" gorm:"primaryKey;autoIncrement"`
	WebhookID      int       `json:"webhook_id" gorm:"index:idx_webhook_delivery_webhook_id"`
	WebhookName    string    `json:"webhook_name" gorm:"type:text"`
	URL            string    `json:"url" gorm:"not null;type:text"`
	EventType      string    `json:"event_type" gorm:"type:text;index:idx_webhook_delivery_event_type"`
	SmsID          int       `json:"sms_id" gorm:"index:idx_webhook_delivery_sms_id"`
	AlertID        int       `json:"alert_id" gorm:"index:idx_webhook_delivery_alert_id"`
	Status         string    `json:"status" gorm:"not null;type:text;index:idx_webhook_delivery_status"`
	HTTPStatusCode int       `json:"http_status_code"`
	ResponseBody   string    `json:"response_body" gorm:"type:text"`
	DurationMs     int64     `json:"duration_ms"`
	Error          string    `json:"error" gorm:"type:text"`
	Attempt        int       `json:"attempt"`
	CreatedAt      time.Time `json:"created_at" gorm:"autoCreateTime;index:idx_webhook_delivery_created_at"`
}

// WebhookDeliveryFilter Webhook发送记录查询过滤器
type WebhookDeliveryFilter struct {
	WebhookID int       `json:"webhook_id,omitempty"`
	EventType string    `json:"event_type,omitempty"`
	Status    string    `json:"status,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
}

// Alert 告警模型
type Alert struct {
	ID         int        `json:"id" gorm:"primaryKey;autoIncrement"`
	Type       string     `json:"type" gorm:"not null;type:text;index:idx_alert_type"`
	Severity   string     `json:"severity" gorm:"not null;type:text;index:idx_alert_severity"`
	Status     string     `json:"status" gorm:"not null;type:text;index:idx_alert_status"`
	Source     string     `json:"source" gorm:"not null;type:text;index:idx_alert_source"`
	Message    string     `json:"message" gorm:"not null;type:text"`
	Detail     string     `json:"detail" gorm:"type:text"`
	Count      int        `json:"count"`
	FirstSeen  time.Time  `json:"first_seen" gorm:"not null;index:idx_alert_first_seen"`
	LastSeen   time.Time  `json:"last_seen" gorm:"not null;index:idx_alert_last_seen"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// AlertFilter 告警查询过滤器
type AlertFilter struct {
	Status   string `json:"status,omitempty"`
	Severity string `json:"severity,omitempty"`
	Type     string `json:"type,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}

// Setting 系统设置模型
type Setting struct {
	Key       string    `json:"key" gorm:"primaryKey;type:text"`
	Value     string    `json:"value" gorm:"not null;type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// Settings 系统设置 (DTO)
type Settings struct {
	SmsdbEnabled   bool `json:"smsdb_enabled"`
	WebhookEnabled bool `json:"webhook_enabled"`
}
