package views

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/PeterHiroshi/clawmon/tui/internal/models"
)

func TestRenderBar(t *testing.T) {
	tests := []struct {
		name    string
		percent float32
	}{
		{"zero", 0},
		{"low", 25.0},
		{"normal boundary", 50.0},
		{"warning", 60.0},
		{"high boundary", 75.0},
		{"high", 85.0},
		{"critical boundary", 90.0},
		{"critical", 95.0},
		{"full", 100.0},
		{"negative clamped", -5.0},
		{"over clamped", 150.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := RenderBar(tt.percent)
			assert.NotEmpty(t, bar)
		})
	}
}

func TestBarStyle(t *testing.T) {
	tests := []struct {
		name    string
		percent float32
	}{
		{"normal green", 25.0},
		{"warning yellow", 55.0},
		{"high orange", 80.0},
		{"critical red", 95.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := BarStyle(tt.percent)
			// Verify the style returns a valid style (just check it doesn't panic)
			rendered := style.Render("test")
			assert.NotEmpty(t, rendered)
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    uint64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{16000000000, "14.9 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderSystemNil(t *testing.T) {
	result := RenderSystem(nil, 80, 24)
	assert.Contains(t, result, "No system data")
}

func TestRenderSystemFull(t *testing.T) {
	sys := &models.SystemResources{
		CPU: models.CpuInfo{
			Model:        "Test CPU",
			CoreCount:    4,
			UsagePercent: 48.2,
			PerCoreUsage: []float32{45.0, 55.0, 48.0, 44.0},
			LoadAvg1m:    2.15,
			LoadAvg5m:    1.87,
			LoadAvg15m:   1.42,
		},
		Memory: models.MemoryInfo{
			TotalBytes:     16000000000,
			UsedBytes:      12400000000,
			AvailableBytes: 3600000000,
			UsagePercent:   78.5,
		},
		Swap: models.SwapInfo{
			TotalBytes:   10000000000,
			UsedBytes:    820000000,
			UsagePercent: 8.2,
		},
		Disks: []models.DiskInfo{
			{
				MountPoint:     "/",
				Filesystem:     "ext4",
				TotalBytes:     200000000000,
				UsedBytes:      136600000000,
				AvailableBytes: 63400000000,
				UsagePercent:   68.3,
			},
		},
		GPUs: []models.GpuInfo{
			{
				Name:               "NVIDIA RTX 4090",
				MemoryUsedMB:       21900,
				MemoryTotalMB:      24000,
				MemoryUsagePercent: 91.2,
				UtilizationPercent: 82.5,
				TemperatureCelsius: 72.0,
			},
		},
		UptimeSeconds: 86400,
		CollectedAt:   "2026-03-18T19:00:00Z",
	}

	result := RenderSystem(sys, 80, 40)
	assert.Contains(t, result, "CPU")
	assert.Contains(t, result, "Test CPU")
	assert.Contains(t, result, "4 cores")
	assert.Contains(t, result, "Memory")
	assert.Contains(t, result, "Disk")
	assert.Contains(t, result, "GPU")
	assert.Contains(t, result, "NVIDIA RTX 4090")
	assert.Contains(t, result, "72°C")
}

func TestRenderSystemNoGPU(t *testing.T) {
	sys := &models.SystemResources{
		CPU: models.CpuInfo{
			Model:        "Test CPU",
			CoreCount:    2,
			UsagePercent: 10.0,
		},
		Memory: models.MemoryInfo{
			TotalBytes:     8000000000,
			UsedBytes:      4000000000,
			AvailableBytes: 4000000000,
			UsagePercent:   50.0,
		},
		Swap:          models.SwapInfo{},
		Disks:         []models.DiskInfo{},
		GPUs:          []models.GpuInfo{},
		UptimeSeconds: 100,
		CollectedAt:   "2026-03-18T19:00:00Z",
	}

	result := RenderSystem(sys, 80, 40)
	assert.Contains(t, result, "No GPU detected")
	assert.Contains(t, result, "No disks detected")
}

func TestRenderSystemMultipleDisks(t *testing.T) {
	sys := &models.SystemResources{
		CPU: models.CpuInfo{Model: "CPU", CoreCount: 1},
		Memory: models.MemoryInfo{
			TotalBytes: 8000000000, UsedBytes: 4000000000,
			AvailableBytes: 4000000000, UsagePercent: 50.0,
		},
		Swap: models.SwapInfo{},
		Disks: []models.DiskInfo{
			{MountPoint: "/", Filesystem: "ext4", TotalBytes: 100000000000, UsedBytes: 50000000000, AvailableBytes: 50000000000, UsagePercent: 50.0},
			{MountPoint: "/home", Filesystem: "ext4", TotalBytes: 200000000000, UsedBytes: 84200000000, AvailableBytes: 115800000000, UsagePercent: 42.1},
		},
		GPUs:          []models.GpuInfo{},
		UptimeSeconds: 100,
		CollectedAt:   "2026-03-18T19:00:00Z",
	}

	result := RenderSystem(sys, 80, 40)
	assert.Contains(t, result, "/")
	assert.Contains(t, result, "/home")
}
