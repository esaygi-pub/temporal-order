package main

import (
	"fmt"
	"log"
	"net/http"

	"order/mockserver"
)

func main() {
	mux := mockserver.NewMux()
	fmt.Println("Mock server listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
