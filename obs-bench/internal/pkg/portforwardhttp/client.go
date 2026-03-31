// Package portforwardhttp — HTTP-клиент для запросов через kubectl-style port-forward на localhost.
// Без keep-alive и с полным сбросом тела ответа, чтобы реже ловить RST/broken pipe в логах port-forward.
package portforwardhttp

import (
	"io"
	"net/http"
	"time"
)

// Client для query к Prometheus-совместимому API за локальным пробросом порта.
var Client = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true,
		// HTTP/2 через SPDY port-forward не нужен и иногда ведёт себя хуже.
		ForceAttemptHTTP2: false,
	},
	Timeout: 60 * time.Second,
}

// CloseResp полностью читает тело и закрывает ответ (важно до следующего запроса по тому же туннелю).
func CloseResp(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}
