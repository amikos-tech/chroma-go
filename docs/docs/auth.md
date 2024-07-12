# Authentication

There are four ways to authenticate with Chroma:

- Manual Header authentication - this approach requires you to be familiar with the server-side auth and generate and insert the necessary headers manually.
- Chroma Basic Auth mechanism
- Chroma Token Auth mechanism with Bearer Authorization header
- Chroma Token Auth mechanism with X-Chroma-Token header

### Manual Header Authentication

```go
package main

import (
	"context"
	"log"
	chroma "github.com/amikos-tech/chroma-go"
)

func main() {
	var defaultHeaders = map[string]string{"Authorization": "Bearer my-custom-token"}
	clientWithTenant, err := chroma.NewClient(chroma.WithBasePath("http://api.trychroma.com/v1/"), chroma.WithDefaultHeaders(defaultHeaders))
	if err != nil {
		log.Fatalf("Error creating client: %s \n", err)
	}
	_, err = clientWithTenant.Heartbeat(context.TODO())
	if err != nil {
		log.Fatalf("Error calling heartbeat: %s \n", err)
	}
}
```

### Chroma Basic Auth mechanism

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
        chroma.WithBasePath("http://api.trychroma.com/v1/"),
        chroma.WithAuth(types.NewBasicAuthCredentialsProvider("myUser", "myPassword")),
    )
    if err != nil {
        log.Fatalf("Error creating client: %s \n", err)
    }
    _, err = client.Heartbeat(context.TODO())
    if err != nil {
        log.Fatalf("Error calling heartbeat: %s \n", err)
    }
}
```

### Chroma Token Auth mechanism with Bearer Authorization header

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
        chroma.WithBasePath("http://api.trychroma.com/v1/"), 
        chroma.WithAuth(types.NewTokenAuthCredentialsProvider("my-auth-token", types.AuthorizationTokenHeader)),
    )
    if err != nil {
        log.Fatalf("Error creating client: %s \n", err)
    }
    _, err = client.Heartbeat(context.TODO())
    if err != nil {
        log.Fatalf("Error calling heartbeat: %s \n", err)
    }
}
```

### Chroma Token Auth mechanism with X-Chroma-Token header

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
        chroma.WithBasePath("http://api.trychroma.com/v1/"), 
        chroma.WithAuth(types.NewTokenAuthCredentialsProvider("my-auth-token", types.XChromaTokenHeader)),
    )
    if err != nil {
        log.Fatalf("Error creating client: %s \n", err)
    }
    _, err = client.Heartbeat(context.TODO())
    if err != nil {
        log.Fatalf("Error calling heartbeat: %s \n", err)
    }
}
```

