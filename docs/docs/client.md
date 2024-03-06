# Chroma Client

Options:

| Options         | Usage                                                                  | Description                         | Value             | Required                         |
|-----------------|------------------------------------------------------------------------|-------------------------------------|-------------------|----------------------------------|
| basePath        | passed as arg to NewClient                                             | The Chroma server base API.         | valid URL         | Yes                              |
| Tenant          | WithTenant("tenant") as ClientOption to the NewClient                  | The default tenant to use.          | string            | No (default: `default_tenant`)   |
| Database        | WithDatabase("database") as ClientOption to the NewClient              | The default database to use.        | string            | No (default: `default_database`) |
| Debug           | WithDebug(true/false) as ClientOption to the NewClient                 | Enable debug mode.                  | bool              | No (default: `false`)            |
| Default Headers | WithDefaultHeaders(map[string]string) as ClientOption to the NewClient | Set default headers for the client. | map[string]string | No (default: `nil`)              |


!!! note "Tenant and Database"
    
    The tenant and database are only supported for Chroma API version `0.4.15+`.


Creating a new client:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    chroma "github.com/amikos-tech/chroma-go"
)

func main() {
    client, err := chroma.NewClient(
		"http://localhost:8000", 
		chroma.WithTenant("my_tenant"), 
		chroma.WithDatabase("my_db"), 
		chroma.WithDebug(true), 
		chroma.WithDefaultHeaders(map[string]string{"Authorization": "Bearer my token"})
		)
    if err != nil {
        fmt.Printf("Failed to create client: %v", err)
    }
    // do something with client
}
```