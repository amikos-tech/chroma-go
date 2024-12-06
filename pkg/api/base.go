package api

type Tenant interface {
	Name() string
	String() string
}

type Database interface {
	Name() string
	Tenant() Tenant
	String() string
}

type Include string

const (
	IncludeMetadatas  Include = "metadatas"
	IncludeDocuments  Include = "documents"
	IncludeEmbeddings Include = "embeddings"
	IncludeURIs       Include = "uris"
	IncludeIDs        Include = "ids"
)

type TenantBase struct {
	TenantName string `json:"name"`
}

func (t *TenantBase) Name() string {
	return t.TenantName
}

func NewTenant(name string) Tenant {
	return &TenantBase{TenantName: name}
}

func (t *TenantBase) String() string {
	return t.Name()
}

func (t *TenantBase) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.Name() + `"`), nil
}
func NewDefaultTenant() Tenant {
	return NewTenant(DefaultTenant)
}

type DatabaseBase struct {
	DBName string `json:"name"`
	tenant Tenant
}

func (d *DatabaseBase) Name() string {
	return d.DBName
}

func (d *DatabaseBase) Tenant() Tenant {
	return d.tenant
}

func (d *DatabaseBase) String() string {
	return d.Name()
}

func (d *DatabaseBase) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.Name() + `"`), nil
}

func NewDatabase(name string, tenant Tenant) Database {
	return &DatabaseBase{DBName: name, tenant: tenant}
}

func NewDefaultDatabase() Database {
	return NewDatabase(DefaultDatabase, NewDefaultTenant())
}
