package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"moshr/internal/server"
)

func main() {
	var (
		port    = flag.String("port", "8080", "server port")
		webMode = flag.Bool("web", false, "run in web mode")
	)
	flag.Parse()

	if *webMode {
		fmt.Printf("Starting web server on port %s\n", *port)
		if err := server.Start(*port); err != nil {
			log.Fatal("Failed to start server:", err)
		}
	} else {
		fmt.Println("Moshr - Video Datamoshing Tool")
		fmt.Println("Usage: moshr -web to start web interface")
		fmt.Println("       moshr -port=8080 -web to specify port")
		os.Exit(0)
	}
}
