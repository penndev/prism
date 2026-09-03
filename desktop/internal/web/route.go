package web

import (
	"desktop/internal/web/rule"
	"desktop/internal/web/subscribe"
	"net/http"
)

// 路由表与内嵌文件服务器只需建一次，不要每个请求重建。
var router = newRouter()

func Route(w http.ResponseWriter, r *http.Request) {
	router.ServeHTTP(w, r)
}

func newRouter() *http.ServeMux {
	router := http.NewServeMux()
	router.HandleFunc("/", handleRoot)
	router.HandleFunc("/api/app-config", handleAppConfig)
	router.Handle("/common/", http.StripPrefix("/common/", handleCommonFileServer()))
	// 订阅页面
	router.HandleFunc("/subscribe", subscribe.HandleSubscribeRedirect)
	router.HandleFunc("/subscribe/logo.png", subscribe.HandleSubscribeLogo)
	router.HandleFunc("/subscribe/api/servers", subscribe.HandleServers)
	router.HandleFunc("/subscribe/api/servers/import", subscribe.HandleImportServers)
	router.HandleFunc("/subscribe/api/servers/export", subscribe.HandleExportServers)
	router.HandleFunc("/subscribe/api/subscription/convert", subscribe.HandleSubscriptionConvert)
	router.Handle("/subscribe/", http.StripPrefix("/subscribe/", subscribe.HandleSubscribeFileServer()))
	// 规则管理
	router.HandleFunc("/rule", rule.HandleRuleRedirect)
	router.Handle("/rule/", http.StripPrefix("/rule/", rule.HandleRuleFileServer()))
	router.HandleFunc("/rule/api/config", rule.HandleRuleConfig)
	router.HandleFunc("/rule/api/areas", rule.HandleRuleAreas)
	router.HandleFunc("/rule/api/db", rule.HandleRuleDB)
	router.HandleFunc("/rule/api/db/download", rule.HandleRuleDBDownload)
	router.HandleFunc("/rule/api/db/upload", rule.HandleRuleDBUpload)
	return router
}
