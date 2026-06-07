package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rehiy/web-modem/database"
	"github.com/rehiy/web-modem/models"
	"github.com/rehiy/web-modem/service"
)

// WebhookHandler Webhook处理器
type WebhookHandler struct {
	ws *service.WebhookService
}

// NewWebhookHandler 创建新的Webhook处理器
func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{
		ws: service.NewWebhookService(),
	}
}

func decodeWebhookRequest(r *http.Request, webhook *models.Webhook) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	raw := map[string]json.RawMessage{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return err
	}
	if err := json.Unmarshal(body, webhook); err != nil {
		return err
	}

	if _, ok := raw["enabled"]; !ok {
		webhook.Enabled = true
	}
	if _, ok := raw["retry_count"]; !ok {
		webhook.RetryCount = 2
	}
	if _, ok := raw["retry_interval_seconds"]; !ok {
		webhook.RetryIntervalSeconds = 2
	}
	if _, ok := raw["retry_backoff"]; !ok {
		webhook.RetryBackoff = true
	}

	return nil
}

// CreateWebhook 创建Webhook配置
func (h *WebhookHandler) CreateWebhook(w http.ResponseWriter, r *http.Request) {
	var webhook models.Webhook
	if err := decodeWebhookRequest(r, &webhook); err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": err.Error()})
		return
	}

	// 验证必填字段
	if webhook.Name == "" || webhook.URL == "" {
		respondJSON(w, http.StatusBadRequest, H{"error": "name and url are required"})
		return
	}

	// 如果模板为空，使用默认模板
	if webhook.Template == "" {
		webhook.Template = "{}"
	}
	normalizeWebhookConfig(&webhook)
	if err := validateWebhookTemplate(webhook.Template); err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": err.Error()})
		return
	}

	if err := database.CreateWebhook(&webhook); err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}
	service.InvalidateWebhookCache()

	respondJSON(w, http.StatusCreated, webhook)
}

// UpdateWebhook 更新Webhook配置
func (h *WebhookHandler) UpdateWebhook(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	idStr := vars.Get("id")
	if idStr == "" {
		respondJSON(w, http.StatusBadRequest, H{"error": "id is required"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": "invalid id"})
		return
	}

	var webhook models.Webhook
	if err := decodeWebhookRequest(r, &webhook); err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": err.Error()})
		return
	}

	webhook.ID = id

	// 验证必填字段
	if webhook.Name == "" || webhook.URL == "" {
		respondJSON(w, http.StatusBadRequest, H{"error": "name and url are required"})
		return
	}

	// 如果模板为空，使用默认模板
	if webhook.Template == "" {
		webhook.Template = "{}"
	}
	normalizeWebhookConfig(&webhook)
	if err := validateWebhookTemplate(webhook.Template); err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": err.Error()})
		return
	}

	if err := database.UpdateWebhook(&webhook); err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}
	service.InvalidateWebhookCache()

	respondJSON(w, http.StatusOK, webhook)
}

// DeleteWebhook 删除Webhook配置
func (h *WebhookHandler) DeleteWebhook(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	idStr := vars.Get("id")
	if idStr == "" {
		respondJSON(w, http.StatusBadRequest, H{"error": "id is required"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": "invalid id"})
		return
	}

	if err := database.DeleteWebhook(id); err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}
	service.InvalidateWebhookCache()

	respondJSON(w, http.StatusOK, H{
		"status": "deleted",
		"id":     id,
	})
}

func normalizeWebhookConfig(webhook *models.Webhook) {
	normalizeWebhookEvent(webhook)
	normalizeWebhookMatch(webhook)
	normalizeWebhookRetry(webhook)
}

func normalizeWebhookEvent(webhook *models.Webhook) {
	webhook.EventType = strings.TrimSpace(webhook.EventType)
	if !service.IsKnownWebhookEventType(webhook.EventType) {
		webhook.EventType = service.WebhookEventSmsReceived
		return
	}
	webhook.EventType = service.NormalizeWebhookEventType(webhook.EventType)
}

func normalizeWebhookMatch(webhook *models.Webhook) {
	webhook.MatchType = strings.TrimSpace(webhook.MatchType)
	webhook.MatchValue = strings.TrimSpace(webhook.MatchValue)

	switch webhook.MatchType {
	case "", "all":
		webhook.MatchType = "all"
		webhook.MatchValue = ""
	case "receive_number", "device_imei", "sim_iccid", "sim_imsi", "operator", "send_number", "modem_name", "content_contains", "alert_type", "alert_source":
	case "direction":
		webhook.MatchValue = strings.ToLower(webhook.MatchValue)
		if webhook.MatchValue != "in" && webhook.MatchValue != "out" {
			webhook.MatchType = "all"
			webhook.MatchValue = ""
		}
	case "alert_severity":
		webhook.MatchValue = strings.ToLower(webhook.MatchValue)
		switch webhook.MatchValue {
		case "critical", "warning", "info":
		default:
			webhook.MatchType = "all"
			webhook.MatchValue = ""
		}
	default:
		webhook.MatchType = "all"
		webhook.MatchValue = ""
	}
}

func normalizeWebhookRetry(webhook *models.Webhook) {
	if webhook.RetryCount < 0 {
		webhook.RetryCount = 0
	}
	if webhook.RetryCount > 10 {
		webhook.RetryCount = 10
	}
	if webhook.RetryIntervalSeconds <= 0 {
		webhook.RetryIntervalSeconds = 2
	}
	if webhook.RetryIntervalSeconds > 3600 {
		webhook.RetryIntervalSeconds = 3600
	}
}

func validateWebhookTemplate(template string) error {
	template = strings.TrimSpace(template)
	if template == "" || template == "{}" {
		return nil
	}

	var payload any
	if err := json.Unmarshal([]byte(template), &payload); err != nil {
		return err
	}
	return nil
}

// GetWebhook 获取单个Webhook配置
func (h *WebhookHandler) GetWebhook(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	idStr := vars.Get("id")
	if idStr == "" {
		respondJSON(w, http.StatusBadRequest, H{"error": "id is required"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": "invalid id"})
		return
	}

	webhook, err := database.GetWebhook(id)
	if err != nil {
		respondJSON(w, http.StatusNotFound, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, webhook)
}

// ListWebhooks 获取所有Webhook配置
func (h *WebhookHandler) ListWebhooks(w http.ResponseWriter, r *http.Request) {
	webhooks, err := database.GetWebhookList()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, webhooks)
}

// PreviewWebhook 预览Webhook模板渲染结果
func (h *WebhookHandler) PreviewWebhook(w http.ResponseWriter, r *http.Request) {
	var webhook models.Webhook
	if err := decodeWebhookRequest(r, &webhook); err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": err.Error()})
		return
	}

	if webhook.Template == "" {
		webhook.Template = "{}"
	}
	normalizeWebhookConfig(&webhook)
	if err := validateWebhookTemplate(webhook.Template); err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": err.Error()})
		return
	}

	payload, err := h.ws.PreviewPayload(&webhook)
	if err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, H{
		"body": formatJSONPreview(payload),
	})
}

func formatJSONPreview(payload []byte) string {
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return string(payload)
	}

	formatted, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return string(payload)
	}
	return string(formatted)
}

// ListWebhookDeliveries 获取Webhook发送记录
func (h *WebhookHandler) ListWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	filter := &models.WebhookDeliveryFilter{
		Limit: 50,
	}

	if webhookID := r.URL.Query().Get("webhook_id"); webhookID != "" {
		id, err := strconv.Atoi(webhookID)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, H{"error": "invalid webhook_id"})
			return
		}
		filter.WebhookID = id
	}

	if eventType := r.URL.Query().Get("event_type"); eventType != "" {
		if !service.IsKnownWebhookEventType(eventType) || eventType == service.WebhookEventAll {
			respondJSON(w, http.StatusBadRequest, H{"error": "invalid event_type"})
			return
		}
		filter.EventType = service.NormalizeWebhookEventType(eventType)
	}

	if status := r.URL.Query().Get("status"); status != "" {
		switch status {
		case "success", "failed":
			filter.Status = status
		default:
			respondJSON(w, http.StatusBadRequest, H{"error": "invalid status"})
			return
		}
	}

	if startTime := r.URL.Query().Get("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}

	if endTime := r.URL.Query().Get("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = t
		}
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 200 {
			filter.Limit = l
		}
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	deliveries, total, err := database.GetWebhookDeliveryList(filter)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, H{
		"data":   deliveries,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// TestWebhook 测试Webhook
func (h *WebhookHandler) TestWebhook(w http.ResponseWriter, r *http.Request) {
	vars := r.URL.Query()
	idStr := vars.Get("id")
	if idStr == "" {
		var webhook models.Webhook
		if err := decodeWebhookRequest(r, &webhook); err != nil {
			respondJSON(w, http.StatusBadRequest, H{"error": err.Error()})
			return
		}
		if webhook.URL == "" {
			respondJSON(w, http.StatusBadRequest, H{"error": "url is required"})
			return
		}
		if webhook.Name == "" {
			webhook.Name = "测试"
		}
		if webhook.Template == "" {
			webhook.Template = "{}"
		}
		normalizeWebhookConfig(&webhook)
		if err := validateWebhookTemplate(webhook.Template); err != nil {
			respondJSON(w, http.StatusBadRequest, H{"error": err.Error()})
			return
		}

		if err := h.ws.TestWebhook(&webhook); err != nil {
			respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
			return
		}
	} else {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, H{"error": "invalid id"})
			return
		}

		webhook, err := database.GetWebhook(id)
		if err != nil {
			respondJSON(w, http.StatusNotFound, H{"error": err.Error()})
			return
		}
		normalizeWebhookConfig(webhook)

		if err := h.ws.TestWebhook(webhook); err != nil {
			respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
			return
		}
	}

	respondJSON(w, http.StatusOK, H{
		"status":  "success",
		"message": "Webhook test sent successfully",
	})
}
