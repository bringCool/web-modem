package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rehiy/web-modem/database"
	"github.com/rehiy/web-modem/models"
	"github.com/rehiy/web-modem/service"
)

// AlertHandler 告警处理器
type AlertHandler struct {
	alertService *service.AlertService
}

// NewAlertHandler 创建告警处理器
func NewAlertHandler() *AlertHandler {
	return &AlertHandler{
		alertService: service.NewAlertService(),
	}
}

// ListAlerts 获取告警列表
func (h *AlertHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	filter := &models.AlertFilter{
		Status: "active",
		Limit:  50,
	}

	if status := r.URL.Query().Get("status"); status != "" {
		switch status {
		case "active", "resolved", "all":
			filter.Status = status
		default:
			respondJSON(w, http.StatusBadRequest, H{"error": "invalid status"})
			return
		}
	}

	if severity := r.URL.Query().Get("severity"); severity != "" {
		switch severity {
		case "critical", "warning", "info":
			filter.Severity = severity
		default:
			respondJSON(w, http.StatusBadRequest, H{"error": "invalid severity"})
			return
		}
	}

	if alertType := r.URL.Query().Get("type"); alertType != "" {
		filter.Type = alertType
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

	alerts, total, err := database.GetAlertList(filter)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	summary, err := database.GetActiveAlertSummary()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, H{
		"data":    alerts,
		"total":   total,
		"limit":   filter.Limit,
		"offset":  filter.Offset,
		"summary": summary,
	})
}

// ScanAlerts 扫描当前系统状态并生成告警
func (h *AlertHandler) ScanAlerts(w http.ResponseWriter, r *http.Request) {
	if err := h.alertService.ScanModemAlerts(); err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	summary, err := database.GetActiveAlertSummary()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, H{
		"status":  "scanned",
		"summary": summary,
	})
}

// ResolveAlerts 解决告警
func (h *AlertHandler) ResolveAlerts(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []int `json:"ids"`
		All bool  `json:"all"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": err.Error()})
		return
	}

	if req.All {
		if err := h.alertService.ResolveAllActiveAlerts(); err != nil {
			respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusOK, H{"status": "resolved"})
		return
	}

	if len(req.IDs) == 0 {
		respondJSON(w, http.StatusBadRequest, H{"error": "no IDs provided"})
		return
	}

	if err := h.alertService.ResolveAlertsByIDs(req.IDs); err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, H{
		"status": "resolved",
		"count":  len(req.IDs),
	})
}
