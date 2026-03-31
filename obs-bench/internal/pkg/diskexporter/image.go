package diskexporter

import (
	"encoding/hex"
	"fmt"

	"golang.org/x/mod/sumdb/dirhash"
)

// ContextPath — каталог с Dockerfile disk-usage-metrics-exporter (от корня модуля obs-bench).
const ContextPath = "./disk-usage-metrics-exporter"

// BuildDevImageTag — детерминированный локальный тег по хешу исходников exporter'а.
func BuildDevImageTag() (string, error) {
	hash, err := dirhash.HashDir(ContextPath, "", dirhash.Hash1)
	if err != nil {
		return "", err
	}
	hexHash := hex.EncodeToString([]byte(hash))
	return fmt.Sprintf("disk-usage-metrics-exporter:dev-%s", hexHash[:12]), nil
}
