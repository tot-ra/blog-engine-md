package profiler

import (
	"testing"
	"time"
)

func TestProfiler_Basic(t *testing.T) {
	p := New(true)
	p.Start()

	p.StartPhase("discovery")
	time.Sleep(10 * time.Millisecond)
	p.EndPhase()

	p.StartPhase("rendering")
	time.Sleep(20 * time.Millisecond)
	p.EndPhase()

	report := p.Report(FileStats{MarkdownFiles: 5, ImageFiles: 3, AssetFiles: 2})

	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if len(report.Phases) != 2 {
		t.Errorf("expected 2 phases, got %d", len(report.Phases))
	}
	if report.Phases[0].Name != "discovery" {
		t.Errorf("expected first phase 'discovery', got '%s'", report.Phases[0].Name)
	}
	if report.TotalTime < 30*time.Millisecond {
		t.Errorf("expected total time >= 30ms, got %s", report.TotalTime)
	}
	if report.FileStats.MarkdownFiles != 5 {
		t.Errorf("expected 5 markdown files, got %d", report.FileStats.MarkdownFiles)
	}
}

func TestProfiler_Disabled(t *testing.T) {
	p := New(false)
	p.Start()
	p.StartPhase("test")
	p.EndPhase()

	report := p.Report(FileStats{})
	if report != nil {
		t.Error("expected nil report when profiler is disabled")
	}
}

func TestProfiler_AutoEndPhase(t *testing.T) {
	p := New(true)
	p.Start()
	p.StartPhase("phase1")
	time.Sleep(5 * time.Millisecond)
	// StartPhase should auto-end previous
	p.StartPhase("phase2")
	time.Sleep(5 * time.Millisecond)

	report := p.Report(FileStats{})

	if len(report.Phases) != 2 {
		t.Errorf("expected 2 phases, got %d", len(report.Phases))
	}
}

func TestFormatReport(t *testing.T) {
	report := &PerformanceReport{
		TotalTime: 100 * time.Millisecond,
		Phases: []PhaseReport{
			{Name: "discovery", Duration: 30 * time.Millisecond, Percent: 30},
			{Name: "rendering", Duration: 70 * time.Millisecond, Percent: 70},
		},
		MemoryStats: MemoryStats{AllocMB: 10.5, SysMB: 20.3, NumGC: 3},
		FileStats:   FileStats{MarkdownFiles: 10, ImageFiles: 5, AssetFiles: 3},
	}

	output := FormatReport(report)
	if output == "" {
		t.Error("expected non-empty formatted report")
	}
	if len(output) < 100 {
		t.Error("expected detailed report")
	}
}

func TestFormatReport_Nil(t *testing.T) {
	output := FormatReport(nil)
	if output != "" {
		t.Error("expected empty string for nil report")
	}
}
