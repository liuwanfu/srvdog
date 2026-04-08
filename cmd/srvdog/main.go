package main

import (
	"log"

	"github.com/liuwanfu/srvdog/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
