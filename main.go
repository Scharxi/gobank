package main

import (
	"fmt"
	"log"
)

func main() {
	storage, err := NewPostgresStorage()
	if err != nil {
		log.Fatalln(err)
	}

	if err := storage.Init(); err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("%+v\n", storage)
	server := NewApiServer(":3000", storage)
	server.Run()
}
