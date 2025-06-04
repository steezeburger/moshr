package server

import (
	"fmt"
	"log"
)

func Start(port string) error {
	server := NewServer()
	
	r := server.SetupRoutes()
	r.GET("/ws", server.handleWebSocket)
	
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Server starting on %s", addr)
	
	return r.Run(addr)
}