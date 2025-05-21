package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// DefaultAddress is the address the server will listen on if no other address is specified.
const DefaultAddress = "localhost:5005"

func main() {
	// Initialize loggers
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	DebugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lmicroseconds)

	// Create a new Chi router.
	router := chi.NewRouter()

	// Use a logger middleware for HTTP requests.
	router.Use(middleware.Logger)

	// Register routes for the main application.
	router.Route("/", mainRouter)

	// Determine the server address.
	addr := getServerAddress()

	// Start listening on the specified address.
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		ErrorLogger.Fatalf("Could not start the application. Check if port is available: %v", err)
	}

	// Construct the URL for browser opening.
	urlAddr := constructURL(addr)
	InfoLogger.Printf("Application is attempting to run on %s", urlAddr)

	// Attempt to open the URL in the default browser.
	if err := openInBrowser(urlAddr); err != nil {
		ErrorLogger.Printf("Failed to open URL in browser: %v. Please open it manually.", err)
	} else {
		InfoLogger.Printf("Application is running on %s 🚀", urlAddr)
	}

	// Start the HTTP server.
	InfoLogger.Printf("Starting server on %s", addr)
	if err := http.Serve(listener, router); err != nil {
		ErrorLogger.Fatalf("Error while serving the application: %v", err)
	}
}

// getServerAddress determines the server address based on environment variables,
// command-line arguments, or a default value.
func getServerAddress() string {
	// Priority 1: Environment variable GO_ADDR.
	if addr := os.Getenv("GO_ADDR"); addr != "" {
		InfoLogger.Printf("Using address from GO_ADDR environment variable: %s", addr)
		return addr
	}

	// Priority 2: Command-line argument --addr.
	for _, arg := range os.Args[1:] { // Skip the program name.
		if strings.HasPrefix(arg, "--addr=") {
			addr := strings.TrimPrefix(arg, "--addr=")
			if addr != "" {
				InfoLogger.Printf("Using address from --addr command-line argument: %s", addr)
				return addr
			}
		}
	}

	// Priority 3: Default address.
	InfoLogger.Printf("Using default address: %s", DefaultAddress)
	return DefaultAddress
}

// constructURL creates a full HTTP URL from an address string.
// If the address does not specify a host, "localhost" is assumed.
func constructURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// If splitting fails, it might be a port-only address like ":5005"
		if strings.HasPrefix(addr, ":") {
			return fmt.Sprintf("http://localhost%s", addr)
		}
		// Fallback for other malformed cases, though getServerAddress should prevent this.
		ErrorLogger.Printf("Malformed address string: %s. Defaulting to localhost.", addr)
		return fmt.Sprintf("http://%s", DefaultAddress)
	}

	if host == "" {
		host = "localhost" // Default host if only port is specified (e.g., ":5005")
	}
	return fmt.Sprintf("http://%s:%s", host, port)
}

// mainRouter sets up routes for the main application, including static file serving and API routes.
func mainRouter(r chi.Router) {
	// Serve static files from the "public" directory.
	workDir, err := os.Getwd()
	if err != nil {
		ErrorLogger.Fatalf("Failed to get current working directory: %v", err)
	}
	filesDir := http.Dir(filepath.Join(workDir, "public"))
	FileServer(r, "/", filesDir)
	InfoLogger.Printf("Serving static files from %s", filesDir)

	// Register API routes under the "/api" prefix.
	r.Route("/api", apiRouter)
	InfoLogger.Println("API routes registered under /api")
}

// FileServer conveniently sets up a static file handler for a given path prefix.
// It ensures that requests to directories are properly redirected.
func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		// Panicking here because this is a programming error: URL parameters are not allowed in the path prefix.
		panic("FileServer does not permit any URL parameters in path prefix.")
	}

	// Ensure the path ends with a slash for directory listings.
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*" // Match all subpaths.

	// Define the handler for serving files.
	r.Get(path, func(w http.ResponseWriter, req *http.Request) {
		rctx := chi.RouteContext(req.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, req)
	})
}

// openInBrowser attempts to open the specified URL in the default web browser.
// It detects the operating system to use the appropriate command.
func openInBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
		args = []string{url}
	}
	DebugLogger.Printf("Attempting to open browser with command: %s %v", cmd, args)
	// Use exec.CommandContext for better control if needed in the future (e.g., timeouts).
	return exec.Command(cmd, args...).Start()
}
