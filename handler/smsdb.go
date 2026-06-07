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

// SmsdbHandler 短信存储处理器
type SmsdbHandler struct {
	smsdbService *service.SmsdbService
}

// NewSmsdbHandler 创建新的短信存储处理器
func NewSmsdbHandler() *SmsdbHandler {
	return &SmsdbHandler{
		smsdbService: service.NewSmsdbService(),
	}
}

// ListSms 获取数据库中的短信列表
func (h *SmsdbHandler) ListSms(w http.ResponseWriter, r *http.Request) {
	filter := &models.SmsFilter{}

	// 解析查询参数
	if direction := r.URL.Query().Get("direction"); direction != "" {
		filter.Direction = direction
	}

	if sendNumber := r.URL.Query().Get("send_number"); sendNumber != "" {
		filter.SendNumber = sendNumber
	}

	if receiveNumber := r.URL.Query().Get("receive_number"); receiveNumber != "" {
		filter.ReceiveNumber = receiveNumber
	}

	if modemName := r.URL.Query().Get("modem_name"); modemName != "" {
		filter.ModemName = modemName
	}

	if deviceIMEI := r.URL.Query().Get("device_imei"); deviceIMEI != "" {
		filter.DeviceIMEI = deviceIMEI
	}

	if simICCID := r.URL.Query().Get("sim_iccid"); simICCID != "" {
		filter.SimICCID = simICCID
	}

	if simIMSI := r.URL.Query().Get("sim_imsi"); simIMSI != "" {
		filter.SimIMSI = simIMSI
	}

	if operator := r.URL.Query().Get("operator"); operator != "" {
		filter.Operator = operator
	}

	if deviceKeys := r.URL.Query()["device_key"]; len(deviceKeys) > 0 {
		filter.DeviceKeys = append(filter.DeviceKeys, deviceKeys...)
	}

	if deviceKeys := r.URL.Query().Get("device_keys"); deviceKeys != "" {
		filter.DeviceKeys = append(filter.DeviceKeys, strings.Split(deviceKeys, ",")...)
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

	// 分页参数
	filter.Limit = 50 // 默认每页50条
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

	smsList, total, err := database.GetSmsList(filter)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, H{
		"data":   smsList,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// DeleteSmsBatch 批量删除数据库中的短信
func (h *SmsdbHandler) DeleteSmsBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []int `json:"ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, H{"error": err.Error()})
		return
	}

	if len(req.IDs) == 0 {
		respondJSON(w, http.StatusBadRequest, H{"error": "no IDs provided"})
		return
	}

	if err := database.BatchDeleteSms(req.IDs); err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, H{
		"status": "deleted",
		"count":  len(req.IDs),
	})
}

// SyncSms 同步短信到数据库；未指定 name 时同步所有已连接Modem。
func (h *SmsdbHandler) SyncSms(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		respondJSON(w, http.StatusBadRequest, H{"error": err.Error()})
		return
	}

	modemName := strings.TrimSpace(req.Name)
	if modemName == "" {
		result, err := h.smsdbService.SyncAllSmsToDB()
		if err != nil {
			respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
			return
		}

		service.NewWebhookService().HandleDataEvent(service.WebhookEventSmsSyncCompleted, H{
			"sync_scope":    "all",
			"total_modems":  result.TotalModems,
			"success_count": result.SuccessCount,
			"failed_count":  result.FailedCount,
			"total_count":   result.TotalCount,
			"new_count":     result.NewCount,
			"results":       result.Results,
		})
		respondJSON(w, http.StatusOK, result)
		return
	}

	result, err := h.smsdbService.SyncSmsToDB(modemName)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	service.NewWebhookService().HandleDataEvent(service.WebhookEventSmsSyncCompleted, H{
		"sync_scope":  "modem",
		"modem_name":  result.ModemName,
		"total_count": result.TotalCount,
		"new_count":   result.NewCount,
	})
	respondJSON(w, http.StatusOK, result)
}
