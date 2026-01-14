package main

import (
	"bwrs/server"
)

func main() {
	if server.InitStart() {
		server.Start()
	}
}
