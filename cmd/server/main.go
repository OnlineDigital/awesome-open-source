package main

import (
	"awesomeopensource.dev/app/server"
	"log"
)

func main() {
	if err := server.RunFromEnvironment(); err != nil {
		log.Fatal(err)
	}
}
