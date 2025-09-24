# Authentication

Chroma supports multiple authentication methods. All methods work with both self-hosted and Chroma Cloud deployments.

> **📁 Complete Examples**: Find runnable authentication examples in the [`examples/v2/auth`](https://github.com/amikos-tech/chroma-go/tree/main/examples/v2/auth) directory.

## API v2 (Recommended)

### Basic Authentication

```go
package main

import (
    "context"
    "log"
    v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
    client, err := v2.NewHTTPClient(
        v2.WithBaseURL("http://localhost:8000"),
        v2.WithAuth(v2.NewBasicAuthCredentialsProvider("admin", "password")),
    )
    if err != nil {
        log.Fatal(err)
    }

    if err := client.Heartbeat(context.TODO()); err != nil {
        log.Fatal(err)
    }
}
```

### Token Authentication - Bearer

```go
package main

import (
    "context"
    "log"
    v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
    client, err := v2.NewHTTPClient(
        v2.WithBaseURL("http://localhost:8000"),
        v2.WithAuth(v2.NewTokenAuthCredentialsProvider("my-token", v2.AuthorizationTokenHeader)),
    )
    if err != nil {
        log.Fatal(err)
    }

    if err := client.Heartbeat(context.TODO()); err != nil {
        log.Fatal(err)
    }
}
```

### Token Authentication - X-Chroma-Token

```go
package main

import (
    "context"
    "log"
    v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
    client, err := v2.NewHTTPClient(
        v2.WithBaseURL("http://localhost:8000"),
        v2.WithAuth(v2.NewTokenAuthCredentialsProvider("my-token", v2.XChromaTokenHeader)),
    )
    if err != nil {
        log.Fatal(err)
    }

    if err := client.Heartbeat(context.TODO()); err != nil {
        log.Fatal(err)
    }
}
```

### Custom Headers

```go
package main

import (
    "context"
    "log"
    v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
    headers := map[string]string{
        "Authorization": "Bearer custom-token",
        "X-Custom-Header": "custom-value",
    }

    client, err := v2.NewHTTPClient(
        v2.WithBaseURL("http://localhost:8000"),
        v2.WithDefaultHeaders(headers),
    )
    if err != nil {
        log.Fatal(err)
    }

    if err := client.Heartbeat(context.TODO()); err != nil {
        log.Fatal(err)
    }
}
```

### Chroma Cloud Authentication

```go
package main

import (
    "context"
    "log"
    v2 "github.com/amikos-tech/chroma-go/pkg/api/v2"
)

func main() {
    client, err := v2.NewCloudClient(
        v2.WithCloudHost("api.trychroma.com"),
        v2.WithCloudAPIKey("your-api-key"),
        v2.WithCloudTenant("your-tenant"),
        v2.WithCloudDatabase("your-database"),
    )
    if err != nil {
        log.Fatal(err)
    }

    if err := client.Heartbeat(context.TODO()); err != nil {
        log.Fatal(err)
    }
}
```

## API v1 (Legacy)

### Basic Authentication

```go
package main

import (
    "context"
    "log"
    chroma "github.com/amikos-tech/chroma-go"
    "github.com/amikos-tech/chroma-go/types"
)

func main() {
    client, err := chroma.NewClient(
        chroma.WithBasePath("http://localhost:8000"),
        chroma.WithAuth(types.NewBasicAuthCredentialsProvider("admin", "password")),
    )
    if err != nil {
        log.Fatal(err)
    }

    if _, err := client.Heartbeat(context.TODO()); err != nil {
        log.Fatal(err)
    }
}
```

### Token Authentication - Bearer

```go
package main

import (
    "context"
    "log"
    chroma "github.com/amikos-tech/chroma-go"
    "github.com/amikos-tech/chroma-go/types"
)

func main() {
    client, err := chroma.NewClient(
        chroma.WithBasePath("http://localhost:8000"),
        chroma.WithAuth(types.NewTokenAuthCredentialsProvider("my-token", types.AuthorizationTokenHeader)),
    )
    if err != nil {
        log.Fatal(err)
    }

    if _, err := client.Heartbeat(context.TODO()); err != nil {
        log.Fatal(err)
    }
}
```

### Token Authentication - X-Chroma-Token

```go
package main

import (
    "context"
    "log"
    chroma "github.com/amikos-tech/chroma-go"
    "github.com/amikos-tech/chroma-go/types"
)

func main() {
    client, err := chroma.NewClient(
        chroma.WithBasePath("http://localhost:8000"),
        chroma.WithAuth(types.NewTokenAuthCredentialsProvider("my-token", types.XChromaTokenHeader)),
    )
    if err != nil {
        log.Fatal(err)
    }

    if _, err := client.Heartbeat(context.TODO()); err != nil {
        log.Fatal(err)
    }
}
```

### Custom Headers

```go
package main

import (
    "context"
    "log"
    chroma "github.com/amikos-tech/chroma-go"
)

func main() {
    headers := map[string]string{
        "Authorization": "Bearer custom-token",
        "X-Custom-Header": "custom-value",
    }

    client, err := chroma.NewClient(
        chroma.WithBasePath("http://localhost:8000"),
        chroma.WithDefaultHeaders(headers),
    )
    if err != nil {
        log.Fatal(err)
    }

    if _, err := client.Heartbeat(context.TODO()); err != nil {
        log.Fatal(err)
    }
}
```
