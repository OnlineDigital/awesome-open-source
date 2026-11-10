package main

import (
	"github.com/andreitelteu/inertia-go-solid-template/app/server"
	"log"
)

func main() {
	if err := server.RunFromEnvironment(); err != nil {
		log.Fatal(err)
	}
}
