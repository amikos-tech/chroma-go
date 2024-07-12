# Chroma Client

Options:

| Options           | Usage                                   | Description                                                                             | Value                      | Required                         |
|-------------------|-----------------------------------------|-----------------------------------------------------------------------------------------|----------------------------|----------------------------------|
| basePath          | passed as arg to NewClient              | The Chroma server base API.                                                             | Non-empty valid URL string | Yes                              |
| Tenant            | `WithTenant("tenant")`                  | The default tenant to use.                                                              | `string`                   | No (default: `default_tenant`)   |
| Database          | `WithDatabase("database")`              | The default database to use.                                                            | `string`                   | No (default: `default_database`) |
| Debug             | `WithDebug(true/false)`                 | Enable debug mode.                                                                      | `bool`                     | No (default: `false`)            |
| Default Headers   | `WithDefaultHeaders(map[string]string)` | Set default headers for the client.                                                     | `map[string]string`        | No (default: `nil`)              |
| SSL Cert          | `WithSSLCert("path/to/cert.pem")`       | Set the path to the SSL certificate.                                                    | valid path to SSL cert.    | No (default: Not Set)            |
| Insecure          | `WithInsecure()`                        | Disable SSL certificate verification                                                    |                            | No (default: Not Set)            |
| Custom HttpClient | `WithHTTPClient(http.Client)`           | Set a custom http client. If this is set then SSL Cert and Insecure options are ignore. | `*http.Client`             | No (default: Default HTTPClient) |

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
		chroma.WithDefaultHeaders(map[string]string{"Authorization": "Bearer my token"}),
		chroma.WithSSLCert("path/to/cert.pem"),
	)
	if err != nil {
		fmt.Printf("Failed to create client: %v", err)
	}
	// do something with client
}
```