package collector

import (
	"bufio"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/liuwanfu/srvdog/internal/model"
)

type DockerCollector struct {
	timeout time.Duration
}

func NewDockerCollector(timeout time.Duration) *DockerCollector {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &DockerCollector{timeout: timeout}
}

func (c *DockerCollector) Collect(ctx context.Context) ([]model.DockerContainer, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{json .}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	type row struct {
		Names  string
		Image  string
		Status string
	}

	containers := make([]model.DockerContainer, 0)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item row
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			return nil, err
		}
		health := ""
		switch {
		case strings.Contains(item.Status, "healthy"):
			health = "healthy"
		case strings.Contains(item.Status, "unhealthy"):
			health = "unhealthy"
		case strings.Contains(item.Status, "starting"):
			health = "starting"
		}
		containers = append(containers, model.DockerContainer{
			Name:    item.Names,
			Image:   item.Image,
			Status:  item.Status,
			Health:  health,
			Running: strings.HasPrefix(item.Status, "Up"),
		})
	}
	return containers, scanner.Err()
}
