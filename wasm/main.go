//go:build wasm

package main

import (
	"context"
	"fmt"
	chroma "github.com/amikos-tech/chroma-go"
	"syscall/js"
)

func getVersion(client chroma.Client) string {
	v, err := client.Version(context.TODO())
	if err != nil {
		fmt.Println(err)
	}
	return v
}
func NewClient(this js.Value, args []js.Value) interface{} {
	host := args[0].String()
	console := js.Global().Get("console")
	console.Call("log", "Hello from Go s.......!"+host)
	client, err := chroma.NewClient(host)
	if err != nil {
		return err.Error()
	}
	c := map[string]interface{}{
		"host": host,
	}

	v := js.ValueOf(c)
	v.Set("version", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return getVersion(*client)
	}))
	v.Set("listCollections", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		collections, err := client.ListCollections(context.TODO())
		if err != nil {
			return err.Error()
		}
		collectionNames := make([]string, 0)
		for _, collection := range collections {
			collectionNames = append(collectionNames, collection.Name)
		}
		return collectionNames
	}))
	return v
}

func exports() {
	js.Global().Set("newChromaClient", js.FuncOf(NewClient))
}

func main() {

	//c := make(chan struct{})

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
	console := js.Global().Get("console")
	console.Call("log", "Hello from Go WASM!")
	//js.Global().Call("alert", "Collections: "+strings.Join(collectionNames, ", "))
	exports()
	//<-c
	select {}
}
