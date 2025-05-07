// main is the entry point for the openapi router application
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cirocosta/openapi-router-go/internal/api"
	"github.com/cirocosta/openapi-router-go/internal/repository"
	"github.com/cirocosta/openapi-router-go/internal/service"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	os.Args = os.Args[1:]

	switch cmd {
	case "run":
		runServer()
	case "openapi-gen":
		generateOpenAPI()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`
Usage: openapi-router-go <command> [options]

Commands:
  run          Start the HTTP server
  openapi-gen  Generate OpenAPI documentation

Run 'openapi-router-go <command> -h' for more information on a command.
`)
}

func runServer() {
	// define command-line flags
	addr := flag.String("addr", ":8080", "HTTP server address")
	flag.Parse()

	// setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// setup dependencies
	todoRepo := repository.NewInMemoryTodoRepository()
	todoService := service.NewTodoService(todoRepo)

	// create router
	r := api.NewRouter(todoService)

	// create server
	server := &http.Server{
		Addr:    *addr,
		Handler: r,
	}

	// create context that listens for interrupts
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// start server in a goroutine
	go func() {
		logger.Info("starting server", "addr", *addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// wait for interrupt
	<-ctx.Done()

	// shutdown server gracefully
	logger.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}

func generateOpenAPI() {
	// define only the output file flag
	output := flag.String("o", "openapi.json", "Output file path")
	flag.Parse()

	// create mock service
	mockService := api.NewMockTodoService()

	// create router
	spec := api.NewRouter(mockService).OpenAPI()

	// Marshal the spec to JSON
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		panic(fmt.Errorf("marshal openapi spec: %w", err))
	}

	// write to file
	if err := os.WriteFile(*output, data, 0644); err != nil {
		panic(fmt.Errorf("write openapi spec to file '%s': %w", *output, err))
	}

	fmt.Printf("OpenAPI spec generated at %s\n", *output)
}
