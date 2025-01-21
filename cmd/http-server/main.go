package main

import (
	"kafka/svcs/internal/frontend"
	"log"
)

func main() {
	log.Fatal(frontend.Run())
}
