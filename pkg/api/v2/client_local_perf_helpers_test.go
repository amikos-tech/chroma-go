//go:build soak && !cloud

package v2

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	embeddingspkg "github.com/amikos-tech/chroma-go/pkg/embeddings"
	defaultef "github.com/amikos-tech/chroma-go/pkg/embeddings/default_ef"
)

const (
	perfProfileSmoke = "smoke"
	perfProfileSoak  = "soak"

	perfScenarioKindSynthetic = "synthetic"
	perfScenarioKindChurn     = "churn"
)

const (
	perfOpSeedAdd        = "seed_add"
	perfOpQuery          = "query"
	perfOpGet            = "get"
	perfOpUpsert         = "upsert"
	perfOpDeleteReinsert = "delete_reinsert"
	perfOpCount          = "count"
	perfOpDurability     = "durability"
	perfOpSample         = "sample"
	perfOpClose          = "close"
	perfOpChurnCycle     = "churn_cycle"
)

var perfLatencyBounds = []time.Duration{
	5 * time.Millisecond,
	10 * time.Millisecond,
	25 * time.Millisecond,
	50 * time.Millisecond,
	100 * time.Millisecond,
	250 * time.Millisecond,
	500 * time.Millisecond,
	750 * time.Millisecond,
	1 * time.Second,
	1500 * time.Millisecond,
	2 * time.Second,
	5 * time.Second,
	10 * time.Second,
}

type perfRuntimeConfig struct {
	Profile          string
	Enforce          bool
	IncludeDefaultEF bool
	EnableDeleteOps  bool
	ReportDir        string
}

type perfScenarioConfig struct {
	Name                 string                `json:"name"`
	Kind                 string                `json:"kind"`
	Mode                 PersistentRuntimeMode `json:"mode"`
	Duration             time.Duration         `json:"duration"`
	DatasetSize          int                   `json:"dataset_size"`
	EmbeddingDim         int                   `json:"embedding_dim"`
	SeedBatchSize        int                   `json:"seed_batch_size"`
	ReadWorkers          int                   `json:"read_workers"`
	WriteInterval        time.Duration         `json:"write_interval"`
	SampleInterval       time.Duration         `json:"sample_interval"`
	RestartDurability    bool                  `json:"restart_durability"`
	ChurnCycles          int                   `json:"churn_cycles"`
	UseDefaultEF         bool                  `json:"use_default_ef"`
	UpsertWeight         int                   `json:"upsert_weight"`
	DeleteReinsertWeight int                   `json:"delete_reinsert_weight"`
}

type perfThresholds struct {
	ErrorRateMax                 float64 `json:"error_rate_max"`
	QueryP95MaxMs                float64 `json:"query_p95_max_ms"`
	GetP95MaxMs                  float64 `json:"get_p95_max_ms"`
	WriteP95MaxMs                float64 `json:"write_p95_max_ms"`
	MaxHeapGrowthPercent         float64 `json:"max_heap_growth_percent"`
	MaxHeapGrowthBytes           uint64  `json:"max_heap_growth_bytes"`
	MaxGoroutineGrowth           int     `json:"max_goroutine_growth"`
	MaxFDGrowth                  int64   `json:"max_fd_growth"`
	SyntheticHeapSlopeAlertMiB   float64 `json:"synthetic_heap_slope_alert_mib_per_min"`
	DefaultEFHeapSlopeAlertMiB   float64 `json:"default_ef_heap_slope_alert_mib_per_min"`
	MaxGoroutineSlopePerMinute   float64 `json:"max_goroutine_slope_per_min"`
	WalAnomalyFactor             float64 `json:"wal_anomaly_factor"`
	ThroughputDriftAlertFraction float64 `json:"throughput_drift_alert_fraction"`
}

type perfResourceSnapshot struct {
	Timestamp      time.Time `json:"timestamp"`
	HeapAllocBytes uint64    `json:"heap_alloc_bytes"`
	NumGoroutines  int       `json:"num_goroutines"`
	FDCount        int64     `json:"fd_count,omitempty"`
	FDMeasured     bool      `json:"fd_measured"`
	PersistBytes   int64     `json:"persist_bytes"`
	WALBytes       int64     `json:"wal_bytes"`
}

type perfSample struct {
	Timestamp        time.Time         `json:"timestamp"`
	ElapsedSeconds   float64           `json:"elapsed_seconds"`
	OpsTotal         uint64            `json:"ops_total"`
	ErrorsTotal      uint64            `json:"errors_total"`
	OpCounts         map[string]uint64 `json:"op_counts"`
	OpErrors         map[string]uint64 `json:"op_errors"`
	HeapAllocBytes   uint64            `json:"heap_alloc_bytes"`
	NumGoroutines    int               `json:"num_goroutines"`
	FDCount          int64             `json:"fd_count,omitempty"`
	FDMeasured       bool              `json:"fd_measured"`
	PersistBytes     int64             `json:"persist_bytes"`
	WALBytes         int64             `json:"wal_bytes"`
	SnapshotCaptured bool              `json:"snapshot_captured"`
	SnapshotError    string            `json:"snapshot_error,omitempty"`
}

type perfOperationSummary struct {
	Count          uint64            `json:"count"`
	Errors         uint64            `json:"errors"`
	P95Millis      float64           `json:"p95_ms"`
	LatencyBuckets map[string]uint64 `json:"latency_buckets"`
}

type perfSummary struct {
	Profile                       string                          `json:"profile"`
	Scenario                      perfScenarioConfig              `json:"scenario"`
	StartedAt                     time.Time                       `json:"started_at"`
	EndedAt                       time.Time                       `json:"ended_at"`
	ElapsedSeconds                float64                         `json:"elapsed_seconds"`
	Enforced                      bool                            `json:"enforced"`
	Thresholds                    perfThresholds                  `json:"thresholds"`
	Baseline                      perfResourceSnapshot            `json:"baseline"`
	Final                         perfResourceSnapshot            `json:"final"`
	RecordCountStart              int                             `json:"record_count_start"`
	RecordCountEnd                int                             `json:"record_count_end"`
	RecordCountEndError           string                          `json:"record_count_end_error,omitempty"`
	DurabilityCheckPassed         bool                            `json:"durability_check_passed"`
	DurabilityError               string                          `json:"durability_error,omitempty"`
	SampleCaptureErrors           int                             `json:"sample_capture_errors"`
	TotalOperations               uint64                          `json:"total_operations"`
	TotalErrors                   uint64                          `json:"total_errors"`
	Operations                    map[string]perfOperationSummary `json:"operations"`
	HeapSlopeMiBPerMinute         float64                         `json:"heap_slope_mib_per_min"`
	GoroutineSlopePerMinute       float64                         `json:"goroutine_slope_per_min"`
	ThroughputFirstQuartileOpsSec float64                         `json:"throughput_first_quartile_ops_per_sec"`
	ThroughputLastQuartileOpsSec  float64                         `json:"throughput_last_quartile_ops_per_sec"`
	WALMedianBytes                int64                           `json:"wal_median_bytes"`
	DeleteReinsertEnabled         bool                            `json:"delete_reinsert_enabled"`
	Samples                       []perfSample                    `json:"samples"`
	Alerts                        []string                        `json:"alerts"`
	Failures                      []string                        `json:"failures"`
	ReportJSONPath                string                          `json:"report_json_path"`
}

type perfDataset struct {
	IDs        []DocumentID
	Texts      []string
	Embeddings []embeddingspkg.Embedding
}

type perfOpAccumulator struct {
	Count      uint64
	Errors     uint64
	Buckets    []uint64
	MaxLatency time.Duration
}

type perfMetrics struct {
	mu  sync.Mutex
	ops map[string]*perfOpAccumulator
}

func newPerfMetrics() *perfMetrics {
	return &perfMetrics{ops: map[string]*perfOpAccumulator{}}
}

func (m *perfMetrics) ensureOp(op string) *perfOpAccumulator {
	acc := m.ops[op]
	if acc != nil {
		return acc
	}
	acc = &perfOpAccumulator{Buckets: make([]uint64, len(perfLatencyBounds)+1)}
	m.ops[op] = acc
	return acc
}

func (m *perfMetrics) observe(op string, latency time.Duration, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	acc := m.ensureOp(op)
	acc.Count++
	if err != nil {
		acc.Errors++
	}
	bucketIdx := perfLatencyBucketIndex(latency)
	acc.Buckets[bucketIdx]++
	if latency > acc.MaxLatency {
		acc.MaxLatency = latency
	}
}

func (m *perfMetrics) snapshotCounts() (map[string]uint64, map[string]uint64, uint64, uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	opCounts := make(map[string]uint64, len(m.ops))
	opErrors := make(map[string]uint64, len(m.ops))
	var totalOps uint64
	var totalErrors uint64
	for name, acc := range m.ops {
		opCounts[name] = acc.Count
		opErrors[name] = acc.Errors
		totalOps += acc.Count
		totalErrors += acc.Errors
	}
	return opCounts, opErrors, totalOps, totalErrors
}

func (m *perfMetrics) operationSummaries() map[string]perfOperationSummary {
	m.mu.Lock()
	defer m.mu.Unlock()

	summaries := make(map[string]perfOperationSummary, len(m.ops))
	for name, acc := range m.ops {
		bucketSnapshot := make(map[string]uint64, len(acc.Buckets))
		for idx, count := range acc.Buckets {
			bucketSnapshot[perfLatencyBucketLabel(idx)] = count
		}
		summaries[name] = perfOperationSummary{
			Count:          acc.Count,
			Errors:         acc.Errors,
			P95Millis:      perfP95FromBuckets(acc.Buckets, acc.Count, acc.MaxLatency),
			LatencyBuckets: bucketSnapshot,
		}
	}
	return summaries
}

func perfLatencyBucketIndex(d time.Duration) int {
	for idx, bound := range perfLatencyBounds {
		if d <= bound {
			return idx
		}
	}
	return len(perfLatencyBounds)
}

func perfLatencyBucketLabel(idx int) string {
	if idx < len(perfLatencyBounds) {
		return fmt.Sprintf("<=%dms", perfLatencyBounds[idx].Milliseconds())
	}
	return fmt.Sprintf(">%dms", perfLatencyBounds[len(perfLatencyBounds)-1].Milliseconds())
}

func perfP95FromBuckets(buckets []uint64, total uint64, maxLatency time.Duration) float64 {
	if total == 0 {
		return 0
	}
	target := uint64(math.Ceil(float64(total) * 0.95))
	if target == 0 {
		target = 1
	}
	var cumulative uint64
	for idx, count := range buckets {
		cumulative += count
		if cumulative >= target {
			if idx < len(perfLatencyBounds) {
				return float64(perfLatencyBounds[idx].Milliseconds())
			}
			lastBound := perfLatencyBounds[len(perfLatencyBounds)-1]
			if maxLatency > lastBound {
				return float64(maxLatency.Milliseconds())
			}
			return float64(lastBound.Milliseconds())
		}
	}
	lastBound := perfLatencyBounds[len(perfLatencyBounds)-1]
	if maxLatency > lastBound {
		return float64(maxLatency.Milliseconds())
	}
	return float64(lastBound.Milliseconds())
}

func defaultPerfThresholds() perfThresholds {
	return perfThresholds{
		ErrorRateMax:                 0,
		QueryP95MaxMs:                750,
		GetP95MaxMs:                  750,
		WriteP95MaxMs:                1500,
		MaxHeapGrowthPercent:         30,
		MaxHeapGrowthBytes:           64 * 1024 * 1024,
		MaxGoroutineGrowth:           8,
		MaxFDGrowth:                  16,
		SyntheticHeapSlopeAlertMiB:   3,
		DefaultEFHeapSlopeAlertMiB:   8,
		MaxGoroutineSlopePerMinute:   0.2,
		WalAnomalyFactor:             4,
		ThroughputDriftAlertFraction: 0.2,
	}
}

func perfBuildRuntimeConfig() (perfRuntimeConfig, error) {
	profile := strings.ToLower(strings.TrimSpace(os.Getenv("CHROMA_PERF_PROFILE")))
	if profile == "" {
		profile = perfProfileSmoke
	}
	if profile != perfProfileSmoke && profile != perfProfileSoak {
		return perfRuntimeConfig{}, errors.Errorf("invalid CHROMA_PERF_PROFILE: %s", profile)
	}

	enforce, err := perfEnvBool("CHROMA_PERF_ENFORCE", profile == perfProfileSmoke)
	if err != nil {
		return perfRuntimeConfig{}, err
	}
	includeDefault, err := perfEnvBool("CHROMA_PERF_INCLUDE_DEFAULT_EF", profile == perfProfileSoak)
	if err != nil {
		return perfRuntimeConfig{}, err
	}
	enableDeleteOps, err := perfEnvBool("CHROMA_PERF_ENABLE_DELETE_REINSERT", false)
	if err != nil {
		return perfRuntimeConfig{}, err
	}

	reportDir := strings.TrimSpace(os.Getenv("CHROMA_PERF_REPORT_DIR"))
	if reportDir == "" {
		tmpDir, err := os.MkdirTemp("", "chroma-perf-reports-*")
		if err != nil {
			return perfRuntimeConfig{}, errors.Wrap(err, "failed to create temporary report directory")
		}
		reportDir = tmpDir
	}
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return perfRuntimeConfig{}, errors.Wrap(err, "failed to create report directory")
	}

	return perfRuntimeConfig{
		Profile:          profile,
		Enforce:          enforce,
		IncludeDefaultEF: includeDefault,
		EnableDeleteOps:  enableDeleteOps,
		ReportDir:        reportDir,
	}, nil
}

func perfBuildScenarios(cfg perfRuntimeConfig) []perfScenarioConfig {
	switch cfg.Profile {
	case perfProfileSoak:
		scenarios := []perfScenarioConfig{
			{
				Name:                 "embedded_synthetic_soak",
				Kind:                 perfScenarioKindSynthetic,
				Mode:                 PersistentRuntimeModeEmbedded,
				Duration:             20 * time.Minute,
				DatasetSize:          3500,
				EmbeddingDim:         64,
				SeedBatchSize:        64,
				ReadWorkers:          7,
				WriteInterval:        20 * time.Millisecond,
				SampleInterval:       5 * time.Second,
				RestartDurability:    false,
				UseDefaultEF:         false,
				UpsertWeight:         2,
				DeleteReinsertWeight: 0,
			},
			{
				Name:                 "server_synthetic_soak",
				Kind:                 perfScenarioKindSynthetic,
				Mode:                 PersistentRuntimeModeServer,
				Duration:             20 * time.Minute,
				DatasetSize:          3500,
				EmbeddingDim:         64,
				SeedBatchSize:        64,
				ReadWorkers:          7,
				WriteInterval:        20 * time.Millisecond,
				SampleInterval:       5 * time.Second,
				RestartDurability:    false,
				UseDefaultEF:         false,
				UpsertWeight:         2,
				DeleteReinsertWeight: 1,
			},
		}
		if cfg.IncludeDefaultEF {
			scenarios = append(scenarios,
				perfScenarioConfig{
					Name:                 "embedded_default_ef_soak",
					Kind:                 perfScenarioKindSynthetic,
					Mode:                 PersistentRuntimeModeEmbedded,
					Duration:             10 * time.Minute,
					DatasetSize:          500,
					EmbeddingDim:         0,
					SeedBatchSize:        24,
					ReadWorkers:          2,
					WriteInterval:        80 * time.Millisecond,
					SampleInterval:       5 * time.Second,
					RestartDurability:    false,
					UseDefaultEF:         true,
					UpsertWeight:         2,
					DeleteReinsertWeight: 0,
				},
				perfScenarioConfig{
					Name:                 "server_default_ef_soak",
					Kind:                 perfScenarioKindSynthetic,
					Mode:                 PersistentRuntimeModeServer,
					Duration:             10 * time.Minute,
					DatasetSize:          500,
					EmbeddingDim:         0,
					SeedBatchSize:        24,
					ReadWorkers:          2,
					WriteInterval:        80 * time.Millisecond,
					SampleInterval:       5 * time.Second,
					RestartDurability:    false,
					UseDefaultEF:         true,
					UpsertWeight:         2,
					DeleteReinsertWeight: 1,
				},
			)
		}
		return scenarios
	default:
		return []perfScenarioConfig{
			{
				Name:                 "embedded_synthetic_smoke",
				Kind:                 perfScenarioKindSynthetic,
				Mode:                 PersistentRuntimeModeEmbedded,
				Duration:             90 * time.Second,
				DatasetSize:          1200,
				EmbeddingDim:         64,
				SeedBatchSize:        64,
				ReadWorkers:          5,
				WriteInterval:        20 * time.Millisecond,
				SampleInterval:       5 * time.Second,
				RestartDurability:    true,
				UseDefaultEF:         false,
				UpsertWeight:         2,
				DeleteReinsertWeight: 0,
			},
			{
				Name:                 "server_synthetic_smoke",
				Kind:                 perfScenarioKindSynthetic,
				Mode:                 PersistentRuntimeModeServer,
				Duration:             90 * time.Second,
				DatasetSize:          1200,
				EmbeddingDim:         64,
				SeedBatchSize:        64,
				ReadWorkers:          5,
				WriteInterval:        20 * time.Millisecond,
				SampleInterval:       5 * time.Second,
				RestartDurability:    true,
				UseDefaultEF:         false,
				UpsertWeight:         2,
				DeleteReinsertWeight: 1,
			},
			{
				Name:           "embedded_churn_smoke",
				Kind:           perfScenarioKindChurn,
				Mode:           PersistentRuntimeModeEmbedded,
				ChurnCycles:    35,
				SampleInterval: 5 * time.Second,
			},
			{
				Name:           "server_churn_smoke",
				Kind:           perfScenarioKindChurn,
				Mode:           PersistentRuntimeModeServer,
				ChurnCycles:    35,
				SampleInterval: 5 * time.Second,
			},
		}
	}
}

func perfEnvBool(key string, defaultValue bool) (bool, error) {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return defaultValue, nil
	}
	switch value {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, errors.Errorf("invalid boolean value for %s: %q (accepted: true/false, 1/0, yes/no, on/off)", key, value)
	}
}

func perfCreatePersistentClient(mode PersistentRuntimeMode, persistPath string) (Client, error) {
	opts := []PersistentClientOption{
		WithPersistentPath(persistPath),
		WithPersistentAllowReset(true),
		WithPersistentRuntimeMode(mode),
	}
	if mode != PersistentRuntimeModeServer {
		client, err := NewPersistentClient(opts...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create persistent client")
		}
		return client, nil
	}

	// Use a concrete loopback port because server URL/readiness can be flaky when the
	// runtime is started with port 0 in test environments.
	const maxServerStartAttempts = 8
	var lastErr error
	for attempt := 1; attempt <= maxServerStartAttempts; attempt++ {
		port, err := perfReserveLoopbackPort()
		if err != nil {
			return nil, errors.Wrap(err, "failed to reserve loopback port for server runtime")
		}
		attemptOpts := make([]PersistentClientOption, 0, len(opts)+2)
		attemptOpts = append(attemptOpts, opts...)
		attemptOpts = append(attemptOpts,
			WithPersistentListenAddress("127.0.0.1"),
			WithPersistentPort(port),
		)
		client, clientErr := NewPersistentClient(attemptOpts...)
		if clientErr == nil {
			return client, nil
		}
		lastErr = clientErr
		time.Sleep(time.Duration(attempt) * 75 * time.Millisecond)
	}
	return nil, errors.Wrapf(lastErr, "failed to create persistent client after %d attempts", maxServerStartAttempts)
}

func perfReserveLoopbackPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = listener.Close()
	}()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok || addr == nil || addr.Port <= 0 {
		return 0, errors.New("failed to resolve reserved loopback TCP port")
	}
	return addr.Port, nil
}

func perfBuildSyntheticDataset(size, dim int, seed int64) perfDataset {
	rng := rand.New(rand.NewSource(seed))
	ids := make([]DocumentID, size)
	texts := make([]string, size)
	embs := make([]embeddingspkg.Embedding, size)
	for i := 0; i < size; i++ {
		ids[i] = DocumentID(fmt.Sprintf("doc-%06d", i))
		texts[i] = fmt.Sprintf("synthetic document %d", i)
		vec := make([]float32, dim)
		for j := 0; j < dim; j++ {
			vec[j] = rng.Float32()
		}
		embs[i] = embeddingspkg.NewEmbeddingFromFloat32(vec)
	}
	return perfDataset{IDs: ids, Texts: texts, Embeddings: embs}
}

func perfBuildTextDataset(size int) perfDataset {
	ids := make([]DocumentID, size)
	texts := make([]string, size)
	for i := 0; i < size; i++ {
		ids[i] = DocumentID(fmt.Sprintf("doc-%06d", i))
		texts[i] = fmt.Sprintf("default ef document %d", i)
	}
	return perfDataset{IDs: ids, Texts: texts}
}

func perfUpdatedEmbedding(index, dim int, revision uint64) embeddingspkg.Embedding {
	vec := make([]float32, dim)
	base := float64(index+1) + float64(revision%997)
	for i := 0; i < dim; i++ {
		value := math.Mod((base*0.0007)+float64(i)*0.031, 1.0)
		vec[i] = float32(value)
	}
	return embeddingspkg.NewEmbeddingFromFloat32(vec)
}

func perfSeedCollection(ctx context.Context, collection Collection, dataset perfDataset, cfg perfScenarioConfig, metrics *perfMetrics) error {
	batchSize := cfg.SeedBatchSize
	if batchSize <= 0 {
		batchSize = 64
	}

	for start := 0; start < len(dataset.IDs); start += batchSize {
		end := start + batchSize
		if end > len(dataset.IDs) {
			end = len(dataset.IDs)
		}

		opts := []CollectionAddOption{
			WithIDs(dataset.IDs[start:end]...),
			WithTexts(dataset.Texts[start:end]...),
		}
		if !cfg.UseDefaultEF {
			opts = append(opts, WithEmbeddings(dataset.Embeddings[start:end]...))
		}

		batchCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		opStarted := time.Now()
		err := collection.Add(batchCtx, opts...)
		cancel()
		metrics.observe(perfOpSeedAdd, time.Since(opStarted), err)
		if err != nil {
			return errors.Wrapf(err, "failed to seed batch %d-%d", start, end)
		}
	}
	return nil
}

func perfCaptureSnapshot(persistPath string) (perfResourceSnapshot, error) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fdCount, fdMeasured := perfFDCount()
	snapshot := perfResourceSnapshot{
		Timestamp:      time.Now().UTC(),
		HeapAllocBytes: mem.HeapAlloc,
		NumGoroutines:  runtime.NumGoroutine(),
		FDCount:        fdCount,
		FDMeasured:     fdMeasured,
		PersistBytes:   0,
		WALBytes:       0,
	}

	persistBytes, walBytes, err := perfPersistAndWALSize(persistPath)
	if err != nil {
		return snapshot, err
	}
	snapshot.PersistBytes = persistBytes
	snapshot.WALBytes = walBytes
	return snapshot, nil
}

func perfPersistAndWALSize(persistPath string) (int64, int64, error) {
	if strings.TrimSpace(persistPath) == "" {
		return 0, 0, errors.New("persist path cannot be empty")
	}
	var persistBytes int64
	var walBytes int64
	err := filepath.WalkDir(persistPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		size := info.Size()
		persistBytes += size
		name := strings.ToLower(d.Name())
		if strings.HasSuffix(name, "-wal") || strings.HasSuffix(name, ".wal") {
			walBytes += size
		}
		return nil
	})
	if err != nil {
		return 0, 0, errors.Wrap(err, "failed to compute persist directory size")
	}
	return persistBytes, walBytes, nil
}

func perfFDCount() (int64, bool) {
	for _, candidate := range []string{"/proc/self/fd", "/dev/fd"} {
		entries, err := os.ReadDir(candidate)
		if err == nil {
			return int64(len(entries)), true
		}
	}
	return 0, false
}

func perfCollectSample(start time.Time, persistPath string, metrics *perfMetrics) perfSample {
	opCounts, opErrors, totalOps, totalErrors := metrics.snapshotCounts()
	resourceSnapshot, err := perfCaptureSnapshot(persistPath)
	sample := perfSample{
		Timestamp:        resourceSnapshot.Timestamp,
		ElapsedSeconds:   resourceSnapshot.Timestamp.Sub(start).Seconds(),
		OpsTotal:         totalOps,
		ErrorsTotal:      totalErrors,
		OpCounts:         opCounts,
		OpErrors:         opErrors,
		HeapAllocBytes:   resourceSnapshot.HeapAllocBytes,
		NumGoroutines:    resourceSnapshot.NumGoroutines,
		FDCount:          resourceSnapshot.FDCount,
		FDMeasured:       resourceSnapshot.FDMeasured,
		PersistBytes:     resourceSnapshot.PersistBytes,
		WALBytes:         resourceSnapshot.WALBytes,
		SnapshotCaptured: err == nil,
	}
	if err != nil {
		sample.SnapshotError = err.Error()
	}
	return sample
}

func perfStabilizeGC() {
	runtime.GC()
	time.Sleep(120 * time.Millisecond)
	runtime.GC()
	time.Sleep(120 * time.Millisecond)
}

func perfShouldIgnoreContextError(opErr error, runCtxErr, opCtxErr error) bool {
	if opErr == nil || runCtxErr == nil || opCtxErr == nil {
		return false
	}
	if stderrors.Is(opErr, context.Canceled) && stderrors.Is(opCtxErr, context.Canceled) {
		return true
	}
	if stderrors.Is(opErr, context.DeadlineExceeded) && stderrors.Is(opCtxErr, context.DeadlineExceeded) {
		return true
	}
	// Some client paths wrap context cancellation into transport-layer errors where
	// errors.Is no longer matches directly. At this point both contexts are already
	// done, so treating these as shutdown noise is safe.
	msg := strings.TrimSpace(strings.ToLower(opErr.Error()))
	switch {
	case msg == "context canceled":
		return stderrors.Is(opCtxErr, context.Canceled)
	case msg == "context deadline exceeded":
		return stderrors.Is(opCtxErr, context.DeadlineExceeded)
	case strings.HasSuffix(msg, ": context canceled"):
		return stderrors.Is(opCtxErr, context.Canceled)
	case strings.HasSuffix(msg, ": context deadline exceeded"):
		return stderrors.Is(opCtxErr, context.DeadlineExceeded)
	}
	return false
}

func perfPanicError(name string, recovered any) error {
	stack := strings.TrimSpace(string(debug.Stack()))
	if stack == "" {
		return errors.Errorf("panic in %s: %v", name, recovered)
	}
	return errors.Errorf("panic in %s: %v\n%s", name, recovered, stack)
}

func perfFilterCapturedSamples(samples []perfSample) []perfSample {
	filtered := make([]perfSample, 0, len(samples))
	for _, sample := range samples {
		if !sample.SnapshotCaptured {
			continue
		}
		filtered = append(filtered, sample)
	}
	return filtered
}

func perfCountSampleCaptureErrors(samples []perfSample) int {
	total := 0
	for _, sample := range samples {
		if sample.SnapshotCaptured {
			continue
		}
		total++
	}
	return total
}

func perfRunSyntheticScenario(cfg perfRuntimeConfig, thresholds perfThresholds, scenario perfScenarioConfig) (summary perfSummary, retErr error) {
	startedAt := time.Now().UTC()
	persistPath, err := os.MkdirTemp("", "chroma-local-perf-*")
	if err != nil {
		return perfSummary{}, errors.Wrap(err, "failed to create persist path")
	}
	defer func() { _ = os.RemoveAll(persistPath) }()

	metrics := newPerfMetrics()
	var closeEF func() error
	client, err := perfCreatePersistentClient(scenario.Mode, persistPath)
	if err != nil {
		return perfSummary{}, err
	}
	defer func() {
		if client != nil {
			started := time.Now()
			closeErr := client.Close()
			metrics.observe(perfOpClose, time.Since(started), closeErr)
			if closeErr != nil {
				if summary.Scenario.Name != "" {
					summary.Failures = append(summary.Failures,
						fmt.Sprintf("failed to close synthetic scenario client: %v", closeErr),
					)
				} else {
					wrapped := errors.Wrap(closeErr, "failed to close synthetic scenario client")
					if retErr != nil {
						retErr = stderrors.Join(retErr, wrapped)
					} else {
						retErr = wrapped
					}
				}
			}
		}
	}()
	defer func() {
		if closeEF != nil {
			started := time.Now()
			closeErr := closeEF()
			metrics.observe(perfOpClose, time.Since(started), closeErr)
			if closeErr != nil {
				if summary.Scenario.Name != "" {
					summary.Failures = append(summary.Failures,
						fmt.Sprintf("failed to close default embedding function: %v", closeErr),
					)
				} else {
					wrapped := errors.Wrap(closeErr, "failed to close default embedding function")
					if retErr != nil {
						retErr = stderrors.Join(retErr, wrapped)
					} else {
						retErr = wrapped
					}
				}
			}
		}
	}()

	ctx := context.Background()
	collectionName := fmt.Sprintf("perf_%s_%d", perfSlugify(scenario.Name), time.Now().UnixNano())

	createOptions := []CreateCollectionOption{}
	if scenario.UseDefaultEF {
		ef, closeFunc, efErr := defaultef.NewDefaultEmbeddingFunction()
		if efErr != nil {
			return perfSummary{}, errors.Wrap(efErr, "failed to create default embedding function")
		}
		closeEF = closeFunc
		createOptions = append(createOptions, WithEmbeddingFunctionCreate(ef))
	} else {
		createOptions = append(createOptions, WithEmbeddingFunctionCreate(embeddingspkg.NewConsistentHashEmbeddingFunction()))
	}

	collection, err := client.GetOrCreateCollection(ctx, collectionName, createOptions...)
	if err != nil {
		return perfSummary{}, errors.Wrap(err, "failed to create performance collection")
	}

	var dataset perfDataset
	if scenario.UseDefaultEF {
		dataset = perfBuildTextDataset(scenario.DatasetSize)
	} else {
		dataset = perfBuildSyntheticDataset(scenario.DatasetSize, scenario.EmbeddingDim, 42)
	}
	if err := perfSeedCollection(ctx, collection, dataset, scenario, metrics); err != nil {
		return perfSummary{}, err
	}

	countStart, err := collection.Count(ctx)
	if err != nil {
		return perfSummary{}, errors.Wrap(err, "failed to count seeded records")
	}

	perfStabilizeGC()
	baseline, err := perfCaptureSnapshot(persistPath)
	if err != nil {
		return perfSummary{}, err
	}

	runCtx, runCancel := context.WithTimeout(context.Background(), scenario.Duration)
	defer runCancel()
	const catastrophicOpsFloor = 200
	const catastrophicErrorRatePercent = 95
	const catastrophicErrorRateThreshold = float64(catastrophicErrorRatePercent) / 100.0
	const catastrophicConsecutiveSamples = 2
	var workerErrMu sync.Mutex
	workerErrs := make([]error, 0, scenario.ReadWorkers+2)

	samples := make([]perfSample, 0, int(scenario.Duration/scenario.SampleInterval)+4)
	sampleMu := sync.Mutex{}
	workloadStart := time.Now().UTC()

	reportWorkerError := func(opName string, opErr error) {
		if opErr == nil {
			return
		}
		metrics.observe(opName, 0, opErr)
		workerErrMu.Lock()
		workerErrs = append(workerErrs, opErr)
		workerErrMu.Unlock()
		runCancel()
	}

	recordSample := func() {
		sample := perfCollectSample(workloadStart, persistPath, metrics)
		sampleMu.Lock()
		samples = append(samples, sample)
		sampleMu.Unlock()
	}
	recordSample()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if recovered := recover(); recovered != nil {
				reportWorkerError(perfOpSample, perfPanicError("sample-worker", recovered))
			}
		}()
		ticker := time.NewTicker(scenario.SampleInterval)
		defer ticker.Stop()
		catastrophicSampleStreak := 0
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				recordSample()
				_, _, totalOps, totalErrors := metrics.snapshotCounts()
				isCatastrophicRate := false
				if totalOps >= catastrophicOpsFloor {
					isCatastrophicRate = float64(totalErrors)/float64(totalOps) >= catastrophicErrorRateThreshold
				}
				if isCatastrophicRate {
					catastrophicSampleStreak++
				} else {
					catastrophicSampleStreak = 0
				}
				if catastrophicSampleStreak >= catastrophicConsecutiveSamples {
					reportWorkerError(
						perfOpSample,
						errors.Errorf("catastrophic failure: error rate remained >=%d%% for %d samples (%d/%d ops failed)",
							catastrophicErrorRatePercent,
							catastrophicConsecutiveSamples,
							totalErrors,
							totalOps,
						),
					)
					return
				}
			}
		}
	}()

	var revisionCounter uint64
	upsertWeight := scenario.UpsertWeight
	deleteWeight := scenario.DeleteReinsertWeight
	if !cfg.EnableDeleteOps {
		deleteWeight = 0
	}
	if upsertWeight < 0 {
		upsertWeight = 0
	}
	if deleteWeight < 0 {
		deleteWeight = 0
	}
	if upsertWeight == 0 && deleteWeight == 0 {
		upsertWeight = 1
	}
	totalWriteWeight := upsertWeight + deleteWeight

	for workerIndex := 0; workerIndex < scenario.ReadWorkers; workerIndex++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					reportWorkerError(perfOpQuery, perfPanicError(fmt.Sprintf("read-worker-%d", index), recovered))
				}
			}()
			rng := rand.New(rand.NewSource(int64(1000 + index)))
			for {
				select {
				case <-runCtx.Done():
					return
				default:
				}

				target := rng.Intn(len(dataset.IDs))
				if rng.Intn(10) < 7 {
					opCtx, opCancel := context.WithTimeout(runCtx, 20*time.Second)
					started := time.Now()
					var opErr error
					if scenario.UseDefaultEF {
						_, opErr = collection.Query(opCtx,
							WithQueryTexts(dataset.Texts[target]),
							WithNResults(3),
							WithInclude(IncludeDistances),
						)
					} else {
						_, opErr = collection.Query(opCtx,
							WithQueryEmbeddings(dataset.Embeddings[target]),
							WithNResults(3),
							WithInclude(IncludeDistances),
						)
					}
					opCtxErr := opCtx.Err()
					opCancel()
					if perfShouldIgnoreContextError(opErr, runCtx.Err(), opCtxErr) {
						opErr = nil
					}
					metrics.observe(perfOpQuery, time.Since(started), opErr)
				} else {
					opCtx, opCancel := context.WithTimeout(runCtx, 20*time.Second)
					started := time.Now()
					_, opErr := collection.Get(opCtx, WithIDs(dataset.IDs[target]), WithInclude(IncludeDocuments))
					opCtxErr := opCtx.Err()
					opCancel()
					if perfShouldIgnoreContextError(opErr, runCtx.Err(), opCtxErr) {
						opErr = nil
					}
					metrics.observe(perfOpGet, time.Since(started), opErr)
				}
			}
		}(workerIndex)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if recovered := recover(); recovered != nil {
				reportWorkerError(perfOpUpsert, perfPanicError("write-worker", recovered))
			}
		}()
		rng := rand.New(rand.NewSource(7331))
		var writeTicker *time.Ticker
		if scenario.WriteInterval > 0 {
			writeTicker = time.NewTicker(scenario.WriteInterval)
			defer writeTicker.Stop()
		}
		for {
			select {
			case <-runCtx.Done():
				return
			default:
			}

			target := rng.Intn(len(dataset.IDs))
			revision := atomic.AddUint64(&revisionCounter, 1)
			updatedText := fmt.Sprintf("updated document %d rev %d", target, revision)

			if deleteWeight > 0 && rng.Intn(totalWriteWeight) < deleteWeight {
				opCtx, opCancel := context.WithTimeout(runCtx, 20*time.Second)
				started := time.Now()
				err := collection.Delete(opCtx, WithIDs(dataset.IDs[target]))
				if err == nil {
					opts := []CollectionAddOption{WithIDs(dataset.IDs[target]), WithTexts(updatedText)}
					if !scenario.UseDefaultEF {
						opts = append(opts, WithEmbeddings(perfUpdatedEmbedding(target, scenario.EmbeddingDim, revision)))
					}
					err = collection.Upsert(opCtx, opts...)
				}
				opCtxErr := opCtx.Err()
				opCancel()
				if perfShouldIgnoreContextError(err, runCtx.Err(), opCtxErr) {
					err = nil
				}
				metrics.observe(perfOpDeleteReinsert, time.Since(started), err)
			} else {
				opCtx, opCancel := context.WithTimeout(runCtx, 20*time.Second)
				started := time.Now()
				opts := []CollectionAddOption{WithIDs(dataset.IDs[target]), WithTexts(updatedText)}
				if !scenario.UseDefaultEF {
					opts = append(opts, WithEmbeddings(perfUpdatedEmbedding(target, scenario.EmbeddingDim, revision)))
				}
				err := collection.Upsert(opCtx, opts...)
				opCtxErr := opCtx.Err()
				opCancel()
				if perfShouldIgnoreContextError(err, runCtx.Err(), opCtxErr) {
					err = nil
				}
				metrics.observe(perfOpUpsert, time.Since(started), err)
			}

			if writeTicker != nil {
				select {
				case <-runCtx.Done():
					return
				case <-writeTicker.C:
				}
			}
		}
	}()

	<-runCtx.Done()
	wg.Wait()
	recordSample()
	workerErrMu.Lock()
	workerErrSnapshot := append([]error(nil), workerErrs...)
	workerErrMu.Unlock()
	if len(workerErrSnapshot) > 0 {
		return perfSummary{}, stderrors.Join(workerErrSnapshot...)
	}

	countEnd := -1
	countEndError := ""
	countStarted := time.Now()
	if count, countErr := collection.Count(ctx); countErr == nil {
		countEnd = count
		metrics.observe(perfOpCount, time.Since(countStarted), nil)
	} else {
		metrics.observe(perfOpCount, time.Since(countStarted), countErr)
		countEndError = countErr.Error()
	}

	durabilityPassed := true
	durabilityError := ""
	if scenario.RestartDurability {
		closeStarted := time.Now()
		closeErr := client.Close()
		metrics.observe(perfOpClose, time.Since(closeStarted), closeErr)
		if closeErr != nil {
			return perfSummary{}, errors.Wrap(closeErr, "failed to close client before durability check")
		}
		client = nil

		durabilityStarted := time.Now()
		durabilityErr := perfDurabilityCheck(scenario, persistPath, collectionName, countStart, dataset)
		metrics.observe(perfOpDurability, time.Since(durabilityStarted), durabilityErr)
		if durabilityErr != nil {
			durabilityPassed = false
			durabilityError = durabilityErr.Error()
		}
	}

	perfStabilizeGC()
	finalSnapshot, err := perfCaptureSnapshot(persistPath)
	if err != nil {
		return perfSummary{}, err
	}

	opSummaries := metrics.operationSummaries()
	totalOps, totalErrors := uint64(0), uint64(0)
	for _, op := range opSummaries {
		totalOps += op.Count
		totalErrors += op.Errors
	}

	sampleMu.Lock()
	samplesSnapshot := make([]perfSample, len(samples))
	copy(samplesSnapshot, samples)
	sampleMu.Unlock()
	capturedSamples := perfFilterCapturedSamples(samplesSnapshot)
	sampleCaptureErrors := perfCountSampleCaptureErrors(samplesSnapshot)

	heapSlope := perfSlopePerMinute(capturedSamples, func(sample perfSample) float64 {
		return float64(sample.HeapAllocBytes) / (1024 * 1024)
	})
	goroutineSlope := perfSlopePerMinute(capturedSamples, func(sample perfSample) float64 {
		return float64(sample.NumGoroutines)
	})
	firstQ, lastQ := perfThroughputQuartiles(samplesSnapshot)
	walMedian := perfMedianWALBytes(capturedSamples)

	summary = perfSummary{
		Profile:                       cfg.Profile,
		Scenario:                      scenario,
		StartedAt:                     startedAt,
		EndedAt:                       time.Now().UTC(),
		ElapsedSeconds:                time.Since(workloadStart).Seconds(),
		Enforced:                      cfg.Enforce,
		Thresholds:                    thresholds,
		Baseline:                      baseline,
		Final:                         finalSnapshot,
		RecordCountStart:              countStart,
		RecordCountEnd:                countEnd,
		RecordCountEndError:           countEndError,
		DurabilityCheckPassed:         durabilityPassed,
		DurabilityError:               durabilityError,
		SampleCaptureErrors:           sampleCaptureErrors,
		TotalOperations:               totalOps,
		TotalErrors:                   totalErrors,
		Operations:                    opSummaries,
		HeapSlopeMiBPerMinute:         heapSlope,
		GoroutineSlopePerMinute:       goroutineSlope,
		ThroughputFirstQuartileOpsSec: firstQ,
		ThroughputLastQuartileOpsSec:  lastQ,
		WALMedianBytes:                walMedian,
		DeleteReinsertEnabled:         deleteWeight > 0,
		Samples:                       samplesSnapshot,
		Alerts:                        []string{},
		Failures:                      []string{},
	}
	if sampleCaptureErrors > 0 {
		summary.Alerts = append(summary.Alerts,
			fmt.Sprintf("resource sampling failed for %d sample(s); trend calculations ignore tainted samples", sampleCaptureErrors),
		)
	}
	perfEvaluateSummary(&summary, thresholds)
	if scenario.DeleteReinsertWeight > 0 && !summary.DeleteReinsertEnabled {
		summary.Alerts = append(summary.Alerts, "delete+reinsert workload is disabled (set CHROMA_PERF_ENABLE_DELETE_REINSERT=true to enable)")
	}
	return summary, retErr
}

func perfRunChurnScenario(cfg perfRuntimeConfig, thresholds perfThresholds, scenario perfScenarioConfig) (perfSummary, error) {
	startedAt := time.Now().UTC()
	persistPath, err := os.MkdirTemp("", "chroma-local-perf-churn-*")
	if err != nil {
		return perfSummary{}, errors.Wrap(err, "failed to create churn persist path")
	}
	defer func() { _ = os.RemoveAll(persistPath) }()

	metrics := newPerfMetrics()
	perfStabilizeGC()
	baseline, err := perfCaptureSnapshot(persistPath)
	if err != nil {
		return perfSummary{}, err
	}

	samples := []perfSample{perfCollectSample(startedAt, persistPath, metrics)}

	ctx := context.Background()
	consecutiveFailures := 0
	const maxConsecutiveChurnFailures = 5
	for i := 0; i < scenario.ChurnCycles; i++ {
		started := time.Now()
		client, clientErr := perfCreatePersistentClient(scenario.Mode, persistPath)
		if clientErr == nil {
			hbCtx, hbCancel := context.WithTimeout(ctx, 20*time.Second)
			heartbeatErr := client.Heartbeat(hbCtx)
			hbCancel()
			closeErr := client.Close()
			if heartbeatErr != nil {
				clientErr = heartbeatErr
				if closeErr != nil {
					clientErr = errors.Wrapf(heartbeatErr, "client close also failed: %v", closeErr)
				}
			} else {
				clientErr = closeErr
			}
		}
		metrics.observe(perfOpChurnCycle, time.Since(started), clientErr)
		if clientErr == nil {
			consecutiveFailures = 0
			continue
		}
		consecutiveFailures++
		if consecutiveFailures >= maxConsecutiveChurnFailures {
			return perfSummary{}, errors.Wrapf(
				clientErr,
				"churn scenario aborted after %d consecutive failures",
				consecutiveFailures,
			)
		}
	}

	perfStabilizeGC()
	finalSnapshot, err := perfCaptureSnapshot(persistPath)
	if err != nil {
		return perfSummary{}, err
	}
	samples = append(samples, perfCollectSample(startedAt, persistPath, metrics))
	capturedSamples := perfFilterCapturedSamples(samples)

	opSummaries := metrics.operationSummaries()
	var totalOps uint64
	var totalErrors uint64
	for _, op := range opSummaries {
		totalOps += op.Count
		totalErrors += op.Errors
	}

	summary := perfSummary{
		Profile:                       cfg.Profile,
		Scenario:                      scenario,
		StartedAt:                     startedAt,
		EndedAt:                       time.Now().UTC(),
		ElapsedSeconds:                time.Since(startedAt).Seconds(),
		Enforced:                      cfg.Enforce,
		Thresholds:                    thresholds,
		Baseline:                      baseline,
		Final:                         finalSnapshot,
		RecordCountStart:              0,
		RecordCountEnd:                0,
		DurabilityCheckPassed:         true,
		SampleCaptureErrors:           perfCountSampleCaptureErrors(samples),
		TotalOperations:               totalOps,
		TotalErrors:                   totalErrors,
		Operations:                    opSummaries,
		HeapSlopeMiBPerMinute:         perfSlopePerMinute(capturedSamples, func(sample perfSample) float64 { return float64(sample.HeapAllocBytes) / (1024 * 1024) }),
		GoroutineSlopePerMinute:       perfSlopePerMinute(capturedSamples, func(sample perfSample) float64 { return float64(sample.NumGoroutines) }),
		ThroughputFirstQuartileOpsSec: 0,
		ThroughputLastQuartileOpsSec:  0,
		WALMedianBytes:                perfMedianWALBytes(capturedSamples),
		DeleteReinsertEnabled:         false,
		Samples:                       samples,
		Alerts:                        []string{},
		Failures:                      []string{},
	}
	if summary.SampleCaptureErrors > 0 {
		summary.Alerts = append(summary.Alerts,
			fmt.Sprintf("resource sampling failed for %d sample(s); trend calculations ignore tainted samples", summary.SampleCaptureErrors),
		)
	}
	perfEvaluateSummary(&summary, thresholds)
	return summary, nil
}

func perfDurabilityCheck(scenario perfScenarioConfig, persistPath, collectionName string, expectedCount int, dataset perfDataset) (retErr error) {
	ctx := context.Background()
	client, err := perfCreatePersistentClient(scenario.Mode, persistPath)
	if err != nil {
		return errors.Wrap(err, "durability check failed to reopen client")
	}
	defer func() {
		closeErr := client.Close()
		if closeErr == nil {
			return
		}
		wrapped := errors.Wrap(closeErr, "durability check failed to close client")
		if retErr != nil {
			retErr = stderrors.Join(retErr, wrapped)
		} else {
			retErr = wrapped
		}
	}()

	collection, err := client.GetCollection(ctx, collectionName)
	if err != nil {
		return errors.Wrap(err, "durability check failed to get collection")
	}
	count, err := collection.Count(ctx)
	if err != nil {
		return errors.Wrap(err, "durability check failed to count records")
	}
	if expectedCount >= 0 && count != expectedCount {
		return errors.Errorf("durability count mismatch: expected %d got %d", expectedCount, count)
	}

	if len(dataset.IDs) == 0 {
		return nil
	}
	id := dataset.IDs[0]
	getResult, err := collection.Get(ctx, WithIDs(id), WithInclude(IncludeDocuments))
	if err != nil {
		return errors.Wrap(err, "durability check failed to get known id")
	}
	if getResult == nil || getResult.Count() == 0 {
		return errors.Errorf("durability check expected id %s to exist", id)
	}

	if !scenario.UseDefaultEF && len(dataset.Embeddings) > 0 {
		queryResult, queryErr := collection.Query(ctx,
			WithQueryEmbeddings(dataset.Embeddings[0]),
			WithNResults(1),
			WithInclude(IncludeDistances),
		)
		if queryErr != nil {
			return errors.Wrap(queryErr, "durability check query failed")
		}
		if queryResult == nil || len(queryResult.GetIDGroups()) == 0 || len(queryResult.GetIDGroups()[0]) == 0 {
			return errors.New("durability check query returned no ids")
		}
	}
	return nil
}

func perfEvaluateSummary(summary *perfSummary, thresholds perfThresholds) {
	if summary == nil {
		return
	}

	switch {
	case summary.TotalOperations > 0:
		errorRate := float64(summary.TotalErrors) / float64(summary.TotalOperations)
		if errorRate > thresholds.ErrorRateMax {
			summary.Failures = append(summary.Failures,
				fmt.Sprintf("error rate %.4f exceeds max %.4f", errorRate, thresholds.ErrorRateMax),
			)
		}
	case summary.TotalErrors > 0:
		summary.Failures = append(summary.Failures, "errors observed with zero successful operations")
	default:
		summary.Failures = append(summary.Failures, "zero operations recorded; workload did not execute")
	}
	if summary.RecordCountEndError != "" {
		summary.Failures = append(summary.Failures,
			fmt.Sprintf("failed to count final records: %s", summary.RecordCountEndError),
		)
	}

	if querySummary, ok := summary.Operations[perfOpQuery]; ok && querySummary.Count > 0 {
		if querySummary.P95Millis > thresholds.QueryP95MaxMs {
			summary.Failures = append(summary.Failures,
				fmt.Sprintf("query p95 %.2fms exceeds max %.2fms", querySummary.P95Millis, thresholds.QueryP95MaxMs),
			)
		}
	}
	if getSummary, ok := summary.Operations[perfOpGet]; ok && getSummary.Count > 0 {
		if getSummary.P95Millis > thresholds.GetP95MaxMs {
			summary.Failures = append(summary.Failures,
				fmt.Sprintf("get p95 %.2fms exceeds max %.2fms", getSummary.P95Millis, thresholds.GetP95MaxMs),
			)
		}
	}

	writeP95 := 0.0
	for _, opName := range []string{perfOpUpsert, perfOpDeleteReinsert} {
		if opSummary, ok := summary.Operations[opName]; ok && opSummary.Count > 0 {
			if opSummary.P95Millis > writeP95 {
				writeP95 = opSummary.P95Millis
			}
		}
	}
	if writeP95 > thresholds.WriteP95MaxMs {
		summary.Failures = append(summary.Failures,
			fmt.Sprintf("write p95 %.2fms exceeds max %.2fms", writeP95, thresholds.WriteP95MaxMs),
		)
	}

	heapGrowthBytes := int64(summary.Final.HeapAllocBytes) - int64(summary.Baseline.HeapAllocBytes)
	if heapGrowthBytes > 0 {
		percentAllowance := uint64(float64(summary.Baseline.HeapAllocBytes) * (thresholds.MaxHeapGrowthPercent / 100.0))
		allowedGrowth := thresholds.MaxHeapGrowthBytes
		if percentAllowance > allowedGrowth {
			allowedGrowth = percentAllowance
		}
		if uint64(heapGrowthBytes) > allowedGrowth {
			summary.Failures = append(summary.Failures,
				fmt.Sprintf("heap growth %d bytes exceeds allowed %d bytes", heapGrowthBytes, allowedGrowth),
			)
		}
	}

	goroutineGrowth := summary.Final.NumGoroutines - summary.Baseline.NumGoroutines
	if goroutineGrowth > thresholds.MaxGoroutineGrowth {
		summary.Failures = append(summary.Failures,
			fmt.Sprintf("goroutine growth %+d exceeds max %+d", goroutineGrowth, thresholds.MaxGoroutineGrowth),
		)
	}

	if summary.Baseline.FDMeasured && summary.Final.FDMeasured {
		fdGrowth := summary.Final.FDCount - summary.Baseline.FDCount
		if fdGrowth > thresholds.MaxFDGrowth {
			summary.Failures = append(summary.Failures,
				fmt.Sprintf("fd growth %+d exceeds max %+d", fdGrowth, thresholds.MaxFDGrowth),
			)
		}
	}

	if summary.Scenario.RestartDurability && !summary.DurabilityCheckPassed {
		if strings.TrimSpace(summary.DurabilityError) != "" {
			summary.Failures = append(summary.Failures,
				fmt.Sprintf("restart durability check failed: %s", summary.DurabilityError),
			)
		} else {
			summary.Failures = append(summary.Failures, "restart durability check failed")
		}
	}

	heapSlopeThreshold := thresholds.SyntheticHeapSlopeAlertMiB
	if summary.Scenario.UseDefaultEF {
		heapSlopeThreshold = thresholds.DefaultEFHeapSlopeAlertMiB
	}
	if summary.HeapSlopeMiBPerMinute > heapSlopeThreshold {
		summary.Alerts = append(summary.Alerts,
			fmt.Sprintf("heap slope %.2f MiB/min exceeds alert threshold %.2f MiB/min", summary.HeapSlopeMiBPerMinute, heapSlopeThreshold),
		)
	}

	if summary.GoroutineSlopePerMinute > thresholds.MaxGoroutineSlopePerMinute {
		summary.Alerts = append(summary.Alerts,
			fmt.Sprintf("goroutine slope %.3f/min exceeds alert threshold %.3f/min", summary.GoroutineSlopePerMinute, thresholds.MaxGoroutineSlopePerMinute),
		)
	}

	if summary.ThroughputFirstQuartileOpsSec > 0 {
		minAllowed := summary.ThroughputFirstQuartileOpsSec * (1 - thresholds.ThroughputDriftAlertFraction)
		if summary.ThroughputLastQuartileOpsSec < minAllowed {
			summary.Alerts = append(summary.Alerts,
				fmt.Sprintf("throughput drift detected: first quartile %.2f ops/s, last quartile %.2f ops/s", summary.ThroughputFirstQuartileOpsSec, summary.ThroughputLastQuartileOpsSec),
			)
		}
	}

	if perfHasWALAnomaly(summary, thresholds.WalAnomalyFactor) {
		summary.Alerts = append(summary.Alerts,
			"wal anomaly detected: mostly monotonic WAL growth with final size above 4x median and no record-count growth",
		)
	}
}

func perfSlopePerMinute(samples []perfSample, value func(sample perfSample) float64) float64 {
	if len(samples) < 2 {
		return 0
	}
	first := samples[0]
	last := samples[len(samples)-1]
	durationMinutes := last.Timestamp.Sub(first.Timestamp).Minutes()
	if durationMinutes <= 0 {
		return 0
	}
	return (value(last) - value(first)) / durationMinutes
}

func perfThroughputQuartiles(samples []perfSample) (float64, float64) {
	if len(samples) < 5 {
		return 0, 0
	}
	intervalRates := make([]float64, 0, len(samples)-1)
	for i := 1; i < len(samples); i++ {
		opsDelta := int64(samples[i].OpsTotal) - int64(samples[i-1].OpsTotal)
		elapsed := samples[i].ElapsedSeconds - samples[i-1].ElapsedSeconds
		if opsDelta < 0 || elapsed <= 0 {
			continue
		}
		intervalRates = append(intervalRates, float64(opsDelta)/elapsed)
	}
	if len(intervalRates) < 4 {
		return 0, 0
	}
	quartileSize := len(intervalRates) / 4
	if quartileSize == 0 {
		quartileSize = 1
	}
	first := perfMean(intervalRates[:quartileSize])
	last := perfMean(intervalRates[len(intervalRates)-quartileSize:])
	return first, last
}

func perfMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func perfMedianWALBytes(samples []perfSample) int64 {
	if len(samples) == 0 {
		return 0
	}
	values := make([]int64, 0, len(samples))
	for _, sample := range samples {
		values = append(values, sample.WALBytes)
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	middle := len(values) / 2
	if len(values)%2 == 0 {
		return (values[middle-1] + values[middle]) / 2
	}
	return values[middle]
}

func perfHasWALAnomaly(summary *perfSummary, factor float64) bool {
	if summary == nil {
		return false
	}
	samples := perfFilterCapturedSamples(summary.Samples)
	if len(samples) < 5 {
		return false
	}
	if summary.RecordCountStart < 0 || summary.RecordCountEnd < 0 {
		return false
	}
	median := perfMedianWALBytes(samples)
	if median <= 0 {
		return false
	}
	finalWAL := samples[len(samples)-1].WALBytes
	if float64(finalWAL) < factor*float64(median) {
		return false
	}

	nondecreasing := 0
	for i := 1; i < len(samples); i++ {
		if samples[i].WALBytes >= samples[i-1].WALBytes {
			nondecreasing++
		}
	}
	ratio := float64(nondecreasing) / float64(len(samples)-1)
	if ratio < 0.9 {
		return false
	}

	recordGrowth := summary.RecordCountEnd > summary.RecordCountStart
	return !recordGrowth
}

func perfWriteScenarioJSON(reportDir string, summary perfSummary) (string, error) {
	if strings.TrimSpace(reportDir) == "" {
		return "", errors.New("report dir cannot be empty")
	}
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return "", errors.Wrap(err, "failed to create report dir")
	}
	fileName := fmt.Sprintf("perf-summary-%s-%s.json", perfSlugify(summary.Profile), perfSlugify(summary.Scenario.Name))
	path := filepath.Join(reportDir, fileName)
	summary.ReportJSONPath = path
	payload, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal scenario summary")
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return "", errors.Wrap(err, "failed to write scenario summary")
	}
	return path, nil
}

func perfWriteMarkdownSummary(reportDir, profile string, summaries []perfSummary) (string, error) {
	if strings.TrimSpace(reportDir) == "" {
		return "", errors.New("report dir cannot be empty")
	}
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return "", errors.Wrap(err, "failed to create report dir")
	}
	path := filepath.Join(reportDir, fmt.Sprintf("perf-summary-%s.md", perfSlugify(profile)))

	var builder strings.Builder
	builder.WriteString("# Local Persistent Chroma Performance Summary\n\n")
	_, _ = fmt.Fprintf(&builder, "Profile: `%s`\n\n", profile)
	builder.WriteString("| Scenario | Mode | Kind | Duration(s) | Ops | Errors | Query p95 (ms) | Write p95 (ms) | Heap Δ (MiB) | Goroutine Δ | Status | JSON |\n")
	builder.WriteString("|---|---|---|---:|---:|---:|---:|---:|---:|---:|---|---|\n")

	for _, summary := range summaries {
		queryP95 := 0.0
		if query, ok := summary.Operations[perfOpQuery]; ok {
			queryP95 = query.P95Millis
		}
		writeP95 := 0.0
		for _, opName := range []string{perfOpUpsert, perfOpDeleteReinsert} {
			if opSummary, ok := summary.Operations[opName]; ok && opSummary.P95Millis > writeP95 {
				writeP95 = opSummary.P95Millis
			}
		}
		heapDeltaMiB := (float64(summary.Final.HeapAllocBytes) - float64(summary.Baseline.HeapAllocBytes)) / (1024 * 1024)
		goroutineDelta := summary.Final.NumGoroutines - summary.Baseline.NumGoroutines
		status := "PASS"
		if len(summary.Failures) > 0 {
			if summary.Enforced {
				status = "FAIL"
			} else {
				status = "WARN"
			}
		} else if len(summary.Alerts) > 0 {
			status = "ALERT"
		}
		jsonCell := "-"
		if summary.ReportJSONPath != "" {
			jsonCell = filepath.Base(summary.ReportJSONPath)
		}
		_, _ = fmt.Fprintf(&builder,
			"| %s | %s | %s | %.0f | %d | %d | %.2f | %.2f | %.2f | %+d | %s | %s |\n",
			summary.Scenario.Name,
			summary.Scenario.Mode,
			summary.Scenario.Kind,
			summary.ElapsedSeconds,
			summary.TotalOperations,
			summary.TotalErrors,
			queryP95,
			writeP95,
			heapDeltaMiB,
			goroutineDelta,
			status,
			jsonCell,
		)
	}

	builder.WriteString("\n## Alerts and failures\n")
	for _, summary := range summaries {
		if len(summary.Alerts) == 0 && len(summary.Failures) == 0 {
			continue
		}
		_, _ = fmt.Fprintf(&builder, "\n### %s\n", summary.Scenario.Name)
		for _, failure := range summary.Failures {
			_, _ = fmt.Fprintf(&builder, "- Failure: %s\n", failure)
		}
		for _, alert := range summary.Alerts {
			_, _ = fmt.Fprintf(&builder, "- Alert: %s\n", alert)
		}
	}

	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		return "", errors.Wrap(err, "failed to write markdown summary")
	}
	return path, nil
}

func perfSlugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	var builder strings.Builder
	builder.Grow(len(value))
	for _, ch := range value {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			builder.WriteRune(ch)
			continue
		}
		builder.WriteByte('_')
	}
	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return "unknown"
	}
	return result
}
