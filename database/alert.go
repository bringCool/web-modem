package database

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/rehiy/web-modem/models"
)

// UpsertActiveAlert 创建或更新同一来源的活跃告警，返回 true 表示新建了活跃告警。
func UpsertActiveAlert(alert *models.Alert) (bool, error) {
	now := time.Now()
	var existing models.Alert
	result := db.Where("type = ? AND source = ? AND status = ?", alert.Type, alert.Source, "active").First(&existing)
	if result.Error == nil {
		update := map[string]any{
			"severity":  alert.Severity,
			"message":   alert.Message,
			"detail":    alert.Detail,
			"count":     existing.Count + 1,
			"last_seen": now,
		}
		if err := db.Model(&existing).Updates(update).Error; err != nil {
			return false, fmt.Errorf("failed to update alert: %w", err)
		}
		return false, nil
	}
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return false, fmt.Errorf("failed to query alert: %w", result.Error)
	}

	alert.Status = "active"
	alert.Count = 1
	alert.FirstSeen = now
	alert.LastSeen = now
	if err := db.Create(alert).Error; err != nil {
		return false, fmt.Errorf("failed to create alert: %w", err)
	}
	return true, nil
}

// ResolveAlertByIdentity 解决同一类型和来源的活跃告警，并返回被恢复的告警。
func ResolveAlertByIdentity(alertType string, source string) (*models.Alert, error) {
	now := time.Now()
	var alert models.Alert
	result := db.Where("type = ? AND source = ? AND status = ?", alertType, source, "active").First(&alert)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query alert: %w", result.Error)
	}

	err := db.Model(&alert).
		Updates(map[string]any{
			"status":      "resolved",
			"resolved_at": &now,
		}).Error
	if err != nil {
		return nil, fmt.Errorf("failed to resolve alert: %w", err)
	}

	alert.Status = "resolved"
	alert.ResolvedAt = &now
	return &alert, nil
}

// GetActiveAlertsByIDs 查询指定 ID 的活跃告警。
func GetActiveAlertsByIDs(ids []int) ([]models.Alert, error) {
	if len(ids) == 0 {
		return []models.Alert{}, nil
	}

	alerts := []models.Alert{}
	err := db.Where("id IN ? AND status = ?", ids, "active").Find(&alerts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query active alerts: %w", err)
	}
	return alerts, nil
}

// GetAllActiveAlerts 查询全部活跃告警。
func GetAllActiveAlerts() ([]models.Alert, error) {
	alerts := []models.Alert{}
	err := db.Where("status = ?", "active").Find(&alerts).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query active alerts: %w", err)
	}
	return alerts, nil
}

// ResolveAlertsByIDs 批量解决告警。
func ResolveAlertsByIDs(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	now := time.Now()
	err := db.Model(&models.Alert{}).
		Where("id IN ? AND status = ?", ids, "active").
		Updates(map[string]any{
			"status":      "resolved",
			"resolved_at": &now,
		}).Error
	if err != nil {
		return fmt.Errorf("failed to resolve alerts: %w", err)
	}
	return nil
}

// ResolveAllActiveAlerts 解决全部活跃告警。
func ResolveAllActiveAlerts() error {
	now := time.Now()
	err := db.Model(&models.Alert{}).
		Where("status = ?", "active").
		Updates(map[string]any{
			"status":      "resolved",
			"resolved_at": &now,
		}).Error
	if err != nil {
		return fmt.Errorf("failed to resolve all alerts: %w", err)
	}
	return nil
}

// GetAlertList 查询告警列表。
func GetAlertList(filter *models.AlertFilter) ([]models.Alert, int, error) {
	query := db.Model(&models.Alert{})

	if filter.Status != "" && filter.Status != "all" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Severity != "" {
		query = query.Where("severity = ?", filter.Severity)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count alerts: %w", err)
	}

	alerts := []models.Alert{}
	err := query.Order("status ASC, last_seen DESC").Limit(filter.Limit).Offset(filter.Offset).Find(&alerts).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query alerts: %w", err)
	}

	return alerts, int(total), nil
}

// GetActiveAlertSummary 获取活跃告警统计。
func GetActiveAlertSummary() (map[string]int, error) {
	type row struct {
		Severity string
		Count    int
	}

	rows := []row{}
	if err := db.Model(&models.Alert{}).
		Select("severity, count(*) as count").
		Where("status = ?", "active").
		Group("severity").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to count active alerts: %w", err)
	}

	summary := map[string]int{
		"active":   0,
		"critical": 0,
		"warning":  0,
		"info":     0,
	}
	for _, item := range rows {
		summary[item.Severity] = item.Count
		summary["active"] += item.Count
	}
	return summary, nil
}
