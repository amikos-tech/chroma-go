package main

import (
	"context"
	"fmt"
	chroma "github.com/amikos-tech/chroma-go"
	"strings"
	"syscall/js"
)

func main() {

	c := make(chan struct{})

	client, err := chroma.NewClient("http://localhost:8000")
	if err != nil {
		fmt.Println(err)
	}
	collections, err := client.ListCollections(context.TODO())
	if err != nil {
		fmt.Println(err)
	}
	collectionNames := make([]string, 0)
	for _, collection := range collections {
		collectionNames = append(collectionNames, collection.Name)
	}

	js.Global().Call("alert", "Collections: "+strings.Join(collectionNames, ", "))
	<-c
}
