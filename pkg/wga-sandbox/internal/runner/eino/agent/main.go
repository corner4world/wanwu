package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	if len(os.Args) < 2 || os.Args[1] == "serve" {
		runServe()
		return
	}

	fmt.Fprintf(os.Stderr, "未知子命令: %s\n\n", os.Args[1])
	os.Exit(1)
}

func runServe() {
	port := "4097"

	// 初始化 CozeLoop（服务级别,只初始化一次）
	if err := Init(); err != nil {
		log.Printf("Warning: failed to initialize cozeloop: %v", err)
	}

	srv := newHTTPServer()
	mux := http.NewServeMux()
	srv.registerRoutes(mux)

	server := &http.Server{
		Addr:           ":" + port,
		Handler:        mux,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   0,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Printf("Server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
