package v2

import "github.com/amikos-tech/chroma-go/pkg/embeddings"

// Result provides simplified access to query and get results.
// This interface provides cleaner method names without the "Get" prefix.
type Result interface {
	// IDs returns the document IDs in the result
	IDs() DocumentIDs
	// Documents returns the documents in the result
	Documents() Documents
	// Metadatas returns the metadata of documents in the result
	Metadatas() DocumentMetadatas
	// Embeddings returns the embeddings in the result
	Embeddings() embeddings.Embeddings
	// Count returns the number of documents in the result
	Count() int
}

// QueryResults provides simplified access to query results with distances.
type QueryResults interface {
	Result
	// Distances returns the distances for each result
	Distances() [][]float32
	// DocumentGroups returns grouped documents for batch queries
	DocumentGroups() []Documents
	// MetadataGroups returns grouped metadata for batch queries
	MetadataGroups() []DocumentMetadatas
}

// SimplifiedGetResult wraps GetResultImpl to implement the simplified Result interface
type SimplifiedGetResult struct {
	*GetResultImpl
}

func (r *SimplifiedGetResult) IDs() DocumentIDs {
	return r.Ids
}

func (r *SimplifiedGetResult) Documents() Documents {
	return r.GetResultImpl.Documents
}

func (r *SimplifiedGetResult) Metadatas() DocumentMetadatas {
	return r.GetResultImpl.Metadatas
}

func (r *SimplifiedGetResult) Embeddings() embeddings.Embeddings {
	return r.GetResultImpl.Embeddings
}

// SimplifiedQueryResult wraps QueryResult to implement the simplified QueryResults interface
type SimplifiedQueryResult struct {
	QueryResult
}

func (r *SimplifiedQueryResult) IDs() DocumentIDs {
	if ids := r.GetIDGroups(); len(ids) > 0 {
		return ids[0]
	}
	return nil
}

func (r *SimplifiedQueryResult) Documents() Documents {
	if docs := r.GetDocumentsGroups(); len(docs) > 0 {
		return docs[0]
	}
	return nil
}

func (r *SimplifiedQueryResult) Metadatas() DocumentMetadatas {
	if metas := r.GetMetadatasGroups(); len(metas) > 0 {
		return metas[0]
	}
	return nil
}

func (r *SimplifiedQueryResult) Embeddings() embeddings.Embeddings {
	if embs := r.GetEmbeddingsGroups(); len(embs) > 0 {
		return embs[0]
	}
	return nil
}

func (r *SimplifiedQueryResult) Distances() [][]float32 {
	distances := r.GetDistancesGroups()
	result := make([][]float32, len(distances))
	for i, group := range distances {
		result[i] = make([]float32, len(group))
		for j, d := range group {
			result[i][j] = float32(d)
		}
	}
	return result
}

func (r *SimplifiedQueryResult) DocumentGroups() []Documents {
	return r.GetDocumentsGroups()
}

func (r *SimplifiedQueryResult) MetadataGroups() []DocumentMetadatas {
	return r.GetMetadatasGroups()
}

func (r *SimplifiedQueryResult) Count() int {
	if ids := r.GetIDGroups(); len(ids) > 0 && len(ids[0]) > 0 {
		return len(ids[0])
	}
	return 0
}

// Helper function to convert GetResult to simplified Result interface
func AsResult(gr GetResult) Result {
	if impl, ok := gr.(*GetResultImpl); ok {
		return &SimplifiedGetResult{GetResultImpl: impl}
	}
	// Wrap other implementations if needed
	return &resultWrapper{GetResult: gr}
}

// Helper function to convert QueryResult to simplified QueryResults interface
func AsQueryResults(qr QueryResult) QueryResults {
	return &SimplifiedQueryResult{QueryResult: qr}
}

// resultWrapper wraps a GetResult to implement the simplified Result interface
type resultWrapper struct {
	GetResult
}

func (w *resultWrapper) IDs() DocumentIDs {
	return w.GetIDs()
}

func (w *resultWrapper) Documents() Documents {
	return w.GetDocuments()
}

func (w *resultWrapper) Metadatas() DocumentMetadatas {
	return w.GetMetadatas()
}

func (w *resultWrapper) Embeddings() embeddings.Embeddings {
	return w.GetEmbeddings()
}
