package sanitise

import (
	"strings"
	"testing"
	"time"
)

// mixedMB builds a deterministic ~1 MB report with extended characters
// (research.md R10: substitution-heavy path).
func mixedMB() string {
	line := "Temp 37.5°C, dose 5µg/L, β-hCG raised, 5–10 mm, “normal”, area m², ±2 all stable.\n"
	var b strings.Builder
	for b.Len() < 1<<20 {
		b.WriteString(line)
	}
	return b.String()
}

// cleanMB is the already-conformant fast path that dominates real traffic
// (research.md R10: ~99% of production messages are clean).
func cleanMB() string {
	line := "Temp 37.5 deg C, dose 5ug/L, beta-hCG raised, 5-10 mm, \"normal\", area m2, +/-2 all stable.\n"
	var b strings.Builder
	for b.Len() < 1<<20 {
		b.WriteString(line)
	}
	return b.String()
}

func BenchmarkSanitiseMixed1MB(b *testing.B) {
	in := mixedMB()
	b.SetBytes(int64(len(in)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Sanitise(in, Options{Charset: "ascii"}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSanitiseClean1MB(b *testing.B) {
	in := cleanMB()
	b.SetBytes(int64(len(in)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Sanitise(in, Options{Charset: "ascii"}); err != nil {
			b.Fatal(err)
		}
	}
}

// NFR-004 gate: 1 MB in under 100 ms. Median of five runs to shrug off
// scheduler noise; both the mixed and clean paths must meet the budget.
func TestPerformanceBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("performance budget skipped in -short")
	}
	for name, in := range map[string]string{"mixed": mixedMB(), "clean": cleanMB()} {
		times := make([]time.Duration, 0, 5)
		for i := 0; i < 5; i++ {
			start := time.Now()
			if _, err := Sanitise(in, Options{Charset: "ascii"}); err != nil {
				t.Fatal(err)
			}
			times = append(times, time.Since(start))
		}
		for i := 0; i < len(times); i++ {
			for j := i + 1; j < len(times); j++ {
				if times[j] < times[i] {
					times[i], times[j] = times[j], times[i]
				}
			}
		}
		if med := times[2]; med > 100*time.Millisecond {
			t.Errorf("%s 1MB median %v exceeds 100ms budget (NFR-004)", name, med)
		}
	}
}
