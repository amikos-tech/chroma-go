package api

type WhereDocumentFilter interface {
	Contains(content string) WhereDocumentFilter
	NotContains(content string) WhereDocumentFilter
	Or(clauses ...WhereDocumentFilter) WhereDocumentFilter
	And(clauses ...WhereDocumentFilter) WhereDocumentFilter
	Validate() error
	String() string
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(b []byte) error
}
