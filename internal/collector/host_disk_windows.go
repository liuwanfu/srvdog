//go:build windows

package collector

import "errors"

func readDiskUsage(rootPath string) (uint64, uint64, error) {
	return 0, 0, errors.New("disk usage collection is only supported on linux targets")
}
