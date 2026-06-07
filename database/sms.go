package database

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/rehiy/web-modem/models"
)

// CreateSms 保存短信到数据库
func CreateSms(sms *models.Sms) error {
	// 确保必要字段已设置
	if sms.Direction == "" {
		sms.Direction = "in"
	}
	if sms.ReceiveTime.IsZero() {
		sms.ReceiveTime = time.Now()
	}

	err := db.Create(sms).Error
	if err != nil {
		return fmt.Errorf("failed to save Sms: %w", err)
	}
	return nil
}

// DeleteSms 根据数据库ID删除短信
func DeleteSms(id int) error {
	ret := db.Delete(&models.Sms{}, id)
	if ret.Error != nil {
		return fmt.Errorf("failed to delete Sms: %w", ret.Error)
	}
	if ret.RowsAffected == 0 {
		return fmt.Errorf("Sms not found")
	}
	return nil
}

// BatchDeleteSms 批量删除短信
func BatchDeleteSms(ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	err := db.Where("id IN ?", ids).Delete(&models.Sms{}).Error
	if err != nil {
		return fmt.Errorf("failed to batch delete Sms: %w", err)
	}
	return nil
}

// GetSmsListByIDs 根据短信模块的ID查询
func GetSmsListByIDs(smsIDs []int) ([]models.Sms, error) {
	if len(smsIDs) == 0 {
		return []models.Sms{}, nil
	}

	smsList := []models.Sms{}
	str := IntArrayToString(smsIDs)
	err := db.Where("sms_ids = ?", str).Order("receive_time DESC").Find(&smsList).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query Sms by IDs: %w", err)
	}
	return smsList, nil
}

// GetSmsListByDeviceIDs 根据设备标识和短信模块ID查询。
func GetSmsListByDeviceIDs(smsIDs []int, deviceIMEI, receiveNumber, modemName string) ([]models.Sms, error) {
	if len(smsIDs) == 0 {
		return []models.Sms{}, nil
	}

	query := db.Where("sms_ids = ?", IntArrayToString(smsIDs))
	switch {
	case isStableDeviceValue(deviceIMEI):
		query = query.Where("device_imei = ?", deviceIMEI)
	case isStableDeviceValue(receiveNumber):
		query = query.Where("receive_number = ?", receiveNumber)
	case modemName != "":
		query = query.Where("modem_name = ?", modemName)
	}

	smsList := []models.Sms{}
	err := query.Order("receive_time DESC").Find(&smsList).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query Sms by device IDs: %w", err)
	}
	return smsList, nil
}

func isStableDeviceValue(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && value != "-" && !strings.EqualFold(value, "unknown")
}

func normalizeSmsDeviceKeys(values []string) []string {
	keys := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := strings.TrimSpace(value)
		if !isStableDeviceValue(key) {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	return keys
}

// GetSmsList 查询短信列表
func GetSmsList(filter *models.SmsFilter) ([]models.Sms, int, error) {
	query := db.Model(&models.Sms{})

	if filter.Direction != "" {
		query = query.Where("direction = ?", filter.Direction)
	}
	if filter.SendNumber != "" {
		query = query.Where("send_number = ?", filter.SendNumber)
	}
	if filter.ReceiveNumber != "" {
		query = query.Where("receive_number = ?", filter.ReceiveNumber)
	}
	if filter.ModemName != "" {
		query = query.Where("modem_name = ?", filter.ModemName)
	}
	if filter.DeviceIMEI != "" {
		query = query.Where("device_imei = ?", filter.DeviceIMEI)
	}
	if filter.SimICCID != "" {
		query = query.Where("sim_icc_id = ?", filter.SimICCID)
	}
	if filter.SimIMSI != "" {
		query = query.Where("sim_imsi = ?", filter.SimIMSI)
	}
	if filter.Operator != "" {
		query = query.Where("operator = ?", filter.Operator)
	}
	deviceKeys := normalizeSmsDeviceKeys(append(filter.DeviceKeys, filter.DeviceKey))
	if len(deviceKeys) > 0 {
		query = query.Where(
			"(device_imei IN ? OR sim_icc_id IN ? OR sim_imsi IN ? OR modem_name IN ? OR (direction = 'in' AND receive_number IN ?) OR (direction = 'out' AND send_number IN ?))",
			deviceKeys,
			deviceKeys,
			deviceKeys,
			deviceKeys,
			deviceKeys,
			deviceKeys,
		)
	}
	if !filter.StartTime.IsZero() {
		query = query.Where("receive_time >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query = query.Where("receive_time <= ?", filter.EndTime)
	}

	// 查询总数
	var total int64
	countQuery := query.Session(&gorm.Session{})
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count Sms: %w", err)
	}

	// 查询列表
	smsList := []models.Sms{}
	err := query.Order("receive_time DESC").Limit(filter.Limit).Offset(filter.Offset).Find(&smsList).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query Sms: %w", err)
	}

	return smsList, int(total), nil
}

// IntArrayToString 将int数组转换为字符串
func IntArrayToString(arr []int) string {
	if len(arr) == 0 {
		return ""
	}

	strs := make([]string, len(arr))
	for i, v := range arr {
		strs[i] = fmt.Sprintf("%d", v)
	}

	return strings.Join(strs, ",")
}
