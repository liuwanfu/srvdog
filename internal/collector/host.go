package collector

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/liuwanfu/srvdog/internal/model"
)

type cpuSnapshot struct {
	idle  uint64
	total uint64
}

type netSnapshot struct {
	at      time.Time
	rxBytes uint64
	txBytes uint64
}

type rateWindow struct {
	values []float64
	limit  int
}

func newRateWindow(limit int) *rateWindow {
	return &rateWindow{limit: limit}
}

func (w *rateWindow) push(v float64) {
	if w.limit <= 0 {
		return
	}
	w.values = append(w.values, v)
	if len(w.values) > w.limit {
		w.values = w.values[len(w.values)-w.limit:]
	}
}

func (w *rateWindow) average() float64 {
	if len(w.values) == 0 {
		return 0
	}
	var total float64
	for _, value := range w.values {
		total += value
	}
	return total / float64(len(w.values))
}

type HostCollector struct {
	rootPath string

	lastCPU   cpuSnapshot
	hasCPU    bool
	lastNet   netSnapshot
	hasNet    bool
	rxWindow  *rateWindow
	txWindow  *rateWindow
	ifaceHint string
}

func NewHostCollector(rootPath string) *HostCollector {
	if rootPath == "" {
		rootPath = "/"
	}
	return &HostCollector{
		rootPath: rootPath,
		rxWindow: newRateWindow(150),
		txWindow: newRateWindow(150),
	}
}

func (c *HostCollector) Collect(now time.Time) (model.Sample, error) {
	cpu, err := readCPUSnapshot()
	if err != nil {
		return model.Sample{}, err
	}
	load1, load5, load15, err := readLoadAvg()
	if err != nil {
		return model.Sample{}, err
	}
	mem, err := readMemInfo()
	if err != nil {
		return model.Sample{}, err
	}
	diskTotal, diskFree, err := readDiskUsage(c.rootPath)
	if err != nil {
		return model.Sample{}, err
	}
	iface, netNow, err := readNetworkSnapshot(c.ifaceHint, now)
	if err != nil {
		return model.Sample{}, err
	}
	c.ifaceHint = iface

	var cpuPercent float64
	if c.hasCPU {
		idleDelta := cpu.idle - c.lastCPU.idle
		totalDelta := cpu.total - c.lastCPU.total
		if totalDelta > 0 {
			cpuPercent = (1 - float64(idleDelta)/float64(totalDelta)) * 100
		}
	}
	c.lastCPU = cpu
	c.hasCPU = true

	var rxRate float64
	var txRate float64
	if c.hasNet {
		seconds := now.Sub(c.lastNet.at).Seconds()
		if seconds > 0 {
			rxRate = float64(netNow.rxBytes-c.lastNet.rxBytes) / seconds
			txRate = float64(netNow.txBytes-c.lastNet.txBytes) / seconds
		}
	}
	c.lastNet = netNow
	c.hasNet = true
	c.rxWindow.push(rxRate)
	c.txWindow.push(txRate)

	memTotal := mem["MemTotal"]
	memAvail := mem["MemAvailable"]
	swapTotal := mem["SwapTotal"]
	swapFree := mem["SwapFree"]

	return model.Sample{
		Timestamp:         now.UTC(),
		CPUPercent:        cpuPercent,
		Load1:             load1,
		Load5:             load5,
		Load15:            load15,
		MemTotalBytes:     memTotal,
		MemUsedBytes:      memTotal - memAvail,
		MemAvailableBytes: memAvail,
		SwapTotalBytes:    swapTotal,
		SwapUsedBytes:     swapTotal - swapFree,
		DiskTotalBytes:    diskTotal,
		DiskUsedBytes:     diskTotal - diskFree,
		DiskFreeBytes:     diskFree,
		NetRxBps:          rxRate,
		NetTxBps:          txRate,
		NetRxAvgBps:       c.rxWindow.average(),
		NetTxAvgBps:       c.txWindow.average(),
		PrimaryInterface:  iface,
	}, nil
}

func readCPUSnapshot() (cpuSnapshot, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return cpuSnapshot{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return cpuSnapshot{}, errors.New("unexpected cpu stat format")
		}
		var total uint64
		for _, field := range fields[1:] {
			value, err := strconv.ParseUint(field, 10, 64)
			if err != nil {
				return cpuSnapshot{}, err
			}
			total += value
		}
		idle, err := strconv.ParseUint(fields[4], 10, 64)
		if err != nil {
			return cpuSnapshot{}, err
		}
		return cpuSnapshot{idle: idle, total: total}, nil
	}
	if err := scanner.Err(); err != nil {
		return cpuSnapshot{}, err
	}
	return cpuSnapshot{}, errors.New("cpu stat not found")
}

func readLoadAvg() (float64, float64, float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return 0, 0, 0, errors.New("unexpected loadavg format")
	}
	load1, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, 0, 0, err
	}
	load5, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, 0, 0, err
	}
	load15, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return 0, 0, 0, err
	}
	return load1, load5, load15, nil
}

func readMemInfo() (map[string]uint64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values := make(map[string]uint64)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return nil, err
		}
		values[key] = value * 1024
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if _, ok := values["MemAvailable"]; !ok {
		values["MemAvailable"] = values["MemFree"] + values["Buffers"] + values["Cached"]
	}
	return values, nil
}

func readNetworkSnapshot(ifaceHint string, now time.Time) (string, netSnapshot, error) {
	stats, err := readNetDev()
	if err != nil {
		return "", netSnapshot{}, err
	}
	iface := ifaceHint
	if iface == "" {
		iface = primaryInterface()
	}
	if iface == "" {
		for name := range stats {
			if name != "lo" {
				iface = name
				break
			}
		}
	}
	stat, ok := stats[iface]
	if !ok {
		return "", netSnapshot{}, errors.New("network interface not found")
	}
	return iface, netSnapshot{
		at:      now,
		rxBytes: stat.rxBytes,
		txBytes: stat.txBytes,
	}, nil
}

type netCounters struct {
	rxBytes uint64
	txBytes uint64
}

func readNetDev() (map[string]netCounters, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values := make(map[string]netCounters)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		name := strings.TrimSpace(parts[0])
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}
		rxBytes, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return nil, err
		}
		txBytes, err := strconv.ParseUint(fields[8], 10, 64)
		if err != nil {
			return nil, err
		}
		values[name] = netCounters{rxBytes: rxBytes, txBytes: txBytes}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func primaryInterface() string {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 || fields[1] != "00000000" {
			continue
		}
		flags, err := strconv.ParseInt(fields[3], 16, 64)
		if err != nil {
			continue
		}
		if flags&0x2 != 0 {
			return fields[0]
		}
	}
	return ""
}
