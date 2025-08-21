# Chroma Client

## Chroma Cloud Client (v0.2.0+)

Options:

| Options           | Usage                                        | Description                                                                                            | Value               | Required                                                   |
|-------------------|----------------------------------------------|--------------------------------------------------------------------------------------------------------|---------------------|------------------------------------------------------------|
| CloudAPIKey       | `WithCloudAPIKey("api_key")`                 | Set Chroma Cloud API Key                                                                               | `string`            | Yes (default: uses `CHROMA_API_KEY` env var if available)  |
| Tenant            | `WithTenant("tenant")`                       | The default tenant to use.                                                                             | `string`            | Yes (default: uses `CHROMA_TENANT` env var if available)   |
| Database          | `WithDatabaseAndTenant("database","tenant")` | The default database and tenant (overrides any tenant set with Tenant). Option precedence is observed! | `string`            | Yes (default: uses `CHROMA_DATABASE` env var if available) |
| Debug             | `WithDebug(true/false)`                      | Enable debug mode, printing http requests to stdout (sensitive information is sanitized)               | `bool`              | No (default: `false`)                                      |
| Default Headers   | `WithDefaultHeaders(map[string]string)`      | Set default HTTP headers for the client. These headers are sent with every request.                    | `map[string]string` | No (default: `nil`)                                        |
| Custom HttpClient | `WithHTTPClient(http.Client)`                | Set a custom http client. If this is set then SSL Cert and Insecure options are ignore.                | `*http.Client`      | No (default: Default HTTPClient)                           |
| Timeout           | `WithTimeout(time.Duration)`                 | Set the timeout for the client.                                                                        | `time.Duration`     | No (default is the default HTTP client timeout duration)   |

```go
package main

import (
	"context"
	"fmt"
	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"log"
)

func main() {
	c, err := chroma.NewHTTPClient(
		chroma.WithCloudAPIKey("my-api-key"),
		chroma.WithDatabaseAndTenant("team-uuid", "my-database"),
		chroma.WithDebug(), // enable debug output (sensitive token information is sanitized)
	)
	if err != nil {
		log.Fatalf("Error creating client: %s \n", err)
	}
	v, err := c.GetVersion(context.Background())
	if err != nil {
		log.Fatalf("Error getting version: %s \n", err)
	}
	fmt.Printf("Chroma API Version: %s \n", v)
}


```

## Chroma Client (v0.2.0+)

Options:

| Options           | Usage                                        | Description                                                                                    | Value                      | Required                                                 |
|-------------------|----------------------------------------------|------------------------------------------------------------------------------------------------|----------------------------|----------------------------------------------------------|
| API Endpoint      | `WithBasePath("http://localhost:8000")`      | The Chroma server base API.                                                                    | Non-empty valid URL string | No (default: `http://localhost:8000`)                    |
| Tenant            | `WithTenant("tenant")`                       | The default tenant to use.                                                                     | `string`                   | No (default: `default_tenant`)                           |
| Database          | `WithDatabaseAndTenant("database","tenant")` | The default database to use.                                                                   | `string`                   | No (default: `default_database`)                         |
| Debug             | `WithDebug(true/false)`                      | Enable debug mode.                                                                             | `bool`                     | No (default: `false`)                                    |
| Default Headers   | `WithDefaultHeaders(map[string]string)`      | Set default HTTP headers for the client. These headers are sent with every request.            | `map[string]string`        | No (default: `nil`)                                      |
| SSL Cert          | `WithSSLCert("path/to/cert.pem")`            | Set the path to the SSL certificate.                                                           | valid path to SSL cert.    | No (default: Not Set)                                    |
| Insecure          | `WithInsecure()`                             | Disable SSL certificate verification                                                           |                            | No (default: Not Set)                                    |
| Custom HttpClient | `WithHTTPClient(http.Client)`                | Set a custom http client. If this is set then SSL Cert and Insecure options are ignore.        | `*http.Client`             | No (default: Default HTTPClient)                         |
| Auth              | `WithAuth(v2.CredentialsProvider)`           | Set the authentication method. The default is `WithAuth(types.NewNoAuthCredentialsProvider())` | `CredentialsProvider`      | No (default: `NoAuth`)                                   |
| Timeout           | `WithTimeout(time.Duration)`                 | Set the timeout for the client.                                                                | `time.Duration`            | No (default is the default HTTP client timeout duration) |
| Transport         | `WithTransport(*http.Transport)`             | Set the transport for the client.                                                              | `http.RoundTripper`        | No (default: Default HTTPClient)                         |

```go
package main

import (
	"context"
	"fmt"
	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
	"log"
)

func main() {
	c, err := chroma.NewHTTPClient(
		chroma.WithBaseURL("http://localhost:8000"),
		chroma.WithDatabaseAndTenant("default_database", "default_tenant"),
		chroma.WithDefaultHeaders(map[string]string{"X-Custom-Header": "header-value"}),
		chroma.WithDebug(), // enable debug output (sensitive token information is sanitized)
	)
	if err != nil {
		log.Fatalf("Error creating client: %s \n", err)
	}
	v, err := c.GetVersion(context.Background())
	if err != nil {
		log.Fatalf("Error getting version: %s \n", err)
	}
	fmt.Printf("Chroma API Version: %s \n", v)
}


```

## Client version v0.1.4 or lower

Options:

| Options           | Usage                                   | Description                                                                             | Value                      | Required                              |
|-------------------|-----------------------------------------|-----------------------------------------------------------------------------------------|----------------------------|---------------------------------------|
| basePath          | `WithBasePath("http://localhost:8000")` | The Chroma server base API.                                                             | Non-empty valid URL string | No (default: `http://localhost:8000`) |
| Tenant            | `WithTenant("tenant")`                  | The default tenant to use.                                                              | `string`                   | No (default: `default_tenant`)        |
| Database          | `WithDatabase("database")`              | The default database to use.                                                            | `string`                   | No (default: `default_database`)      |
| Debug             | `WithDebug(true/false)`                 | Enable debug mode.                                                                      | `bool`                     | No (default: `false`)                 |
| Default Headers   | `WithDefaultHeaders(map[string]string)` | Set default headers for the client.                                                     | `map[string]string`        | No (default: `nil`)                   |
| SSL Cert          | `WithSSLCert("path/to/cert.pem")`       | Set the path to the SSL certificate.                                                    | valid path to SSL cert.    | No (default: Not Set)                 |
| Insecure          | `WithInsecure()`                        | Disable SSL certificate verification                                                    |                            | No (default: Not Set)                 |
| Custom HttpClient | `WithHTTPClient(http.Client)`           | Set a custom http client. If this is set then SSL Cert and Insecure options are ignore. | `*http.Client`             | No (default: Default HTTPClient)      |

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
		chroma.WithBasePath("http://localhost:8000"),
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

	// Close the client to release any resources such as local embedding functions
	err = client.Close()
	if err != nil {
		fmt.Printf("Failed to close client: %v", err)
	}
}
```
