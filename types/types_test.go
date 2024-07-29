//go:build basic

package types

import (
	"context"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/oklog/ulid"

	openapi "github.com/amikos-tech/chroma-go/swagger"
	"github.com/amikos-tech/chroma-go/where"
	wheredoc "github.com/amikos-tech/chroma-go/where_document"
)

func TestConsistentHashEmbeddingFunction_EmbedDocuments(t *testing.T) {
	type args struct {
		documents []string
		dim       int
	}
	tests := []struct {
		name    string
		args    args
		want    func(got []*Embedding) bool
		wantErr bool
	}{
		{
			name:    "empty document list, expect empty embeddings list",
			args:    args{documents: []string{}, dim: 10},
			want:    func(got []*Embedding) bool { return len(got) == 0 },
			wantErr: false,
		},
		{
			name:    "with single document, expect single embedding",
			args:    args{documents: []string{"test document"}, dim: 10},
			want:    func(got []*Embedding) bool { return len(got) == 1 },
			wantErr: false,
		},
		{
			name:    "with single document and 100 dims",
			args:    args{documents: []string{"test document"}, dim: 100},
			want:    func(got []*Embedding) bool { return len(got) == 1 && len(*got[0].ArrayOfFloat32) == 100 },
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ef := &ConsistentHashEmbeddingFunction{dim: tt.args.dim}
			got, err := ef.EmbedDocuments(context.TODO(), tt.args.documents)
			if (err != nil) != tt.wantErr {
				t.Errorf("EmbedDocuments() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.want(got) {
				t.Errorf("generateServiceId() got = %v, want %v", got, tt.want(got))
			}
		})
	}
}

func TestConsistentHashEmbeddingFunction_EmbedQuery(t *testing.T) {
	type args struct {
		text string
		dim  int
	}
	tests := []struct {
		name    string
		args    args
		want    func(got *Embedding) bool
		wantErr bool
	}{
		{
			name:    "empty text, expect empty embedding",
			args:    args{text: "", dim: 10},
			want:    func(got *Embedding) bool { return got == nil },
			wantErr: true,
		},
		{
			name:    "with single document, expect single embedding",
			args:    args{text: "test document", dim: 10},
			want:    func(got *Embedding) bool { return len(*got.ArrayOfFloat32) == 10 },
			wantErr: false,
		},
		{
			name:    "with single document and 100 dims",
			args:    args{text: "test document", dim: 100},
			want:    func(got *Embedding) bool { return len(*got.ArrayOfFloat32) == 100 },
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ef := &ConsistentHashEmbeddingFunction{dim: tt.args.dim}
			got, err := ef.EmbedQuery(context.TODO(), tt.args.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("EmbedQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.want(got) {
				t.Errorf("EmbedQuery() got = %v, want %v", got, tt.want(got))
			}
		})
	}
}

func TestConsistentHashEmbeddingFunction_EmbedDocuments1(t *testing.T) {
	type fields struct {
		dim int
	}
	type args struct {
		ctx       context.Context
		documents []string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*Embedding
		wantErr bool
	}{
		{name: "empty document list, expect empty embeddings list",
			fields:  fields{dim: 10},
			args:    args{documents: []string{}, ctx: context.TODO()},
			want:    []*Embedding{},
			wantErr: false,
		},
		{name: "with single document, expect single embedding",
			fields:  fields{dim: 10},
			args:    args{documents: []string{"test document"}, ctx: context.TODO()},
			want:    []*Embedding{{ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334}, ArrayOfInt32: nil}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ConsistentHashEmbeddingFunction{
				dim: tt.fields.dim,
			}
			got, err := e.EmbedDocuments(tt.args.ctx, tt.args.documents)
			if (err != nil) != tt.wantErr {
				t.Errorf("EmbedDocuments() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EmbedDocuments() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConsistentHashEmbeddingFunction_EmbedQuery1(t *testing.T) {
	type fields struct {
		dim int
	}
	type args struct {
		ctx      context.Context
		document string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Embedding
		wantErr bool
	}{
		{name: "empty document list, expect empty embeddings list",
			fields:  fields{dim: 10},
			args:    args{document: "", ctx: context.TODO()},
			want:    nil,
			wantErr: true,
		},
		{name: "with single document, expect single embedding",
			fields:  fields{dim: 10},
			args:    args{document: "test document", ctx: context.TODO()},
			want:    NewEmbeddingFromFloat32([]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334}),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ConsistentHashEmbeddingFunction{
				dim: tt.fields.dim,
			}
			got, err := e.EmbedQuery(tt.args.ctx, tt.args.document)
			if (err != nil) != tt.wantErr {
				t.Errorf("EmbedQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EmbedQuery() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConsistentHashEmbeddingFunction_EmbedRecords(t *testing.T) {
	type fields struct {
		dim int
	}
	type args struct {
		ctx     context.Context
		records []*Record
		force   bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "empty document list, expect empty embeddings list",
			fields: fields{dim: 10},
			args: args{
				ctx:     context.TODO(),
				records: []*Record{},
			},
			wantErr: false,
		},
		{
			name:   "with single document, expect single embedding",
			fields: fields{dim: 10},
			args: args{
				ctx: context.TODO(),
				records: []*Record{
					{
						Document: "test document",
						ID:       "1",
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "record without doc content",
			fields: fields{dim: 10},
			args: args{
				ctx: context.TODO(),
				records: []*Record{
					{
						ID: "1",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ConsistentHashEmbeddingFunction{
				dim: tt.fields.dim,
			}
			if err := e.EmbedRecords(tt.args.ctx, tt.args.records, tt.args.force); (err != nil) != tt.wantErr {
				t.Errorf("EmbedRecords() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmbedRecordsDefaultImpl(t *testing.T) {
	type args struct {
		e       EmbeddingFunction
		ctx     context.Context
		records []*Record
		force   bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "empty document list, expect empty embeddings list",
			args: args{
				e:       &ConsistentHashEmbeddingFunction{dim: 10},
				ctx:     context.TODO(),
				records: []*Record{},
				force:   false,
			},
			wantErr: false,
		},
		{
			name: "with single document, expect single embedding",
			args: args{
				e:   &ConsistentHashEmbeddingFunction{dim: 10},
				ctx: context.TODO(),
				records: []*Record{
					{
						Document: "test document",
					},
				},
				force: false,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := EmbedRecordsDefaultImpl(tt.args.e, tt.args.ctx, tt.args.records, tt.args.force); (err != nil) != tt.wantErr {
				t.Errorf("EmbedRecordsDefaultImpl() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmbedding_GetFloat32(t *testing.T) {
	type fields struct {
		ArrayOfFloat32 *[]float32
		ArrayOfInt32   *[]int32
	}
	tests := []struct {
		name   string
		fields fields
		want   *[]float32
	}{
		{
			name: "empty embedding",
			fields: fields{
				ArrayOfFloat32: &[]float32{},
				ArrayOfInt32:   &[]int32{},
			},
			want: &[]float32{},
		},
		{
			name: "embedding with 10 dims",
			fields: fields{
				ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334},
				ArrayOfInt32:   &[]int32{},
			},
			want: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Embedding{
				ArrayOfFloat32: tt.fields.ArrayOfFloat32,
				ArrayOfInt32:   tt.fields.ArrayOfInt32,
			}
			if got := e.GetFloat32(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetFloat32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmbedding_GetInt32(t *testing.T) {
	type fields struct {
		ArrayOfFloat32 *[]float32
		ArrayOfInt32   *[]int32
	}
	tests := []struct {
		name   string
		fields fields
		want   *[]int32
	}{
		{
			name: "empty embedding",
			fields: fields{
				ArrayOfFloat32: &[]float32{},
				ArrayOfInt32:   &[]int32{},
			},
			want: &[]int32{},
		},
		{
			name: "embedding with 10 dims",
			fields: fields{
				ArrayOfFloat32: &[]float32{},
				ArrayOfInt32:   &[]int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			},
			want: &[]int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Embedding{
				ArrayOfFloat32: tt.fields.ArrayOfFloat32,
				ArrayOfInt32:   tt.fields.ArrayOfInt32,
			}
			if got := e.GetInt32(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetInt32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmbedding_IsDefined(t *testing.T) {
	type fields struct {
		ArrayOfFloat32 *[]float32
		ArrayOfInt32   *[]int32
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "empty embedding",
			fields: fields{
				ArrayOfFloat32: &[]float32{},
				ArrayOfInt32:   &[]int32{},
			},
			want: false,
		},
		{
			name: "embedding with 10 dims float32",
			fields: fields{
				ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334},
				ArrayOfInt32:   &[]int32{},
			},
			want: true,
		},
		{
			name: "embedding with 10 dims int32",
			fields: fields{
				ArrayOfFloat32: &[]float32{},
				ArrayOfInt32:   &[]int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Embedding{
				ArrayOfFloat32: tt.fields.ArrayOfFloat32,
				ArrayOfInt32:   tt.fields.ArrayOfInt32,
			}
			if got := e.IsDefined(); got != tt.want {
				t.Errorf("IsDefined() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmbedding_ToAPI(t *testing.T) {
	type fields struct {
		ArrayOfFloat32 *[]float32
		ArrayOfInt32   *[]int32
	}
	tests := []struct {
		name   string
		fields fields
		want   openapi.EmbeddingsInner
	}{
		{
			name: "empty embedding",
			fields: fields{
				ArrayOfFloat32: &[]float32{},
				ArrayOfInt32:   &[]int32{},
			},
			want: openapi.EmbeddingsInner{
				ArrayOfFloat32: &[]float32{},
				ArrayOfInt32:   &[]int32{},
			},
		},
		{
			name: "embedding with 10 dims float32",
			fields: fields{
				ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334},
				ArrayOfInt32:   &[]int32{},
			},
			want: openapi.EmbeddingsInner{
				ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334},
				ArrayOfInt32:   &[]int32{},
			},
		},
		{
			name: "embedding with 10 dims int32",
			fields: fields{
				ArrayOfFloat32: &[]float32{},
				ArrayOfInt32:   &[]int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			},
			want: openapi.EmbeddingsInner{
				ArrayOfFloat32: &[]float32{},
				ArrayOfInt32:   &[]int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Embedding{
				ArrayOfFloat32: tt.fields.ArrayOfFloat32,
				ArrayOfInt32:   tt.fields.ArrayOfInt32,
			}
			if got := e.ToAPI(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToAPI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewConsistentHashEmbeddingFunction(t *testing.T) {
	tests := []struct {
		name string
		want EmbeddingFunction
	}{
		{
			name: "default consistent hash embedding function",
			want: &ConsistentHashEmbeddingFunction{
				dim: 378,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewConsistentHashEmbeddingFunction(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewConsistentHashEmbeddingFunction() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewEmbeddingFromAPI(t *testing.T) {
	type args struct {
		apiEmbedding openapi.EmbeddingsInner
	}
	tests := []struct {
		name string
		args args
		want *Embedding
	}{
		{
			name: "empty embedding",
			args: args{apiEmbedding: openapi.EmbeddingsInner{
				ArrayOfFloat32: &[]float32{},
				ArrayOfInt32:   &[]int32{},
			},
			},
			want: &Embedding{
				ArrayOfFloat32: &[]float32{},
				ArrayOfInt32:   &[]int32{},
			},
		},
		{
			name: "embedding with 10 dims float32",
			args: args{apiEmbedding: openapi.EmbeddingsInner{
				ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334},
				ArrayOfInt32:   &[]int32{},
			}},
			want: &Embedding{
				ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334},
				ArrayOfInt32:   &[]int32{},
			},
		},
		{
			name: "embedding with 10 dims int32",
			args: args{apiEmbedding: openapi.EmbeddingsInner{
				ArrayOfFloat32: &[]float32{},
				ArrayOfInt32:   &[]int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			}},
			want: &Embedding{
				ArrayOfFloat32: &[]float32{},
				ArrayOfInt32:   &[]int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewEmbeddingFromAPI(tt.args.apiEmbedding); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEmbeddingFromAPI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewEmbeddingFromFloat32(t *testing.T) {
	type args struct {
		embedding []float32
	}
	tests := []struct {
		name string
		args args
		want *Embedding
	}{
		{
			name: "empty embedding",
			args: args{embedding: []float32{}},
			want: &Embedding{
				ArrayOfFloat32: &[]float32{},
			},
		},
		{
			name: "embedding with 10 dims",
			args: args{embedding: []float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334}},
			want: &Embedding{
				ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewEmbeddingFromFloat32(tt.args.embedding); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEmbeddingFromFloat32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewEmbeddingFromInt32(t *testing.T) {
	type args struct {
		embedding []int32
	}
	tests := []struct {
		name string
		args args
		want *Embedding
	}{
		{
			name: "empty embedding",
			args: args{embedding: []int32{}},
			want: &Embedding{
				ArrayOfInt32: &[]int32{},
			},
		},
		{
			name: "embedding with 10 dims",
			args: args{embedding: []int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}},
			want: &Embedding{
				ArrayOfInt32: &[]int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewEmbeddingFromInt32(tt.args.embedding); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEmbeddingFromInt32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewEmbedding(t *testing.T) {
	type args struct {
		embeddings []interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    *Embedding
		wantErr bool
	}{
		{
			name: "empty embedding",
			args: args{embeddings: []interface{}{}},
			want: &Embedding{
				ArrayOfFloat32: &[]float32{},
				ArrayOfInt32:   &[]int32{},
			},
			wantErr: false,
		},
		{
			name: "embedding with 10 dims float32",
			args: args{
				embeddings: func() []interface{} {
					var floats = []float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334}
					var interfaces = make([]interface{}, len(floats))
					for i, v := range floats {
						interfaces[i] = v
					}
					return interfaces
				}(),
			},
			want: &Embedding{
				ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334},
				ArrayOfInt32:   &[]int32{},
			},
			wantErr: false,
		},
		{
			name: "embedding with 10 dims int32",
			args: args{
				embeddings: func() []interface{} {
					var ints = []int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
					var interfaces = make([]interface{}, len(ints))
					for i, v := range ints {
						interfaces[i] = v
					}
					return interfaces
				}(),
			},
			want: &Embedding{
				ArrayOfFloat32: &[]float32{},
				ArrayOfInt32:   &[]int32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEmbedding(tt.args.embeddings)
			if !tt.wantErr {
				require.NoError(t, err)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEmbedding() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			require.True(t, got.Compare(tt.want), "NewEmbedding() got = %v, want %v", got, tt.want)
		})
	}
}

func TestNewEmbeddings(t *testing.T) {
	type args struct {
		embeddings []interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []*Embedding
		wantErr bool
	}{
		{
			name:    "empty embedding",
			args:    args{embeddings: []interface{}{}},
			want:    make([]*Embedding, 0),
			wantErr: false,
		},
		{
			name: "float32 embeddings",
			args: args{
				embeddings: []interface{}{
					[]interface{}{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334},
					[]interface{}{0.26666668, 0.5, .2, 0.46666667, 0.2, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.3},
				},
			},
			want: []*Embedding{
				{
					ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334},
				},
				{
					ArrayOfFloat32: &[]float32{0.26666668, 0.5, .2, 0.46666667, 0.2, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.3},
				},
			},
			wantErr: false,
		},
		{
			name: "int32 embeddings",
			args: args{
				embeddings: []interface{}{
					[]interface{}{1, 2, 3, 4, 5, 6},
					[]interface{}{6, 5, 4, 3, 2, 1},
				},
			},
			want: []*Embedding{
				{
					ArrayOfInt32: &[]int32{1, 2, 3, 4, 5, 6},
				},
				{
					ArrayOfInt32: &[]int32{6, 5, 4, 3, 2, 1},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewEmbeddings(tt.args.embeddings)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEmbeddings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !CompareEmbeddings(got, tt.want) {
				t.Errorf("NewEmbeddings() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewEmbeddingsFromFloat32(t *testing.T) {
	type args struct {
		embeddings [][]float32
	}
	tests := []struct {
		name string
		args args
		want []*Embedding
	}{
		{
			name: "empty embedding",
			args: args{embeddings: [][]float32{}},
			want: []*Embedding{},
		},
		{
			name: "embedding with 10 dims float32",
			args: args{embeddings: [][]float32{{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334}}},
			want: []*Embedding{{ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334}}},
		},
		{
			name: "embedding with 10 dims ints",
			args: args{embeddings: [][]float32{{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}}},
			want: []*Embedding{{ArrayOfFloat32: &[]float32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewEmbeddingsFromFloat32(tt.args.embeddings); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEmbeddingsFromFloat32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSHA256Generator(t *testing.T) {
	tests := []struct {
		name string
		want *SHA256Generator
	}{
		{
			name: "default sha256 generator",
			want: NewSHA256Generator(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewSHA256Generator(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSHA256Generator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewULIDGenerator(t *testing.T) {
	tests := []struct {
		name string
		want *ULIDGenerator
	}{
		{
			name: "default ulid generator",
			want: NewULIDGenerator(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewULIDGenerator(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewULIDGenerator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewUUIDGenerator(t *testing.T) {
	tests := []struct {
		name string
		want *UUIDGenerator
	}{
		{
			name: "default uuid generator",
			want: NewUUIDGenerator(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewUUIDGenerator(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUUIDGenerator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSHA256Generator_Generate(t *testing.T) {
	type args struct {
		document string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty document",
			args: args{document: ""},
			want: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name: "non-empty document",
			args: args{document: "test document"},
			want: "4837479125758add3ba4c99153bb855c8519f86a7f672b26b155bea6adcbb41a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SHA256Generator{}
			if got := s.Generate(tt.args.document); got != tt.want {
				t.Errorf("Generate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToAPIEmbeddings(t *testing.T) {
	type args struct {
		embeddings []*Embedding
	}
	tests := []struct {
		name string
		args args
		want []openapi.EmbeddingsInner
	}{
		{
			name: "empty embeddings",
			args: args{embeddings: []*Embedding{}},
			want: make([]openapi.EmbeddingsInner, 0),
		},
		{
			name: "single embedding",
			args: args{embeddings: []*Embedding{{ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334}}}},
			want: []openapi.EmbeddingsInner{{ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334}}},
		},
		{
			name: "multiple embeddings",
			args: args{embeddings: []*Embedding{
				{ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334}},
				{ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334}},
			}},
			want: []openapi.EmbeddingsInner{
				{ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334}},
				{ArrayOfFloat32: &[]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToAPIEmbeddings(tt.args.embeddings); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToAPIEmbeddings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToDistanceFunction(t *testing.T) {
	type args struct {
		str any
	}
	tests := []struct {
		name    string
		args    args
		want    DistanceFunction
		wantErr bool
	}{
		{
			name:    "empty string, should get default L2",
			args:    args{str: ""},
			want:    L2,
			wantErr: false,
		},
		{
			name:    "L2 string, should get L2",
			args:    args{str: "L2"},
			want:    L2,
			wantErr: false,
		},
		{
			name:    "COSINE all caps string, should get COSINE",
			args:    args{str: "COSINE"},
			want:    COSINE,
			wantErr: false,
		},
		{
			name:    "COSINE lowecase string, should get COSINE",
			args:    args{str: "cosine"},
			want:    COSINE,
			wantErr: false,
		},
		{
			name:    "IP string, should get IP",
			args:    args{str: "ip"},
			want:    IP,
			wantErr: false,
		},
		{
			name:    "invliad string, should fail",
			args:    args{str: "L1"},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToDistanceFunction(tt.args.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToDistanceFunction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ToDistanceFunction() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestULIDGenerator_Generate(t *testing.T) {
	type args struct {
		in0 string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "empty document",
			args: args{in0: ""},
		},
		{
			name: "non-empty document",
			args: args{in0: "test document"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &ULIDGenerator{}
			if got, err := ulid.Parse(u.Generate(tt.args.in0)); err != nil {
				t.Errorf("Generate() did not produce valid ULID: %v", got)
			}
		})
	}
}

func TestUUIDGenerator_Generate(t *testing.T) {
	type args struct {
		in0 string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "empty document",
			args: args{in0: ""},
		},
		{
			name: "non-empty document",
			args: args{in0: "test document"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UUIDGenerator{}
			if got, err := uuid.Parse(u.Generate(tt.args.in0)); err != nil {
				t.Errorf("Generate() - created an invalid uuid %v", got)
			}
		})
	}
}

func TestWithIds(t *testing.T) {
	type args struct {
		ids          []string
		queryBuilder *CollectionQueryBuilder
	}
	tests := []struct {
		name string
		args args
		want *CollectionQueryBuilder
	}{
		{
			name: "with ids",
			args: args{ids: []string{"1", "2", "3"}, queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Ids: []string{"1", "2", "3"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _ = WithIds(tt.args.ids)(tt.args.queryBuilder); !reflect.DeepEqual(tt.args.queryBuilder, tt.want) {
				t.Errorf("WithIds() = %v, want %v", tt.args.queryBuilder, tt.want)
			}
		})
	}
}

func TestWithInclude(t *testing.T) {
	type args struct {
		include      []QueryEnum
		queryBuilder *CollectionQueryBuilder
	}
	tests := []struct {
		name string
		args args
		want *CollectionQueryBuilder
	}{
		{
			name: "with include",
			args: args{include: []QueryEnum{QueryEnum(IDistances)}, queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Include: []QueryEnum{QueryEnum(IDistances)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _ = WithInclude(tt.args.include...)(tt.args.queryBuilder); !reflect.DeepEqual(tt.args.queryBuilder, tt.want) {
				t.Errorf("WithInclude() = %v, want %v", tt.args.queryBuilder, tt.want)
			}
		})
	}
}

func TestWithLimit(t *testing.T) {
	type args struct {
		limit        int32
		queryBuilder *CollectionQueryBuilder
	}
	tests := []struct {
		name string
		args args
		want *CollectionQueryBuilder
	}{
		{
			name: "with limit",
			args: args{limit: 10, queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Limit: 10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _ = WithLimit(tt.args.limit)(tt.args.queryBuilder); !reflect.DeepEqual(tt.args.queryBuilder, tt.want) {
				t.Errorf("WithLimit() = %v, want %v", tt.args.queryBuilder, tt.want)
			}
		})
	}
}

func TestWithNResults(t *testing.T) {
	type args struct {
		nResults     int32
		queryBuilder *CollectionQueryBuilder
	}
	tests := []struct {
		name string
		args args
		want *CollectionQueryBuilder
	}{
		{
			name: "with n results",
			args: args{nResults: 10, queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{NResults: 10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _ = WithNResults(tt.args.nResults)(tt.args.queryBuilder); !reflect.DeepEqual(tt.args.queryBuilder, tt.want) {
				t.Errorf("WithNResults() = %v, want %v", tt.args.queryBuilder, tt.want)
			}
		})
	}
}

func TestWithOffset(t *testing.T) {
	type args struct {
		offset       int32
		queryBuilder *CollectionQueryBuilder
	}
	tests := []struct {
		name string
		args args
		want *CollectionQueryBuilder
	}{
		{
			name: "with offset",
			args: args{offset: 10, queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Offset: 10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _ = WithOffset(tt.args.offset)(tt.args.queryBuilder); !reflect.DeepEqual(tt.args.queryBuilder, tt.want) {
				t.Errorf("WithOffset() = %v, want %v", tt.args.queryBuilder, tt.want)
			}
		})
	}
}

func TestWithQueryEmbedding(t *testing.T) {
	type args struct {
		queryEmbedding *Embedding
		queryBuilder   *CollectionQueryBuilder
	}
	tests := []struct {
		name string
		args args
		want *CollectionQueryBuilder
	}{
		{
			name: "with query embedding",
			args: args{queryEmbedding: NewEmbeddingFromFloat32([]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334}), queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{QueryEmbeddings: NewEmbeddingsFromFloat32([][]float32{{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334}})},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _ = WithQueryEmbedding(tt.args.queryEmbedding)(tt.args.queryBuilder); !reflect.DeepEqual(tt.args.queryBuilder, tt.want) {
				t.Errorf("WithQueryEmbedding() = %v, want %v", tt.args.queryBuilder, tt.want)
			}
		})
	}
}

func TestWithQueryEmbeddings(t *testing.T) {
	type args struct {
		queryEmbeddings []*Embedding
		queryBuilder    *CollectionQueryBuilder
	}
	tests := []struct {
		name string
		args args
		want *CollectionQueryBuilder
	}{
		{
			name: "with query embeddings",
			args: args{queryEmbeddings: []*Embedding{NewEmbeddingFromFloat32([]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334}), NewEmbeddingFromFloat32([]float32{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334})}, queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{
				QueryEmbeddings: NewEmbeddingsFromFloat32([][]float32{
					{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334},
					{0.26666668, 0.53333336, .2, 0.46666667, 0.26666668, 0.46666667, 0.6, 0.06666667, 0.13333334, 0.33333334},
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _ = WithQueryEmbeddings(tt.args.queryEmbeddings)(tt.args.queryBuilder); !reflect.DeepEqual(tt.args.queryBuilder, tt.want) {
				t.Errorf("WithQueryEmbeddings() = %v, want %v", tt.args.queryBuilder, tt.want)
			}
		})
	}
}

func TestWithQueryText(t *testing.T) {
	type args struct {
		queryText    string
		queryBuilder *CollectionQueryBuilder
	}
	tests := []struct {
		name string
		args args
		want *CollectionQueryBuilder
	}{
		{
			name: "with query text",
			args: args{queryText: "test", queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{QueryTexts: []string{"test"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _ = WithQueryText(tt.args.queryText)(tt.args.queryBuilder); !reflect.DeepEqual(tt.args.queryBuilder, tt.want) {
				t.Errorf("WithQueryText() = %v, want %v", tt.args.queryBuilder, tt.want)
			}
		})
	}
}

func TestWithQueryTexts(t *testing.T) {
	type args struct {
		queryTexts   []string
		queryBuilder *CollectionQueryBuilder
	}
	tests := []struct {
		name string
		args args
		want *CollectionQueryBuilder
	}{
		{
			name: "with query texts",
			args: args{queryTexts: []string{"test1", "test2"}, queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{QueryTexts: []string{"test1", "test2"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _ = WithQueryTexts(tt.args.queryTexts)(tt.args.queryBuilder); !reflect.DeepEqual(tt.args.queryBuilder, tt.want) {
				t.Errorf("WithQueryTexts() = %v, want %v", tt.args.queryBuilder, tt.want)
			}
		})
	}
}

func TestWithWhere(t *testing.T) {
	type args struct {
		operation    where.WhereOperation
		queryBuilder *CollectionQueryBuilder
	}
	tests := []struct {
		name string
		args args
		want *CollectionQueryBuilder
	}{
		{
			name: "with where for eq int",
			args: args{operation: where.Eq("test", 1), queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Where: map[string]interface{}{"test": map[string]interface{}{"$eq": 1}}},
		},
		{
			name: "with where for eq string",
			args: args{operation: where.Eq("test", "my string"), queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Where: map[string]interface{}{"test": map[string]interface{}{"$eq": "my string"}}},
		},
		{
			name: "with where for eq bool",
			args: args{operation: where.Eq("test", true), queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Where: map[string]interface{}{"test": map[string]interface{}{"$eq": true}}},
		},
		{
			name: "with where for eq bool",
			args: args{operation: where.Eq("test", float32(101.99)), queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Where: map[string]interface{}{"test": map[string]interface{}{"$eq": float32(101.99)}}},
		},
		{
			name: "with where for ne string",
			args: args{operation: where.Ne("test", "not equal"), queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Where: map[string]interface{}{"test": map[string]interface{}{"$ne": "not equal"}}},
		},
		{
			name: "with where for gt int",
			args: args{operation: where.Gt("test", 10), queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Where: map[string]interface{}{"test": map[string]interface{}{"$gt": 10}}},
		},
		{
			name: "with where for gte float",
			args: args{operation: where.Gte("test", float32(10.1)), queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Where: map[string]interface{}{"test": map[string]interface{}{"$gte": float32(10.1)}}},
		},
		{
			name: "with where for lt int",
			args: args{operation: where.Lt("test", 10), queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Where: map[string]interface{}{"test": map[string]interface{}{"$lt": 10}}},
		},
		{
			name: "with where for lte float",
			args: args{operation: where.Lte("test", float32(10.9)), queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Where: map[string]interface{}{"test": map[string]interface{}{"$lte": float32(10.9)}}},
		},
		{
			name: "with where for in list of strings",
			args: args{operation: where.In("test", []interface{}{"one", "two"}), queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Where: map[string]interface{}{"test": map[string]interface{}{"$in": []interface{}{"one", "two"}}}},
		},
		{
			name: "with where for nin list of ints",
			args: args{operation: where.Nin("test", []interface{}{1, 2, 3}), queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Where: map[string]interface{}{"test": map[string]interface{}{"$nin": []interface{}{1, 2, 3}}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _ = WithWhere(tt.args.operation)(tt.args.queryBuilder); !reflect.DeepEqual(tt.args.queryBuilder, tt.want) {
				t.Errorf("WithWhere() = %v, want %v", tt.args.queryBuilder, tt.want)
			}
		})
	}
}

func TestWithWhereDocument(t *testing.T) {
	type args struct {
		operation    wheredoc.WhereDocumentOperation
		queryBuilder *CollectionQueryBuilder
	}
	tests := []struct {
		name string
		args args
		want *CollectionQueryBuilder
	}{
		{
			name: "with where document",
			args: args{operation: wheredoc.Contains("test"), queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{WhereDocument: map[string]interface{}{"$contains": "test"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _ = WithWhereDocument(tt.args.operation)(tt.args.queryBuilder); !reflect.DeepEqual(tt.args.queryBuilder, tt.want) {
				t.Errorf("WithWhereDocument() = %v, want %v", tt.args.queryBuilder, tt.want)
			}
		})
	}
}

func TestWithWhereDocumentMap(t *testing.T) {
	type args struct {
		where        map[string]interface{}
		queryBuilder *CollectionQueryBuilder
	}
	tests := []struct {
		name string
		args args
		want *CollectionQueryBuilder
	}{
		{
			name: "with where document map",
			args: args{where: map[string]interface{}{"test": "test"}, queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{WhereDocument: map[string]interface{}{"test": "test"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _ = WithWhereDocumentMap(tt.args.where)(tt.args.queryBuilder); !reflect.DeepEqual(tt.args.queryBuilder, tt.want) {
				t.Errorf("WithWhereDocumentMap() = %v, want %v", tt.args.queryBuilder, tt.want)
			}
		})
	}
}

func TestWithWhereMap(t *testing.T) {
	type args struct {
		where        map[string]interface{}
		queryBuilder *CollectionQueryBuilder
	}
	tests := []struct {
		name string
		args args
		want *CollectionQueryBuilder
	}{
		{
			name: "with where map",
			args: args{where: map[string]interface{}{"test": "test"}, queryBuilder: &CollectionQueryBuilder{}},
			want: &CollectionQueryBuilder{Where: map[string]interface{}{"test": "test"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _ = WithWhereMap(tt.args.where)(tt.args.queryBuilder); !reflect.DeepEqual(tt.args.queryBuilder, tt.want) {
				t.Errorf("WithWhereMap() = %v, want %v", tt.args.queryBuilder, tt.want)
			}
		})
	}
}
