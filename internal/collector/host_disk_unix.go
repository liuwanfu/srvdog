//go:build !windows

package collector

import "syscall"

func readDiskUsage(rootPath string) (uint64, uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(rootPath, &stat); err != nil {
		return 0, 0, err
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	return total, free, nil
}
