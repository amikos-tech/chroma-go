//go:build basicv2 && !cloud
// +build basicv2,!cloud

package v2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	localchroma "github.com/amikos-tech/chroma-go-local"
	embeddingspkg "github.com/amikos-tech/chroma-go/pkg/embeddings"
)

type scriptedEmbeddedRuntime struct {
	*stubEmbeddedRuntime
	healthResponses   []*localchroma.EmbeddedHealthCheckResponse
	healthErr         error
	heartbeatErr      error
	resetErr          error
	createTenantErr   error
	createDatabaseErr error
	deleteDatabaseErr error

	healthCalls         int
	heartbeatCalls      int
	createTenantCalls   int
	createDatabaseCalls int
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
		dst[key] = value
	}
	return dst
}

func cloneEmbedding(src []float32) []float32 {
	if src == nil {
		return nil
	}
	dst := make([]float32, len(src))
	copy(dst, src)
	return dst
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
		ID:       fmt.Sprintf("mem-col-%d", s.nextCollectionID),
		Name:     request.Name,
		Tenant:   normalizeEmbeddedTenant(request.TenantID),
		Database: normalizeEmbeddedDatabase(request.DatabaseName),
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
	newKey := collectionRuntimeKey(col.Tenant, request.DatabaseName, request.NewName)
	col.Name = request.NewName
	if request.DatabaseName != "" {
		col.Database = request.DatabaseName
	}

	delete(s.collections, oldKey)
	s.collections[newKey] = col
	s.collectionByID[col.ID] = newKey
	return nil
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
	}
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

func (s *scriptedEmbeddedRuntime) CreateDatabase(localchroma.EmbeddedCreateDatabaseRequest) error {
	s.createDatabaseCalls++
	return s.createDatabaseErr
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

	testDB := NewTenant(DefaultTenant).Database("test_db")
	_, err = client.CreateDatabase(ctx, testDB)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 0, runtime.createDatabaseCalls)

	err = client.DeleteDatabase(ctx, testDB)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 0, runtime.deleteDatabaseCalls)
}

func TestEmbeddedCollectionModifyMetadataReturnsExplicitError(t *testing.T) {
	client := &embeddedLocalClient{
		state:           &APIClientV2{},
		embedded:        newScriptedEmbeddedRuntime(),
		collectionState: map[string]*embeddedCollectionState{},
	}
	oldMetadata := NewMetadataFromMap(map[string]interface{}{"old": "value"})
	collection := &embeddedCollection{
		id:       "c1",
		metadata: oldMetadata,
		client:   client,
	}

	err := collection.ModifyMetadata(context.Background(), NewMetadataFromMap(map[string]interface{}{"new": "value"}))
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not support persisting collection metadata updates")
	require.Equal(t, oldMetadata, collection.metadata)
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
