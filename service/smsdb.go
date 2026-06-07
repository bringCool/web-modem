package service

import (
	"fmt"
	"log"

	"github.com/rehiy/web-modem/database"
	"github.com/rehiy/web-modem/models"
)

// SmsdbService 短信数据库服务
type SmsdbService struct {
	modemService *ModemService
}

type SmsdbSyncResult struct {
	ModemName  string `json:"modemName"`
	TotalCount int    `json:"totalCount"`
	NewCount   int    `json:"newCount"`
	Error      string `json:"error,omitempty"`
}

type SmsdbSyncAllResult struct {
	Results      []SmsdbSyncResult `json:"results"`
	TotalModems  int               `json:"totalModems"`
	SuccessCount int               `json:"successCount"`
	FailedCount  int               `json:"failedCount"`
	TotalCount   int               `json:"totalCount"`
	NewCount     int               `json:"newCount"`
}

// NewSmsdbService 创建短信数据库服务
func NewSmsdbService() *SmsdbService {
	return &SmsdbService{
		modemService: GetModemService(),
	}
}

// SyncSmsToDB 从指定Modem同步所有短信到数据库
func (s *SmsdbService) SyncSmsToDB(modemName string) (*SmsdbSyncResult, error) {
	// 获取连接
	conn, err := s.modemService.GetConn(modemName)
	if err != nil {
		return nil, fmt.Errorf("获取连接失败: %v", err)
	}

	// 列出所有短信（stat=4 表示所有短信）
	smsList, err := conn.ListSmsPdu(4)
	if err != nil {
		return nil, fmt.Errorf("读取短信失败: %v", err)
	}

	totalCount := len(smsList)
	newCount := 0

	// 同步每条短信
	for _, atSms := range smsList {
		// 转换为数据库模型
		modelSms := atSmsToModelSms(atSms, conn)

		// 检查是否已存在
		if res, err := database.GetSmsListByDeviceIDs(atSms.Indices, conn.IMEI, conn.Number, modemName); err == nil && len(res) > 0 {
			log.Printf("[%s] Sms already exists in database, skipping: %s", modemName, res[0].SmsIDs)
			continue
		}

		// 保存到数据库
		if err := database.CreateSms(modelSms); err != nil {
			log.Printf("[%s] Failed to save Sms to database: %v", modemName, err)
			continue
		}

		newCount++
		log.Printf("[%s] Synced Sms from %s to database: %s", modemName, atSms.Number, atSms.Text)
	}

	return &SmsdbSyncResult{
		ModemName:  modemName,
		TotalCount: totalCount,
		NewCount:   newCount,
	}, nil
}

// SyncAllSmsToDB 从所有已连接Modem同步短信到数据库
func (s *SmsdbService) SyncAllSmsToDB() (*SmsdbSyncAllResult, error) {
	s.modemService.ScanModems()
	modemNames := s.modemService.GetConnectedNames()

	result := &SmsdbSyncAllResult{
		Results:     make([]SmsdbSyncResult, 0, len(modemNames)),
		TotalModems: len(modemNames),
	}

	for _, modemName := range modemNames {
		syncResult, err := s.SyncSmsToDB(modemName)
		if err != nil {
			result.FailedCount++
			result.Results = append(result.Results, SmsdbSyncResult{
				ModemName: modemName,
				Error:     err.Error(),
			})
			continue
		}

		result.SuccessCount++
		result.TotalCount += syncResult.TotalCount
		result.NewCount += syncResult.NewCount
		result.Results = append(result.Results, *syncResult)
	}

	return result, nil
}

// HandleIncomingSms 处理接收到的短信：保存到数据库
func (w *SmsdbService) HandleIncomingSms(dbSms *models.Sms) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Sms] Panic recovered: %v", r)
		}
	}()
	if database.IsSmsdbEnabled() {
		if err := database.CreateSms(dbSms); err != nil {
			log.Printf("[Sms] Failed to save incoming Sms: %v", err)
		}
	}
}
