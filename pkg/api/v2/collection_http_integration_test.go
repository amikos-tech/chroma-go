//go:build basicv2 && !cloud

package v2

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/amikos-tech/chroma-go/pkg/embeddings"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func TestCollectionAddIntegration(t *testing.T) {
	ctx := context.Background()
	var chromaVersion = "1.0.20"
	var chromaImage = "ghcr.io/chroma-core/chroma"
	if os.Getenv("CHROMA_VERSION") != "" {
		chromaVersion = os.Getenv("CHROMA_VERSION")
	}
	if os.Getenv("CHROMA_IMAGE") != "" {
		chromaImage = os.Getenv("CHROMA_IMAGE")
	}
	cwd, err := os.Getwd()
	require.NoError(t, err)
	mounts := []HostMount{
		{
			Source: filepath.Join(cwd, "v1-config.yaml"),
			Target: "/config.yaml",
		},
	}
	req := testcontainers.ContainerRequest{
		Image:        fmt.Sprintf("%s:%s", chromaImage, chromaVersion),
		ExposedPorts: []string{"8000/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("8000/tcp"),
			wait.ForHTTP("/api/v2/heartbeat").WithStatusCodeMatcher(func(status int) bool {
				return status == 200
			}),
		),
		Env: map[string]string{
			"ALLOW_RESET": "true", // does not work with Chroma v1.0.x
		},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			dockerMounts := make([]mount.Mount, 0)
			for _, mnt := range mounts {
				dockerMounts = append(dockerMounts, mount.Mount{
					Type:   mount.TypeBind,
					Source: mnt.Source,
					Target: mnt.Target,
				})
			}
			hostConfig.Mounts = dockerMounts
		},
	}
	chromaContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, chromaContainer.Terminate(ctx))
	})

	ip, err := chromaContainer.Host(ctx)
	require.NoError(t, err)
	port, err := chromaContainer.MappedPort(ctx, "8000")
	require.NoError(t, err)
	endpoint := fmt.Sprintf("http://%s:%s", ip, port.Port())

	chromaURL := os.Getenv("CHROMA_URL")
	if chromaURL == "" {
		chromaURL = endpoint
	}
	c, err := NewHTTPClient(WithBaseURL(chromaURL), WithDebug())
	require.NoError(t, err)

	t.Cleanup(func() {
		err := c.Close()
		require.NoError(t, err)
	})

	t.Run("add documents", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		count, err := collection.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, count)

		err = collection.Add(ctx, WithIDs("4", "5", "6"), WithTexts("test_document_4", "test_document_5", "test_document_6"))
		require.NoError(t, err)

		err = collection.Add(ctx, WithIDGenerator(NewSHA256Generator()), WithTexts("test_document_7", "test_document_8", "test_document_9"))
		require.NoError(t, err)

		err = collection.Add(ctx, WithIDGenerator(NewULIDGenerator()), WithTexts("test_document_10", "test_document_11", "test_document_12"))
		require.NoError(t, err)
	})

	t.Run("add documents with errors", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		// no ids or id generator
		err = collection.Add(ctx, WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one ID or record is required. Alternatively, an ID generator can be provided")

		err = collection.Add(ctx, WithEmbeddings(embeddings.NewEmbeddingFromFloat32([]float32{1.0, 2.0, 3.0})))
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one ID or record is required. Alternatively, an ID generator can be provided")

		// no documents or embeddings
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()))
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one document or embedding is required")

	})

	t.Run("get documents", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		res, err := collection.Get(ctx, WithIDsGet("1", "2", "3"))
		require.NoError(t, err)
		require.Equal(t, 3, len(res.GetIDs()))

		res, err = collection.Get(ctx, WithIDsGet("1_1", "2_3", "3_0"))
		require.NoError(t, err)
		require.Equal(t, 0, len(res.GetIDs()))

		res, err = collection.Get(ctx, WithIncludeGet(IncludeEmbeddings))
		require.NoError(t, err)
		require.Equal(t, 3, len(res.GetIDs()))

	})

	t.Run("get documents with errors", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)

		// wrong limit
		_, err = collection.Get(ctx, WithLimitGet(-1))
		require.Error(t, err)
		require.Contains(t, err.Error(), "limit must be greater than 0")

		_, err = collection.Get(ctx, WithLimitGet(0))
		require.Error(t, err)
		require.Contains(t, err.Error(), "limit must be greater than 0")

		// wrong offset
		_, err = collection.Get(ctx, WithOffsetGet(-1))
		require.Error(t, err)
		require.Contains(t, err.Error(), "offset must be greater than or equal to 0")
	})

	t.Run("get documents with limit and offset", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		res, err := collection.Get(ctx, WithLimitGet(1), WithOffsetGet(0))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDs()))
	})

	t.Run("get documents where_document regex", func(t *testing.T) {
		if chromaVersion != "latest" {
			cVersion, err := semver.NewVersion(chromaVersion)
			require.NoError(t, err)
			if !semver.MustParse("1.0.8").LessThan(cVersion) {
				t.Skipf("skipping for chroma version %s", cVersion)
			}
		}
		err = c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("this is document 1", "another document", "384km is the distance between the earth and the moon"))
		require.NoError(t, err)
		res, err := collection.Get(ctx, WithWhereDocumentGet(Regex("[0-9]+km")))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDs()))
		require.Equal(t, "384km is the distance between the earth and the moon", res.GetDocuments()[0].ContentString())
	})

	t.Run("get documents with where", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("test_document_1", "test_document_2", "test_document_3"),
			WithMetadatas(
				NewDocumentMetadata(NewStringAttribute("test_key", "doc1")),
				NewDocumentMetadata(NewStringAttribute("test_key", "doc2")),
				NewDocumentMetadata(NewStringAttribute("test_key", "doc3")),
			),
		)
		require.NoError(t, err)
		res, err := collection.Get(ctx, WithWhereGet(EqString("test_key", "doc1")))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDs()))
		require.Equal(t, "test_document_1", res.GetDocuments()[0].ContentString())
	})
	t.Run("count documents", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		count, err := collection.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, count)
	})

	t.Run("delete documents", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		err = collection.Delete(ctx, WithIDsDelete("1", "2", "3"))
		require.NoError(t, err)
		count, err := collection.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("delete documents with errors", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)

		// No Filters
		err = collection.Delete(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one filter is required, ids, where or whereDocument")
	})

	t.Run("upsert documents", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)

		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		err = collection.Upsert(ctx, WithIDs("1", "2", "3"), WithTexts("test_document_1_updated", "test_document_2_updated", "test_document_3_updated"))
		require.NoError(t, err)
		count, err := collection.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, count)
		res, err := collection.Get(ctx, WithIDsGet("1", "2", "3"))
		require.NoError(t, err)
		require.Equal(t, 3, len(res.GetIDs()))
		require.Equal(t, "test_document_1_updated", res.GetDocuments()[0].ContentString())
		require.Equal(t, "test_document_2_updated", res.GetDocuments()[1].ContentString())
		require.Equal(t, "test_document_3_updated", res.GetDocuments()[2].ContentString())
	})

	t.Run("upsert with errors", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		// no ids or id generator
		err = collection.Upsert(ctx, WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one ID or record is required. Alternatively, an ID generator can be provided")

		err = collection.Upsert(ctx, WithEmbeddings(embeddings.NewEmbeddingFromFloat32([]float32{1.0, 2.0, 3.0})))
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one ID or record is required. Alternatively, an ID generator can be provided")

		// no documents or embeddings
		err = collection.Upsert(ctx, WithIDGenerator(NewUUIDGenerator()))
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one document or embedding is required")
	})

	t.Run("update documents", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx,
			"test_collection",
			WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()),
		)
		require.NoError(t, err)
		err = collection.Add(ctx,
			WithIDs("1", "2", "3"),
			WithTexts("test_document_1", "test_document_2", "test_document_3"),
			WithMetadatas(
				NewMetadata(NewStringAttribute("test_key_1", "original")),
				NewMetadata(NewStringAttribute("test_key_2", "original")),
				NewMetadata(NewStringAttribute("test_key_3", "original")),
			),
		)
		require.NoError(t, err)
		err = collection.Update(ctx,
			WithIDsUpdate("1", "2", "3"),
			WithTextsUpdate("test_document_1_updated", "test_document_2_updated", "test_document_3_updated"),
			WithMetadatasUpdate(
				NewMetadata(NewIntAttribute("test_key_1", 1)),
				NewMetadata(RemoveAttribute("test_key_2"), NewStringAttribute("test_key_3", "updated")),
				NewMetadata(NewFloatAttribute("test_key_3", 2.0)),
			),
		)
		require.NoError(t, err)
		count, err := collection.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, count)
		res, err := collection.Get(ctx, WithIDsGet("1", "2", "3"))
		require.NoError(t, err)
		require.Equal(t, 3, len(res.GetIDs()))
		require.Equal(t, "test_document_1_updated", res.GetDocuments()[0].ContentString())
		require.Equal(t, "test_document_2_updated", res.GetDocuments()[1].ContentString())
		require.Equal(t, "test_document_3_updated", res.GetDocuments()[2].ContentString())
		mv1, ok := res.GetMetadatas()[0].GetInt("test_key_1")
		require.True(t, ok)
		require.Equal(t, int64(1), mv1)
		mv2, ok := res.GetMetadatas()[1].GetString("test_key_3")
		require.True(t, ok)
		require.Equal(t, "updated", mv2)
		_, nok := res.GetMetadatas()[1].GetString("test_key_2")
		require.False(t, nok, "test_key_2 should be removed")
		mv3, ok := res.GetMetadatas()[2].GetFloat("test_key_3")
		require.True(t, ok)
		require.Equal(t, 2.0, mv3)
	})

	t.Run("update documents with errors", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		// silent ignore of update
		err = collection.Update(ctx, WithIDsUpdate("1", "2", "3"), WithTextsUpdate("test_document_1_updated", "test_document_2_updated", "test_document_3_updated"))
		require.NoError(t, err)
		count, err := collection.Count(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, count)

		// no ids
		err = collection.Update(ctx, WithTextsUpdate("test_document_1_updated", "test_document_2_updated", "test_document_3_updated"))
		require.Error(t, err)
		fmt.Println("error", err)
		require.Contains(t, err.Error(), "at least one ID or record is required.")

	})

	t.Run("query documents", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		res, err := collection.Query(ctx, WithQueryTexts("test_document_1"))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDGroups()))
		require.Equal(t, 3, len(res.GetIDGroups()[0]))
		require.Equal(t, "test_document_1", res.GetDocumentsGroups()[0][0].ContentString())
	})
	t.Run("query documents with where", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(
			NewUUIDGenerator()),
			WithTexts("test_document_1", "test_document_2", "test_document_3"),
			WithMetadatas(
				NewDocumentMetadata(NewStringAttribute("test_key", "doc1")),
				NewDocumentMetadata(NewStringAttribute("test_key", "doc2")),
				NewDocumentMetadata(NewStringAttribute("test_key", "doc3")),
			),
		)
		require.NoError(t, err)
		res, err := collection.Query(ctx, WithQueryTexts("test_document_1"), WithWhereQuery(EqString("test_key", "doc1")))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDGroups()))
		require.Equal(t, 1, len(res.GetIDGroups()[0]))
		require.Equal(t, "test_document_1", res.GetDocumentsGroups()[0][0].ContentString())
	})
	t.Run("query documents with where document", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		res, err := collection.Query(ctx, WithQueryTexts("test_document_1"), WithWhereDocumentQuery(Contains("test_document_1")))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDGroups()))
		require.Equal(t, 1, len(res.GetIDGroups()[0]))
		require.Equal(t, "test_document_1", res.GetDocumentsGroups()[0][0].ContentString())
	})

	t.Run("query documents with where document - regex", func(t *testing.T) {
		if chromaVersion != "latest" {
			cVersion, err := semver.NewVersion(chromaVersion)
			require.NoError(t, err)
			if !semver.MustParse("1.0.8").LessThan(cVersion) {
				t.Skipf("skipping for chroma version %s", cVersion)
			}
		}
		err = c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("this is document about cats", "123141231", "$@!123115"))
		require.NoError(t, err)
		res, err := collection.Query(ctx, WithQueryTexts("123"), WithWhereDocumentQuery(Regex("^\\d+$")))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDGroups()))
		require.Equal(t, 1, len(res.GetIDGroups()[0]))
		require.Equal(t, "123141231", res.GetDocumentsGroups()[0][0].ContentString())
	})

	t.Run("query documents with include", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		res, err := collection.Query(ctx, WithQueryTexts("test_document_1"), WithWhereDocumentQuery(Contains("test_document_1")), WithIncludeQuery(IncludeMetadatas))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDGroups()))
		require.Equal(t, 1, len(res.GetIDGroups()[0]))
		require.Equal(t, 1, len(res.GetMetadatasGroups()))
		require.Equal(t, 0, len(res.GetDocumentsGroups()))
		require.Equal(t, 0, len(res.GetDistancesGroups()))
	})

	t.Run("query with n_results", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		res, err := collection.Query(ctx, WithQueryTexts("test_document_1"), WithNResults(2))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDGroups()))
		require.Equal(t, 2, len(res.GetIDGroups()[0]))
		require.Equal(t, 1, len(res.GetMetadatasGroups()))
		require.Equal(t, 2, len(res.GetMetadatasGroups()[0]))
		require.Equal(t, 1, len(res.GetDocumentsGroups()))
		require.Equal(t, 2, len(res.GetDocumentsGroups()[0]))
		require.Equal(t, 1, len(res.GetDistancesGroups()))
		require.Equal(t, 2, len(res.GetDistancesGroups()[0]))
	})

	t.Run("query with query embeddings", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDGenerator(NewUUIDGenerator()), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		ef := embeddings.NewConsistentHashEmbeddingFunction()
		embedding, err := ef.EmbedQuery(ctx, "test_document_1")
		require.NoError(t, err)
		res, err := collection.Query(ctx, WithQueryEmbeddings(embedding))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDGroups()))
		require.Equal(t, 3, len(res.GetIDGroups()[0]))
		require.Equal(t, 1, len(res.GetDocumentsGroups()))
		require.Equal(t, 3, len(res.GetDocumentsGroups()[0]))
		require.Equal(t, "test_document_1", res.GetDocumentsGroups()[0][0].ContentString())
	})

	t.Run("query with query IDs", func(t *testing.T) {
		v, err := c.GetVersion(ctx)
		require.NoError(t, err)
		if !strings.HasPrefix(v, "1.") {
			t.Skipf("skipping for chroma version %s", v)
		}
		err = c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		err = collection.Add(ctx, WithIDs("1", "2", "3"), WithTexts("test_document_1", "test_document_2", "test_document_3"))
		require.NoError(t, err)
		res, err := collection.Query(ctx, WithQueryTexts("test_document_1"), WithIDsQuery("1", "3"))
		require.NoError(t, err)
		require.Equal(t, 1, len(res.GetIDGroups()))
		require.Equal(t, 2, len(res.GetIDGroups()[0]))
		require.Equal(t, 1, len(res.GetDocumentsGroups()))
		require.Equal(t, 2, len(res.GetDocumentsGroups()[0]))
		require.Equal(t, "test_document_1", res.GetDocumentsGroups()[0][0].ContentString())
	})

	t.Run("query with errors ", func(t *testing.T) {
		err := c.Reset(ctx)
		require.NoError(t, err)
		collection, err := c.CreateCollection(ctx, "test_collection", WithEmbeddingFunctionCreate(embeddings.NewConsistentHashEmbeddingFunction()))
		require.NoError(t, err)
		// no options
		_, err = collection.Query(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one query embedding or query text is required")

		// empty query texts

		_, err = collection.Query(ctx, WithQueryTexts())
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one query text is required")
		// empty query embeddings
		_, err = collection.Query(ctx, WithQueryEmbeddings())
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one query embedding is required")
		// empty query IDs
		_, err = collection.Query(ctx, WithIDsQuery(), WithQueryTexts("test"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one id is required")

		// empty where
		_, err = collection.Query(ctx, WithWhereQuery(EqString("", "")), WithQueryTexts("test"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid key for $eq, expected non-empty")
	})
}
