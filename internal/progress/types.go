package progress

import (
	"strings"
	"time"
)

// EventType represents the type of progress event
type EventType int

const (
	EventScanStart EventType = iota
	EventScanComplete
	EventEnterDirectory
	EventLeaveDirectory
	EventComponentDetected
	EventFileProcessingStart
	EventFileProcessingEnd
	EventFolderFileProcessingStart
	EventFolderFileProcessingEnd
	EventSkipped
	EventProgress
	EventScanInitializing
	EventFileWriting
	EventFileWritten
	EventInfo
	EventRuleCheck
	EventRuleResult
	EventGitIgnoreEnter
	EventGitIgnoreLeave
)

// Event represents something that happened during scanning
type Event struct {
	Type      EventType
	Path      string
	Name      string
	Tech      string
	Info      string
	Reason    string
	FileCount int
	DirCount  int
	Duration  time.Duration
	Timestamp time.Time // For timing calculations
	Matched   bool      // For rule matching results
	Details   []string  // For detailed rule check information
}

// Reporter is the interface the scanner uses to report events
type Reporter interface {
	Report(event Event)
}

// Handler processes events and produces output
type Handler interface {
	Handle(event Event)
}

// TimingEntry represents a directory timing for analysis
type TimingEntry struct {
	Path     string
	Duration time.Duration
	Depth    int
}

// RuleEntry represents a rule match for analysis
type RuleEntry struct {
	Tech    string
	Reason  string
	Path    string
	Matched bool
}

// getTimingIcon returns the appropriate icon for a duration
func getTimingIcon(seconds float64) string {
	if seconds >= 10.0 {
		return "🔴" // Slow
	} else if seconds >= 1.0 {
		return "🟡" // Medium
	}
	return "🟢" // Fast
}

// shortenPath shortens a path for display if it's too long
func shortenPath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	parts := strings.Split(path, "/")
	if len(parts) > 3 {
		return "..." + "/" + strings.Join(parts[len(parts)-2:], "/")
	}
	return path
}

// sortTimingsByDuration sorts timings by duration descending (partial sort for top N)
func sortTimingsByDuration(timings []TimingEntry, topN int) []TimingEntry {
	sorted := make([]TimingEntry, len(timings))
	copy(sorted, timings)
	for i := 0; i < len(sorted)-1 && i < topN; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Duration < sorted[j+1].Duration {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	return sorted
}
