package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/rehiy/libgo/upgrade"
	"github.com/rehiy/web-modem/service"
)

// UpdateHandler 自更新处理器
type UpdateHandler struct {
	updateService *service.UpdateService
}

// NewUpdateHandler 创建新的自更新处理器
func NewUpdateHandler() *UpdateHandler {
	return &UpdateHandler{updateService: service.NewUpdateService()}
}

// CheckUpdate 检查是否存在新版本
func (h *UpdateHandler) CheckUpdate(w http.ResponseWriter, r *http.Request) {
	result, err := h.updateService.Check(r.Context())
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// ApplyUpdate 下载并应用新版本
func (h *UpdateHandler) ApplyUpdate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Restart bool `json:"restart"`
	}

	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	result, err := h.updateService.Apply(r.Context())
	if err != nil {
		if errors.Is(err, upgrade.ErrNoUpdate) {
			respondJSON(w, http.StatusOK, H{
				"status": "no_update",
				"result": result,
			})
			return
		}
		respondJSON(w, http.StatusInternalServerError, H{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, H{
		"status":  "updated",
		"restart": req.Restart,
		"result":  result,
	})

	service.NewWebhookService().HandleDataEvent(service.WebhookEventSystemUpdateCompleted, H{
		"status":           "updated",
		"current_version":  result.CurrentVersion,
		"latest_version":   result.LatestVersion,
		"package_checksum": result.PackageChecksum,
		"restart":          req.Restart,
	})

	if req.Restart {
		restartLater()
	}
}

// Restart 重启当前应用
func (h *UpdateHandler) Restart(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, H{"status": "restarting"})
	restartLater()
}

func restartLater() {
	go func() {
		time.Sleep(800 * time.Millisecond)
		if err := upgrade.Restart(); err != nil {
			log.Printf("restart failed: %v", err)
		}
	}()
}
