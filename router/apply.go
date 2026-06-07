package router

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"github.com/rehiy/web-modem/handler"
	"github.com/rehiy/web-modem/webview"
)

func Apply() http.Handler {
	r := mux.NewRouter()

	if target := os.Getenv("API_PROXY_TARGET"); target != "" {
		if err := ProxyRegister(r, target); err != nil {
			log.Printf("API proxy disabled: %v", err)
			LocalAPIRegister(r)
		}
	} else {
		LocalAPIRegister(r)
	}

	// 静态文件服务
	StaticServer(r)

	// 应用 Basic Auth 中间件
	return BasicAuthMiddleware(r)
}

func LocalAPIRegister(r *mux.Router) {
	api := r.PathPrefix("/api").Subrouter()
	ModemRegister(api)
	SmsdbRegister(api)
	WebhookRegister(api)
	AlertRegister(api)
	SettingRegister(api)
	UpdateRegister(api)

	// SSE
	SseRegister(r)
}

func ModemRegister(r *mux.Router) {
	mh := handler.NewModemHandler()

	// 模块列表
	r.HandleFunc("/modem/list", mh.ListModems).Methods("GET")

	// 模块操作
	r.HandleFunc("/modem/send", mh.SendModemCommand).Methods("POST")
	r.HandleFunc("/modem/info", mh.GetModemBasicInfo).Methods("GET")
	r.HandleFunc("/modem/signal", mh.GetModemSignal).Methods("GET")

	// 短信读写
	r.HandleFunc("/modem/sms/list", mh.ListModemSms).Methods("GET")
	r.HandleFunc("/modem/sms/send", mh.SendModemSms).Methods("POST")
	r.HandleFunc("/modem/sms/delete", mh.DeleteModemSms).Methods("POST")
}

func SmsdbRegister(r *mux.Router) {
	dh := handler.NewSmsdbHandler()

	// 短信存储管理
	r.HandleFunc("/smsdb/list", dh.ListSms).Methods("GET")
	r.HandleFunc("/smsdb/delete", dh.DeleteSmsBatch).Methods("POST")
	r.HandleFunc("/smsdb/sync", dh.SyncSms).Methods("POST")
}

func WebhookRegister(r *mux.Router) {
	wh := handler.NewWebhookHandler()

	// Webhook配置管理
	r.HandleFunc("/webhook", wh.CreateWebhook).Methods("POST")
	r.HandleFunc("/webhook/list", wh.ListWebhooks).Methods("GET")
	r.HandleFunc("/webhook/get", wh.GetWebhook).Methods("GET")
	r.HandleFunc("/webhook/update", wh.UpdateWebhook).Methods("PUT")
	r.HandleFunc("/webhook/delete", wh.DeleteWebhook).Methods("DELETE")
	r.HandleFunc("/webhook/preview", wh.PreviewWebhook).Methods("POST")
	r.HandleFunc("/webhook/deliveries", wh.ListWebhookDeliveries).Methods("GET")
	r.HandleFunc("/webhook/test", wh.TestWebhook).Methods("POST")
}

func AlertRegister(r *mux.Router) {
	ah := handler.NewAlertHandler()

	r.HandleFunc("/alerts/list", ah.ListAlerts).Methods("GET")
	r.HandleFunc("/alerts/scan", ah.ScanAlerts).Methods("POST")
	r.HandleFunc("/alerts/resolve", ah.ResolveAlerts).Methods("POST")
}

func SettingRegister(r *mux.Router) {
	sh := handler.NewSettingHandler()

	// 设置管理
	r.HandleFunc("/settings", sh.GetSettings).Methods("GET")
	r.HandleFunc("/settings/smsdb", sh.UpdateSmsdbSettings).Methods("PUT")
	r.HandleFunc("/settings/webhook", sh.UpdateWebhookSettings).Methods("PUT")
}

func UpdateRegister(r *mux.Router) {
	uh := handler.NewUpdateHandler()

	r.HandleFunc("/update/check", uh.CheckUpdate).Methods("GET")
	r.HandleFunc("/update/apply", uh.ApplyUpdate).Methods("POST")
	r.HandleFunc("/update/restart", uh.Restart).Methods("POST")
}

func SseRegister(r *mux.Router) {
	sse := handler.NewSseHandler()

	r.HandleFunc("/events/modem", sse.HandleModemEvents).Methods("GET")
}

func StaticServer(r *mux.Router) {
	hfs := http.FileServer(http.FS(webview.FS))
	if _, err := os.Stat("./webview"); err == nil {
		hfs = http.FileServer(http.Dir("./webview"))
	}

	r.PathPrefix("/").Handler(hfs)
}
