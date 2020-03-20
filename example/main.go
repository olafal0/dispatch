package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/olafal0/dispatch"
)

func rootHandler(ctx *dispatch.Context) string {
	return fmt.Sprintf("Hello, %s!", ctx.PathVars["name"])
}

func main() {
	api := &dispatch.API{}
	api.AddEndpoint("GET/{name}", rootHandler)
	http.HandleFunc("/", api.GetHandler())
	log.Fatal(http.ListenAndServe(":8000", nil))
}
