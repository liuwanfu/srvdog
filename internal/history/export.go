package history

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strconv"
	"time"

	"github.com/liuwanfu/srvdog/internal/model"
)

func ExportJSON(window string, samples, realtime []model.Sample) ([]byte, string, error) {
	payload := model.HistoryExport{
		GeneratedAt: time.Now().UTC(),
		Window:      window,
		Samples:     samples,
		Realtime:    realtime,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, "", err
	}
	return data, "application/json", nil
}

func ExportCSV(samples, realtime []model.Sample) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	header := []string{
		"scope", "timestamp", "cpu_percent", "load_1", "load_5", "load_15",
		"mem_total_bytes", "mem_used_bytes", "mem_available_bytes",
		"swap_total_bytes", "swap_used_bytes",
		"disk_total_bytes", "disk_used_bytes", "disk_free_bytes",
		"net_rx_bps", "net_tx_bps", "net_rx_avg_bps", "net_tx_avg_bps", "primary_interface",
	}
	if err := writer.Write(header); err != nil {
		return nil, "", err
	}
	writeSample := func(scope string, sample model.Sample) error {
		record := []string{
			scope,
			sample.Timestamp.Format(time.RFC3339),
			strconv.FormatFloat(sample.CPUPercent, 'f', 3, 64),
			strconv.FormatFloat(sample.Load1, 'f', 3, 64),
			strconv.FormatFloat(sample.Load5, 'f', 3, 64),
			strconv.FormatFloat(sample.Load15, 'f', 3, 64),
			strconv.FormatUint(sample.MemTotalBytes, 10),
			strconv.FormatUint(sample.MemUsedBytes, 10),
			strconv.FormatUint(sample.MemAvailableBytes, 10),
			strconv.FormatUint(sample.SwapTotalBytes, 10),
			strconv.FormatUint(sample.SwapUsedBytes, 10),
			strconv.FormatUint(sample.DiskTotalBytes, 10),
			strconv.FormatUint(sample.DiskUsedBytes, 10),
			strconv.FormatUint(sample.DiskFreeBytes, 10),
			strconv.FormatFloat(sample.NetRxBps, 'f', 3, 64),
			strconv.FormatFloat(sample.NetTxBps, 'f', 3, 64),
			strconv.FormatFloat(sample.NetRxAvgBps, 'f', 3, 64),
			strconv.FormatFloat(sample.NetTxAvgBps, 'f', 3, 64),
			sample.PrimaryInterface,
		}
		return writer.Write(record)
	}
	for _, sample := range samples {
		if err := writeSample("history", sample); err != nil {
			return nil, "", err
		}
	}
	for _, sample := range realtime {
		if err := writeSample("realtime", sample); err != nil {
			return nil, "", err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "text/csv", nil
}
