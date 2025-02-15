package main

import (
	"log"

	"forum/server"
)

func main() {
	if err := server.Run(); err != nil {
		log.Fatal("Server error:", err)
	}
}
