package profiler

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Profiler tracks build performance
type Profiler struct {
	mu        sync.Mutex
	startTime time.Time
	phases    []PhaseRecord
	current   *PhaseRecord
	enabled   bool
}

// PhaseRecord holds timing for a single build phase
type PhaseRecord struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
}

// PerformanceReport holds the complete profiling results
type PerformanceReport struct {
	TotalTime   time.Duration
	Phases      []PhaseReport
	MemoryStats MemoryStats
	FileStats   FileStats
}

// PhaseReport holds a single phase's stats
type PhaseReport struct {
	Name     string
	Duration time.Duration
	Percent  float64
}

// MemoryStats holds memory usage information
type MemoryStats struct {
	AllocMB   float64
	SysMB     float64
	NumGC     uint32
}

// FileStats holds file processing stats
type FileStats struct {
	MarkdownFiles int
	ImageFiles    int
	AssetFiles    int
}

// New creates a new profiler
func New(enabled bool) *Profiler {
	return &Profiler{
		enabled: enabled,
		phases:  make([]PhaseRecord, 0),
	}
}

// Start begins profiling
func (p *Profiler) Start() {
	if !p.enabled {
		return
	}
	p.startTime = time.Now()
}

// StartPhase begins timing a build phase
func (p *Profiler) StartPhase(name string) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	// End current phase if any
	if p.current != nil {
		p.current.EndTime = time.Now()
		p.current.Duration = p.current.EndTime.Sub(p.current.StartTime)
		p.phases = append(p.phases, *p.current)
	}

	p.current = &PhaseRecord{
		Name:      name,
		StartTime: time.Now(),
	}
}

// EndPhase ends the current phase
func (p *Profiler) EndPhase() {
	if !p.enabled || p.current == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current.EndTime = time.Now()
	p.current.Duration = p.current.EndTime.Sub(p.current.StartTime)
	p.phases = append(p.phases, *p.current)
	p.current = nil
}

// Report generates the performance report
func (p *Profiler) Report(fileStats FileStats) *PerformanceReport {
	if !p.enabled {
		return nil
	}

	// End current phase if still running
	p.EndPhase()

	totalTime := time.Since(p.startTime)

	report := &PerformanceReport{
		TotalTime: totalTime,
		FileStats: fileStats,
	}

	// Build phase reports
	for _, phase := range p.phases {
		percent := 0.0
		if totalTime > 0 {
			percent = float64(phase.Duration) / float64(totalTime) * 100
		}
		report.Phases = append(report.Phases, PhaseReport{
			Name:     phase.Name,
			Duration: phase.Duration,
			Percent:  percent,
		})
	}

	// Memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	report.MemoryStats = MemoryStats{
		AllocMB: float64(m.Alloc) / 1024 / 1024,
		SysMB:   float64(m.Sys) / 1024 / 1024,
		NumGC:   m.NumGC,
	}

	return report
}

// FormatReport formats a performance report as a string
func FormatReport(report *PerformanceReport) string {
	if report == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("\nBuild Performance Report\n")
	sb.WriteString("========================\n")
	sb.WriteString(fmt.Sprintf("Total time: %s\n", report.TotalTime.Round(time.Millisecond)))
	sb.WriteString(fmt.Sprintf("Memory: %.1f MB alloc, %.1f MB sys, %d GC cycles\n",
		report.MemoryStats.AllocMB, report.MemoryStats.SysMB, report.MemoryStats.NumGC))

	sb.WriteString("\nPhases:\n")
	for _, phase := range report.Phases {
		bar := strings.Repeat("█", int(phase.Percent/5))
		sb.WriteString(fmt.Sprintf("  %-25s %8s (%4.1f%%) %s\n",
			phase.Name, phase.Duration.Round(time.Millisecond), phase.Percent, bar))
	}

	sb.WriteString(fmt.Sprintf("\nFiles: %d markdown, %d images, %d assets\n",
		report.FileStats.MarkdownFiles, report.FileStats.ImageFiles, report.FileStats.AssetFiles))

	return sb.String()
}
