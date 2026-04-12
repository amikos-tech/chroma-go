//go:build basicv2 && !cloud
// +build basicv2,!cloud

package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	localchroma "github.com/amikos-tech/chroma-go-local"
	embeddingspkg "github.com/amikos-tech/chroma-go/pkg/embeddings"
	ortpkg "github.com/amikos-tech/chroma-go/pkg/embeddings/ort"
	loggerpkg "github.com/amikos-tech/chroma-go/pkg/logger"
)

type scriptedEmbeddedRuntime struct {
	*stubEmbeddedRuntime
	healthResponses   []*localchroma.EmbeddedHealthCheckResponse
	healthErr         error
	heartbeatErr      error
	resetErr          error
	createTenantErr   error
	getTenantErr      error
	createDatabaseErr error
	listDatabasesErr  error
	getDatabaseErr    error
	deleteDatabaseErr error

	healthCalls         int
	heartbeatCalls      int
	createTenantCalls   int
	getTenantCalls      atomic.Int32
	createDatabaseCalls int
	listDatabasesCalls  int
	getDatabaseCalls    atomic.Int32
	deleteDatabaseCalls int
}

type memoryEmbeddedRecord struct {
	embedding []float32
	document  *string
	metadata  map[string]any
}

type memoryEmbeddedRuntime struct {
	*stubEmbeddedRuntime
	mu sync.Mutex

	nextCollectionID int
	collections      map[string]localchroma.EmbeddedCollection
	collectionByID   map[string]string

	records     map[string]map[string]memoryEmbeddedRecord
	recordOrder map[string][]string
}

type blockingRenameEmbeddedRuntime struct {
	*memoryEmbeddedRuntime

	firstUpdateStarted chan struct{}
	unblockFirstUpdate chan struct{}

	updateMu    sync.Mutex
	updateCalls int
}

type panicRenameEmbeddedRuntime struct {
	*memoryEmbeddedRuntime
}

type mismatchedQueryEmbeddedRuntime struct {
	*memoryEmbeddedRuntime
}

type missingProjectionEmbeddedRuntime struct {
	*memoryEmbeddedRuntime
	dropProjectionOnce sync.Once
}

type emptyProjectionEmbeddingRuntime struct {
	*memoryEmbeddedRuntime
	makeEmptyOnce sync.Once
}

type countingMemoryEmbeddedRuntime struct {
	*memoryEmbeddedRuntime
	mu sync.Mutex

	createCollectionCalls int
	getCollectionCalls    int
}

type jsonRoundTripMissingGetCollectionOnceRuntime struct {
	*missingGetCollectionOnceRuntime
}

type errorGetCollectionOnceRuntime struct {
	*memoryEmbeddedRuntime
	lookupErr error
	failOnce  atomic.Bool
}

type blockingGetMemoryEmbeddedRuntime struct {
	*memoryEmbeddedRuntime

	firstSnapshotTaken chan struct{}
	unblockFirstGet    chan struct{}
	getCalls           atomic.Int32
}

type blockingCreateAfterMissMemoryEmbeddedRuntime struct {
	*memoryEmbeddedRuntime

	targetCollectionName string
	firstCreateReady     chan struct{}
	allowFirstCreate     chan struct{}
	createCalls          atomic.Int32
}

type failingRevalidationGetMemoryEmbeddedRuntime struct {
	*memoryEmbeddedRuntime

	targetCollectionName string

	mu                 sync.Mutex
	getCollectionCalls map[string]int
}

type failingCreateCollectionRuntime struct {
	*stubEmbeddedRuntime
	createErr error
}

type invalidCreateResponseDeleteTrackingRuntime struct {
	*memoryEmbeddedRuntime
	deleteCalls []localchroma.EmbeddedDeleteCollectionRequest
}

type invalidCreateResponseMissingDeleteRuntime struct {
	*memoryEmbeddedRuntime
	deleteCalls []localchroma.EmbeddedDeleteCollectionRequest
}

type invalidCreateResponseDeleteErrorRuntime struct {
	*memoryEmbeddedRuntime
	deleteCalls []localchroma.EmbeddedDeleteCollectionRequest
	deleteErr   error
}

type preExistingInvalidCreateRuntime struct {
	*memoryEmbeddedRuntime
	preflightErr error
	deleteCalls  []localchroma.EmbeddedDeleteCollectionRequest
}

type missingGetCollectionOnceRuntime struct {
	*memoryEmbeddedRuntime
	missNextGet atomic.Bool
}

type staleGetCollectionDeleteRuntime struct {
	*memoryEmbeddedRuntime
	staleNextGet atomic.Bool
}

type failingUpdateEmbeddedRuntime struct {
	*memoryEmbeddedRuntime

	updateErr error
	updateMu  sync.Mutex
	updateCnt int
}

type recordingDeleteEmbeddedRuntime struct {
	*stubEmbeddedRuntime

	lastDeleteRequest localchroma.EmbeddedDeleteRecordsRequest
	deleteCalls       int
}

func newScriptedEmbeddedRuntime() *scriptedEmbeddedRuntime {
	return &scriptedEmbeddedRuntime{
		stubEmbeddedRuntime: &stubEmbeddedRuntime{},
	}
}

func newMemoryEmbeddedRuntime() *memoryEmbeddedRuntime {
	return &memoryEmbeddedRuntime{
		stubEmbeddedRuntime: &stubEmbeddedRuntime{},
		collections:         map[string]localchroma.EmbeddedCollection{},
		collectionByID:      map[string]string{},
		records:             map[string]map[string]memoryEmbeddedRecord{},
		recordOrder:         map[string][]string{},
	}
}

func newBlockingRenameEmbeddedRuntime() *blockingRenameEmbeddedRuntime {
	return &blockingRenameEmbeddedRuntime{
		memoryEmbeddedRuntime: newMemoryEmbeddedRuntime(),
		firstUpdateStarted:    make(chan struct{}),
		unblockFirstUpdate:    make(chan struct{}),
	}
}

func newCountingMemoryEmbeddedRuntime() *countingMemoryEmbeddedRuntime {
	return &countingMemoryEmbeddedRuntime{
		memoryEmbeddedRuntime: newMemoryEmbeddedRuntime(),
	}
}

func newBlockingGetMemoryEmbeddedRuntime() *blockingGetMemoryEmbeddedRuntime {
	return &blockingGetMemoryEmbeddedRuntime{
		memoryEmbeddedRuntime: newMemoryEmbeddedRuntime(),
		firstSnapshotTaken:    make(chan struct{}),
		unblockFirstGet:       make(chan struct{}),
	}
}

func newBlockingCreateAfterMissMemoryEmbeddedRuntime(targetCollectionName string) *blockingCreateAfterMissMemoryEmbeddedRuntime {
	return &blockingCreateAfterMissMemoryEmbeddedRuntime{
		memoryEmbeddedRuntime: newMemoryEmbeddedRuntime(),
		targetCollectionName:  targetCollectionName,
		firstCreateReady:      make(chan struct{}),
		allowFirstCreate:      make(chan struct{}),
	}
}

func newFailingRevalidationGetMemoryEmbeddedRuntime(targetCollectionName string) *failingRevalidationGetMemoryEmbeddedRuntime {
	return &failingRevalidationGetMemoryEmbeddedRuntime{
		memoryEmbeddedRuntime: newMemoryEmbeddedRuntime(),
		targetCollectionName:  targetCollectionName,
		getCollectionCalls:    map[string]int{},
	}
}

func newJSONRoundTripMissingGetCollectionOnceRuntime() *jsonRoundTripMissingGetCollectionOnceRuntime {
	return &jsonRoundTripMissingGetCollectionOnceRuntime{
		missingGetCollectionOnceRuntime: newMissingGetCollectionOnceRuntime(),
	}
}

func newErrorGetCollectionOnceRuntime(lookupErr error) *errorGetCollectionOnceRuntime {
	runtime := &errorGetCollectionOnceRuntime{
		memoryEmbeddedRuntime: newMemoryEmbeddedRuntime(),
		lookupErr:             lookupErr,
	}
	runtime.failOnce.Store(true)
	return runtime
}

func newInvalidCreateResponseDeleteTrackingRuntime() *invalidCreateResponseDeleteTrackingRuntime {
	return &invalidCreateResponseDeleteTrackingRuntime{
		memoryEmbeddedRuntime: newMemoryEmbeddedRuntime(),
		deleteCalls:           make([]localchroma.EmbeddedDeleteCollectionRequest, 0, 1),
	}
}

func newInvalidCreateResponseMissingDeleteRuntime() *invalidCreateResponseMissingDeleteRuntime {
	return &invalidCreateResponseMissingDeleteRuntime{
		memoryEmbeddedRuntime: newMemoryEmbeddedRuntime(),
		deleteCalls:           make([]localchroma.EmbeddedDeleteCollectionRequest, 0, 1),
	}
}

func newInvalidCreateResponseDeleteErrorRuntime(deleteErr error) *invalidCreateResponseDeleteErrorRuntime {
	return &invalidCreateResponseDeleteErrorRuntime{
		memoryEmbeddedRuntime: newMemoryEmbeddedRuntime(),
		deleteCalls:           make([]localchroma.EmbeddedDeleteCollectionRequest, 0, 1),
		deleteErr:             deleteErr,
	}
}

func newPreExistingInvalidCreateRuntime(preflightErr error) *preExistingInvalidCreateRuntime {
	return &preExistingInvalidCreateRuntime{
		memoryEmbeddedRuntime: newMemoryEmbeddedRuntime(),
		preflightErr:          preflightErr,
		deleteCalls:           make([]localchroma.EmbeddedDeleteCollectionRequest, 0, 1),
	}
}

func newMissingGetCollectionOnceRuntime() *missingGetCollectionOnceRuntime {
	runtime := &missingGetCollectionOnceRuntime{
		memoryEmbeddedRuntime: newMemoryEmbeddedRuntime(),
	}
	runtime.missNextGet.Store(true)
	return runtime
}

func newStaleGetCollectionDeleteRuntime() *staleGetCollectionDeleteRuntime {
	runtime := &staleGetCollectionDeleteRuntime{
		memoryEmbeddedRuntime: newMemoryEmbeddedRuntime(),
	}
	runtime.staleNextGet.Store(true)
	return runtime
}

func newFailingUpdateEmbeddedRuntime(updateErr error) *failingUpdateEmbeddedRuntime {
	return &failingUpdateEmbeddedRuntime{
		memoryEmbeddedRuntime: newMemoryEmbeddedRuntime(),
		updateErr:             updateErr,
	}
}

func newRecordingDeleteEmbeddedRuntime() *recordingDeleteEmbeddedRuntime {
	return &recordingDeleteEmbeddedRuntime{
		stubEmbeddedRuntime: &stubEmbeddedRuntime{},
	}
}

func (s *countingMemoryEmbeddedRuntime) CreateCollection(request localchroma.EmbeddedCreateCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	s.mu.Lock()
	s.createCollectionCalls++
	s.mu.Unlock()
	return s.memoryEmbeddedRuntime.CreateCollection(request)
}

func (s *countingMemoryEmbeddedRuntime) GetCollection(request localchroma.EmbeddedGetCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	s.mu.Lock()
	s.getCollectionCalls++
	s.mu.Unlock()
	return s.memoryEmbeddedRuntime.GetCollection(request)
}

func (s *blockingGetMemoryEmbeddedRuntime) GetCollection(request localchroma.EmbeddedGetCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	col, err := s.memoryEmbeddedRuntime.GetCollection(request)
	if err != nil {
		return nil, err
	}
	if s.getCalls.Add(1) == 1 {
		close(s.firstSnapshotTaken)
		<-s.unblockFirstGet
	}
	return col, nil
}

func (s *blockingCreateAfterMissMemoryEmbeddedRuntime) CreateCollection(request localchroma.EmbeddedCreateCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	if request.Name == s.targetCollectionName && s.createCalls.Add(1) == 1 {
		close(s.firstCreateReady)
		select {
		case <-s.allowFirstCreate:
		case <-time.After(5 * time.Second):
			return nil, errors.New("timed out waiting to resume first create")
		}
	}
	return s.memoryEmbeddedRuntime.CreateCollection(request)
}

func (s *failingRevalidationGetMemoryEmbeddedRuntime) GetCollection(request localchroma.EmbeddedGetCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	if request.Name != s.targetCollectionName {
		return s.memoryEmbeddedRuntime.GetCollection(request)
	}

	s.mu.Lock()
	s.getCollectionCalls[request.Name]++
	callNo := s.getCollectionCalls[request.Name]
	s.mu.Unlock()

	if callNo == 2 {
		return nil, errors.New("revalidation boom")
	}
	return s.memoryEmbeddedRuntime.GetCollection(request)
}

func (s *jsonRoundTripMissingGetCollectionOnceRuntime) CreateCollection(request localchroma.EmbeddedCreateCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	col, err := s.missingGetCollectionOnceRuntime.CreateCollection(request)
	if err != nil {
		return nil, err
	}
	return roundTripEmbeddedCollectionModel(col)
}

func (s *jsonRoundTripMissingGetCollectionOnceRuntime) GetCollection(request localchroma.EmbeddedGetCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	col, err := s.missingGetCollectionOnceRuntime.GetCollection(request)
	if err != nil {
		return nil, err
	}
	return roundTripEmbeddedCollectionModel(col)
}

func (s *errorGetCollectionOnceRuntime) GetCollection(request localchroma.EmbeddedGetCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	if s.failOnce.CompareAndSwap(true, false) {
		return nil, s.lookupErr
	}
	return s.memoryEmbeddedRuntime.GetCollection(request)
}

func (s *failingCreateCollectionRuntime) CreateCollection(localchroma.EmbeddedCreateCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	return nil, s.createErr
}

func (s *invalidCreateResponseDeleteTrackingRuntime) CreateCollection(request localchroma.EmbeddedCreateCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	col, err := s.memoryEmbeddedRuntime.CreateCollection(request)
	if err != nil {
		return nil, err
	}
	invalid := *col
	invalid.Metadata = map[string]any{
		"invalid": map[string]any{"nested": "object"},
	}
	return &invalid, nil
}

func (s *invalidCreateResponseDeleteTrackingRuntime) DeleteCollection(request localchroma.EmbeddedDeleteCollectionRequest) error {
	s.deleteCalls = append(s.deleteCalls, request)
	return s.memoryEmbeddedRuntime.DeleteCollection(request)
}

func (s *invalidCreateResponseMissingDeleteRuntime) CreateCollection(request localchroma.EmbeddedCreateCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	return &localchroma.EmbeddedCollection{
		ID:       "dangling-invalid-id",
		Name:     request.Name,
		Tenant:   normalizeEmbeddedTenant(request.TenantID),
		Database: normalizeEmbeddedDatabase(request.DatabaseName),
		Metadata: map[string]any{
			"invalid": map[string]any{"nested": "object"},
		},
	}, nil
}

func (s *invalidCreateResponseMissingDeleteRuntime) DeleteCollection(request localchroma.EmbeddedDeleteCollectionRequest) error {
	s.deleteCalls = append(s.deleteCalls, request)
	return s.memoryEmbeddedRuntime.DeleteCollection(request)
}

func (s *invalidCreateResponseDeleteErrorRuntime) CreateCollection(request localchroma.EmbeddedCreateCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	col, err := s.memoryEmbeddedRuntime.CreateCollection(request)
	if err != nil {
		return nil, err
	}
	invalid := *col
	invalid.Metadata = map[string]any{
		"invalid": map[string]any{"nested": "object"},
	}
	return &invalid, nil
}

func (s *invalidCreateResponseDeleteErrorRuntime) DeleteCollection(request localchroma.EmbeddedDeleteCollectionRequest) error {
	s.deleteCalls = append(s.deleteCalls, request)
	return s.deleteErr
}

func (s *preExistingInvalidCreateRuntime) GetCollection(_ localchroma.EmbeddedGetCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	return nil, s.preflightErr
}

func (s *preExistingInvalidCreateRuntime) CreateCollection(request localchroma.EmbeddedCreateCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	col, err := s.memoryEmbeddedRuntime.CreateCollection(request)
	if err != nil {
		return nil, err
	}
	invalid := *col
	invalid.Metadata = map[string]any{
		"invalid": map[string]any{"nested": "object"},
	}
	return &invalid, nil
}

func (s *preExistingInvalidCreateRuntime) DeleteCollection(request localchroma.EmbeddedDeleteCollectionRequest) error {
	s.deleteCalls = append(s.deleteCalls, request)
	return s.memoryEmbeddedRuntime.DeleteCollection(request)
}

func (s *missingGetCollectionOnceRuntime) GetCollection(request localchroma.EmbeddedGetCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	if s.missNextGet.CompareAndSwap(true, false) {
		return nil, errors.New("collection not found")
	}
	return s.memoryEmbeddedRuntime.GetCollection(request)
}

func (s *staleGetCollectionDeleteRuntime) GetCollection(request localchroma.EmbeddedGetCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	if !s.staleNextGet.CompareAndSwap(true, false) {
		return s.memoryEmbeddedRuntime.GetCollection(request)
	}

	s.memoryEmbeddedRuntime.mu.Lock()
	defer s.memoryEmbeddedRuntime.mu.Unlock()

	key := collectionRuntimeKey(request.TenantID, request.DatabaseName, request.Name)
	col, ok := s.memoryEmbeddedRuntime.collections[key]
	if !ok {
		return nil, errors.New("collection not found")
	}

	delete(s.memoryEmbeddedRuntime.collections, key)
	delete(s.memoryEmbeddedRuntime.collectionByID, col.ID)
	delete(s.memoryEmbeddedRuntime.records, col.ID)
	delete(s.memoryEmbeddedRuntime.recordOrder, col.ID)

	copyCol := col
	return &copyCol, nil
}

func (s *failingUpdateEmbeddedRuntime) UpdateCollection(request localchroma.EmbeddedUpdateCollectionRequest) error {
	s.updateMu.Lock()
	s.updateCnt++
	s.updateMu.Unlock()
	if s.updateErr != nil {
		return s.updateErr
	}
	return s.memoryEmbeddedRuntime.UpdateCollection(request)
}

func (s *failingUpdateEmbeddedRuntime) UpdateCalls() int {
	s.updateMu.Lock()
	defer s.updateMu.Unlock()
	return s.updateCnt
}

func (s *recordingDeleteEmbeddedRuntime) DeleteRecords(request localchroma.EmbeddedDeleteRecordsRequest) error {
	s.lastDeleteRequest = request
	s.deleteCalls++
	return nil
}

func (s *countingMemoryEmbeddedRuntime) callCounts() (createCalls int, getCalls int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createCollectionCalls, s.getCollectionCalls
}

func normalizeEmbeddedTenant(tenant string) string {
	if tenant == "" {
		return DefaultTenant
	}
	return tenant
}

func normalizeEmbeddedDatabase(database string) string {
	if database == "" {
		return DefaultDatabase
	}
	return database
}

func collectionRuntimeKey(tenant, database, name string) string {
	return fmt.Sprintf("%s|%s|%s", normalizeEmbeddedTenant(tenant), normalizeEmbeddedDatabase(database), name)
}

func cloneMetadataMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = cloneMetadataValue(value)
	}
	return dst
}

func cloneMetadataValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMetadataMap(typed)
	case []any:
		cloned := make([]any, len(typed))
		for i, item := range typed {
			cloned[i] = cloneMetadataValue(item)
		}
		return cloned
	case []string:
		return append([]string(nil), typed...)
	case []int:
		return append([]int(nil), typed...)
	case []int64:
		return append([]int64(nil), typed...)
	case []float32:
		return append([]float32(nil), typed...)
	case []float64:
		return append([]float64(nil), typed...)
	case []bool:
		return append([]bool(nil), typed...)
	default:
		return typed
	}
}

func cloneEmbedding(src []float32) []float32 {
	if src == nil {
		return nil
	}
	dst := make([]float32, len(src))
	copy(dst, src)
	return dst
}

func roundTripEmbeddedCollectionModel(model *localchroma.EmbeddedCollection) (*localchroma.EmbeddedCollection, error) {
	if model == nil {
		return nil, nil
	}
	payload, err := json.Marshal(model)
	if err != nil {
		return nil, err
	}
	var roundTripped localchroma.EmbeddedCollection
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(&roundTripped); err != nil {
		return nil, err
	}
	return &roundTripped, nil
}

func (s *memoryEmbeddedRuntime) CreateCollection(request localchroma.EmbeddedCreateCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := collectionRuntimeKey(request.TenantID, request.DatabaseName, request.Name)
	if existing, ok := s.collections[key]; ok {
		if request.GetOrCreate {
			copyCol := existing
			return &copyCol, nil
		}
		return nil, errors.New("collection already exists")
	}

	s.nextCollectionID++
	col := localchroma.EmbeddedCollection{
		ID:                fmt.Sprintf("mem-col-%d", s.nextCollectionID),
		Name:              request.Name,
		Tenant:            normalizeEmbeddedTenant(request.TenantID),
		Database:          normalizeEmbeddedDatabase(request.DatabaseName),
		Metadata:          cloneMetadataMap(request.Metadata),
		ConfigurationJSON: cloneMetadataMap(request.Configuration),
		Schema:            cloneMetadataMap(request.Schema),
	}
	s.collections[key] = col
	s.collectionByID[col.ID] = key
	s.records[col.ID] = map[string]memoryEmbeddedRecord{}
	s.recordOrder[col.ID] = []string{}

	copyCol := col
	return &copyCol, nil
}

func (s *memoryEmbeddedRuntime) GetCollection(request localchroma.EmbeddedGetCollectionRequest) (*localchroma.EmbeddedCollection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := collectionRuntimeKey(request.TenantID, request.DatabaseName, request.Name)
	col, ok := s.collections[key]
	if !ok {
		return nil, errors.New("collection not found")
	}
	copyCol := col
	return &copyCol, nil
}

func (s *memoryEmbeddedRuntime) DeleteCollection(request localchroma.EmbeddedDeleteCollectionRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := collectionRuntimeKey(request.TenantID, request.DatabaseName, request.Name)
	col, ok := s.collections[key]
	if !ok {
		return errors.New("collection not found")
	}
	delete(s.collections, key)
	delete(s.collectionByID, col.ID)
	delete(s.records, col.ID)
	delete(s.recordOrder, col.ID)
	return nil
}

func (s *memoryEmbeddedRuntime) ListCollections(request localchroma.EmbeddedListCollectionsRequest) ([]localchroma.EmbeddedCollection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tenant := normalizeEmbeddedTenant(request.TenantID)
	database := normalizeEmbeddedDatabase(request.DatabaseName)

	list := make([]localchroma.EmbeddedCollection, 0, len(s.collections))
	for _, col := range s.collections {
		if col.Tenant == tenant && col.Database == database {
			list = append(list, col)
		}
	}

	start := int(request.Offset)
	if start >= len(list) {
		return []localchroma.EmbeddedCollection{}, nil
	}
	list = list[start:]
	if request.Limit > 0 && int(request.Limit) < len(list) {
		list = list[:request.Limit]
	}
	return list, nil
}

func (s *memoryEmbeddedRuntime) CountCollections(request localchroma.EmbeddedCountCollectionsRequest) (uint32, error) {
	list, err := s.ListCollections(localchroma.EmbeddedListCollectionsRequest{
		TenantID:     request.TenantID,
		DatabaseName: request.DatabaseName,
	})
	if err != nil {
		return 0, err
	}
	return uint32(len(list)), nil
}

func (s *memoryEmbeddedRuntime) UpdateCollection(request localchroma.EmbeddedUpdateCollectionRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldKey, ok := s.collectionByID[request.CollectionID]
	if !ok {
		return errors.New("collection not found")
	}
	col := s.collections[oldKey]
	if request.NewName != "" {
		col.Name = request.NewName
	}
	if request.DatabaseName != "" {
		col.Database = request.DatabaseName
	}
	if request.NewMetadata != nil {
		for key, value := range request.NewMetadata {
			if value == nil {
				return fmt.Errorf("invalid new_metadata: metadata.%s cannot be null", key)
			}
		}
		col.Metadata = cloneMetadataMap(request.NewMetadata)
	}

	newKey := collectionRuntimeKey(col.Tenant, col.Database, col.Name)

	delete(s.collections, oldKey)
	s.collections[newKey] = col
	s.collectionByID[col.ID] = newKey
	return nil
}

func (s *blockingRenameEmbeddedRuntime) UpdateCollection(request localchroma.EmbeddedUpdateCollectionRequest) error {
	s.updateMu.Lock()
	s.updateCalls++
	callNo := s.updateCalls
	s.updateMu.Unlock()

	if callNo == 1 {
		close(s.firstUpdateStarted)
		<-s.unblockFirstUpdate
	}

	return s.memoryEmbeddedRuntime.UpdateCollection(request)
}

func (s *panicRenameEmbeddedRuntime) UpdateCollection(localchroma.EmbeddedUpdateCollectionRequest) error {
	panic("panic in UpdateCollection")
}

func (s *memoryEmbeddedRuntime) Add(request localchroma.EmbeddedAddRequest) error {
	return s.upsertOrAddRecords(request.CollectionID, request.IDs, request.Embeddings, request.Documents, request.Metadatas)
}

func (s *memoryEmbeddedRuntime) UpsertRecords(request localchroma.EmbeddedUpsertRecordsRequest) error {
	return s.upsertOrAddRecords(request.CollectionID, request.IDs, request.Embeddings, request.Documents, request.Metadatas)
}

func (s *memoryEmbeddedRuntime) upsertOrAddRecords(collectionID string, ids []string, embeddings [][]float32, documents []string, metadatas []map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	recordMap, ok := s.records[collectionID]
	if !ok {
		return errors.New("collection not found")
	}
	for i, id := range ids {
		_, existed := recordMap[id]
		record := memoryEmbeddedRecord{}
		if i < len(embeddings) {
			record.embedding = cloneEmbedding(embeddings[i])
		}
		if i < len(documents) {
			doc := documents[i]
			record.document = &doc
		}
		if i < len(metadatas) {
			record.metadata = cloneMetadataMap(metadatas[i])
		}
		recordMap[id] = record
		if !existed {
			s.recordOrder[collectionID] = append(s.recordOrder[collectionID], id)
		}
	}
	return nil
}

func (s *memoryEmbeddedRuntime) UpdateRecords(request localchroma.EmbeddedUpdateRecordsRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	recordMap, ok := s.records[request.CollectionID]
	if !ok {
		return errors.New("collection not found")
	}
	for i, id := range request.IDs {
		record, exists := recordMap[id]
		if !exists {
			continue
		}
		if i < len(request.Embeddings) {
			record.embedding = cloneEmbedding(request.Embeddings[i])
		}
		if i < len(request.Documents) {
			doc := request.Documents[i]
			record.document = &doc
		}
		if i < len(request.Metadatas) {
			record.metadata = cloneMetadataMap(request.Metadatas[i])
		}
		recordMap[id] = record
	}
	return nil
}

func (s *memoryEmbeddedRuntime) DeleteRecords(request localchroma.EmbeddedDeleteRecordsRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	recordMap, ok := s.records[request.CollectionID]
	if !ok {
		return errors.New("collection not found")
	}
	if len(request.IDs) == 0 {
		return nil
	}
	for _, id := range request.IDs {
		delete(recordMap, id)
	}
	order := s.recordOrder[request.CollectionID]
	filtered := make([]string, 0, len(order))
	for _, id := range order {
		if _, exists := recordMap[id]; exists {
			filtered = append(filtered, id)
		}
	}
	s.recordOrder[request.CollectionID] = filtered
	return nil
}

func (s *memoryEmbeddedRuntime) GetRecords(request localchroma.EmbeddedGetRecordsRequest) (*localchroma.EmbeddedGetRecordsResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	recordMap, ok := s.records[request.CollectionID]
	if !ok {
		return nil, errors.New("collection not found")
	}

	var ids []string
	if len(request.IDs) > 0 {
		ids = make([]string, 0, len(request.IDs))
		for _, id := range request.IDs {
			if _, exists := recordMap[id]; exists {
				ids = append(ids, id)
			}
		}
	} else {
		ids = append([]string{}, s.recordOrder[request.CollectionID]...)
		start := int(request.Offset)
		if start > len(ids) {
			start = len(ids)
		}
		ids = ids[start:]
		if request.Limit > 0 && int(request.Limit) < len(ids) {
			ids = ids[:request.Limit]
		}
	}

	response := &localchroma.EmbeddedGetRecordsResponse{
		IDs:        make([]string, 0, len(ids)),
		Embeddings: make([][]float32, 0, len(ids)),
		Documents:  make([]*string, 0, len(ids)),
		Metadatas:  make([]map[string]any, 0, len(ids)),
	}
	for _, id := range ids {
		record := recordMap[id]
		response.IDs = append(response.IDs, id)
		response.Embeddings = append(response.Embeddings, cloneEmbedding(record.embedding))
		if record.document != nil {
			doc := *record.document
			response.Documents = append(response.Documents, &doc)
		} else {
			response.Documents = append(response.Documents, nil)
		}
		response.Metadatas = append(response.Metadatas, cloneMetadataMap(record.metadata))
	}
	return response, nil
}

func (s *memoryEmbeddedRuntime) CountRecords(request localchroma.EmbeddedCountRecordsRequest) (uint32, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	recordMap, ok := s.records[request.CollectionID]
	if !ok {
		return 0, errors.New("collection not found")
	}
	return uint32(len(recordMap)), nil
}

func (s *memoryEmbeddedRuntime) Query(request localchroma.EmbeddedQueryRequest) (*localchroma.EmbeddedQueryResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	recordMap, ok := s.records[request.CollectionID]
	if !ok {
		return nil, errors.New("collection not found")
	}
	candidates := make([]string, 0)
	if len(request.IDs) > 0 {
		for _, id := range request.IDs {
			if _, exists := recordMap[id]; exists {
				candidates = append(candidates, id)
			}
		}
	} else {
		for _, id := range s.recordOrder[request.CollectionID] {
			if _, exists := recordMap[id]; exists {
				candidates = append(candidates, id)
			}
		}
	}

	n := len(candidates)
	if request.NResults > 0 && int(request.NResults) < n {
		n = int(request.NResults)
	}
	group := append([]string{}, candidates[:n]...)
	ids := make([][]string, len(request.QueryEmbeddings))
	for i := range request.QueryEmbeddings {
		ids[i] = append([]string{}, group...)
	}
	return &localchroma.EmbeddedQueryResponse{IDs: ids}, nil
}

func (s *mismatchedQueryEmbeddedRuntime) Query(request localchroma.EmbeddedQueryRequest) (*localchroma.EmbeddedQueryResponse, error) {
	response, err := s.memoryEmbeddedRuntime.Query(request)
	if err != nil {
		return nil, err
	}

	duplicate := []string{}
	if len(response.IDs) > 0 {
		duplicate = append(duplicate, response.IDs[0]...)
	}
	response.IDs = append(response.IDs, duplicate)
	return response, nil
}

func (s *missingProjectionEmbeddedRuntime) GetRecords(request localchroma.EmbeddedGetRecordsRequest) (*localchroma.EmbeddedGetRecordsResponse, error) {
	response, err := s.memoryEmbeddedRuntime.GetRecords(request)
	if err != nil {
		return nil, err
	}
	s.dropProjectionOnce.Do(func() {
		if len(response.IDs) < 2 {
			return
		}
		lastIdx := len(response.IDs) - 1
		response.IDs = response.IDs[:lastIdx]
		response.Embeddings = response.Embeddings[:lastIdx]
		response.Documents = response.Documents[:lastIdx]
		response.Metadatas = response.Metadatas[:lastIdx]
	})
	return response, nil
}

func (s *emptyProjectionEmbeddingRuntime) GetRecords(request localchroma.EmbeddedGetRecordsRequest) (*localchroma.EmbeddedGetRecordsResponse, error) {
	response, err := s.memoryEmbeddedRuntime.GetRecords(request)
	if err != nil {
		return nil, err
	}
	s.makeEmptyOnce.Do(func() {
		if len(response.Embeddings) > 0 {
			response.Embeddings[0] = nil
		}
	})
	return response, nil
}

func newEmbeddedClientForRuntime(t *testing.T, runtime localEmbeddedRuntime) *embeddedLocalClient {
	t.Helper()

	stateClient, err := NewHTTPClient()
	require.NoError(t, err)
	apiState, ok := stateClient.(*APIClientV2)
	require.True(t, ok)
	t.Cleanup(func() {
		_ = apiState.Close()
	})

	return &embeddedLocalClient{
		state:           apiState,
		embedded:        runtime,
		collectionState: map[string]*embeddedCollectionState{},
		logger:          loggerpkg.NewNoopLogger(),
	}
}

func newRealPersistentClientForTest(t *testing.T, persistPath string) *PersistentClient {
	t.Helper()

	opts := []PersistentClientOption{
		WithPersistentPath(persistPath),
		WithPersistentLibraryAutoDownload(true),
	}
	if libPath := strings.TrimSpace(os.Getenv("CHROMA_LIB_PATH")); libPath != "" {
		opts = append(opts, WithPersistentLibraryPath(libPath))
	}

	client, err := NewPersistentClient(opts...)
	if err != nil {
		t.Skipf("embedded local runtime unavailable in this environment: %v", err)
	}
	persistent, ok := client.(*PersistentClient)
	require.True(t, ok)
	return persistent
}

// countingCloseableEF is a test helper that records every Close() invocation
// on the wrapped EmbeddingFunction. It deliberately does NOT deduplicate Close
// via sync.Once so tests can detect double-close regressions -- in particular
// the PR #504 review item 2 scenario where a failed contentEF-promotion close
// could cause an outer defer to invoke Close() a second time on a C-backed EF.
// If you need idempotent Close in a test, wrap the EF with wrapEFCloseOnce
// instead so the production close-once wrapper is exercised.
type countingCloseableEF struct {
	embeddingspkg.EmbeddingFunction

	closeFn    func() error
	closeCount atomic.Int32
}

var _ io.Closer = (*countingCloseableEF)(nil)

func (e *countingCloseableEF) Close() error {
	e.closeCount.Add(1)
	if e.closeFn != nil {
		return e.closeFn()
	}
	return nil
}

func seedEmbeddedCollectionForTest(t *testing.T, runtime *memoryEmbeddedRuntime, name string, configuration *CollectionConfigurationImpl) string {
	t.Helper()

	var configurationMap map[string]any
	var err error
	if configuration != nil {
		configurationMap, err = marshalToMap(configuration)
		require.NoError(t, err)
	}

	runtime.mu.Lock()
	defer runtime.mu.Unlock()

	runtime.nextCollectionID++
	collectionID := fmt.Sprintf("seed-col-%d", runtime.nextCollectionID)
	key := collectionRuntimeKey(DefaultTenant, DefaultDatabase, name)
	runtime.collections[key] = localchroma.EmbeddedCollection{
		ID:                collectionID,
		Name:              name,
		Tenant:            DefaultTenant,
		Database:          DefaultDatabase,
		ConfigurationJSON: cloneMetadataMap(configurationMap),
	}
	runtime.collectionByID[collectionID] = key
	runtime.records[collectionID] = map[string]memoryEmbeddedRecord{}
	runtime.recordOrder[collectionID] = []string{}

	return collectionID
}

func (s *scriptedEmbeddedRuntime) Heartbeat() (uint64, error) {
	s.heartbeatCalls++
	if s.heartbeatErr != nil {
		return 0, s.heartbeatErr
	}
	return 1, nil
}

func (s *scriptedEmbeddedRuntime) Healthcheck() (*localchroma.EmbeddedHealthCheckResponse, error) {
	s.healthCalls++
	if s.healthErr != nil {
		return nil, s.healthErr
	}
	if len(s.healthResponses) == 0 {
		return &localchroma.EmbeddedHealthCheckResponse{
			IsExecutorReady:  true,
			IsLogClientReady: true,
		}, nil
	}
	idx := s.healthCalls - 1
	if idx >= len(s.healthResponses) {
		idx = len(s.healthResponses) - 1
	}
	return s.healthResponses[idx], nil
}

func (s *scriptedEmbeddedRuntime) Reset() error {
	return s.resetErr
}

func (s *scriptedEmbeddedRuntime) CreateTenant(localchroma.EmbeddedCreateTenantRequest) error {
	s.createTenantCalls++
	return s.createTenantErr
}

func (s *scriptedEmbeddedRuntime) GetTenant(request localchroma.EmbeddedGetTenantRequest) (*localchroma.EmbeddedTenant, error) {
	s.getTenantCalls.Add(1)
	if s.getTenantErr != nil {
		return nil, s.getTenantErr
	}
	tenantName := request.Name
	if tenantName == "" {
		tenantName = DefaultTenant
	}
	return &localchroma.EmbeddedTenant{Name: tenantName}, nil
}

func (s *scriptedEmbeddedRuntime) CreateDatabase(localchroma.EmbeddedCreateDatabaseRequest) error {
	s.createDatabaseCalls++
	return s.createDatabaseErr
}

func (s *scriptedEmbeddedRuntime) ListDatabases(request localchroma.EmbeddedListDatabasesRequest) ([]localchroma.EmbeddedDatabase, error) {
	s.listDatabasesCalls++
	if s.listDatabasesErr != nil {
		return nil, s.listDatabasesErr
	}
	tenantName := request.TenantID
	if tenantName == "" {
		tenantName = DefaultTenant
	}
	return []localchroma.EmbeddedDatabase{{Name: DefaultDatabase, Tenant: tenantName}}, nil
}

func (s *scriptedEmbeddedRuntime) GetDatabase(request localchroma.EmbeddedGetDatabaseRequest) (*localchroma.EmbeddedDatabase, error) {
	s.getDatabaseCalls.Add(1)
	if s.getDatabaseErr != nil {
		return nil, s.getDatabaseErr
	}
	tenantName := request.TenantID
	if tenantName == "" {
		tenantName = DefaultTenant
	}
	databaseName := request.Name
	if databaseName == "" {
		databaseName = DefaultDatabase
	}
	return &localchroma.EmbeddedDatabase{Name: databaseName, Tenant: tenantName}, nil
}

func (s *scriptedEmbeddedRuntime) DeleteDatabase(localchroma.EmbeddedDeleteDatabaseRequest) error {
	s.deleteDatabaseCalls++
	return s.deleteDatabaseErr
}

func TestWaitForLocalEmbeddedReady_DoesNotBypassHealthcheckReadiness(t *testing.T) {
	runtime := newScriptedEmbeddedRuntime()
	runtime.healthResponses = []*localchroma.EmbeddedHealthCheckResponse{
		{IsExecutorReady: false, IsLogClientReady: false},
		{IsExecutorReady: true, IsLogClientReady: true},
	}

	err := waitForLocalEmbeddedReady(runtime)
	require.NoError(t, err)
	require.Equal(t, 2, runtime.healthCalls)
	require.Equal(t, 0, runtime.heartbeatCalls)
}

func TestWaitForLocalEmbeddedReady_FallsBackToHeartbeatWhenHealthcheckFails(t *testing.T) {
	runtime := newScriptedEmbeddedRuntime()
	runtime.healthErr = errors.New("healthcheck unavailable")

	err := waitForLocalEmbeddedReady(runtime)
	require.NoError(t, err)
	require.GreaterOrEqual(t, runtime.heartbeatCalls, 1)
}

func TestEmbeddedLocalClient_ContextCancellationShortCircuits(t *testing.T) {
	lockLocalTestHooks(t)

	runtime := newScriptedEmbeddedRuntime()
	client := &embeddedLocalClient{
		state:    &APIClientV2{},
		embedded: runtime,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	origVersionFunc := localVersionWithErrorFunc
	localVersionWithErrorFunc = func() (string, error) {
		t.Fatal("version function must not be called when context is canceled")
		return "", nil
	}
	t.Cleanup(func() {
		localVersionWithErrorFunc = origVersionFunc
	})

	err := client.Heartbeat(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 0, runtime.heartbeatCalls)

	_, err = client.GetVersion(ctx)
	require.ErrorIs(t, err, context.Canceled)

	_, err = client.GetIdentity(ctx)
	require.ErrorIs(t, err, context.Canceled)

	err = client.Reset(ctx)
	require.ErrorIs(t, err, context.Canceled)

	collection := &embeddedCollection{client: client}
	err = collection.ModifyMetadata(ctx, NewMetadataFromMap(map[string]interface{}{"k": "v"}))
	require.ErrorIs(t, err, context.Canceled)

	_, err = client.CreateTenant(ctx, NewTenant(DefaultTenant))
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 0, runtime.createTenantCalls)

	_, err = client.GetTenant(ctx, NewTenant(DefaultTenant))
	require.ErrorIs(t, err, context.Canceled)
	require.EqualValues(t, 0, runtime.getTenantCalls.Load())

	testDB := NewTenant(DefaultTenant).Database("test_db")
	_, err = client.CreateDatabase(ctx, testDB)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 0, runtime.createDatabaseCalls)

	_, err = client.ListDatabases(ctx, NewTenant(DefaultTenant))
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 0, runtime.listDatabasesCalls)

	_, err = client.GetDatabase(ctx, testDB)
	require.ErrorIs(t, err, context.Canceled)
	require.EqualValues(t, 0, runtime.getDatabaseCalls.Load())

	err = client.DeleteDatabase(ctx, testDB)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 0, runtime.deleteDatabaseCalls)
}

func TestEmbeddedCollectionModifyMetadataUsesReplacementSemantics(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(
		ctx,
		"modify-metadata",
		WithCollectionMetadataCreate(NewMetadataFromMap(map[string]interface{}{
			"old": "value",
		})),
	)
	require.NoError(t, err)

	newMetadata := NewMetadataFromMap(map[string]interface{}{
		"new": "value",
	})
	err = collection.ModifyMetadata(ctx, newMetadata)
	require.NoError(t, err)

	immediateNew, ok := collection.Metadata().GetString("new")
	require.True(t, ok)
	require.Equal(t, "value", immediateNew)
	_, hasOld := collection.Metadata().GetString("old")
	require.False(t, hasOld)

	reloaded, err := client.GetCollection(ctx, "modify-metadata")
	require.NoError(t, err)
	reloadedNew, ok := reloaded.Metadata().GetString("new")
	require.True(t, ok)
	require.Equal(t, "value", reloadedNew)
	_, hasReloadedOld := reloaded.Metadata().GetString("old")
	require.False(t, hasReloadedOld)
}

func TestEmbeddedCollectionModifyMetadataRejectsNilValues(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(
		ctx,
		"modify-metadata-nil",
		WithCollectionMetadataCreate(NewMetadataFromMap(map[string]interface{}{
			"old": "value",
		})),
	)
	require.NoError(t, err)

	err = collection.ModifyMetadata(ctx, NewMetadata(RemoveAttribute("old")))
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be null")

	immediateOld, ok := collection.Metadata().GetString("old")
	require.True(t, ok)
	require.Equal(t, "value", immediateOld)

	reloaded, err := client.GetCollection(ctx, "modify-metadata-nil")
	require.NoError(t, err)
	reloadedOld, ok := reloaded.Metadata().GetString("old")
	require.True(t, ok)
	require.Equal(t, "value", reloadedOld)
}

func TestEmbeddedCollectionModifyMetadataRejectsNilArgument(t *testing.T) {
	runtime := newFailingUpdateEmbeddedRuntime(nil)
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(
		ctx,
		"modify-metadata-nil-arg",
		WithCollectionMetadataCreate(NewMetadataFromMap(map[string]interface{}{"old": "value"})),
	)
	require.NoError(t, err)

	err = collection.ModifyMetadata(ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "newMetadata cannot be nil")
	require.Equal(t, 0, runtime.UpdateCalls())
}

func TestEmbeddedCollectionModifyMetadataRejectsEmptyMetadata(t *testing.T) {
	runtime := newFailingUpdateEmbeddedRuntime(nil)
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(
		ctx,
		"modify-metadata-empty",
		WithCollectionMetadataCreate(NewMetadataFromMap(map[string]interface{}{"old": "value"})),
	)
	require.NoError(t, err)

	err = collection.ModifyMetadata(ctx, NewMetadata())
	require.Error(t, err)
	require.Contains(t, err.Error(), "newMetadata cannot be empty")
	require.Equal(t, 0, runtime.UpdateCalls())
}

func TestEmbeddedCollectionModifyMetadataUpdateFailureLeavesStateUntouched(t *testing.T) {
	runtime := newFailingUpdateEmbeddedRuntime(errors.New("update failed"))
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(
		ctx,
		"modify-metadata-update-failure",
		WithCollectionMetadataCreate(NewMetadataFromMap(map[string]interface{}{"old": "value"})),
	)
	require.NoError(t, err)

	err = collection.ModifyMetadata(ctx, NewMetadataFromMap(map[string]interface{}{"new": "value"}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "error modifying collection metadata")
	require.Equal(t, 1, runtime.UpdateCalls())

	_, hasNew := collection.Metadata().GetString("new")
	require.False(t, hasNew)
	immediateOld, ok := collection.Metadata().GetString("old")
	require.True(t, ok)
	require.Equal(t, "value", immediateOld)

	client.collectionStateMu.RLock()
	state := client.collectionState[collection.ID()]
	client.collectionStateMu.RUnlock()
	require.NotNil(t, state)
	_, stateHasNew := state.metadata.GetString("new")
	require.False(t, stateHasNew)
	stateOld, ok := state.metadata.GetString("old")
	require.True(t, ok)
	require.Equal(t, "value", stateOld)

	reloaded, err := client.GetCollection(ctx, "modify-metadata-update-failure")
	require.NoError(t, err)
	_, reloadedHasNew := reloaded.Metadata().GetString("new")
	require.False(t, reloadedHasNew)
	reloadedOld, ok := reloaded.Metadata().GetString("old")
	require.True(t, ok)
	require.Equal(t, "value", reloadedOld)
}

func TestEmbeddedCollectionDimensionReadsStateSnapshot(t *testing.T) {
	client := &embeddedLocalClient{
		state:           &APIClientV2{},
		embedded:        newScriptedEmbeddedRuntime(),
		collectionState: map[string]*embeddedCollectionState{},
	}
	collection := &embeddedCollection{
		id:        "c1",
		dimension: 3,
		client:    client,
	}

	require.Equal(t, 3, collection.Dimension())
	client.setCollectionDimension("c1", 9)
	require.Equal(t, 9, collection.Dimension())
}

func TestEmbeddedCollectionAdd_DoesNotOverrideKnownDimension(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(ctx, "dimension-guard-add")
	require.NoError(t, err)

	client.setCollectionDimension(collection.ID(), 42)

	err = collection.Add(
		ctx,
		WithIDs("a1"),
		WithEmbeddings(embeddingspkg.NewEmbeddingFromFloat32([]float32{1, 2, 3})),
		WithTexts("doc-a1"),
	)
	require.NoError(t, err)
	require.Equal(t, 42, collection.Dimension())
}

func TestEmbeddedCollectionUpsert_DoesNotOverrideKnownDimension(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(ctx, "dimension-guard-upsert")
	require.NoError(t, err)

	client.setCollectionDimension(collection.ID(), 42)

	err = collection.Upsert(
		ctx,
		WithIDs("u1"),
		WithEmbeddings(embeddingspkg.NewEmbeddingFromFloat32([]float32{1, 2, 3})),
		WithTexts("doc-u1"),
	)
	require.NoError(t, err)
	require.Equal(t, 42, collection.Dimension())
}

func TestEmbeddedCollectionUpdate_DoesNotOverrideKnownDimension(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(ctx, "dimension-guard-update")
	require.NoError(t, err)

	err = collection.Add(
		ctx,
		WithIDs("x1"),
		WithEmbeddings(embeddingspkg.NewEmbeddingFromFloat32([]float32{9, 9, 9})),
		WithTexts("doc-x1"),
	)
	require.NoError(t, err)

	client.setCollectionDimension(collection.ID(), 42)

	err = collection.Update(
		ctx,
		WithIDs("x1"),
		WithEmbeddings(embeddingspkg.NewEmbeddingFromFloat32([]float32{1, 2, 3, 4})),
		WithTexts("doc-x1-updated"),
	)
	require.NoError(t, err)
	require.Equal(t, 42, collection.Dimension())
}

func TestEmbeddingsToFloat32Matrix_RejectsNilOrEmptyEmbeddings(t *testing.T) {
	_, err := embeddingsToFloat32Matrix([]embeddingspkg.Embedding{nil})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be nil")

	empty := embeddingspkg.NewEmbeddingFromFloat32([]float32{})
	_, err = embeddingsToFloat32Matrix([]embeddingspkg.Embedding{empty})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be empty")

	valid := embeddingspkg.NewEmbeddingFromFloat32([]float32{1, 2, 3})
	matrix, err := embeddingsToFloat32Matrix([]embeddingspkg.Embedding{valid})
	require.NoError(t, err)
	require.Equal(t, [][]float32{{1, 2, 3}}, matrix)
}

func TestIntToUint32_ValidatesRange(t *testing.T) {
	value, err := intToUint32(42, "limit")
	require.NoError(t, err)
	require.Equal(t, uint32(42), value)

	_, err = intToUint32(-1, "limit")
	require.Error(t, err)
	require.Contains(t, err.Error(), "limit must be greater than or equal to 0")

	maxInt := int(^uint(0) >> 1)
	if uint64(maxInt) > uint64(math.MaxUint32) {
		_, err = intToUint32(maxInt, "offset")
		require.Error(t, err)
		require.Contains(t, err.Error(), "offset cannot exceed")
	}
}

func TestMarshalFilterToMap_ReturnsNilForTypedNilMarshaler(t *testing.T) {
	var typedNilWhere *WhereClauseString
	var typedNilWhereDocument *WhereDocumentClauseContainsOrNotContains

	testCases := []struct {
		name   string
		filter json.Marshaler
	}{
		{name: "where filter", filter: typedNilWhere},
		{name: "where document filter", filter: typedNilWhereDocument},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := marshalFilterToMap(tc.filter)
			require.NoError(t, err)
			require.Nil(t, result)
		})
	}
}

func TestEmbeddedCollectionModifyName_SerializesRenameAndCacheUpdate(t *testing.T) {
	runtime := newBlockingRenameEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(ctx, "rename-start")
	require.NoError(t, err)
	embeddedCollection, ok := collection.(*embeddedCollection)
	require.True(t, ok)

	firstDone := make(chan error, 1)
	go func() {
		firstDone <- embeddedCollection.ModifyName(ctx, "rename-first")
	}()
	<-runtime.firstUpdateStarted

	secondDone := make(chan error, 1)
	go func() {
		secondDone <- embeddedCollection.ModifyName(ctx, "rename-second")
	}()

	select {
	case err := <-secondDone:
		require.Failf(t, "second rename completed before first released", "err=%v", err)
	case <-time.After(50 * time.Millisecond):
		// Expected: second rename is blocked while the first rename holds collection lock.
	}

	close(runtime.unblockFirstUpdate)
	require.NoError(t, <-firstDone)
	require.NoError(t, <-secondDone)

	require.Nil(t, client.cachedCollectionByName("rename-start"))
	require.Nil(t, client.cachedCollectionByName("rename-first"))
	renamed := client.cachedCollectionByName("rename-second")
	require.NotNil(t, renamed)
	require.Equal(t, "rename-second", embeddedCollection.Name())
}

func TestEmbeddedCollectionModifyName_ReleasesLockWhenRuntimePanics(t *testing.T) {
	runtime := &panicRenameEmbeddedRuntime{memoryEmbeddedRuntime: newMemoryEmbeddedRuntime()}
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(ctx, "rename-panic")
	require.NoError(t, err)
	embeddedCollection, ok := collection.(*embeddedCollection)
	require.True(t, ok)

	require.PanicsWithValue(t, "panic in UpdateCollection", func() {
		_ = embeddedCollection.ModifyName(ctx, "rename-panic-new")
	})

	nameDone := make(chan string, 1)
	go func() {
		nameDone <- embeddedCollection.Name()
	}()
	select {
	case gotName := <-nameDone:
		require.Equal(t, "rename-panic", gotName)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("collection lock remained held after panic in ModifyName")
	}
}

func TestEmbeddedCollectionQuery_ReturnsErrorOnDistanceEmbeddingMismatch(t *testing.T) {
	runtime := &mismatchedQueryEmbeddedRuntime{memoryEmbeddedRuntime: newMemoryEmbeddedRuntime()}
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(ctx, "distance-mismatch")
	require.NoError(t, err)

	queryEmbedding := embeddingspkg.NewEmbeddingFromFloat32([]float32{1, 0, 0})
	err = collection.Add(
		ctx,
		WithIDs("d1"),
		WithEmbeddings(queryEmbedding),
		WithTexts("doc-1"),
	)
	require.NoError(t, err)

	_, err = collection.Query(
		ctx,
		WithQueryEmbeddings(queryEmbedding),
		WithNResults(1),
		WithInclude(IncludeDistances),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "query response returned")
}

func TestEmbeddedCollectionQuery_ReturnsErrorWhenProjectionRowsDisappear(t *testing.T) {
	runtime := &missingProjectionEmbeddedRuntime{memoryEmbeddedRuntime: newMemoryEmbeddedRuntime()}
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(ctx, "query-projection-toctou")
	require.NoError(t, err)

	emb1 := embeddingspkg.NewEmbeddingFromFloat32([]float32{1, 0, 0})
	emb2 := embeddingspkg.NewEmbeddingFromFloat32([]float32{0, 1, 0})
	err = collection.Add(
		ctx,
		WithIDs("q1", "q2"),
		WithEmbeddings(emb1, emb2),
		WithTexts("doc-1", "doc-2"),
	)
	require.NoError(t, err)

	_, err = collection.Query(
		ctx,
		WithQueryEmbeddings(emb1),
		WithNResults(2),
		WithInclude(IncludeDocuments),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "query projections changed during read")
}

func TestEmbeddedCollectionQuery_ReturnsErrorOnEmptyProjectionEmbedding(t *testing.T) {
	runtime := &emptyProjectionEmbeddingRuntime{memoryEmbeddedRuntime: newMemoryEmbeddedRuntime()}
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(ctx, "query-empty-projection-embedding")
	require.NoError(t, err)

	queryEmbedding := embeddingspkg.NewEmbeddingFromFloat32([]float32{1, 0, 0})
	err = collection.Add(
		ctx,
		WithIDs("q1"),
		WithEmbeddings(queryEmbedding),
		WithTexts("doc-1"),
	)
	require.NoError(t, err)

	_, err = collection.Query(
		ctx,
		WithQueryEmbeddings(queryEmbedding),
		WithNResults(1),
		WithInclude(IncludeDistances),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot compute distance for empty embedding vectors")
}

func TestEmbeddedCollectionQuery_RejectsNResultsOverflow(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("requires 64-bit int to exceed uint32 range")
	}

	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(ctx, "query-overflow")
	require.NoError(t, err)

	queryEmbedding := embeddingspkg.NewEmbeddingFromFloat32([]float32{1, 0, 0})
	tooLargeNResults := int(uint64(math.MaxUint32) + 1)
	_, err = collection.Query(
		ctx,
		WithQueryEmbeddings(queryEmbedding),
		WithNResults(tooLargeNResults),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nResults cannot exceed")
}

func TestEmbeddedCollectionDelete_PassesLimitAndWhereDocument(t *testing.T) {
	runtime := newRecordingDeleteEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(ctx, "delete-limit")
	require.NoError(t, err)

	err = collection.Delete(
		ctx,
		WithWhereDocument(Contains("draft")),
		WithLimit(2),
	)
	require.NoError(t, err)
	require.Equal(t, 1, runtime.deleteCalls)
	require.NotNil(t, runtime.lastDeleteRequest.Limit)
	require.Equal(t, uint32(2), *runtime.lastDeleteRequest.Limit)
	require.Equal(t, map[string]any{"$contains": "draft"}, runtime.lastDeleteRequest.WhereDocument)
}

func TestAPIClientV2LocalCollectionCacheHelpers_InitializeNilMap(t *testing.T) {
	client := &APIClientV2{}
	first := &embeddedCollection{name: "first"}
	second := &embeddedCollection{name: "second"}

	client.localAddCollectionToCache(first)
	require.Equal(t, first, client.localCollectionByName("first"))

	client.localRenameCollectionInCache("first", second)
	require.Nil(t, client.localCollectionByName("first"))
	require.Equal(t, second, client.localCollectionByName("second"))
}

func TestEmbeddedLocalClientCRUD_CollectionsLifecycle(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	created, err := client.CreateCollection(ctx, "books")
	require.NoError(t, err)
	require.Equal(t, "books", created.Name())
	require.NotEmpty(t, created.ID())

	retrieved, err := client.GetCollection(ctx, "books")
	require.NoError(t, err)
	require.Equal(t, created.ID(), retrieved.ID())

	listed, err := client.ListCollections(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, "books", listed[0].Name())

	count, err := client.CountCollections(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	require.NoError(t, client.DeleteCollection(ctx, "books"))

	count, err = client.CountCollections(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestEmbeddedLocalClientDeleteCollection_PropagatesStateCloseError(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	failingEF := &mockFailingCloseEF{closeErr: errors.New("delete cleanup failure")}
	_, err := client.CreateCollection(ctx, "delete-close-error", WithEmbeddingFunctionCreate(failingEF))
	require.NoError(t, err)

	err = client.DeleteCollection(ctx, "delete-close-error")
	require.Error(t, err)
	require.Contains(t, err.Error(), "delete cleanup failure")
	require.Equal(t, int32(1), failingEF.closeCount.Load(), "DeleteCollection must still physically close the EF once")
}

func TestEmbeddedLocalClientCreateCollection_PersistsMetadataConfigurationAndSchemaAcrossClients(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	writer := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	metadata := NewMetadataFromMap(map[string]interface{}{
		"owner":  "qa",
		"active": true,
		"count":  7,
		"score":  0.75,
	})
	configuration := NewCollectionConfiguration()
	configuration.SetRaw("hnsw", map[string]any{
		"space": "cosine",
	})
	schema, err := NewSchemaWithDefaults()
	require.NoError(t, err)

	created, err := writer.CreateCollection(
		ctx,
		"collection-with-config",
		WithCollectionMetadataCreate(metadata),
		WithConfigurationCreate(configuration),
		WithSchemaCreate(schema),
	)
	require.NoError(t, err)
	require.NotEmpty(t, created.ID())

	reader := newEmbeddedClientForRuntime(t, runtime)

	got, err := reader.GetCollection(ctx, "collection-with-config")
	require.NoError(t, err)
	require.Equal(t, created.ID(), got.ID())

	owner, ok := got.Metadata().GetString("owner")
	require.True(t, ok)
	require.Equal(t, "qa", owner)
	active, ok := got.Metadata().GetBool("active")
	require.True(t, ok)
	require.True(t, active)
	count, ok := got.Metadata().GetInt("count")
	require.True(t, ok)
	require.EqualValues(t, 7, count)
	score, ok := got.Metadata().GetFloat("score")
	require.True(t, ok)
	require.InDelta(t, 0.75, score, 1e-9)

	hnswRaw, ok := got.Configuration().GetRaw("hnsw")
	require.True(t, ok)
	hnswMap, ok := hnswRaw.(map[string]any)
	require.True(t, ok)
	space, ok := hnswMap["space"].(string)
	require.True(t, ok)
	require.Equal(t, "cosine", space)

	require.NotNil(t, got.Schema())

	listed, err := reader.ListCollections(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.Equal(t, created.ID(), listed[0].ID())

	listedOwner, ok := listed[0].Metadata().GetString("owner")
	require.True(t, ok)
	require.Equal(t, "qa", listedOwner)
	listedCount, ok := listed[0].Metadata().GetInt("count")
	require.True(t, ok)
	require.EqualValues(t, 7, listedCount)
}

func TestEmbeddedLocalClientGetCollectionFailsOnInvalidRuntimeMetadata(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	created, err := client.CreateCollection(
		ctx,
		"invalid-runtime-metadata",
		WithCollectionMetadataCreate(NewMetadataFromMap(map[string]interface{}{"owner": "qa"})),
	)
	require.NoError(t, err)

	runtime.mu.Lock()
	key := runtime.collectionByID[created.ID()]
	col := runtime.collections[key]
	col.Metadata = map[string]any{
		"invalid": map[string]any{"nested": "object"},
	}
	runtime.collections[key] = col
	runtime.mu.Unlock()

	_, err = client.GetCollection(ctx, "invalid-runtime-metadata")
	require.Error(t, err)
	require.Contains(t, err.Error(), "error parsing collection metadata")
}

func TestEmbeddedLocalClientListCollectionsFailsOnInvalidRuntimeSchema(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()
	schema, err := NewSchemaWithDefaults()
	require.NoError(t, err)

	created, err := client.CreateCollection(
		ctx,
		"invalid-runtime-schema",
		WithSchemaCreate(schema),
	)
	require.NoError(t, err)

	runtime.mu.Lock()
	key := runtime.collectionByID[created.ID()]
	col := runtime.collections[key]
	col.Schema = map[string]any{
		"keys": "invalid",
	}
	runtime.collections[key] = col
	runtime.mu.Unlock()

	_, err = client.ListCollections(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error parsing collection schema")
}

func TestEmbeddedLocalClientGetOrCreateCollection_ExistingWithoutEFPreservesLocalState(t *testing.T) {
	runtime := newCountingMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	initialEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	created, err := client.CreateCollection(ctx, "idempotent-existing", WithEmbeddingFunctionCreate(initialEF))
	require.NoError(t, err)

	got, err := client.GetOrCreateCollection(ctx, "idempotent-existing")
	require.NoError(t, err)
	require.Equal(t, created.ID(), got.ID())

	gotCollection, ok := got.(*embeddedCollection)
	require.True(t, ok)
	require.Same(t, initialEF, unwrapCloseOnceEF(gotCollection.embeddingFunctionSnapshot()))

	createCalls, getCalls := runtime.callCounts()
	require.Equal(t, 1, createCalls)
	require.Equal(t, 2, getCalls, "GetCollection revalidates the collection ID before publishing to cache")
}

func TestEmbeddedLocalClientGetOrCreateCollection_ExistingWithEFUpdatesLocalState(t *testing.T) {
	runtime := newCountingMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	initialEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	_, err := client.CreateCollection(ctx, "idempotent-existing-override", WithEmbeddingFunctionCreate(initialEF))
	require.NoError(t, err)

	overrideEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	got, err := client.GetOrCreateCollection(
		ctx,
		"idempotent-existing-override",
		WithEmbeddingFunctionCreate(overrideEF),
	)
	require.NoError(t, err)
	gotCollection, ok := got.(*embeddedCollection)
	require.True(t, ok)
	require.Same(t, overrideEF, unwrapCloseOnceEF(gotCollection.embeddingFunctionSnapshot()))

	again, err := client.GetCollection(ctx, "idempotent-existing-override")
	require.NoError(t, err)
	againEmbedded, ok := again.(*embeddedCollection)
	require.True(t, ok)
	require.Same(t, overrideEF, unwrapCloseOnceEF(againEmbedded.embeddingFunctionSnapshot()))

	createCalls, getCalls := runtime.callCounts()
	require.Equal(t, 1, createCalls)
	require.Equal(t, 4, getCalls, "each GetCollection call performs an ID revalidation round-trip")
}

func TestEmbeddedLocalClientGetOrCreateCollection_CreatesWhenMissing(t *testing.T) {
	runtime := newCountingMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	ef := embeddingspkg.NewConsistentHashEmbeddingFunction()
	got, err := client.GetOrCreateCollection(
		ctx,
		"idempotent-missing-create",
		WithEmbeddingFunctionCreate(ef),
	)
	require.NoError(t, err)
	require.NotEmpty(t, got.ID())
	require.Equal(t, "idempotent-missing-create", got.Name())

	gotCollection, ok := got.(*embeddedCollection)
	require.True(t, ok)
	require.Same(t, ef, unwrapCloseOnceEF(gotCollection.embeddingFunctionSnapshot()))

	createCalls, getCalls := runtime.callCounts()
	require.Equal(t, 1, createCalls)
	require.Equal(t, 2, getCalls)
}

func TestEmbeddedLocalClientGetOrCreateCollection_FallbackAfterProvisionalGetFailureKeepsCallerEFOpen(t *testing.T) {
	ctx := context.Background()

	type scenario struct {
		name    string
		run     func(t *testing.T, got *embeddedCollection)
		options func() ([]CreateCollectionOption, *atomic.Int32)
	}

	tests := []scenario{
		{
			name: "dense caller ef",
			options: func() ([]CreateCollectionOption, *atomic.Int32) {
				denseEF := &mockCloseableEF{}
				return []CreateCollectionOption{WithEmbeddingFunctionCreate(denseEF)}, &denseEF.closeCount
			},
			run: func(t *testing.T, got *embeddedCollection) {
				t.Helper()
				denseEF := unwrapCloseOnceEF(got.embeddingFunctionSnapshot())
				require.NotNil(t, denseEF, "dense EF must remain usable after fallback")
				_, err := denseEF.EmbedQuery(ctx, "hello")
				require.NoError(t, err)
				require.NotErrorIs(t, err, errEFClosed)
			},
		},
		{
			name: "content caller ef",
			options: func() ([]CreateCollectionOption, *atomic.Int32) {
				contentEF := &mockCloseableContentEF{}
				return []CreateCollectionOption{WithContentEmbeddingFunctionCreate(contentEF)}, &contentEF.closeCount
			},
			run: func(t *testing.T, got *embeddedCollection) {
				t.Helper()
				got.mu.RLock()
				contentEF := got.contentEmbeddingFunction
				got.mu.RUnlock()
				require.NotNil(t, contentEF, "content EF must remain usable after fallback")
				_, err := contentEF.EmbedContent(ctx, embeddingspkg.Content{})
				require.NoError(t, err)
				require.NotErrorIs(t, err, errEFClosed)
			},
		},
		{
			name: "dual-interface content ef",
			options: func() ([]CreateCollectionOption, *atomic.Int32) {
				dualEF := &mockDualEF{}
				return []CreateCollectionOption{WithContentEmbeddingFunctionCreate(dualEF)}, &dualEF.closeCount
			},
			run: func(t *testing.T, got *embeddedCollection) {
				t.Helper()
				denseEF := unwrapCloseOnceEF(got.embeddingFunctionSnapshot())
				require.NotNil(t, denseEF, "dense view must remain usable for dual-interface content EF")
				_, err := denseEF.EmbedQuery(ctx, "hello")
				require.NoError(t, err)
				require.NotErrorIs(t, err, errEFClosed)

				got.mu.RLock()
				contentEF := got.contentEmbeddingFunction
				got.mu.RUnlock()
				require.NotNil(t, contentEF, "content view must remain usable for dual-interface content EF")
				_, err = contentEF.EmbedContent(ctx, embeddingspkg.Content{})
				require.NoError(t, err)
				require.NotErrorIs(t, err, errEFClosed)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collectionName := "fallback-after-provisional-" + strings.ReplaceAll(tt.name, " ", "-")
			runtime := newFailingRevalidationGetMemoryEmbeddedRuntime(collectionName)
			seedEmbeddedCollectionForTest(t, runtime.memoryEmbeddedRuntime, collectionName, nil)

			client := newEmbeddedClientForRuntime(t, runtime)
			options, closeCount := tt.options()

			// Call-order contract for the revalidation failure helper:
			// 1. GetOrCreateCollection -> GetCollection triggers embedded.GetCollection call 1.
			// 2. The same GetCollection path triggers embedded.GetCollection call 2 for revalidation.
			// 3. After call 2 fails, GetOrCreateCollection falls back to CreateCollection(..., WithIfNotExistsCreate()),
			//    whose own preflight lookup / reuse logic becomes call 3+ and delegates to the real runtime.
			got, err := client.GetOrCreateCollection(ctx, collectionName, options...)
			require.NoError(t, err)
			require.NotEmpty(t, got.ID())

			gotEmbedded, ok := got.(*embeddedCollection)
			require.True(t, ok)
			tt.run(t, gotEmbedded)
			require.Equal(t, int32(0), closeCount.Load(), "caller-provided EF must stay open until collection Close")

			require.NoError(t, got.Close())
			require.Equal(t, int32(1), closeCount.Load(), "collection Close must own exactly one cleanup")
		})
	}
}

func TestEmbeddedGetCollection_RevalidationPreservesCallerOwnedEFsInState(t *testing.T) {
	ctx := context.Background()

	type scenario struct {
		name      string
		getOpts   func() ([]GetCollectionOption, *atomic.Int32)
		assertEFs func(t *testing.T, got *embeddedCollection)
	}

	tests := []scenario{
		{
			name: "dense caller ef",
			getOpts: func() ([]GetCollectionOption, *atomic.Int32) {
				denseEF := &mockCloseableEF{}
				return []GetCollectionOption{WithEmbeddingFunctionGet(denseEF)}, &denseEF.closeCount
			},
			assertEFs: func(t *testing.T, got *embeddedCollection) {
				t.Helper()
				require.NotNil(t, got.embeddingFunctionSnapshot())
			},
		},
		{
			name: "content caller ef",
			getOpts: func() ([]GetCollectionOption, *atomic.Int32) {
				contentEF := &mockCloseableContentEF{}
				return []GetCollectionOption{WithContentEmbeddingFunctionGet(contentEF)}, &contentEF.closeCount
			},
			assertEFs: func(t *testing.T, got *embeddedCollection) {
				t.Helper()
				got.mu.RLock()
				defer got.mu.RUnlock()
				require.NotNil(t, got.contentEmbeddingFunction)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtime := newMemoryEmbeddedRuntime()
			writer := newEmbeddedClientForRuntime(t, runtime)
			reader := newEmbeddedClientForRuntime(t, runtime)

			collectionName := "get-revalidation-preserves-" + strings.ReplaceAll(tt.name, " ", "-")
			_, err := writer.CreateCollection(ctx, collectionName,
				WithEmbeddingFunctionCreate(embeddingspkg.NewConsistentHashEmbeddingFunction()),
			)
			require.NoError(t, err)

			getOpts, closeCount := tt.getOpts()
			got, err := reader.GetCollection(ctx, collectionName, getOpts...)
			require.NoError(t, err)

			gotEmbedded, ok := got.(*embeddedCollection)
			require.True(t, ok)
			tt.assertEFs(t, gotEmbedded)

			require.NoError(t, reader.deleteCollectionState(got.ID()))
			require.Equal(t, int32(0), closeCount.Load(),
				"state cleanup must not close caller-provided EFs after successful revalidation")

			require.NoError(t, got.Close())
			require.Equal(t, int32(1), closeCount.Load(),
				"collection Close must retain ownership of the caller-provided EF wrapper")
		})
	}
}

func TestEmbeddedLocalClientGetOrCreateCollection_ConcurrentRaceReturnsUsableCollection(t *testing.T) {
	const collectionName = "concurrent-race-usable"

	runtime := newBlockingCreateAfterMissMemoryEmbeddedRuntime(collectionName)
	sharedClient := newEmbeddedClientForRuntime(t, runtime)
	sharedDualEF := &mockDualEF{}
	ctx := context.Background()

	type result struct {
		collection Collection
		err        error
	}

	runGetOrCreate := func(resultCh chan<- result) {
		collection, err := sharedClient.GetOrCreateCollection(
			ctx,
			collectionName,
			WithEmbeddingFunctionCreate(sharedDualEF),
			WithContentEmbeddingFunctionCreate(sharedDualEF),
		)
		resultCh <- result{collection: collection, err: err}
	}

	results := make(chan result, 2)
	go runGetOrCreate(results)

	select {
	case <-runtime.firstCreateReady:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for first create to block")
	}

	go runGetOrCreate(results)

	var first result
	select {
	case first = <-results:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for winning get-or-create result")
	}
	require.NoError(t, first.err)
	require.NotNil(t, first.collection)

	close(runtime.allowFirstCreate)

	var second result
	select {
	case second = <-results:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for losing get-or-create result")
	}
	require.NoError(t, second.err)
	require.NotNil(t, second.collection)

	require.NotEmpty(t, first.collection.ID())
	require.NotEmpty(t, second.collection.ID())
	require.Equal(t, first.collection.ID(), second.collection.ID())

	for _, got := range []Collection{first.collection, second.collection} {
		gotEmbedded, ok := got.(*embeddedCollection)
		require.True(t, ok)

		denseEF := unwrapCloseOnceEF(gotEmbedded.embeddingFunctionSnapshot())
		require.NotNil(t, denseEF, "dense EF must remain usable after concurrent get-or-create race")
		_, err := denseEF.EmbedQuery(ctx, "hello")
		require.NoError(t, err)
		require.NotErrorIs(t, err, errEFClosed)

		gotEmbedded.mu.RLock()
		contentEF := gotEmbedded.contentEmbeddingFunction
		gotEmbedded.mu.RUnlock()
		require.NotNil(t, contentEF, "content EF must remain usable after concurrent get-or-create race")
		_, err = contentEF.EmbedContent(ctx, embeddingspkg.Content{})
		require.NoError(t, err)
		require.NotErrorIs(t, err, errEFClosed)
	}

	require.NoError(t, first.collection.Close())
	require.NoError(t, second.collection.Close())
	require.Equal(t, int32(1), sharedDualEF.closeCount.Load(), "shared dual-interface EF must be closed exactly once")
}

func TestEmbeddedLocalClientCreateCollection_IfNotExistsExistingDoesNotOverrideState(t *testing.T) {
	runtime := newCountingMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	initialEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	initialMetadata := NewMetadataFromMap(map[string]interface{}{"source": "initial"})
	created, err := client.CreateCollection(
		ctx,
		"create-if-not-exists-idempotent",
		WithEmbeddingFunctionCreate(initialEF),
		WithCollectionMetadataCreate(initialMetadata),
	)
	require.NoError(t, err)

	overrideEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	overrideMetadata := NewMetadataFromMap(map[string]interface{}{"source": "override"})
	got, err := client.CreateCollection(
		ctx,
		"create-if-not-exists-idempotent",
		WithIfNotExistsCreate(),
		WithEmbeddingFunctionCreate(overrideEF),
		WithCollectionMetadataCreate(overrideMetadata),
	)
	require.NoError(t, err)
	require.Equal(t, created.ID(), got.ID())

	gotCollection, ok := got.(*embeddedCollection)
	require.True(t, ok)
	require.Same(t, initialEF, unwrapCloseOnceEF(gotCollection.embeddingFunctionSnapshot()))
	source, ok := gotCollection.Metadata().GetString("source")
	require.True(t, ok)
	require.Equal(t, "initial", source)

	createCalls, getCalls := runtime.callCounts()
	require.Equal(t, 2, createCalls)
	require.Equal(t, 1, getCalls)
}

func TestEmbeddedCreateCollection_DefaultORTExistingCollectionClosesTemporaryDefaultAndPreservesState(t *testing.T) {
	runtime := newCountingMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	initialEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	initialMetadata := NewMetadataFromMap(map[string]interface{}{"source": "initial"})
	created, err := client.CreateCollection(
		ctx,
		"default-ort-existing-close",
		WithEmbeddingFunctionCreate(initialEF),
		WithCollectionMetadataCreate(initialMetadata),
	)
	require.NoError(t, err)

	temporaryDefaultEF := &mockCloseableEF{}
	overrideMetadata := NewMetadataFromMap(map[string]interface{}{"source": "override"})
	got, err := client.CreateCollection(
		ctx,
		"default-ort-existing-close",
		WithIfNotExistsCreate(),
		WithCollectionMetadataCreate(overrideMetadata),
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return temporaryDefaultEF, func() error { return nil }, nil
		}),
	)
	require.NoError(t, err)
	require.Equal(t, created.ID(), got.ID())

	gotCollection, ok := got.(*embeddedCollection)
	require.True(t, ok)
	require.Same(t, initialEF, unwrapCloseOnceEF(gotCollection.embeddingFunctionSnapshot()))
	require.Equal(t, int32(1), temporaryDefaultEF.closeCount.Load())

	source, ok := gotCollection.Metadata().GetString("source")
	require.True(t, ok)
	require.Equal(t, "initial", source)
}

func TestEmbeddedCreateCollection_DefaultORTExistingCollectionProbeMissStillClosesTemporaryDefault(t *testing.T) {
	runtime := newMissingGetCollectionOnceRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	initialEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	initialMetadata := NewMetadataFromMap(map[string]interface{}{"source": "initial"})
	created, err := client.CreateCollection(
		ctx,
		"default-ort-existing-missed-probe",
		WithEmbeddingFunctionCreate(initialEF),
		WithCollectionMetadataCreate(initialMetadata),
	)
	require.NoError(t, err)

	temporaryDefaultEF := &mockCloseableEF{}
	overrideMetadata := NewMetadataFromMap(map[string]interface{}{"source": "override"})
	got, err := client.CreateCollection(
		ctx,
		"default-ort-existing-missed-probe",
		WithIfNotExistsCreate(),
		WithCollectionMetadataCreate(overrideMetadata),
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return temporaryDefaultEF, func() error { return nil }, nil
		}),
	)
	require.NoError(t, err)
	require.Equal(t, created.ID(), got.ID())

	gotCollection, ok := got.(*embeddedCollection)
	require.True(t, ok)
	require.Same(t, initialEF, unwrapCloseOnceEF(gotCollection.embeddingFunctionSnapshot()))
	require.Equal(t, int32(1), temporaryDefaultEF.closeCount.Load())

	source, ok := gotCollection.Metadata().GetString("source")
	require.True(t, ok)
	require.Equal(t, "initial", source)
}

func TestEmbeddedCreateCollection_DefaultORTReloadForReuseForwardsCallerContentEF(t *testing.T) {
	runtime := newMissingGetCollectionOnceRuntime()
	writer := newEmbeddedClientForRuntime(t, runtime)
	reader := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	initialEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	created, err := writer.CreateCollection(
		ctx,
		"default-ort-reload-content-forward",
		WithEmbeddingFunctionCreate(initialEF),
		WithCollectionMetadataCreate(NewMetadataFromMap(map[string]interface{}{"source": "initial"})),
	)
	require.NoError(t, err)

	contentEF := &mockCloseableContentEF{}
	temporaryDefaultEF := &mockCloseableEF{}
	got, err := reader.CreateCollection(
		ctx,
		"default-ort-reload-content-forward",
		WithIfNotExistsCreate(),
		WithCollectionMetadataCreate(NewMetadataFromMap(map[string]interface{}{"source": "override"})),
		WithContentEmbeddingFunctionCreate(contentEF),
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return temporaryDefaultEF, func() error { return nil }, nil
		}),
	)
	require.NoError(t, err)
	require.Equal(t, created.ID(), got.ID())
	require.Equal(t, int32(1), temporaryDefaultEF.closeCount.Load(),
		"temporary default EF must still be cleaned up on existing-collection reuse")

	gotEmbedded, ok := got.(*embeddedCollection)
	require.True(t, ok)
	gotEmbedded.mu.RLock()
	gotContentEF := gotEmbedded.contentEmbeddingFunction
	gotEmbedded.mu.RUnlock()
	require.NotNil(t, gotContentEF, "reload-for-reuse must forward the caller-provided content EF")
	require.Same(t, contentEF, unwrapCloseOnceContentEF(gotContentEF))
	require.Equal(t, int32(0), contentEF.closeCount.Load())

	require.NoError(t, got.Close())
	require.Equal(t, int32(1), contentEF.closeCount.Load(),
		"collection Close must own the forwarded content EF wrapper")
}

// TestEmbeddedCreateCollection_DefaultORTExistingCollectionLogsCleanupErrorAndReturnsCollection
// verifies that when cleaning up the temporary SDK-owned default embedding
// function fails after an existing-collection reuse, the user still receives
// the built collection (since it wraps state EFs unaffected by the cleanup
// failure) and the failure is surfaced via the logger. Replaces the Phase 23
// "synchronous error" contract, which incorrectly discarded a valid handle.
func TestEmbeddedCreateCollection_DefaultORTExistingCollectionLogsCleanupErrorAndReturnsCollection(t *testing.T) {
	runtime := newCountingMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	log := &capturingLogger{}
	client.logger = log
	ctx := context.Background()

	initialEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	created, err := client.CreateCollection(
		ctx,
		"default-ort-existing-close-error",
		WithEmbeddingFunctionCreate(initialEF),
	)
	require.NoError(t, err)

	temporaryDefaultEF := &mockFailingCloseEF{closeErr: errors.New("close boom")}
	got, err := client.CreateCollection(
		ctx,
		"default-ort-existing-close-error",
		WithIfNotExistsCreate(),
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return temporaryDefaultEF, func() error { return nil }, nil
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, created.ID(), got.ID())
	require.Equal(t, int32(1), temporaryDefaultEF.closeCount.Load())

	log.mu.Lock()
	defer log.mu.Unlock()
	require.GreaterOrEqual(t, log.errorCount, 1,
		"cleanup failure must be surfaced to the logger when the collection is still returned")
	require.Contains(t, log.lastMsg, "close",
		"logged message should describe the cleanup failure")
}

func TestEmbeddedCreateCollection_DefaultORTNewCollectionDoesNotCloseTemporaryDefault(t *testing.T) {
	runtime := newCountingMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	temporaryDefaultEF := &mockCloseableEF{}
	got, err := client.CreateCollection(
		ctx,
		"default-ort-new-collection",
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return temporaryDefaultEF, func() error { return nil }, nil
		}),
	)
	require.NoError(t, err)
	require.Equal(t, int32(0), temporaryDefaultEF.closeCount.Load())

	gotCollection, ok := got.(*embeddedCollection)
	require.True(t, ok)
	require.Same(t, temporaryDefaultEF, unwrapCloseOnceEF(gotCollection.embeddingFunctionSnapshot()))

	require.NoError(t, got.Close())
	require.Equal(t, int32(1), temporaryDefaultEF.closeCount.Load())
}

func TestEmbeddedCreateCollection_DefaultORTExistingCollectionOnFreshClientUsesStoredEFWhenProbeMisses(t *testing.T) {
	runtime := newMissingGetCollectionOnceRuntime()
	writer := newEmbeddedClientForRuntime(t, runtime)
	reader := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	initialEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	created, err := writer.CreateCollection(
		ctx,
		"default-ort-existing-fresh-client-probe-miss",
		WithEmbeddingFunctionCreate(initialEF),
	)
	require.NoError(t, err)

	temporaryDefaultEF := &mockCloseableEF{}
	got, err := reader.CreateCollection(
		ctx,
		"default-ort-existing-fresh-client-probe-miss",
		WithIfNotExistsCreate(),
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return temporaryDefaultEF, func() error { return nil }, nil
		}),
	)
	require.NoError(t, err)
	require.Equal(t, created.ID(), got.ID())
	require.Equal(t, int32(1), temporaryDefaultEF.closeCount.Load(),
		"temporary default EF must be closed when CreateCollection reused an existing collection")

	gotCollection, ok := got.(*embeddedCollection)
	require.True(t, ok)
	gotDenseEF := unwrapCloseOnceEF(gotCollection.embeddingFunctionSnapshot())
	require.NotNil(t, gotDenseEF)
	require.Equal(t, initialEF.Name(), gotDenseEF.Name())
	require.NotSame(t, temporaryDefaultEF, gotDenseEF)

	again, err := reader.GetCollection(ctx, "default-ort-existing-fresh-client-probe-miss")
	require.NoError(t, err)
	againEmbedded, ok := again.(*embeddedCollection)
	require.True(t, ok)
	require.Equal(t, initialEF.Name(), unwrapCloseOnceEF(againEmbedded.embeddingFunctionSnapshot()).Name())
	require.NotSame(t, temporaryDefaultEF, unwrapCloseOnceEF(againEmbedded.embeddingFunctionSnapshot()))
}

func TestEmbeddedCreateCollection_DefaultORTExistingCollectionOnFreshClientUsesStoredEFWhenProbeMisses_AfterJSONRoundTrip(t *testing.T) {
	runtime := newJSONRoundTripMissingGetCollectionOnceRuntime()
	writer := newEmbeddedClientForRuntime(t, runtime)
	reader := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	initialEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	created, err := writer.CreateCollection(
		ctx,
		"default-ort-existing-fresh-client-probe-miss-json-roundtrip",
		WithEmbeddingFunctionCreate(initialEF),
	)
	require.NoError(t, err)

	temporaryDefaultEF := &mockCloseableEF{}
	got, err := reader.CreateCollection(
		ctx,
		"default-ort-existing-fresh-client-probe-miss-json-roundtrip",
		WithIfNotExistsCreate(),
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return temporaryDefaultEF, func() error { return nil }, nil
		}),
	)
	require.NoError(t, err)
	require.Equal(t, created.ID(), got.ID())
	require.Equal(t, int32(1), temporaryDefaultEF.closeCount.Load())

	gotCollection, ok := got.(*embeddedCollection)
	require.True(t, ok)
	require.Equal(t, initialEF.Name(), unwrapCloseOnceEF(gotCollection.embeddingFunctionSnapshot()).Name())
	require.NotSame(t, temporaryDefaultEF, unwrapCloseOnceEF(gotCollection.embeddingFunctionSnapshot()))
}

// TestEmbeddedCreateCollection_DefaultORTExistingEmptyCollectionOnFreshClientDoesNotPromoteTemporaryDefaultWhenPreflightErrors
// pins the item 3 race from PR #504 review: when a pre-existing collection has
// all-empty metadata/config/schema (e.g. created via a path that skipped EF
// config persistence) AND the reader is a fresh client with no cached state AND
// the preflight GetCollection returned a transient (non-not-found) error, the
// JSON-equality heuristic in collectionModelMatchesCreateRequest can't detect
// reuse. The current behaviour installs the reader's throwaway default EF as
// persistent state; the fix routes this case through the reload branch so the
// throwaway is closed instead.
func TestEmbeddedCreateCollection_DefaultORTExistingEmptyCollectionOnFreshClientDoesNotPromoteTemporaryDefaultWhenPreflightErrors(t *testing.T) {
	runtime := newErrorGetCollectionOnceRuntime(errors.New("transient backend error"))
	writer := newEmbeddedClientForRuntime(t, runtime)
	reader := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	const name = "default-ort-existing-empty-preflight-error"

	// Create a persisted collection with empty metadata/config/schema by using
	// WithDisableEFConfigStorage on the writer and omitting metadata/schema.
	_, err := writer.CreateCollection(
		ctx,
		name,
		WithDisableEFConfigStorage(),
	)
	require.NoError(t, err)

	temporaryDefaultEF := &mockCloseableEF{}
	got, err := reader.CreateCollection(
		ctx,
		name,
		WithIfNotExistsCreate(),
		WithDisableEFConfigStorage(),
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return temporaryDefaultEF, func() error { return nil }, nil
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, got)

	// PR #504 review item 3 fix: when a fresh client lands in the all-empty
	// ambiguous branch with a transient preflight error, we cannot tell from
	// the runtime whether the collection was just created or was a pre-existing
	// empty one. In both cases there is no server-side EF config to reload, so
	// the reader's temporary default EF is the best (and only) fallback. Install
	// it as state rather than closing it -- a usable collection beats a
	// semantically-"pure" nil-EF collection that fails on the first Add/Query.
	require.Equal(t, int32(0), temporaryDefaultEF.closeCount.Load(),
		"temporary default EF must remain alive so the reader's collection has a usable EF")

	gotCollection, ok := got.(*embeddedCollection)
	require.True(t, ok)
	require.Same(t, temporaryDefaultEF, unwrapCloseOnceEF(gotCollection.embeddingFunctionSnapshot()),
		"returned collection must wrap the reader's temporary default EF as fallback")
}

// TestEmbeddedCreateCollection_DefaultORTFreshClientTransientPreflightOnNewEmptyInstallsTemporaryDefault
// pins the PR #504 review item 3 fix for the genuinely-new path: a fresh
// client creates a brand-new empty collection via CreateIfNotExists +
// DisableEFConfigStorage, and the preflight GetCollection returns a transient
// (non-not-found) error. Previously the all-empty reload branch wrongly fired
// and closed the temp EF, handing the caller a nil-EF collection. The fix
// ensures the temp EF is installed as state so Add/Query work.
func TestEmbeddedCreateCollection_DefaultORTFreshClientTransientPreflightOnNewEmptyInstallsTemporaryDefault(t *testing.T) {
	runtime := newErrorGetCollectionOnceRuntime(errors.New("transient backend error"))
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	temporaryDefaultEF := &mockCloseableEF{}
	got, err := client.CreateCollection(
		ctx,
		"fresh-new-empty-preflight-error",
		WithIfNotExistsCreate(),
		WithDisableEFConfigStorage(),
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return temporaryDefaultEF, func() error { return nil }, nil
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.Equal(t, int32(0), temporaryDefaultEF.closeCount.Load(),
		"temporary default EF must remain alive on genuinely-new empty collection even after transient preflight error")

	gotCollection, ok := got.(*embeddedCollection)
	require.True(t, ok)
	require.Same(t, temporaryDefaultEF, unwrapCloseOnceEF(gotCollection.embeddingFunctionSnapshot()),
		"returned collection must wrap the temporary default EF as fallback")
}

// TestEmbeddedCreateCollection_DefaultORTNewEmptyCollectionWithIfNotExistsStillInstallsTemporaryDefault
// is a regression guard for the item 3 fix: when the preflight probe
// conclusively reports not-found for a genuinely-new collection (no transient
// error), the reader's throwaway default EF must still be promoted to state
// so that subsequent Add/Query calls on the fresh collection have a usable EF.
// This is the "legitimately new, all empty, CreateIfNotExists" path that the
// narrow reload widening must NOT touch.
func TestEmbeddedCreateCollection_DefaultORTNewEmptyCollectionWithIfNotExistsStillInstallsTemporaryDefault(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	temporaryDefaultEF := &mockCloseableEF{}
	got, err := client.CreateCollection(
		ctx,
		"default-ort-new-empty-if-not-exists",
		WithIfNotExistsCreate(),
		WithDisableEFConfigStorage(),
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return temporaryDefaultEF, func() error { return nil }, nil
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, int32(0), temporaryDefaultEF.closeCount.Load(),
		"brand-new collection must keep its temporary default EF alive as state")

	gotCollection, ok := got.(*embeddedCollection)
	require.True(t, ok)
	require.Same(t, temporaryDefaultEF, unwrapCloseOnceEF(gotCollection.embeddingFunctionSnapshot()))
}

func TestEmbeddedCreateCollection_DefaultORTReplacementCollectionKeepsTemporaryDefaultWhenProbeIsStale(t *testing.T) {
	runtime := newStaleGetCollectionDeleteRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	original, err := client.CreateCollection(ctx, "default-ort-recreated-after-stale-probe")
	require.NoError(t, err)

	temporaryDefaultEF := &mockCloseableEF{}
	got, err := client.CreateCollection(
		ctx,
		"default-ort-recreated-after-stale-probe",
		WithIfNotExistsCreate(),
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return temporaryDefaultEF, func() error { return nil }, nil
		}),
	)
	require.NoError(t, err)
	require.NotEqual(t, original.ID(), got.ID())

	gotCollection, ok := got.(*embeddedCollection)
	require.True(t, ok)
	require.Same(t, temporaryDefaultEF, unwrapCloseOnceEF(gotCollection.embeddingFunctionSnapshot()))
	require.Equal(t, int32(0), temporaryDefaultEF.closeCount.Load())

	require.NoError(t, got.Close())
	require.Equal(t, int32(1), temporaryDefaultEF.closeCount.Load())
}

func TestEmbeddedCreateCollection_DefaultORTCreateFailureClosesTemporaryDefault(t *testing.T) {
	runtime := &failingCreateCollectionRuntime{
		stubEmbeddedRuntime: &stubEmbeddedRuntime{},
		createErr:           errors.New("create boom"),
	}
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	temporaryDefaultEF := &mockCloseableEF{}
	_, err := client.CreateCollection(
		ctx,
		"default-ort-create-failure",
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return temporaryDefaultEF, func() error { return nil }, nil
		}),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error creating collection")
	require.Equal(t, int32(1), temporaryDefaultEF.closeCount.Load())
}

func TestEmbeddedCreateCollection_DefaultORTCreateFailureJoinsCleanupError(t *testing.T) {
	runtime := &failingCreateCollectionRuntime{
		stubEmbeddedRuntime: &stubEmbeddedRuntime{},
		createErr:           errors.New("create boom"),
	}
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	temporaryDefaultEF := &mockFailingCloseEF{closeErr: errors.New("close boom")}
	_, err := client.CreateCollection(
		ctx,
		"default-ort-create-failure-cleanup-join",
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return temporaryDefaultEF, func() error { return nil }, nil
		}),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error creating collection")
	require.Contains(t, err.Error(), "error cleaning up default embedding function during collection create")
	require.Equal(t, int32(1), temporaryDefaultEF.closeCount.Load())
}

func TestEmbeddedCreateCollection_DefaultORTCanceledContextClosesTemporaryDefault(t *testing.T) {
	runtime := newCountingMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	temporaryDefaultEF := &mockCloseableEF{}
	_, err := client.CreateCollection(
		ctx,
		"default-ort-canceled-context",
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return temporaryDefaultEF, func() error { return nil }, nil
		}),
	)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, int32(1), temporaryDefaultEF.closeCount.Load())
}

func TestEmbeddedCreateCollection_DefaultORTExistingCollectionWithNonClosableDefaultLogsAndReturnsCollection(t *testing.T) {
	runtime := newCountingMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	log := &capturingLogger{}
	client.logger = log
	ctx := context.Background()

	created, err := client.CreateCollection(
		ctx,
		"default-ort-existing-non-closable",
		WithEmbeddingFunctionCreate(embeddingspkg.NewConsistentHashEmbeddingFunction()),
	)
	require.NoError(t, err)

	got, err := client.CreateCollection(
		ctx,
		"default-ort-existing-non-closable",
		WithIfNotExistsCreate(),
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return &mockNonCloseableEF{}, func() error { return nil }, nil
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, created.ID(), got.ID())

	log.mu.Lock()
	defer log.mu.Unlock()
	require.GreaterOrEqual(t, log.errorCount, 1)
}

func TestEmbeddedCreateCollection_BuildFailureDeletesRuntimeCollection(t *testing.T) {
	runtime := newInvalidCreateResponseDeleteTrackingRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	_, err := client.CreateCollection(
		ctx,
		"invalid-create-response",
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return &mockCloseableEF{}, func() error { return nil }, nil
		}),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error parsing collection metadata")

	require.Len(t, runtime.deleteCalls, 1)
	require.Equal(t, "invalid-create-response", runtime.deleteCalls[0].Name)

	_, getErr := runtime.GetCollection(localchroma.EmbeddedGetCollectionRequest{
		Name:         "invalid-create-response",
		TenantID:     DefaultTenant,
		DatabaseName: DefaultDatabase,
	})
	require.Error(t, getErr)
}

func TestEmbeddedCreateCollection_BuildFailureSuppressesDeleteCollectionNotFound(t *testing.T) {
	runtime := newInvalidCreateResponseMissingDeleteRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	_, err := client.CreateCollection(
		ctx,
		"invalid-create-response-delete-missing",
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			return &mockCloseableEF{}, func() error { return nil }, nil
		}),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error building collection")
	require.NotContains(t, err.Error(), "error deleting collection after build failure")
	require.Len(t, runtime.deleteCalls, 1)
}

func TestEmbeddedCreateCollection_BuildFailureJoinsDeleteAndCleanupErrors(t *testing.T) {
	runtime := newInvalidCreateResponseDeleteErrorRuntime(errors.New("delete boom"))
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	_, err := client.CreateCollection(ctx, "invalid-create-response-delete-error", WithEmbeddingFunctionCreate(&mockFailingCloseEF{
		closeErr: errors.New("cleanup boom"),
	}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "error building collection")
	require.Contains(t, err.Error(), "error deleting collection after build failure: delete boom")
	require.Contains(t, err.Error(), "cleanup boom")
	require.Len(t, runtime.deleteCalls, 1)
}

// TestEmbeddedCreateCollection_BuildFailureDoesNotDeletePreExistingCollection
// reproduces the dangerous scenario where a failing post-create build could
// destroy a pre-existing user collection: preflight GetCollection fails
// transiently (non-not-found), runtime CreateCollection(GetOrCreate:true)
// returns the pre-existing collection, and buildEmbeddedCollection then fails.
// The delete that used to fire in that path has been gated on positive
// evidence that this call created the collection.
func TestEmbeddedCreateCollection_BuildFailureDoesNotDeletePreExistingCollection(t *testing.T) {
	runtime := newPreExistingInvalidCreateRuntime(errors.New("transient preflight failure"))
	_, preseedErr := runtime.memoryEmbeddedRuntime.CreateCollection(localchroma.EmbeddedCreateCollectionRequest{
		Name:         "preexisting",
		TenantID:     DefaultTenant,
		DatabaseName: DefaultDatabase,
	})
	require.NoError(t, preseedErr)

	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	_, err := client.CreateCollection(
		ctx,
		"preexisting",
		WithIfNotExistsCreate(),
		WithEmbeddingFunctionCreate(embeddingspkg.NewConsistentHashEmbeddingFunction()),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "error parsing collection metadata")

	require.Empty(t, runtime.deleteCalls,
		"pre-existing user collection must not be deleted on build failure when preflight did not conclusively confirm absence")

	stillThere, getErr := runtime.memoryEmbeddedRuntime.GetCollection(localchroma.EmbeddedGetCollectionRequest{
		Name:         "preexisting",
		TenantID:     DefaultTenant,
		DatabaseName: DefaultDatabase,
	})
	require.NoError(t, getErr)
	require.NotNil(t, stillThere)
}

func TestComparableCollectionMap_NormalizesJSONNumberAndFloat64(t *testing.T) {
	left := map[string]any{
		"hnsw": map[string]any{
			"construction_ef": float64(200),
			"m":               float64(16),
		},
	}
	right := map[string]any{
		"hnsw": map[string]any{
			"construction_ef": json.Number("200"),
			"m":               json.Number("16"),
		},
	}

	require.True(t, comparableCollectionMap(left, right))
	require.True(t, comparableCollectionMap(right, left))
}

func TestComparableCollectionMap_EmptyAndNilMatch(t *testing.T) {
	require.True(t, comparableCollectionMap(nil, map[string]any{}))
	require.True(t, comparableCollectionMap(map[string]any{}, nil))
}

func TestRoundTripEmbeddedCollectionModel_PreservesJSONNumbers(t *testing.T) {
	model := &localchroma.EmbeddedCollection{
		ID:   "roundtrip-json-number",
		Name: "roundtrip-json-number",
		ConfigurationJSON: map[string]any{
			"hnsw": map[string]any{
				"construction_ef": float64(200),
			},
		},
	}

	roundTripped, err := roundTripEmbeddedCollectionModel(model)
	require.NoError(t, err)

	hnsw, ok := roundTripped.ConfigurationJSON["hnsw"].(map[string]any)
	require.True(t, ok)
	require.IsType(t, json.Number("200"), hnsw["construction_ef"])
}

func TestIsEmbeddedCollectionNotFoundError(t *testing.T) {
	require.True(t, isEmbeddedCollectionNotFoundError(errors.New("collection not found")))
	require.True(t, isEmbeddedCollectionNotFoundError(errors.New("Collection NOT FOUND in runtime")))
	require.True(t, isEmbeddedCollectionNotFoundError(errors.New("Collection [missing] does not exist")))
	require.False(t, isEmbeddedCollectionNotFoundError(errors.New("embedded runtime unavailable")))
	require.False(t, isEmbeddedCollectionNotFoundError(nil))
}

// TestIsEmbeddedCollectionNotFoundError_SentinelPath pins the PR #504 review
// item 5 defensive improvement: classification is keyed on errors.Is against
// ErrEmbeddedCollectionNotFound so future upstream message rewording cannot
// silently flip the classifier. Substring matching remains as a fallback for
// localchroma versions that have not yet been wrapped with the sentinel.
func TestIsEmbeddedCollectionNotFoundError_SentinelPath(t *testing.T) {
	t.Run("direct sentinel is classified", func(t *testing.T) {
		require.True(t, isEmbeddedCollectionNotFoundError(ErrEmbeddedCollectionNotFound))
	})

	t.Run("wrapped sentinel is classified via errors.Is", func(t *testing.T) {
		wrapped := fmt.Errorf("failed to fetch: %w", ErrEmbeddedCollectionNotFound)
		require.True(t, isEmbeddedCollectionNotFoundError(wrapped))
	})

	t.Run("sentinel with message that would not match substring is still classified", func(t *testing.T) {
		// Simulate upstream rewording the message to something like "missing
		// target" -- substring match would not catch it, but the sentinel does.
		reworded := fmt.Errorf("%w: missing target entity", ErrEmbeddedCollectionNotFound)
		require.True(t, isEmbeddedCollectionNotFoundError(reworded))
	})

	t.Run("unrelated error is not classified", func(t *testing.T) {
		require.False(t, isEmbeddedCollectionNotFoundError(errors.New("permission denied")))
	})
}

func TestIsEmbeddedCollectionNotFoundError_RealEmbeddedRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real embedded runtime test in short mode")
	}

	client := newRealPersistentClientForTest(t, t.TempDir())
	defer func() {
		_ = client.Close()
	}()

	embeddedClient, ok := client.Client.(*embeddedLocalClient)
	require.True(t, ok)
	embedded := embeddedClient.embedded
	databaseName := fmt.Sprintf("not-found-db-%d", time.Now().UnixNano())
	require.NoError(t, embedded.CreateDatabase(localchroma.EmbeddedCreateDatabaseRequest{
		Name: databaseName,
	}))

	_, getErr := embedded.GetCollection(localchroma.EmbeddedGetCollectionRequest{
		Name:         "missing-collection",
		TenantID:     DefaultTenant,
		DatabaseName: databaseName,
	})
	require.Error(t, getErr)
	require.True(t, isEmbeddedCollectionNotFoundError(getErr))

	deleteErr := embedded.DeleteCollection(localchroma.EmbeddedDeleteCollectionRequest{
		Name:         "missing-collection",
		TenantID:     DefaultTenant,
		DatabaseName: databaseName,
	})
	require.Error(t, deleteErr)
	require.True(t, isEmbeddedCollectionNotFoundError(deleteErr))
}

func TestEmbeddedCollectionCRUD_AddUpsertQueryDelete(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(ctx, "records")
	require.NoError(t, err)

	emb1 := embeddingspkg.NewEmbeddingFromFloat32([]float32{0.1, 0.2, 0.3})
	emb2 := embeddingspkg.NewEmbeddingFromFloat32([]float32{0.4, 0.5, 0.6})
	meta1 := NewDocumentMetadata(NewStringAttribute("kind", "alpha"))
	meta2 := NewDocumentMetadata(NewStringAttribute("kind", "beta"))

	err = collection.Add(ctx,
		WithIDs("r1", "r2"),
		WithEmbeddings(emb1, emb2),
		WithTexts("doc-1", "doc-2"),
		WithMetadatas(meta1, meta2),
	)
	require.NoError(t, err)

	recordCount, err := collection.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, recordCount)

	getResult, err := collection.Get(ctx, WithIDs("r1", "r2"), WithInclude(IncludeDocuments, IncludeMetadatas, IncludeEmbeddings))
	require.NoError(t, err)
	require.Equal(t, 2, getResult.Count())
	require.Equal(t, DocumentIDs{"r1", "r2"}, getResult.GetIDs())
	require.Equal(t, "doc-1", getResult.GetDocuments()[0].ContentString())
	require.Equal(t, "doc-2", getResult.GetDocuments()[1].ContentString())

	emb2Updated := embeddingspkg.NewEmbeddingFromFloat32([]float32{1.1, 1.2, 1.3})
	emb3 := embeddingspkg.NewEmbeddingFromFloat32([]float32{2.1, 2.2, 2.3})
	err = collection.Upsert(ctx,
		WithIDs("r2", "r3"),
		WithEmbeddings(emb2Updated, emb3),
		WithTexts("doc-2-updated", "doc-3"),
	)
	require.NoError(t, err)

	recordCount, err = collection.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, recordCount)

	queryResult, err := collection.Query(ctx,
		WithQueryEmbeddings(emb1),
		WithNResults(2),
		WithInclude(IncludeDocuments, IncludeEmbeddings),
	)
	require.NoError(t, err)
	require.Equal(t, 1, queryResult.CountGroups())
	require.Len(t, queryResult.GetIDGroups()[0], 2)
	require.Len(t, queryResult.GetDocumentsGroups(), 1)
	require.Equal(t, "doc-1", queryResult.GetDocumentsGroups()[0][0].ContentString())

	require.NoError(t, collection.Delete(ctx, WithIDs("r1")))
	recordCount, err = collection.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, recordCount)
}

func TestEmbeddedCollectionAdd_WithIDGeneratorGeneratesAndPersistsIDs(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(ctx, "generated-ids")
	require.NoError(t, err)

	err = collection.Add(ctx,
		WithIDGenerator(NewSHA256Generator()),
		WithTexts("doc-1", "doc-2"),
	)
	require.NoError(t, err)

	count, err := collection.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	result, err := collection.Get(ctx)
	require.NoError(t, err)
	require.Len(t, result.GetIDs(), 2)
	require.NotEqual(t, DocumentID(""), result.GetIDs()[0])
	require.NotEqual(t, DocumentID(""), result.GetIDs()[1])
}

func TestEmbeddedCollectionQuery_DefaultIncludesReturnProjections(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(ctx, "default-query-projections")
	require.NoError(t, err)

	queryEmbedding := embeddingspkg.NewEmbeddingFromFloat32([]float32{1, 0, 0})
	otherEmbedding := embeddingspkg.NewEmbeddingFromFloat32([]float32{0, 1, 0})
	err = collection.Add(
		ctx,
		WithIDs("q1", "q2"),
		WithEmbeddings(queryEmbedding, otherEmbedding),
		WithTexts("doc-1", "doc-2"),
		WithMetadatas(
			NewDocumentMetadata(NewStringAttribute("kind", "one")),
			NewDocumentMetadata(NewStringAttribute("kind", "two")),
		),
	)
	require.NoError(t, err)

	result, err := collection.Query(
		ctx,
		WithQueryEmbeddings(queryEmbedding),
		WithNResults(2),
	)
	require.NoError(t, err)
	require.Len(t, result.GetIDGroups(), 1)
	require.Len(t, result.GetDocumentsGroups(), 1)
	require.Len(t, result.GetMetadatasGroups(), 1)
	require.Len(t, result.GetDistancesGroups(), 1)
	require.Len(t, result.GetDocumentsGroups()[0], 2)
	require.Len(t, result.GetMetadatasGroups()[0], 2)
	require.Len(t, result.GetDistancesGroups()[0], 2)
	require.Equal(t, "doc-1", result.GetDocumentsGroups()[0][0].ContentString())

	kind, ok := result.GetMetadatasGroups()[0][0].GetString("kind")
	require.True(t, ok)
	require.Equal(t, "one", kind)
	require.InDelta(t, 0, float64(result.GetDistancesGroups()[0][0]), 1e-6)
}

func TestEmbeddedCollectionQuery_IncludeDistancesExplicitly(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	collection, err := client.CreateCollection(ctx, "query-distances-include")
	require.NoError(t, err)

	queryEmbedding := embeddingspkg.NewEmbeddingFromFloat32([]float32{1, 0, 0})
	otherEmbedding := embeddingspkg.NewEmbeddingFromFloat32([]float32{0, 1, 0})
	err = collection.Add(
		ctx,
		WithIDs("d1", "d2"),
		WithEmbeddings(queryEmbedding, otherEmbedding),
		WithTexts("doc-1", "doc-2"),
	)
	require.NoError(t, err)

	result, err := collection.Query(
		ctx,
		WithQueryEmbeddings(queryEmbedding),
		WithNResults(2),
		WithInclude(IncludeDistances),
	)
	require.NoError(t, err)
	require.Len(t, result.GetIDGroups(), 1)
	require.Len(t, result.GetDistancesGroups(), 1)
	require.Len(t, result.GetDistancesGroups()[0], 2)
	require.Len(t, result.GetDocumentsGroups(), 0)
	require.Len(t, result.GetMetadatasGroups(), 0)
	require.Len(t, result.GetEmbeddingsGroups(), 0)
	require.InDelta(t, 0, float64(result.GetDistancesGroups()[0][0]), 1e-6)
	require.Greater(t, float64(result.GetDistancesGroups()[0][1]), 0.0)
}

func TestEmbeddedLocalClientDeleteCollection_UsesScopedCollectionStateCleanup(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	defaultCollection, err := client.CreateCollection(
		ctx,
		"shared_name",
		WithEmbeddingFunctionCreate(embeddingspkg.NewConsistentHashEmbeddingFunction()),
	)
	require.NoError(t, err)

	otherDatabase := NewDatabase("other_db", NewTenant(DefaultTenant))
	otherCollection, err := client.CreateCollection(
		ctx,
		"shared_name",
		WithDatabaseCreate(otherDatabase),
		WithEmbeddingFunctionCreate(embeddingspkg.NewConsistentHashEmbeddingFunction()),
	)
	require.NoError(t, err)

	client.collectionStateMu.RLock()
	_, hasDefaultBefore := client.collectionState[defaultCollection.ID()]
	_, hasOtherBefore := client.collectionState[otherCollection.ID()]
	client.collectionStateMu.RUnlock()
	require.True(t, hasDefaultBefore)
	require.True(t, hasOtherBefore)

	err = client.DeleteCollection(ctx, "shared_name")
	require.NoError(t, err)

	client.collectionStateMu.RLock()
	_, hasDefaultAfter := client.collectionState[defaultCollection.ID()]
	_, hasOtherAfter := client.collectionState[otherCollection.ID()]
	client.collectionStateMu.RUnlock()
	require.False(t, hasDefaultAfter)
	require.True(t, hasOtherAfter)
}

func TestEmbeddedLocalClientConcurrentTenantDatabaseStateAccess(t *testing.T) {
	runtime := newScriptedEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	stateClient := client.state.(*APIClientV2)
	ctx := context.Background()

	runConcurrencyTest(t, stateClient.TenantAndDatabase, 500,
		func(i int) error {
			return client.UseTenant(ctx, NewTenant(fmt.Sprintf("tenant-%d", i%32)))
		},
		func(i int) error {
			tenant := NewTenant(fmt.Sprintf("tenant-%d", i%32))
			return client.UseDatabase(ctx, NewDatabase(fmt.Sprintf("database-%d", i%32), tenant))
		},
	)

	require.NotNil(t, client.CurrentTenant())
	require.NotNil(t, client.CurrentDatabase())
}

func TestEmbeddedLocalClientConcurrentUseTenantDatabase(t *testing.T) {
	runtime := newScriptedEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	stateClient, ok := client.state.(*APIClientV2)
	require.True(t, ok)
	ctx := context.Background()

	runConcurrencyTest(t, stateClient.TenantAndDatabase, 300,
		func(i int) error {
			tenant := NewTenant(fmt.Sprintf("tenant-%d", i%16))
			database := NewDatabase(fmt.Sprintf("database-%d", i%16), tenant)
			return client.UseTenantDatabase(ctx, tenant, database)
		},
		func(i int) error {
			return client.UseTenantDatabase(ctx, NewTenant(fmt.Sprintf("tenant-%d", (i+7)%16)), nil)
		},
	)
}

func TestEmbeddedLocalClientUseTenantDatabase_NilDatabaseDefaults(t *testing.T) {
	runtime := newScriptedEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	err := client.UseTenantDatabase(ctx, NewTenant("tenant-default-db"), nil)
	require.NoError(t, err)

	tenant := client.CurrentTenant()
	require.NotNil(t, tenant)
	require.Equal(t, "tenant-default-db", tenant.Name())

	database := client.CurrentDatabase()
	require.NotNil(t, database)
	require.Equal(t, DefaultDatabase, database.Name())
	require.NotNil(t, database.Tenant())
	require.Equal(t, tenant.Name(), database.Tenant().Name())
}

func TestEmbeddedLocalClientUseTenantDatabase_NilTenantReturnsError(t *testing.T) {
	runtime := newScriptedEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)

	err := client.UseTenantDatabase(context.Background(), nil, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tenant cannot be nil")
}

func TestEmbeddedLocalClientUseTenant_NilTenantReturnsError(t *testing.T) {
	runtime := newScriptedEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)

	err := client.UseTenant(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tenant cannot be nil")
}

func TestAnyToFloat32Slice_TableDriven(t *testing.T) {
	embedding := embeddingspkg.NewEmbeddingFromFloat32([]float32{1, 2, 3})

	tests := []struct {
		name    string
		input   any
		want    []float32
		wantErr string
	}{
		{name: "nil", input: nil, wantErr: "embedding cannot be nil"},
		{name: "embedding type", input: embedding, want: []float32{1, 2, 3}},
		{name: "float32 slice", input: []float32{1, 2}, want: []float32{1, 2}},
		{name: "float64 slice", input: []float64{1.5, 2.5}, want: []float32{1.5, 2.5}},
		{name: "int slice", input: []int{1, 2}, want: []float32{1, 2}},
		{name: "int32 slice", input: []int32{3, 4}, want: []float32{3, 4}},
		{name: "int64 slice", input: []int64{5, 6}, want: []float32{5, 6}},
		{
			name:  "any slice mixed numerics",
			input: []any{float32(1), float64(2), int(3), int32(4), int64(5), json.Number("6.75")},
			want:  []float32{1, 2, 3, 4, 5, 6.75},
		},
		{
			name:    "any slice invalid number",
			input:   []any{json.Number("NaN-not-number")},
			wantErr: "invalid numeric embedding value",
		},
		{
			name:    "any slice unsupported element",
			input:   []any{"bad"},
			wantErr: "unsupported embedding element type string",
		},
		{
			name:    "unsupported input type",
			input:   "bad",
			wantErr: "unsupported embedding type string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := anyToFloat32Slice(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestAnyToFloat32Slice_ReturnsCopyForFloat32Slice(t *testing.T) {
	input := []float32{1, 2, 3}
	got, err := anyToFloat32Slice(input)
	require.NoError(t, err)
	require.Equal(t, []float32{1, 2, 3}, got)

	input[0] = 99
	require.Equal(t, []float32{1, 2, 3}, got)
}

func TestEmbeddedGetCollection_WithExplicitContentEF(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	denseEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	_, err := client.CreateCollection(ctx, "test-cef", WithEmbeddingFunctionCreate(denseEF))
	require.NoError(t, err)

	contentEF := &mockCloseableContentEF{}
	got, err := client.GetCollection(ctx, "test-cef", WithContentEmbeddingFunctionGet(contentEF))
	require.NoError(t, err)

	ec, ok := got.(*embeddedCollection)
	require.True(t, ok)

	ec.mu.RLock()
	gotContentEF := ec.contentEmbeddingFunction
	ec.mu.RUnlock()
	require.NotNil(t, gotContentEF, "contentEF must be wired into the embedded collection")
}

func TestEmbeddedGetCollection_DerivesDenseEFFromDualContentEF(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	ctx := context.Background()

	writer := newEmbeddedClientForRuntime(t, runtime)
	_, err := writer.CreateCollection(ctx, "test-dual-content")
	require.NoError(t, err)

	client := newEmbeddedClientForRuntime(t, runtime)
	contentEF := &mockDualEF{}
	got, err := client.GetCollection(ctx, "test-dual-content", WithContentEmbeddingFunctionGet(contentEF))
	require.NoError(t, err)

	ec, ok := got.(*embeddedCollection)
	require.True(t, ok)

	ec.mu.RLock()
	gotDenseEF := ec.embeddingFunction
	gotContentEF := ec.contentEmbeddingFunction
	ec.mu.RUnlock()

	require.Same(t, contentEF, unwrapCloseOnceEF(gotDenseEF), "dense EF should be derived from a dual-interface content EF")
	require.Same(t, contentEF, unwrapCloseOnceContentEF(gotContentEF), "explicit dual-interface content EF should be preserved")
}

func TestEmbeddedGetCollection_AutoWiresContentEFFromDenseEF(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	denseEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	_, err := client.CreateCollection(ctx, "test-auto", WithEmbeddingFunctionCreate(denseEF))
	require.NoError(t, err)

	got, err := client.GetCollection(ctx, "test-auto", WithEmbeddingFunctionGet(denseEF))
	require.NoError(t, err)

	ec, ok := got.(*embeddedCollection)
	require.True(t, ok)

	ec.mu.RLock()
	gotDenseEF := ec.embeddingFunction
	ec.mu.RUnlock()
	require.NotNil(t, gotDenseEF, "denseEF must be wired into the embedded collection")
}

func TestEmbeddedGetCollection_AutoWiresFromConfigurationOnly(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	ctx := context.Background()

	configuration := NewCollectionConfiguration()
	configuration.SetEmbeddingFunctionInfo(&EmbeddingFunctionInfo{
		Type: "known",
		Name: "consistent_hash",
		Config: map[string]any{
			"dim": float64(128),
		},
	})

	seedEmbeddedCollectionForTest(t, runtime, "test-config-only", configuration)

	client := newEmbeddedClientForRuntime(t, runtime)
	got, err := client.GetCollection(ctx, "test-config-only")
	require.NoError(t, err)

	ec, ok := got.(*embeddedCollection)
	require.True(t, ok)

	ec.mu.RLock()
	gotDenseEF := ec.embeddingFunction
	gotContentEF := ec.contentEmbeddingFunction
	ec.mu.RUnlock()

	require.NotNil(t, gotContentEF, "content EF should be auto-wired from collection configuration")
	require.NotNil(t, gotDenseEF, "dense EF should be auto-wired from collection configuration")
	require.Equal(t, "consistent_hash", gotDenseEF.Name())

	unwrapper, ok := gotContentEF.(embeddingspkg.EmbeddingFunctionUnwrapper)
	require.True(t, ok, "config-built content EF should unwrap to the dense EF")
	require.Same(t, unwrapCloseOnceEF(gotDenseEF), unwrapper.UnwrapEmbeddingFunction())
}

func TestEmbeddedGetCollection_ContentEFStateRoundTrip(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	denseEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	_, err := client.CreateCollection(ctx, "test-roundtrip", WithEmbeddingFunctionCreate(denseEF))
	require.NoError(t, err)

	contentEF := &mockCloseableContentEF{}
	col1, err := client.GetCollection(ctx, "test-roundtrip", WithContentEmbeddingFunctionGet(contentEF))
	require.NoError(t, err)

	ec1, ok := col1.(*embeddedCollection)
	require.True(t, ok)
	ec1.mu.RLock()
	firstContentEF := ec1.contentEmbeddingFunction
	ec1.mu.RUnlock()
	require.NotNil(t, firstContentEF, "contentEF must be wired on first GetCollection")

	// Second GetCollection without explicit contentEF — state guard should preserve it
	col2, err := client.GetCollection(ctx, "test-roundtrip", WithEmbeddingFunctionGet(denseEF))
	require.NoError(t, err)

	ec2, ok := col2.(*embeddedCollection)
	require.True(t, ok)
	ec2.mu.RLock()
	secondContentEF := ec2.contentEmbeddingFunction
	ec2.mu.RUnlock()
	require.Same(t, firstContentEF, secondContentEF, "contentEF must survive state round-trip via collectionState")
}

func TestEmbeddedGetCollection_LogsAutoWireErrorsToStderr(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	ctx := context.Background()

	t.Setenv("OPENAI_API_KEY", "")
	configuration := NewCollectionConfiguration()
	configuration.SetEmbeddingFunctionInfo(&EmbeddingFunctionInfo{
		Type: "known",
		Name: "openai",
		Config: map[string]any{
			"api_key_env_var": "OPENAI_API_KEY",
			"model_name":      "text-embedding-3-small",
		},
	})

	seedEmbeddedCollectionForTest(t, runtime, "test-autowire-log", configuration)

	client := newEmbeddedClientForRuntime(t, runtime)
	client.logger = nil
	output := captureStderr(t, func() {
		got, getErr := client.GetCollection(ctx, "test-autowire-log")
		require.NoError(t, getErr)
		require.NotNil(t, got)
	})

	require.Contains(t, output, "failed to auto-wire content embedding function")
	require.Contains(t, output, "failed to auto-wire embedding function")
	require.Contains(t, output, "OPENAI_API_KEY")
}

func TestEmbeddedGetCollection_PreservesExistingDenseEFWhenOnlyContentEFChanges(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	initialDenseEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	_, err := client.CreateCollection(ctx, "test-dense-guard", WithEmbeddingFunctionCreate(initialDenseEF))
	require.NoError(t, err)

	contentEF := &mockDualEF{}
	got, err := client.GetCollection(ctx, "test-dense-guard", WithContentEmbeddingFunctionGet(contentEF))
	require.NoError(t, err)

	ec, ok := got.(*embeddedCollection)
	require.True(t, ok)

	ec.mu.RLock()
	gotDenseEF := ec.embeddingFunction
	gotContentEF := ec.contentEmbeddingFunction
	ec.mu.RUnlock()

	require.Same(t, initialDenseEF, unwrapCloseOnceEF(gotDenseEF), "existing dense EF should survive when only content EF is provided")
	require.Same(t, contentEF, unwrapCloseOnceContentEF(gotContentEF), "new content EF should still be stored")
}

func TestEmbeddedCollection_CloseLifecycleWithSharedAdapter(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	denseEF := &mockCloseableEF{}
	_, err := client.CreateCollection(ctx, "test-close-lifecycle", WithEmbeddingFunctionCreate(denseEF))
	require.NoError(t, err)

	contentAdapter := &mockSharedContentAdapter{inner: denseEF}
	col, err := client.GetCollection(ctx, "test-close-lifecycle",
		WithEmbeddingFunctionGet(denseEF),
		WithContentEmbeddingFunctionGet(contentAdapter),
	)
	require.NoError(t, err)

	err = col.Close()
	require.NoError(t, err)

	require.Equal(t, 1, contentAdapter.closeCount, "content adapter closed once through lifecycle")
	require.Equal(t, int32(1), denseEF.closeCount.Load(), "shared dense EF should only be closed through the content adapter")
}

func TestEmbeddedCollection_CloseLifecycleWithIndependentEFs(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	denseEF := &mockCloseableEF{}
	_, err := client.CreateCollection(ctx, "test-close-indep", WithEmbeddingFunctionCreate(denseEF))
	require.NoError(t, err)

	contentEF := &mockCloseableContentEF{}
	col, err := client.GetCollection(ctx, "test-close-indep",
		WithEmbeddingFunctionGet(denseEF),
		WithContentEmbeddingFunctionGet(contentEF),
	)
	require.NoError(t, err)

	err = col.Close()
	require.NoError(t, err)

	require.Equal(t, int32(1), contentEF.closeCount.Load(), "independent contentEF closed once through lifecycle")
	require.Equal(t, int32(1), denseEF.closeCount.Load(), "independent denseEF closed once through lifecycle")
}

func TestEmbeddedGetCollection_ConcurrentAutoWire(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	ctx := context.Background()

	providerName := fmt.Sprintf("concurrent-test-ef-%d", time.Now().UnixNano())
	var buildCount atomic.Int32
	err := embeddingspkg.RegisterDense(providerName, func(_ embeddingspkg.EmbeddingFunctionConfig) (embeddingspkg.EmbeddingFunction, error) {
		buildCount.Add(1)
		return &mockCloseableEF{}, nil
	})
	require.NoError(t, err)

	configuration := NewCollectionConfiguration()
	configuration.SetEmbeddingFunctionInfo(&EmbeddingFunctionInfo{
		Type:   "known",
		Name:   providerName,
		Config: map[string]any{},
	})
	seedEmbeddedCollectionForTest(t, runtime, "concurrent-autowire", configuration)

	client := newEmbeddedClientForRuntime(t, runtime)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	ready := make(chan struct{})
	results := make([]Collection, goroutines)
	errs := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			<-ready
			results[idx], errs[idx] = client.GetCollection(ctx, "concurrent-autowire")
		}(i)
	}
	close(ready)
	wg.Wait()

	for i := 0; i < goroutines; i++ {
		require.NoError(t, errs[i], "goroutine %d must not error", i)
		ec, ok := results[i].(*embeddedCollection)
		require.True(t, ok)
		ec.mu.RLock()
		require.NotNil(t, ec.embeddingFunction, "goroutine %d must have non-nil dense EF", i)
		ec.mu.RUnlock()
	}

	require.Equal(t, int32(1), buildCount.Load(), "factory must be invoked exactly once (write lock prevents duplicate auto-wire)")

	client.collectionStateMu.RLock()
	require.Equal(t, 1, len(client.collectionState), "exactly one state entry for the collection")
	client.collectionStateMu.RUnlock()
}

func TestEmbeddedDeleteCollectionState_ClosesEFs(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)

	mockEF := &mockCloseableEF{}
	mockContentEF := &mockCloseableContentEF{}
	wrappedEF := wrapEFCloseOnce(mockEF)
	wrappedContentEF := wrapContentEFCloseOnce(mockContentEF)

	client.collectionStateMu.Lock()
	client.collectionState["test-id"] = &embeddedCollectionState{
		embeddingFunction:            wrappedEF,
		ownsEmbeddingFunction:        true,
		contentEmbeddingFunction:     wrappedContentEF,
		ownsContentEmbeddingFunction: true,
	}
	client.collectionStateMu.Unlock()

	client.deleteCollectionState("test-id")

	require.Equal(t, int32(1), mockEF.closeCount.Load(), "dense EF must be closed exactly once")
	require.Equal(t, int32(1), mockContentEF.closeCount.Load(), "content EF must be closed exactly once")

	client.collectionStateMu.RLock()
	require.Nil(t, client.collectionState["test-id"], "map entry must be removed")
	client.collectionStateMu.RUnlock()
}

func TestEmbeddedLocalClient_UpsertCollectionStateSnapshotIncludesOwnershipFlags(t *testing.T) {
	client := newEmbeddedClientForRuntime(t, newMemoryEmbeddedRuntime())

	snapshot := client.upsertCollectionState("snapshot-id", func(state *embeddedCollectionState) {
		state.ownsEmbeddingFunction = true
		state.ownsContentEmbeddingFunction = true
	})

	require.True(t, snapshot.ownsEmbeddingFunction)
	require.True(t, snapshot.ownsContentEmbeddingFunction)
}

func TestEmbeddedLocalClient_Close_RespectsCollectionStateOwnershipFlags(t *testing.T) {
	tests := []struct {
		name              string
		ownDense          bool
		ownContent        bool
		wantDenseCloses   int32
		wantContentCloses int32
	}{
		{
			name:              "owns both",
			ownDense:          true,
			ownContent:        true,
			wantDenseCloses:   1,
			wantContentCloses: 1,
		},
		{
			name:              "owns dense only",
			ownDense:          true,
			ownContent:        false,
			wantDenseCloses:   1,
			wantContentCloses: 0,
		},
		{
			name:              "owns content only",
			ownDense:          false,
			ownContent:        true,
			wantDenseCloses:   0,
			wantContentCloses: 1,
		},
		{
			name:              "owns neither",
			ownDense:          false,
			ownContent:        false,
			wantDenseCloses:   0,
			wantContentCloses: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newEmbeddedClientForRuntime(t, newMemoryEmbeddedRuntime())

			mockEF := &mockCloseableEF{}
			mockContentEF := &mockCloseableContentEF{}

			client.collectionStateMu.Lock()
			client.collectionState["test-id"] = &embeddedCollectionState{
				embeddingFunction:            wrapEFCloseOnce(mockEF),
				ownsEmbeddingFunction:        tt.ownDense,
				contentEmbeddingFunction:     wrapContentEFCloseOnce(mockContentEF),
				ownsContentEmbeddingFunction: tt.ownContent,
			}
			client.collectionStateMu.Unlock()

			require.NoError(t, client.Close())
			require.Equal(t, tt.wantDenseCloses, mockEF.closeCount.Load())
			require.Equal(t, tt.wantContentCloses, mockContentEF.closeCount.Load())
		})
	}
}

func TestEmbeddedLocalClient_Close_CleansUpCollectionState(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)

	mockEF1 := &mockCloseableEF{}
	mockContentEF1 := &mockCloseableContentEF{}
	mockEF2 := &mockCloseableEF{}
	mockContentEF2 := &mockFailingCloseContentEF{closeErr: errors.New("mock close failure")}

	client.collectionStateMu.Lock()
	client.collectionState["col-1"] = &embeddedCollectionState{
		embeddingFunction:            wrapEFCloseOnce(mockEF1),
		ownsEmbeddingFunction:        true,
		contentEmbeddingFunction:     wrapContentEFCloseOnce(mockContentEF1),
		ownsContentEmbeddingFunction: true,
	}
	client.collectionState["col-2"] = &embeddedCollectionState{
		embeddingFunction:            wrapEFCloseOnce(mockEF2),
		ownsEmbeddingFunction:        true,
		contentEmbeddingFunction:     wrapContentEFCloseOnce(mockContentEF2),
		ownsContentEmbeddingFunction: true,
	}
	client.collectionStateMu.Unlock()

	err := client.Close()
	require.Error(t, err, "Close must propagate aggregated errors")
	require.Contains(t, err.Error(), "mock close failure")

	require.Equal(t, int32(1), mockEF1.closeCount.Load(), "EF1 must be closed")
	require.Equal(t, int32(1), mockContentEF1.closeCount.Load(), "content EF1 must be closed")
	require.Equal(t, int32(1), mockEF2.closeCount.Load(), "EF2 must be closed even after EF1 error")
	require.Equal(t, int32(1), mockContentEF2.closeCount.Load(), "content EF2 must be closed")

	client.collectionStateMu.RLock()
	require.Equal(t, 0, len(client.collectionState), "state map must be cleared")
	client.collectionStateMu.RUnlock()
}

func TestEmbeddedLocalClient_Close_LogsCloseErrors(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	log := &capturingLogger{}

	client := newEmbeddedClientForRuntime(t, runtime)
	client.logger = log

	mockEF := &mockFailingCloseEF{closeErr: errors.New("mock close failure")}
	client.collectionStateMu.Lock()
	client.collectionState["test-id"] = &embeddedCollectionState{
		embeddingFunction:     wrapEFCloseOnce(mockEF),
		ownsEmbeddingFunction: true,
	}
	client.collectionStateMu.Unlock()

	err := client.Close()
	require.Error(t, err)
	require.GreaterOrEqual(t, log.errorCount, 1, "logger must receive at least one Error call")
	require.Contains(t, log.lastMsg, "failed to close EF during client shutdown")
}

func TestEmbeddedLocalClient_Close_IsSafeToCallTwice(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)

	mockEF := &mockCloseableEF{}
	mockContentEF := &mockCloseableContentEF{}

	client.collectionStateMu.Lock()
	client.collectionState["test-id"] = &embeddedCollectionState{
		embeddingFunction:            wrapEFCloseOnce(mockEF),
		ownsEmbeddingFunction:        true,
		contentEmbeddingFunction:     wrapContentEFCloseOnce(mockContentEF),
		ownsContentEmbeddingFunction: true,
	}
	client.collectionStateMu.Unlock()

	require.NoError(t, client.Close())
	require.NoError(t, client.Close())
	require.Equal(t, int32(1), mockEF.closeCount.Load(), "dense EF must still be closed exactly once after repeated Close")
	require.Equal(t, int32(1), mockContentEF.closeCount.Load(), "content EF must still be closed exactly once after repeated Close")
}

func TestEmbeddedLocalClient_Close_NoLoggerFallsBackToStderrWithShutdownContext(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	client.logger = nil

	mockEF := &mockFailingCloseEF{closeErr: errors.New("shutdown stderr test error")}
	client.collectionStateMu.Lock()
	client.collectionState["shutdown-test"] = &embeddedCollectionState{
		embeddingFunction:     wrapEFCloseOnce(mockEF),
		ownsEmbeddingFunction: true,
	}
	client.collectionStateMu.Unlock()

	output := captureStderr(t, func() {
		err := client.Close()
		require.Error(t, err)
	})

	require.Contains(t, output, "failed to close EF during client shutdown")
	require.NotContains(t, output, "collection cache cleanup")
}

func TestEmbeddedBuildCollection_CloseOnceWrapping(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	colID := seedEmbeddedCollectionForTest(t, runtime, "wrap-test", nil)

	rawEF := &mockCloseableEF{}
	rawContentEF := &mockCloseableContentEF{}

	runtime.mu.Lock()
	key := collectionRuntimeKey(DefaultTenant, DefaultDatabase, "wrap-test")
	model := runtime.collections[key]
	runtime.mu.Unlock()

	collection, err := client.buildEmbeddedCollection(model, nil, rawEF, rawContentEF, true, true)
	require.NoError(t, err)
	_ = colID

	_, ok := collection.embeddingFunction.(*closeOnceEF)
	require.True(t, ok, "embeddingFunction must be wrapped in *closeOnceEF")

	_, ok = collection.contentEmbeddingFunction.(*closeOnceContentEF)
	require.True(t, ok, "contentEmbeddingFunction must be wrapped in *closeOnceContentEF")
}

func TestEmbeddedGetCollection_BuildErrorGuard(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	ctx := context.Background()

	configuration := NewCollectionConfiguration()
	configuration.SetEmbeddingFunctionInfo(&EmbeddingFunctionInfo{
		Type:   "known",
		Name:   "nonexistent_provider_xyz",
		Config: map[string]any{},
	})
	colID := seedEmbeddedCollectionForTest(t, runtime, "error-guard", configuration)

	client := newEmbeddedClientForRuntime(t, runtime)

	got, err := client.GetCollection(ctx, "error-guard")
	require.NoError(t, err)

	ec, ok := got.(*embeddedCollection)
	require.True(t, ok)
	ec.mu.RLock()
	require.Nil(t, ec.embeddingFunction, "EF must be nil when build fails")
	require.Nil(t, ec.contentEmbeddingFunction, "content EF must be nil when build fails")
	ec.mu.RUnlock()

	client.collectionStateMu.RLock()
	state := client.collectionState[colID]
	require.NotNil(t, state, "state entry must exist")
	require.Nil(t, state.embeddingFunction, "state EF must be nil (not poisoned)")
	require.Nil(t, state.contentEmbeddingFunction, "state content EF must be nil (not poisoned)")
	client.collectionStateMu.RUnlock()

	explicitEF := &mockCloseableEF{}
	got2, err := client.GetCollection(ctx, "error-guard", WithEmbeddingFunctionGet(explicitEF))
	require.NoError(t, err)

	ec2, ok := got2.(*embeddedCollection)
	require.True(t, ok)
	ec2.mu.RLock()
	require.NotNil(t, ec2.embeddingFunction, "explicit EF must be set on subsequent GetCollection")
	ec2.mu.RUnlock()
}

func TestEmbeddedGetCollection_AutoWireRecoveryAfterInitialBuildFailure(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	ctx := context.Background()

	origValue, hadOrig := os.LookupEnv("OPENAI_API_KEY")
	require.NoError(t, os.Unsetenv("OPENAI_API_KEY"))
	defer func() {
		if hadOrig {
			require.NoError(t, os.Setenv("OPENAI_API_KEY", origValue))
			return
		}
		require.NoError(t, os.Unsetenv("OPENAI_API_KEY"))
	}()

	configuration := NewCollectionConfiguration()
	configuration.SetEmbeddingFunctionInfo(&EmbeddingFunctionInfo{
		Type: "known",
		Name: "openai",
		Config: map[string]any{
			"api_key_env_var": "OPENAI_API_KEY",
			"model_name":      "text-embedding-3-small",
		},
	})
	colID := seedEmbeddedCollectionForTest(t, runtime, "autowire-recovery", configuration)

	client := newEmbeddedClientForRuntime(t, runtime)

	got, err := client.GetCollection(ctx, "autowire-recovery")
	require.NoError(t, err)

	ec, ok := got.(*embeddedCollection)
	require.True(t, ok)
	ec.mu.RLock()
	require.Nil(t, ec.embeddingFunction, "dense EF must stay nil after initial build failure")
	require.Nil(t, ec.contentEmbeddingFunction, "content EF must stay nil after initial build failure")
	ec.mu.RUnlock()

	client.collectionStateMu.RLock()
	state := client.collectionState[colID]
	require.NotNil(t, state, "state entry must exist after failed auto-wire")
	require.Nil(t, state.embeddingFunction, "state dense EF must stay nil after failed auto-wire")
	require.Nil(t, state.contentEmbeddingFunction, "state content EF must stay nil after failed auto-wire")
	client.collectionStateMu.RUnlock()

	require.NoError(t, os.Setenv("OPENAI_API_KEY", "test-api-key-123"))

	got2, err := client.GetCollection(ctx, "autowire-recovery")
	require.NoError(t, err)

	ec2, ok := got2.(*embeddedCollection)
	require.True(t, ok)
	ec2.mu.RLock()
	require.NotNil(t, ec2.embeddingFunction, "dense EF must auto-wire on retry after config becomes valid")
	require.NotNil(t, ec2.contentEmbeddingFunction, "content EF must auto-wire on retry after config becomes valid")
	ec2.mu.RUnlock()

	client.collectionStateMu.RLock()
	state = client.collectionState[colID]
	require.NotNil(t, state.embeddingFunction, "state dense EF must be populated after successful retry")
	require.NotNil(t, state.contentEmbeddingFunction, "state content EF must be populated after successful retry")
	client.collectionStateMu.RUnlock()
}

func TestEmbeddedGetCollection_RaceWithDeleteCollectionDoesNotResurrectState(t *testing.T) {
	runtime := newBlockingGetMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	created, err := client.CreateCollection(ctx, "race-delete-get")
	require.NoError(t, err)

	type getResult struct {
		collection Collection
		err        error
	}
	resultCh := make(chan getResult, 1)

	go func() {
		col, getErr := client.GetCollection(ctx, "race-delete-get")
		resultCh <- getResult{collection: col, err: getErr}
	}()

	<-runtime.firstSnapshotTaken

	err = client.DeleteCollection(ctx, "race-delete-get")
	require.NoError(t, err)

	close(runtime.unblockFirstGet)
	result := <-resultCh

	require.Error(t, result.err)
	require.Nil(t, result.collection)

	client.collectionStateMu.RLock()
	_, hasState := client.collectionState[created.ID()]
	client.collectionStateMu.RUnlock()
	require.False(t, hasState, "concurrent GetCollection must not resurrect deleted collection state")
	require.Nil(t, client.cachedCollectionByName("race-delete-get"), "concurrent GetCollection must not resurrect deleted collection cache entry")

	_, err = client.GetCollection(ctx, "race-delete-get")
	require.Error(t, err)
}

func TestEmbeddedGetCollection_RaceWithDeleteAndRecreateDoesNotReturnStaleCollection(t *testing.T) {
	runtime := newBlockingGetMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	original, err := client.CreateCollection(ctx, "race-delete-recreate")
	require.NoError(t, err)

	type getResult struct {
		collection Collection
		err        error
	}
	resultCh := make(chan getResult, 1)

	go func() {
		col, getErr := client.GetCollection(ctx, "race-delete-recreate")
		resultCh <- getResult{collection: col, err: getErr}
	}()

	<-runtime.firstSnapshotTaken

	err = client.DeleteCollection(ctx, "race-delete-recreate")
	require.NoError(t, err)

	recreated, err := client.CreateCollection(ctx, "race-delete-recreate")
	require.NoError(t, err)
	require.NotEqual(t, original.ID(), recreated.ID(), "recreated collection must have a new ID")

	close(runtime.unblockFirstGet)
	result := <-resultCh

	require.Error(t, result.err)
	require.Nil(t, result.collection)

	client.collectionStateMu.RLock()
	_, hasOriginalState := client.collectionState[original.ID()]
	_, hasRecreatedState := client.collectionState[recreated.ID()]
	client.collectionStateMu.RUnlock()
	require.False(t, hasOriginalState, "stale collection state must be cleaned up")
	require.True(t, hasRecreatedState, "replacement collection state must remain intact")

	cached := client.cachedCollectionByName("race-delete-recreate")
	require.NotNil(t, cached, "replacement collection must remain cached")
	require.Equal(t, recreated.ID(), cached.ID(), "cache must point at the replacement collection")
}

func TestEmbeddedCreateCollection_StoresWrappedEFInState(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	mockEF := &mockCloseableEF{}

	collection, err := client.CreateCollection(ctx, "create-wrap-test", WithEmbeddingFunctionCreate(mockEF))
	require.NoError(t, err)

	embedded, ok := collection.(*embeddedCollection)
	require.True(t, ok)

	client.collectionStateMu.RLock()
	state := client.collectionState[embedded.ID()]
	require.NotNil(t, state, "state entry must exist after CreateCollection")
	_, wrapped := state.embeddingFunction.(*closeOnceEF)
	require.True(t, wrapped, "state embeddingFunction must be stored as *closeOnceEF")
	client.collectionStateMu.RUnlock()

	client.deleteCollectionState(embedded.ID())

	err = embedded.Close()
	require.NoError(t, err)
	require.Equal(t, int32(1), mockEF.closeCount.Load(), "CreateCollection state and collection must share one close-once wrapper")
}

func TestEmbeddedDeleteAndCloseShareWrapper(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)

	mockEF := &mockCloseableEF{}
	mockContentEF := &mockCloseableContentEF{}
	wrappedEF := wrapEFCloseOnce(mockEF)
	wrappedContentEF := wrapContentEFCloseOnce(mockContentEF)

	client.collectionStateMu.Lock()
	client.collectionState["test-id"] = &embeddedCollectionState{
		embeddingFunction:            wrappedEF,
		ownsEmbeddingFunction:        true,
		contentEmbeddingFunction:     wrappedContentEF,
		ownsContentEmbeddingFunction: true,
	}
	client.collectionStateMu.Unlock()

	// Build an embeddedCollection holding the SAME wrapped EFs
	collection := &embeddedCollection{
		embeddingFunction:        wrappedEF,
		contentEmbeddingFunction: wrappedContentEF,
	}
	collection.ownsEF.Store(true)

	// Close via state path
	client.deleteCollectionState("test-id")

	// Close via collection path -- sync.Once prevents double-close
	err := collection.Close()
	require.NoError(t, err)

	require.Equal(t, int32(1), mockEF.closeCount.Load(), "shared wrapper sync.Once must prevent double-close of dense EF")
	require.Equal(t, int32(1), mockContentEF.closeCount.Load(), "shared wrapper sync.Once must prevent double-close of content EF")
}

func TestEmbeddedDeleteAndCloseRace(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)

	mockEF := &mockCloseableEF{}
	mockContentEF := &mockCloseableContentEF{}
	wrappedEF := wrapEFCloseOnce(mockEF)
	wrappedContentEF := wrapContentEFCloseOnce(mockContentEF)

	client.collectionStateMu.Lock()
	client.collectionState["race-id"] = &embeddedCollectionState{
		embeddingFunction:            wrappedEF,
		ownsEmbeddingFunction:        true,
		contentEmbeddingFunction:     wrappedContentEF,
		ownsContentEmbeddingFunction: true,
	}
	client.collectionStateMu.Unlock()

	var wg sync.WaitGroup
	wg.Add(2)
	ready := make(chan struct{})

	go func() {
		defer wg.Done()
		<-ready
		client.deleteCollectionState("race-id")
	}()

	go func() {
		defer wg.Done()
		<-ready
		_ = client.Close()
	}()

	close(ready)
	wg.Wait()

	require.Equal(t, int32(1), mockEF.closeCount.Load(), "shared wrapper sync.Once must close dense EF exactly once")
	require.Equal(t, int32(1), mockContentEF.closeCount.Load(), "shared wrapper sync.Once must close content EF exactly once")
}

func TestEmbeddedListCollections_ContentEFIsNil(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	denseEF := embeddingspkg.NewConsistentHashEmbeddingFunction()
	_, err := client.CreateCollection(ctx, "test-list-cef", WithEmbeddingFunctionCreate(denseEF))
	require.NoError(t, err)

	// Set contentEF via GetCollection so state has it
	contentEF := &mockCloseableContentEF{}
	_, err = client.GetCollection(ctx, "test-list-cef", WithContentEmbeddingFunctionGet(contentEF))
	require.NoError(t, err)

	// ListCollections passes nil,nil to buildEmbeddedCollection — contentEF comes from state
	cols, err := client.ListCollections(ctx)
	require.NoError(t, err)
	require.Len(t, cols, 1)

	ec, ok := cols[0].(*embeddedCollection)
	require.True(t, ok)
	// ListCollections rebuilds from state, so contentEF should be present from prior state
	ec.mu.RLock()
	gotContentEF := ec.contentEmbeddingFunction
	ec.mu.RUnlock()
	require.NotNil(t, gotContentEF, "ListCollections should pick up contentEF from state")
}

func TestEmbeddedClient_LoggerReceivesErrors(t *testing.T) {
	t.Run("auto-wire errors route to logger at Warn level", func(t *testing.T) {
		const providerName = "failing_logger_test_provider"
		require.NoError(t, embeddingspkg.RegisterContent(providerName,
			func(_ embeddingspkg.EmbeddingFunctionConfig) (embeddingspkg.ContentEmbeddingFunction, error) {
				return nil, errors.New("intentional factory failure for logger test")
			}))

		runtime := newMemoryEmbeddedRuntime()
		log := &capturingLogger{}

		configuration := NewCollectionConfiguration()
		configuration.SetEmbeddingFunctionInfo(&EmbeddingFunctionInfo{
			Type:   "known",
			Name:   providerName,
			Config: map[string]any{},
		})
		seedEmbeddedCollectionForTest(t, runtime, "logger-warn-test", configuration)

		client := newEmbeddedClientForRuntime(t, runtime)
		client.logger = log

		_, err := client.GetCollection(context.Background(), "logger-warn-test")
		require.NoError(t, err)

		require.GreaterOrEqual(t, log.warnCount, 1, "logger must receive at least one Warn call")
		require.Contains(t, log.lastMsg, "failed to auto-wire")
	})

	t.Run("close errors route to logger at Error level", func(t *testing.T) {
		runtime := newMemoryEmbeddedRuntime()
		log := &capturingLogger{}

		client := newEmbeddedClientForRuntime(t, runtime)
		client.logger = log

		mockEF := &mockFailingCloseEF{closeErr: errors.New("mock close failure")}
		wrappedEF := wrapEFCloseOnce(mockEF)

		client.collectionStateMu.Lock()
		client.collectionState["test-id"] = &embeddedCollectionState{
			embeddingFunction:     wrappedEF,
			ownsEmbeddingFunction: true,
		}
		client.collectionStateMu.Unlock()

		client.deleteCollectionState("test-id")

		require.GreaterOrEqual(t, log.errorCount, 1, "logger must receive at least one Error call")
		require.Contains(t, log.lastMsg, "failed to close EF")
	})
}

func TestNewEmbeddedClientForRuntime_DefaultsToLoggerInvariant(t *testing.T) {
	client := newEmbeddedClientForRuntime(t, newMemoryEmbeddedRuntime())
	require.NotNil(t, client.logger)
}

func TestEmbeddedCreateCollection_IfNotExistsPreflightFallsBackToStderrWhenLoggerNil(t *testing.T) {
	runtime := newErrorGetCollectionOnceRuntime(errors.New("runtime temporarily unavailable"))
	client := newEmbeddedClientForRuntime(t, runtime)
	client.logger = nil

	output := captureStderr(t, func() {
		got, err := client.CreateCollection(context.Background(), "preflight-warn-nil-logger", WithIfNotExistsCreate())
		require.NoError(t, err)
		require.NotNil(t, got)
	})

	require.Contains(t, output, "create-if-not-exists preflight GetCollection failed")
	require.Contains(t, output, "runtime temporarily unavailable")
}

func TestEmbeddedCreateCollection_IfNotExistsPreflightWarnsViaLogger(t *testing.T) {
	runtime := newErrorGetCollectionOnceRuntime(errors.New("runtime temporarily unavailable"))
	client := newEmbeddedClientForRuntime(t, runtime)
	log := &capturingLogger{}
	client.logger = log

	got, err := client.CreateCollection(context.Background(), "preflight-warn-logger", WithIfNotExistsCreate())
	require.NoError(t, err)
	require.NotNil(t, got)
	require.GreaterOrEqual(t, log.warnCount, 1)
	require.Contains(t, log.lastMsg, "create-if-not-exists preflight GetCollection failed")
}

func TestEmbeddedCreateCollection_RealORTIfNotExistsRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real ORT/local runtime test in short mode")
	}

	persistPath := t.TempDir()
	ctx := context.Background()
	collectionName := fmt.Sprintf("real-ort-if-not-exists-%d", time.Now().UnixNano())

	writer := newRealPersistentClientForTest(t, persistPath)
	created, err := writer.CreateCollection(ctx, collectionName, WithIfNotExistsCreate())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "error creating default embedding function") {
			t.Skipf("ORT embedding function unavailable in this environment: %v", err)
		}
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	createdEmbedded, ok := created.(*embeddedCollection)
	require.True(t, ok)
	createdDenseEF := unwrapCloseOnceEF(createdEmbedded.embeddingFunctionSnapshot())
	require.NotNil(t, createdDenseEF)
	require.Equal(t, "default", createdDenseEF.Name())

	reader := newRealPersistentClientForTest(t, persistPath)
	defer func() {
		_ = reader.Close()
	}()

	var temporaryDefaultEF *countingCloseableEF
	got, err := reader.CreateCollection(
		ctx,
		collectionName,
		WithIfNotExistsCreate(),
		withDefaultDenseEFFactoryCreate(func() (embeddingspkg.EmbeddingFunction, func() error, error) {
			ef, closeFn, factoryErr := ortpkg.NewDefaultEmbeddingFunction()
			if factoryErr != nil {
				return nil, nil, factoryErr
			}
			temporaryDefaultEF = &countingCloseableEF{
				EmbeddingFunction: ef,
				closeFn:           closeFn,
			}
			return temporaryDefaultEF, temporaryDefaultEF.Close, nil
		}),
	)
	require.NoError(t, err)
	require.Equal(t, created.ID(), got.ID())
	require.NotNil(t, temporaryDefaultEF)
	require.Equal(t, int32(1), temporaryDefaultEF.closeCount.Load(),
		"temporary default EF must be closed when CreateCollection reused an existing collection")

	gotEmbedded, ok := got.(*embeddedCollection)
	require.True(t, ok)
	gotDenseEF := unwrapCloseOnceEF(gotEmbedded.embeddingFunctionSnapshot())
	require.NotNil(t, gotDenseEF)
	require.Equal(t, "default", gotDenseEF.Name())
}

func TestEmbeddedClient_NoLoggerFallsBackToStderr(t *testing.T) {
	runtime := newMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	client.logger = nil

	mockEF := &mockFailingCloseEF{closeErr: errors.New("stderr test error")}
	wrappedEF := wrapEFCloseOnce(mockEF)

	client.collectionStateMu.Lock()
	client.collectionState["stderr-test"] = &embeddedCollectionState{
		embeddingFunction:     wrappedEF,
		ownsEmbeddingFunction: true,
	}
	client.collectionStateMu.Unlock()

	output := captureStderr(t, func() {
		client.deleteCollectionState("stderr-test")
	})

	require.Contains(t, output, "failed to close EF during collection state cleanup")
	require.NotContains(t, output, "collection cache cleanup")
}

func TestWithPersistentLogger_PropagatesToStateClient(t *testing.T) {
	log := &capturingLogger{}

	cfg := defaultLocalClientConfig()
	err := WithPersistentLogger(log)(cfg)
	require.NoError(t, err)

	require.Equal(t, log, cfg.logger, "embedded client logger must be set")
	require.GreaterOrEqual(t, len(cfg.clientOptions), 1, "WithLogger must be appended to clientOptions")

	// Verify the appended ClientOption actually sets the logger on a BaseAPIClient.
	base, baseErr := newBaseAPIClient(cfg.clientOptions...)
	require.NoError(t, baseErr)
	require.Equal(t, log, base.logger, "state client logger must be the same capturingLogger")
}

func TestWithPersistentLogger_RejectsNil(t *testing.T) {
	cfg := defaultLocalClientConfig()

	err := WithPersistentLogger(nil)(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "persistent logger cannot be nil")
}

func TestEmbeddedCreateCollection_ContentEF_NewCollection(t *testing.T) {
	runtime := newCountingMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	contentEF := &mockCloseableContentEF{}
	col, err := client.CreateCollection(ctx, "test-content-ef",
		WithEmbeddingFunctionCreate(embeddingspkg.NewConsistentHashEmbeddingFunction()),
		WithContentEmbeddingFunctionCreate(contentEF),
	)
	require.NoError(t, err)

	ec := col.(*embeddedCollection)
	ec.mu.RLock()
	gotContentEF := ec.contentEmbeddingFunction
	ec.mu.RUnlock()
	require.NotNil(t, gotContentEF, "contentEmbeddingFunction must be set on new embedded collection")
}

func TestEmbeddedCreateCollection_ContentEF_ExistingCollection(t *testing.T) {
	runtime := newCountingMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	originalContentEF := &mockCloseableContentEF{}
	_, err := client.CreateCollection(ctx, "test-col",
		WithEmbeddingFunctionCreate(embeddingspkg.NewConsistentHashEmbeddingFunction()),
		WithContentEmbeddingFunctionCreate(originalContentEF),
	)
	require.NoError(t, err)

	newContentEF := &mockCloseableContentEF{}
	col2, err := client.CreateCollection(ctx, "test-col",
		WithEmbeddingFunctionCreate(embeddingspkg.NewConsistentHashEmbeddingFunction()),
		WithContentEmbeddingFunctionCreate(newContentEF),
		WithIfNotExistsCreate(),
	)
	require.NoError(t, err)

	ec := col2.(*embeddedCollection)
	ec.mu.RLock()
	gotContentEF := ec.contentEmbeddingFunction
	ec.mu.RUnlock()
	require.NotNil(t, gotContentEF, "contentEF should come from state for existing collection")
	require.Same(t, originalContentEF, unwrapCloseOnceContentEF(gotContentEF),
		"existing collection must preserve original contentEF, not use the new one")
}

func TestEmbeddedGetOrCreateCollection_ContentEF_ForwardedToGetCollection(t *testing.T) {
	runtime := newCountingMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	_, err := client.CreateCollection(ctx, "test-col",
		WithEmbeddingFunctionCreate(embeddingspkg.NewConsistentHashEmbeddingFunction()),
	)
	require.NoError(t, err)

	contentEF := &mockCloseableContentEF{}
	got, err := client.GetOrCreateCollection(ctx, "test-col",
		WithEmbeddingFunctionCreate(embeddingspkg.NewConsistentHashEmbeddingFunction()),
		WithContentEmbeddingFunctionCreate(contentEF),
	)
	require.NoError(t, err)

	ec := got.(*embeddedCollection)
	ec.mu.RLock()
	gotContentEF := ec.contentEmbeddingFunction
	ec.mu.RUnlock()
	require.NotNil(t, gotContentEF, "contentEF must be forwarded to existing collection via GetCollection")
}

func TestEmbeddedGetOrCreateCollection_ContentEF_VerifyViaSubsequentGetCollection(t *testing.T) {
	runtime := newCountingMemoryEmbeddedRuntime()
	client := newEmbeddedClientForRuntime(t, runtime)
	ctx := context.Background()

	contentEF := &mockCloseableContentEF{}
	_, err := client.CreateCollection(ctx, "test-col",
		WithEmbeddingFunctionCreate(embeddingspkg.NewConsistentHashEmbeddingFunction()),
		WithContentEmbeddingFunctionCreate(contentEF),
	)
	require.NoError(t, err)

	got, err := client.GetCollection(ctx, "test-col",
		WithEmbeddingFunctionGet(embeddingspkg.NewConsistentHashEmbeddingFunction()),
	)
	require.NoError(t, err)

	ec := got.(*embeddedCollection)
	ec.mu.RLock()
	gotContentEF := ec.contentEmbeddingFunction
	ec.mu.RUnlock()
	require.NotNil(t, gotContentEF, "contentEF should be available from embedded state even without explicit option")
	require.Same(t, contentEF, unwrapCloseOnceContentEF(gotContentEF), "should be the same contentEF stored in state")
}
