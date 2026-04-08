package model

import "time"

type Sample struct {
	Timestamp         time.Time `json:"timestamp"`
	CPUPercent        float64   `json:"cpu_percent"`
	Load1             float64   `json:"load_1"`
	Load5             float64   `json:"load_5"`
	Load15            float64   `json:"load_15"`
	MemTotalBytes     uint64    `json:"mem_total_bytes"`
	MemUsedBytes      uint64    `json:"mem_used_bytes"`
	MemAvailableBytes uint64    `json:"mem_available_bytes"`
	SwapTotalBytes    uint64    `json:"swap_total_bytes"`
	SwapUsedBytes     uint64    `json:"swap_used_bytes"`
	DiskTotalBytes    uint64    `json:"disk_total_bytes"`
	DiskUsedBytes     uint64    `json:"disk_used_bytes"`
	DiskFreeBytes     uint64    `json:"disk_free_bytes"`
	NetRxBps          float64   `json:"net_rx_bps"`
	NetTxBps          float64   `json:"net_tx_bps"`
	NetRxAvgBps       float64   `json:"net_rx_avg_bps"`
	NetTxAvgBps       float64   `json:"net_tx_avg_bps"`
	PrimaryInterface  string    `json:"primary_interface"`
}

type DockerContainer struct {
	Name    string `json:"name"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	Health  string `json:"health"`
	Running bool   `json:"running"`
}

type Summary struct {
	UpdatedAt     time.Time         `json:"updated_at"`
	Mode          string            `json:"mode"`
	RetentionDays int               `json:"retention_days"`
	Sample        Sample            `json:"sample"`
	Docker        []DockerContainer `json:"docker"`
	DockerError   string            `json:"docker_error,omitempty"`
}

type HistoryExport struct {
	GeneratedAt time.Time `json:"generated_at"`
	Window      string    `json:"window"`
	Samples     []Sample  `json:"samples"`
	Realtime    []Sample  `json:"realtime"`
}
