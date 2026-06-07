package service

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rehiy/modem/at"
	"go.bug.st/serial"
)

var (
	modemOnce     sync.Once
	modemInstance *ModemService
	ModemEvent    = make(chan string, 100)
)

// ModemConn 端口连接
type ModemConn struct {
	Name       string `json:"name"`
	Number     string `json:"number"`
	IMEI       string `json:"imei"`
	ICCID      string `json:"iccid"`
	IMSI       string `json:"imsi"`
	Operator   string `json:"operator"`
	Connected  bool   `json:"connected"`
	*at.Device `json:"-"`
}

type serialPort struct {
	serial.Port
}

func (p *serialPort) Flush() error {
	if err := p.ResetInputBuffer(); err != nil {
		return err
	}
	return p.ResetOutputBuffer()
}

// ModemService 管理多个串口连接
type ModemService struct {
	pool map[string]*ModemConn
	mu   sync.Mutex
}

// GetModemService 返回单例实例
func GetModemService() *ModemService {
	modemOnce.Do(func() {
		modemInstance = &ModemService{
			pool: map[string]*ModemConn{},
		}
	})
	return modemInstance
}

// ScanModems 扫描可用的调制解调器并连接到它们
func (m *ModemService) ScanModems(devs ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 环境变量
	if len(devs) == 0 {
		port := os.Getenv("MODEM_PORT")
		if port != "" {
			devs = strings.Split(port, ",")
		}
	}

	// 查找潜在设备
	switch runtime.GOOS {
	case "windows":
		if len(devs) == 0 {
			devs = []string{"COM1", "COM2", "COM3", "COM4", "COM5"}
		}
	case "darwin":
		if len(devs) == 0 {
			devs = []string{
				"/dev/cu.usbmodem*",
				"/dev/cu.usbserial*",
				"/dev/cu.wchusbserial*",
				"/dev/cu.SLAB_USBtoUART*",
				"/dev/tty.usbmodem*",
				"/dev/tty.usbserial*",
				"/dev/tty.wchusbserial*",
				"/dev/tty.SLAB_USBtoUART*",
			}
		}
		pps := []string{}
		for _, p := range devs {
			matches, _ := filepath.Glob(p)
			pps = append(pps, matches...)
		}
		devs = pps
	default:
		if len(devs) == 0 {
			devs = []string{"/dev/ttyUSB*", "/dev/ttyACM*"}
		}
		pps := []string{}
		for _, p := range devs {
			matches, _ := filepath.Glob(p)
			pps = append(pps, matches...)
		}
		devs = pps
	}

	// 尝试连接到新设备
	for _, u := range devs {
		m.makeConnect(u)
	}
}

// GetConnList 返回已连接的端口信息
func (m *ModemService) GetConnList() []*ModemConn {
	m.mu.Lock()
	defer m.mu.Unlock()

	conns := make([]*ModemConn, 0, len(m.pool))
	for _, model := range m.pool {
		conns = append(conns, model)
	}
	return conns
}

// GetConnectedNames 返回当前可用连接名称列表
func (m *ModemService) GetConnectedNames() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	names := []string{}
	for _, conn := range m.pool {
		if conn == nil || !conn.Connected || conn.Device == nil || !conn.Device.IsOpen() {
			continue
		}
		names = append(names, conn.Name)
	}
	sort.Strings(names)
	return names
}

// GetConn 返回给定端口名称的 AT 接口
func (m *ModemService) GetConn(u string) (*ModemConn, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	n := path.Base(u)
	conn, ok := m.pool[n]
	if !ok {
		return nil, fmt.Errorf("[%s] not found", n)
	}
	if !conn.Connected || conn.Device == nil || conn.Device.IsOpen() == false {
		return nil, fmt.Errorf("[%s] not connected", n)
	}
	return conn, nil
}

// handleIncomingSms 处理指定端口的新接收短信
func (m *ModemService) handleIncomingSms(portName string, smsIndex int) {
	conn, err := m.GetConn(portName)
	if err != nil {
		log.Printf("[%s] Failed to get connection for incoming Sms: %v", portName, err)
		return
	}

	// 获取短信列表（只获取新短信）
	smsList, err := conn.ListSmsPdu(4)
	if err != nil {
		log.Printf("[%s] Failed to list Sms: %v", portName, err)
		return
	}

	smsdbService := NewSmsdbService()
	webhookService := NewWebhookService()

	// 处理每条短信
	for _, atSms := range smsList {
		hasNewSms := false
		for _, idx := range atSms.Indices {
			if idx == smsIndex {
				hasNewSms = true
				break
			}
		}
		if hasNewSms {
			log.Printf("[%s] New Sms from %s: %s", portName, atSms.Number, atSms.Text)
			modelSms := atSmsToModelSms(atSms, conn)
			smsdbService.HandleIncomingSms(modelSms)
			webhookService.HandleIncomingSms(modelSms)
			// 自动删除设备上的短信
			go func() {
				if err := conn.DeleteSms(atSms.Indices); err != nil {
					log.Printf("[%s] failed to delete Sms: %v", portName, err)
				} else {
					log.Printf("[%s] Sms deleted automatically, indices: %v", portName, atSms.Indices)
				}
			}()
		}
	}
}

// makeConnect 添加新的 AT 接口
func (m *ModemService) makeConnect(u string) error {
	n := path.Base(u)

	// 创建日志函数
	pf := func(s string, v ...any) {
		log.Printf(fmt.Sprintf("[%s] %s", n, s), v...)
	}

	// 检查是否已连接
	if conn, ok := m.pool[n]; ok {
		if conn.Test() == nil {
			pf("already connected")
			return nil
		}
		conn.Connected = false
		conn.Close()
	}

	// 创建事件处理函数
	hf := func(e string, p map[int]string) {
		select {
		case ModemEvent <- fmt.Sprintf("%s, %s, %v", n, e, p):
		default:
			log.Printf("[%s] urc dropped: %s", n, e)
		}
		// 处理收到的短信通知
		if e == "+CMTI" && len(p) > 0 {
			if indexStr, ok := p[1]; ok {
				if index, err := strconv.Atoi(indexStr); err == nil {
					m.handleIncomingSms(n, index)
				}
			}
		}
	}

	// 打开串口
	pf("connecting")
	port, err := serial.Open(u, &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	})
	if err != nil {
		pf("connect failed: %v", err)
		return err
	}
	if err := port.SetReadTimeout(1 * time.Second); err != nil {
		port.Close()
		pf("set read timeout failed: %v", err)
		return err
	}

	// 链接新设备
	modem := at.New(&serialPort{Port: port}, hf, &at.Config{Printf: pf})
	if err := modem.Test(); err != nil {
		pf("at test failed: %v", err)
		modem.Close()
		return err
	}

	// 设置默认参数
	modem.EchoOff()     // 关闭回显
	modem.SetSmsMode(0) // PDU 模式
	modem.SetSmsStore("ME", "ME", "ME")

	// 添加到连接池
	m.pool[n] = &ModemConn{
		Name:      n,
		Number:    "unknown",
		Connected: true,
		Device:    modem,
	}

	if imei, err := modem.GetIMEI(); err == nil {
		m.pool[n].IMEI = imei
	} else {
		pf("connected, but failed to get IMEI: %v", err)
	}

	// 获取手机号，用于接收号码
	if number, _, err := modem.GetNumber(); err == nil {
		pf("connected, phone number: %s", number)
		m.pool[n].Number = number
	} else {
		pf("connected, but failed to get phone number: %v", err)
	}

	return nil
}
