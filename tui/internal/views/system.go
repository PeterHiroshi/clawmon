package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/PeterHiroshi/clawmon/tui/internal/models"
)

// Progress bar constants.
const (
	BarWidth = 20

	ThresholdNormal  float32 = 50.0
	ThresholdWarning float32 = 75.0
	ThresholdHigh    float32 = 90.0
)

// Color thresholds for usage bars.
var (
	ColorOrange     = lipgloss.Color("#ff8800")
	BarNormalStyle  = lipgloss.NewStyle().Foreground(ColorGreen)
	BarWarningStyle = lipgloss.NewStyle().Foreground(ColorYellow)
	BarHighStyle    = lipgloss.NewStyle().Foreground(ColorOrange)
	BarCritStyle    = lipgloss.NewStyle().Foreground(ColorRed)
)

// RenderSystem renders the system resources tab.
func RenderSystem(sys *models.SystemResources, width, height int) string {
	if sys == nil {
		return lipgloss.Place(width, height/2, lipgloss.Center, lipgloss.Center,
			DimStyle.Render("No system data"))
	}

	cardWidth := width - 4
	if cardWidth > MaxCardWidth*2 {
		cardWidth = MaxCardWidth * 2
	}
	if cardWidth < MinCardWidth {
		cardWidth = MinCardWidth
	}

	var sections []string
	sections = append(sections, renderCpuSection(sys.CPU, cardWidth))
	sections = append(sections, renderMemorySection(sys.Memory, sys.Swap, cardWidth))
	sections = append(sections, renderDiskSection(sys.Disks, cardWidth))
	sections = append(sections, renderGpuSection(sys.GPUs, cardWidth))

	return strings.Join(sections, "\n")
}

func renderCpuSection(cpu models.CpuInfo, width int) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("Model: %s (%d cores)", cpu.Model, cpu.CoreCount))
	lines = append(lines, fmt.Sprintf("Usage: %s %.1f%%", RenderBar(cpu.UsagePercent), cpu.UsagePercent))
	lines = append(lines, fmt.Sprintf("Load:  %.2f / %.2f / %.2f", cpu.LoadAvg1m, cpu.LoadAvg5m, cpu.LoadAvg15m))

	content := TitleStyle.Render("CPU") + "\n" + strings.Join(lines, "\n")
	return CardStyle.Width(width).Render(content)
}

func renderMemorySection(mem models.MemoryInfo, swap models.SwapInfo, width int) string {
	var lines []string

	lines = append(lines, fmt.Sprintf("RAM:  %s %.1f%%", RenderBar(mem.UsagePercent), mem.UsagePercent))
	lines = append(lines, fmt.Sprintf("      %s / %s (%s free)",
		FormatBytes(mem.UsedBytes), FormatBytes(mem.TotalBytes), FormatBytes(mem.AvailableBytes)))

	lines = append(lines, fmt.Sprintf("Swap: %s %.1f%%", RenderBar(swap.UsagePercent), swap.UsagePercent))
	lines = append(lines, fmt.Sprintf("      %s / %s",
		FormatBytes(swap.UsedBytes), FormatBytes(swap.TotalBytes)))

	content := TitleStyle.Render("Memory") + "\n" + strings.Join(lines, "\n")
	return CardStyle.Width(width).Render(content)
}

func renderDiskSection(disks []models.DiskInfo, width int) string {
	if len(disks) == 0 {
		content := TitleStyle.Render("Disk") + "\n" + DimStyle.Render("No disks detected")
		return CardStyle.Width(width).Render(content)
	}

	var lines []string
	for _, d := range disks {
		lines = append(lines, fmt.Sprintf("%-5s %s %.1f%%",
			Truncate(d.MountPoint, 5), RenderBar(d.UsagePercent), d.UsagePercent))
		lines = append(lines, fmt.Sprintf("      %s / %s",
			FormatBytes(d.UsedBytes), FormatBytes(d.TotalBytes)))
	}

	content := TitleStyle.Render("Disk") + "\n" + strings.Join(lines, "\n")
	return CardStyle.Width(width).Render(content)
}

func renderGpuSection(gpus []models.GpuInfo, width int) string {
	if len(gpus) == 0 {
		content := TitleStyle.Render("GPU") + "\n" + DimStyle.Render("No GPU detected")
		return CardStyle.Width(width).Render(content)
	}

	var lines []string
	for _, g := range gpus {
		lines = append(lines, g.Name)
		lines = append(lines, fmt.Sprintf("VRAM: %s %.1f%%",
			RenderBar(g.MemoryUsagePercent), g.MemoryUsagePercent))
		lines = append(lines, fmt.Sprintf("      %d MB / %d MB", g.MemoryUsedMB, g.MemoryTotalMB))
		lines = append(lines, fmt.Sprintf("Util: %s %.1f%%",
			RenderBar(g.UtilizationPercent), g.UtilizationPercent))
		lines = append(lines, fmt.Sprintf("Temp: %.0f°C", g.TemperatureCelsius))
	}

	content := TitleStyle.Render("GPU") + "\n" + strings.Join(lines, "\n")
	return CardStyle.Width(width).Render(content)
}

// RenderBar renders a color-coded progress bar for a percentage value.
func RenderBar(percent float32) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	filled := int(percent / 100.0 * float32(BarWidth))
	if filled > BarWidth {
		filled = BarWidth
	}
	empty := BarWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return BarStyle(percent).Render(bar)
}

// BarStyle returns the lipgloss style for a given usage percentage.
func BarStyle(percent float32) lipgloss.Style {
	switch {
	case percent >= ThresholdHigh:
		return BarCritStyle
	case percent >= ThresholdWarning:
		return BarHighStyle
	case percent >= ThresholdNormal:
		return BarWarningStyle
	default:
		return BarNormalStyle
	}
}

// FormatBytes formats a byte count as a human-readable string.
func FormatBytes(bytes uint64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
		tb = 1024 * gb
	)

	switch {
	case bytes >= tb:
		return fmt.Sprintf("%.1f TB", float64(bytes)/float64(tb))
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
