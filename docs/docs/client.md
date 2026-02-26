# Chroma Client

## Chroma Cloud Client (v0.2.0+)

Options:

| Options           | Usage                                        | Description                                                                                            | Value               | Required                                                   |
|-------------------|----------------------------------------------|--------------------------------------------------------------------------------------------------------|---------------------|------------------------------------------------------------|
| CloudAPIKey       | `WithCloudAPIKey("api_key")`                 | Set Chroma Cloud API Key                                                                               | `string`            | Yes (default: uses `CHROMA_API_KEY` env var if available)  |
| Tenant            | `WithTenant("tenant")`                       | The default tenant to use.                                                                             | `string`            | Yes (default: uses `CHROMA_TENANT` env var if available)   |
| Database          | `WithDatabaseAndTenant("database","tenant")` | The default database and tenant (overrides any tenant set with Tenant). Option precedence is observed! | `string`            | Yes (default: uses `CHROMA_DATABASE` env var if available) |
| ~~Debug~~         | ~~`WithDebug()`~~ **DEPRECATED**             | ~~Enable debug mode~~ **Use `WithLogger` with debug level instead**                                    | `bool`              | No (deprecated - use WithLogger)                           |
| Logger            | `WithLogger(logger)`                         | Set a custom logger for HTTP request/response logging. See [logging docs](./logging.md)                | `logger.Logger`     | No (default: NoopLogger)                                   |
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
		// chroma.WithDebug() is deprecated - use WithLogger instead for debug output
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
| API Endpoint      | `WithBaseURL("http://localhost:8000")`       | The Chroma server base API.                                                                    | Non-empty valid URL string | No (default: `http://localhost:8000`)                    |
| Tenant            | `WithTenant("tenant")`                       | The default tenant to use.                                                                     | `string`                   | No (default: `default_tenant`)                           |
| Database          | `WithDatabaseAndTenant("database","tenant")` | The default database to use.                                                                   | `string`                   | No (default: `default_database`)                         |
| ~~Debug~~         | ~~`WithDebug()`~~ **DEPRECATED**             | ~~Enable debug mode~~ **Use `WithLogger` with debug level instead**                            | `bool`                     | No (deprecated - use WithLogger)                         |
| Logger            | `WithLogger(logger)`                          | Set a custom logger for HTTP request/response logging. See [logging docs](./logging.md)        | `logger.Logger`            | No (default: NoopLogger)                                 |
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
		// chroma.WithDebug() is deprecated - use WithLogger instead for debug output
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

## Local Persistent Client (v0.3.6+)

`NewLocalClient` starts and manages a local Chroma runtime (via `chroma-go-local`) and exposes the same `Client` interface.
By default it uses embedded mode (no HTTP server). You can opt into server mode when needed.

### Runtime Options

| Options               | Usage                                             | Description                                    |
|-----------------------|---------------------------------------------------|------------------------------------------------|
| Local Runtime Mode    | `WithLocalRuntimeMode(chroma.LocalRuntimeModeServer)`    | Select `embedded` (default) or `server`.       |
| Local Library Path    | `WithLocalLibraryPath("/path/to/lib...")`         | Explicit local runtime library path.           |
| Local Library Version | `WithLocalLibraryVersion("v0.2.0")`               | Release tag used for auto-download (default `v0.2.0`). |
| Local Library Cache   | `WithLocalLibraryCacheDir("./.cache/chroma")`     | Cache location for downloaded shim libraries.  |
| Local Library Download| `WithLocalLibraryAutoDownload(true)`              | Enable/disable auto-download fallback.         |
| Local Persist Path    | `WithLocalPersistPath("./chroma_data")`           | Persistent storage directory.                  |
| Local Listen Address  | `WithLocalListenAddress("127.0.0.1")`             | Server-mode bind address.                      |
| Local Port            | `WithLocalPort(0)`                                | Server-mode port (default `8000`; `0` auto-selects). |
| Local Allow Reset     | `WithLocalAllowReset(true)`                       | Enable reset endpoint/behavior.                |
| Local Config Path     | `WithLocalConfigPath("./chroma.yaml")`            | Start runtime from YAML file.                  |
| Local Raw YAML        | `WithLocalRawYAML("port: 8010\npersist_path:...")` | Start runtime from inline YAML.                |
| Wrapped Client Option | `WithLocalClientOption(chroma.WithDatabaseAndTenant(...))` | Apply regular `ClientOption` to local client state. |

### Example

```go
package main

import (
	"context"
	"fmt"
	"log"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
	client, err := chroma.NewLocalClient(
		chroma.WithLocalPersistPath("./chroma_data"),
		chroma.WithLocalAllowReset(true),
		chroma.WithLocalClientOption(chroma.WithDatabaseAndTenant("default_database", "default_tenant")),
	)
	if err != nil {
		log.Fatalf("Error creating local client: %v", err)
	}
	defer client.Close()

	version, err := client.GetVersion(context.Background())
	if err != nil {
		log.Fatalf("Error getting version: %v", err)
	}
	fmt.Println("Chroma version:", version)
}
```

### Library Path Resolution

`NewLocalClient` resolves the runtime shared library in this order:

1. `WithLocalLibraryPath(...)`
2. `CHROMA_LIB_PATH`
3. Auto-download from `chroma-go-local` GitHub releases (enabled by default)

## Client version v0.1.4 or lower

!!! warning "V1 API Deprecation Notice"

    The V1 API has been removed in version `v0.3.0` and later. To use the examples below, pin your dependency to `v0.2.4` or earlier:
    ```bash
    go get github.com/amikos-tech/chroma-go@v0.2.4
    ```

Options:

| Options           | Usage                                   | Description                                                                             | Value                      | Required                              |
|-------------------|-----------------------------------------|-----------------------------------------------------------------------------------------|----------------------------|---------------------------------------|
| basePath          | `WithBasePath("http://localhost:8000")` | The Chroma server base API.                                                             | Non-empty valid URL string | No (default: `http://localhost:8000`) |
| Tenant            | `WithTenant("tenant")`                  | The default tenant to use.                                                              | `string`                   | No (default: `default_tenant`)        |
| Database          | `WithDatabase("database")`              | The default database to use.                                                            | `string`                   | No (default: `default_database`)      |
| ~~Debug~~         | ~~`WithDebug()`~~ **DEPRECATED**        | ~~Enable debug mode~~ **Use `WithLogger` with debug level instead**                     | `bool`                     | No (deprecated - use WithLogger)     |
| Logger            | `WithLogger(logger)`                     | Set a custom logger for HTTP request/response logging. See [logging docs](./logging.md) | `logger.Logger`            | No (default: NoopLogger)             |
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
		// chroma.WithDebug(true) is deprecated - use WithLogger instead for debug output
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
