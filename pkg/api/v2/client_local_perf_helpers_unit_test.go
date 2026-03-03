//go:build soak && !cloud

package v2

import (
	"strings"
	"testing"
	"time"
)

func TestPerfLatencyBucketIndexAndLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    time.Duration
		want int
	}{
		{name: "zero", d: 0, want: 0},
		{name: "first boundary", d: 5 * time.Millisecond, want: 0},
		{name: "second bucket", d: 6 * time.Millisecond, want: 1},
		{name: "overflow", d: 15 * time.Second, want: len(perfLatencyBounds)},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := perfLatencyBucketIndex(tc.d)
			if got != tc.want {
				t.Fatalf("perfLatencyBucketIndex(%s)=%d want %d", tc.d, got, tc.want)
			}
		})
	}

	if got := perfLatencyBucketLabel(0); got != "<=5ms" {
		t.Fatalf("perfLatencyBucketLabel(0)=%q want <=5ms", got)
	}
	overflow := perfLatencyBucketLabel(len(perfLatencyBounds))
	if overflow != ">10000ms" {
		t.Fatalf("overflow label=%q want >10000ms", overflow)
	}
}

func TestPerfP95FromBuckets(t *testing.T) {
	t.Parallel()

	buckets := make([]uint64, len(perfLatencyBounds)+1)
	if got := perfP95FromBuckets(buckets, 0, 0); got != 0 {
		t.Fatalf("zero-total p95=%f want 0", got)
	}

	buckets = make([]uint64, len(perfLatencyBounds)+1)
	buckets[0] = 10
	if got := perfP95FromBuckets(buckets, 10, 5*time.Millisecond); got != 5 {
		t.Fatalf("first-bucket p95=%f want 5", got)
	}

	buckets = make([]uint64, len(perfLatencyBounds)+1)
	buckets[0] = 94
	buckets[1] = 6
	if got := perfP95FromBuckets(buckets, 100, 10*time.Millisecond); got != 10 {
		t.Fatalf("second-bucket p95=%f want 10", got)
	}

	buckets = make([]uint64, len(perfLatencyBounds)+1)
	buckets[len(perfLatencyBounds)] = 1
	if got := perfP95FromBuckets(buckets, 1, 15*time.Second); got != 15000 {
		t.Fatalf("overflow-max p95=%f want 15000", got)
	}
	if got := perfP95FromBuckets(buckets, 1, 2*time.Second); got != 10000 {
		t.Fatalf("overflow-capped p95=%f want 10000", got)
	}
}

func TestPerfFilterCapturedSamplesAndCountErrors(t *testing.T) {
	t.Parallel()

	samples := []perfSample{
		{SnapshotCaptured: true},
		{SnapshotCaptured: false},
		{SnapshotCaptured: true},
	}
	filtered := perfFilterCapturedSamples(samples)
	if len(filtered) != 2 {
		t.Fatalf("filtered len=%d want 2", len(filtered))
	}
	if got := perfCountSampleCaptureErrors(samples); got != 1 {
		t.Fatalf("capture errors=%d want 1", got)
	}
}

func TestPerfSlopePerMinute(t *testing.T) {
	t.Parallel()

	if got := perfSlopePerMinute(nil, func(sample perfSample) float64 { return sample.ElapsedSeconds }); got != 0 {
		t.Fatalf("nil slope=%f want 0", got)
	}
	if got := perfSlopePerMinute([]perfSample{{Timestamp: time.Now().UTC()}}, func(sample perfSample) float64 { return 1 }); got != 0 {
		t.Fatalf("single-sample slope=%f want 0", got)
	}

	ts := time.Now().UTC()
	sameTime := []perfSample{
		{Timestamp: ts, HeapAllocBytes: 10},
		{Timestamp: ts, HeapAllocBytes: 20},
	}
	if got := perfSlopePerMinute(sameTime, func(sample perfSample) float64 { return float64(sample.HeapAllocBytes) }); got != 0 {
		t.Fatalf("same-time slope=%f want 0", got)
	}

	samples := []perfSample{
		{Timestamp: ts, HeapAllocBytes: 10},
		{Timestamp: ts.Add(2 * time.Minute), HeapAllocBytes: 16},
	}
	got := perfSlopePerMinute(samples, func(sample perfSample) float64 { return float64(sample.HeapAllocBytes) })
	if got != 3 {
		t.Fatalf("slope=%f want 3", got)
	}
}

func TestPerfThroughputQuartiles(t *testing.T) {
	t.Parallel()

	if first, last := perfThroughputQuartiles([]perfSample{{}, {}, {}, {}}); first != 0 || last != 0 {
		t.Fatalf("insufficient quartiles=(%f,%f) want (0,0)", first, last)
	}

	base := time.Now().UTC()
	opsTotals := []uint64{0, 10, 20, 40, 60, 90, 120, 160, 200}
	samples := make([]perfSample, 0, len(opsTotals))
	for i, ops := range opsTotals {
		samples = append(samples, perfSample{
			Timestamp:      base.Add(time.Duration(i) * time.Second),
			ElapsedSeconds: float64(i),
			OpsTotal:       ops,
		})
	}
	first, last := perfThroughputQuartiles(samples)
	if first != 10 {
		t.Fatalf("first quartile=%f want 10", first)
	}
	if last != 40 {
		t.Fatalf("last quartile=%f want 40", last)
	}
}

func TestPerfMedianWALBytes(t *testing.T) {
	t.Parallel()

	if got := perfMedianWALBytes(nil); got != 0 {
		t.Fatalf("nil median=%d want 0", got)
	}

	odd := []perfSample{{WALBytes: 1}, {WALBytes: 5}, {WALBytes: 3}}
	if got := perfMedianWALBytes(odd); got != 3 {
		t.Fatalf("odd median=%d want 3", got)
	}

	even := []perfSample{{WALBytes: 1}, {WALBytes: 3}, {WALBytes: 7}, {WALBytes: 9}}
	if got := perfMedianWALBytes(even); got != 5 {
		t.Fatalf("even median=%d want 5", got)
	}
}

func TestPerfHasWALAnomaly(t *testing.T) {
	t.Parallel()

	if perfHasWALAnomaly(nil, 4) {
		t.Fatal("nil summary should not report anomaly")
	}

	summary := &perfSummary{
		RecordCountStart: 100,
		RecordCountEnd:   100,
		Samples: []perfSample{
			{SnapshotCaptured: true, WALBytes: 10},
			{SnapshotCaptured: true, WALBytes: 10},
			{SnapshotCaptured: true, WALBytes: 10},
			{SnapshotCaptured: true, WALBytes: 20},
			{SnapshotCaptured: true, WALBytes: 50},
		},
	}
	if !perfHasWALAnomaly(summary, 4) {
		t.Fatal("expected WAL anomaly to be detected")
	}

	summary.RecordCountEnd = 110
	if perfHasWALAnomaly(summary, 4) {
		t.Fatal("record growth should suppress WAL anomaly")
	}

	summary.RecordCountEnd = 100
	summary.RecordCountStart = -1
	if perfHasWALAnomaly(summary, 4) {
		t.Fatal("unknown record count should suppress WAL anomaly")
	}

	summary.RecordCountStart = 100
	summary.Samples = []perfSample{
		{SnapshotCaptured: true, WALBytes: 10},
		{SnapshotCaptured: true, WALBytes: 9},
		{SnapshotCaptured: true, WALBytes: 8},
		{SnapshotCaptured: true, WALBytes: 7},
		{SnapshotCaptured: true, WALBytes: 50},
	}
	if perfHasWALAnomaly(summary, 4) {
		t.Fatal("non-monotonic WAL growth should suppress anomaly")
	}

	summary.Samples = []perfSample{
		{SnapshotCaptured: true, WALBytes: 10},
		{SnapshotCaptured: true, WALBytes: 10},
		{SnapshotCaptured: true, WALBytes: 20},
		{SnapshotCaptured: true, WALBytes: 50},
	}
	if perfHasWALAnomaly(summary, 4) {
		t.Fatal("exactly four samples should suppress anomaly")
	}
}

func TestPerfEvaluateSummary(t *testing.T) {
	t.Parallel()

	thresholds := defaultPerfThresholds()
	t.Run("failure branches", func(t *testing.T) {
		t.Parallel()

		summary := &perfSummary{
			Scenario:              perfScenarioConfig{RestartDurability: true},
			TotalOperations:       100,
			TotalErrors:           1,
			RecordCountEndError:   "sqlite busy",
			DurabilityCheckPassed: false,
			DurabilityError:       "count mismatch",
			Baseline:              perfResourceSnapshot{HeapAllocBytes: 1000, NumGoroutines: 1, FDCount: 10, FDMeasured: true},
			Final:                 perfResourceSnapshot{HeapAllocBytes: 1000, NumGoroutines: 1, FDCount: 10, FDMeasured: true},
			Operations: map[string]perfOperationSummary{
				perfOpQuery:  {Count: 1, P95Millis: thresholds.QueryP95MaxMs + 1},
				perfOpGet:    {Count: 1, P95Millis: thresholds.GetP95MaxMs + 1},
				perfOpUpsert: {Count: 1, P95Millis: thresholds.WriteP95MaxMs + 1},
			},
		}
		perfEvaluateSummary(summary, thresholds)

		wantContains := []string{
			"error rate",
			"failed to count final records",
			"query p95",
			"get p95",
			"write p95",
			"restart durability check failed: count mismatch",
		}
		for _, fragment := range wantContains {
			if !containsFragment(summary.Failures, fragment) {
				t.Fatalf("expected failures to contain %q, got %v", fragment, summary.Failures)
			}
		}
	})

	t.Run("resource growth branches", func(t *testing.T) {
		t.Parallel()

		summary := &perfSummary{
			TotalOperations: 1,
			TotalErrors:     0,
			Baseline: perfResourceSnapshot{
				HeapAllocBytes: 100,
				NumGoroutines:  1,
				FDCount:        5,
				FDMeasured:     true,
			},
			Final: perfResourceSnapshot{
				HeapAllocBytes: 100 + thresholds.MaxHeapGrowthBytes + 1,
				NumGoroutines:  1 + thresholds.MaxGoroutineGrowth + 1,
				FDCount:        5 + thresholds.MaxFDGrowth + 1,
				FDMeasured:     true,
			},
		}
		perfEvaluateSummary(summary, thresholds)
		for _, fragment := range []string{"heap growth", "goroutine growth", "fd growth"} {
			if !containsFragment(summary.Failures, fragment) {
				t.Fatalf("expected failures to contain %q, got %v", fragment, summary.Failures)
			}
		}
	})

	t.Run("durability fallback message", func(t *testing.T) {
		t.Parallel()

		summary := &perfSummary{
			Scenario:              perfScenarioConfig{RestartDurability: true},
			TotalOperations:       1,
			DurabilityCheckPassed: false,
		}
		perfEvaluateSummary(summary, thresholds)
		if !containsFragment(summary.Failures, "restart durability check failed") {
			t.Fatalf("expected generic durability failure, got %v", summary.Failures)
		}
	})

	t.Run("errors with zero ops branch", func(t *testing.T) {
		t.Parallel()

		summary := &perfSummary{TotalErrors: 1}
		perfEvaluateSummary(summary, thresholds)
		if !containsFragment(summary.Failures, "errors observed with zero successful operations") {
			t.Fatalf("expected zero-op error failure, got %v", summary.Failures)
		}
	})

	t.Run("zero operations branch", func(t *testing.T) {
		t.Parallel()

		summary := &perfSummary{}
		perfEvaluateSummary(summary, thresholds)
		if !containsFragment(summary.Failures, "zero operations recorded") {
			t.Fatalf("expected zero-operation failure, got %v", summary.Failures)
		}
	})

	t.Run("alerts branches", func(t *testing.T) {
		t.Parallel()

		now := time.Now().UTC()
		summary := &perfSummary{
			TotalOperations: 10,
			Baseline: perfResourceSnapshot{
				HeapAllocBytes: 100,
				NumGoroutines:  1,
			},
			Final: perfResourceSnapshot{
				HeapAllocBytes: 100,
				NumGoroutines:  1,
			},
			HeapSlopeMiBPerMinute:         thresholds.SyntheticHeapSlopeAlertMiB + 1,
			GoroutineSlopePerMinute:       thresholds.MaxGoroutineSlopePerMinute + 0.1,
			ThroughputFirstQuartileOpsSec: 100,
			ThroughputLastQuartileOpsSec:  50,
			RecordCountStart:              100,
			RecordCountEnd:                100,
			Samples: []perfSample{
				{Timestamp: now, SnapshotCaptured: true, WALBytes: 10},
				{Timestamp: now.Add(1 * time.Second), SnapshotCaptured: true, WALBytes: 10},
				{Timestamp: now.Add(2 * time.Second), SnapshotCaptured: true, WALBytes: 10},
				{Timestamp: now.Add(3 * time.Second), SnapshotCaptured: true, WALBytes: 20},
				{Timestamp: now.Add(4 * time.Second), SnapshotCaptured: true, WALBytes: 50},
			},
		}
		perfEvaluateSummary(summary, thresholds)
		for _, fragment := range []string{"heap slope", "goroutine slope", "throughput drift detected", "wal anomaly detected"} {
			if !containsFragment(summary.Alerts, fragment) {
				t.Fatalf("expected alerts to contain %q, got %v", fragment, summary.Alerts)
			}
		}
	})

	t.Run("default ef heap alert threshold branch", func(t *testing.T) {
		t.Parallel()

		summary := &perfSummary{
			Scenario:                      perfScenarioConfig{UseDefaultEF: true},
			TotalOperations:               1,
			HeapSlopeMiBPerMinute:         thresholds.DefaultEFHeapSlopeAlertMiB + 0.1,
			Baseline:                      perfResourceSnapshot{},
			Final:                         perfResourceSnapshot{},
			RecordCountStart:              0,
			RecordCountEnd:                0,
			ThroughputFirstQuartileOpsSec: 0,
		}
		perfEvaluateSummary(summary, thresholds)
		if !containsFragment(summary.Alerts, "heap slope") {
			t.Fatalf("expected default_ef heap slope alert, got %v", summary.Alerts)
		}
	})
}

func TestPerfEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		value        string
		defaultValue bool
		want         bool
		wantErr      bool
	}{
		{name: "empty uses default true", key: "CHROMA_PERF_TEST_BOOL", value: "", defaultValue: true, want: true},
		{name: "empty uses default false", key: "CHROMA_PERF_TEST_BOOL", value: "", defaultValue: false, want: false},
		{name: "true literal", key: "CHROMA_PERF_TEST_BOOL", value: "true", defaultValue: false, want: true},
		{name: "TRUE upper", key: "CHROMA_PERF_TEST_BOOL", value: "TRUE", defaultValue: false, want: true},
		{name: "1 literal", key: "CHROMA_PERF_TEST_BOOL", value: "1", defaultValue: false, want: true},
		{name: "yes literal", key: "CHROMA_PERF_TEST_BOOL", value: "yes", defaultValue: false, want: true},
		{name: "on literal", key: "CHROMA_PERF_TEST_BOOL", value: "on", defaultValue: false, want: true},
		{name: "false literal", key: "CHROMA_PERF_TEST_BOOL", value: "false", defaultValue: true, want: false},
		{name: "0 literal", key: "CHROMA_PERF_TEST_BOOL", value: "0", defaultValue: true, want: false},
		{name: "no literal", key: "CHROMA_PERF_TEST_BOOL", value: "no", defaultValue: true, want: false},
		{name: "off literal", key: "CHROMA_PERF_TEST_BOOL", value: "off", defaultValue: true, want: false},
		{name: "invalid literal", key: "CHROMA_PERF_TEST_BOOL", value: "tru", defaultValue: true, wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			got, err := perfEnvBool(tc.key, tc.defaultValue)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (value=%q)", tc.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("perfEnvBool(%q)=%t want %t", tc.value, got, tc.want)
			}
		})
	}
}

func TestPerfSlugify(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: "unknown"},
		{in: "  ", want: "unknown"},
		{in: "Server Synthetic Smoke", want: "server_synthetic_smoke"},
		{in: "A/B+C", want: "a_b_c"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := perfSlugify(tc.in); got != tc.want {
				t.Fatalf("perfSlugify(%q)=%q want %q", tc.in, got, tc.want)
			}
		})
	}
}

func containsFragment(values []string, fragment string) bool {
	for _, value := range values {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}
