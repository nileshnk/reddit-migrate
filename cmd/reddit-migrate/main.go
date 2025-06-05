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

	"github.com/nileshnk/reddit-migrate/internal/api"
	"github.com/nileshnk/reddit-migrate/internal/auth"
	"github.com/nileshnk/reddit-migrate/internal/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Version is the application version, injected at build time.
var Version = "dev" // Default to "dev" if not built with version info

// DefaultAddress is the address the server will listen on if no other address is specified.
const DefaultAddress = "localhost:5005"

func main() {
	// Initialize loggers
	config.InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	config.ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lmicroseconds)
	config.DebugLogger = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lmicroseconds)

	config.InfoLogger.Printf("Application version: %s", Version) // Print the version

	// Initialize OAuth configuration
	// Get OAuth credentials from environment variables (you can also use a config file)
	clientID := os.Getenv("REDDIT_CLIENT_ID")
	clientSecret := os.Getenv("REDDIT_CLIENT_SECRET")
	redirectURI := os.Getenv("REDDIT_REDIRECT_URI")

	// Use default redirect URI if not specified
	if redirectURI == "" {
		redirectURI = fmt.Sprintf("http://%s/api/oauth/callback", getServerAddress())
	}

	// Initialize OAuth only if credentials are provided
	if clientID != "" && clientSecret != "" {
		auth.InitOAuth(clientID, clientSecret, redirectURI)
		config.InfoLogger.Printf("OAuth initialized with client ID: %s", clientID)
		config.InfoLogger.Printf("OAuth redirect URI: %s", redirectURI)
	} else {
		config.InfoLogger.Println("OAuth not initialized. Set REDDIT_CLIENT_ID and REDDIT_CLIENT_SECRET environment variables to enable OAuth.")
	}

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
		config.ErrorLogger.Fatalf("Could not start the application. Check if port is available: %v", err)
	}

	// Construct the URL for browser opening.
	urlAddr := constructURL(addr)
	config.InfoLogger.Printf("Application is attempting to run on %s", urlAddr)

	// Attempt to open the URL in the default browser.
	if err := openInBrowser(urlAddr); err != nil {
		config.ErrorLogger.Printf("Failed to open URL in browser: %v. Please open it manually.", err)
	} else {
		config.InfoLogger.Printf("Application is running on %s 🚀", urlAddr)
	}

	// Start the HTTP server.
	config.InfoLogger.Printf("Starting server on %s", addr)
	if err := http.Serve(listener, router); err != nil {
		config.ErrorLogger.Fatalf("Error while serving the application: %v", err)
	}
}

// getServerAddress determines the server address based on environment variables,
// command-line arguments, or a default value.
func getServerAddress() string {
	// Priority 1: Environment variable GO_ADDR.
	if addr := os.Getenv("GO_ADDR"); addr != "" {
		config.InfoLogger.Printf("Using address from GO_ADDR environment variable: %s", addr)
		return addr
	}

	// Priority 2: Command-line argument --addr.
	for _, arg := range os.Args[1:] { // Skip the program name.
		if strings.HasPrefix(arg, "--addr=") {
			addr := strings.TrimPrefix(arg, "--addr=")
			if addr != "" {
				config.InfoLogger.Printf("Using address from --addr command-line argument: %s", addr)
				return addr
			}
		}
	}

	// Priority 3: Default address.
	config.InfoLogger.Printf("Using default address: %s", DefaultAddress)
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
		config.ErrorLogger.Printf("Malformed address string: %s. Defaulting to localhost.", addr)
		return fmt.Sprintf("http://%s", DefaultAddress)
	}

	if host == "" {
		host = "localhost" // Default host if only port is specified (e.g., ":5005")
	}
	return fmt.Sprintf("http://%s:%s", host, port)
}

// mainRouter sets up routes for the main application, including static file serving and API routes.
func mainRouter(r chi.Router) {
	// Determine the path to static files relative to the executable.
	exePath, err := os.Executable()
	if err != nil {
		config.ErrorLogger.Fatalf("Failed to get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)
	cleanedExeDir := filepath.Clean(exeDir)

	var baseResourcePath string

	// Check for macOS .app structure: AppName.app/Contents/MacOS/
	// If the executable is in AppName.app/Contents/MacOS, baseResourcePath is the dir containing AppName.app.
	if filepath.Base(cleanedExeDir) == "MacOS" &&
		filepath.Base(filepath.Dir(cleanedExeDir)) == "Contents" &&
		strings.HasSuffix(filepath.Base(filepath.Dir(filepath.Dir(cleanedExeDir))), ".app") {
		baseResourcePath = filepath.Dir(filepath.Dir(filepath.Dir(cleanedExeDir)))
	} else if filepath.Base(cleanedExeDir) == "bin" {
		// Executable is in a 'bin' subdirectory of the distribution root (e.g., for macOS raw binaries).
		// baseResourcePath is the parent directory of 'bin'.
		baseResourcePath = filepath.Dir(cleanedExeDir)
	} else {
		// Executable is directly in the distribution root.
		baseResourcePath = cleanedExeDir
	}

	staticFilesPath := filepath.Join(baseResourcePath, "web", "static")

	// Check if the directory exists at the executable-relative path.
	_, errStat := os.Stat(staticFilesPath)
	if os.IsNotExist(errStat) {
		config.InfoLogger.Printf("Static files directory not found at executable-relative path: %s. Executable was at: %s", staticFilesPath, exePath)
		// Fallback for development: try CWD-relative path.
		workDir, errWd := os.Getwd()
		if errWd != nil {
			config.ErrorLogger.Fatalf("Failed to get current working directory for fallback: %v", errWd)
		}
		cwdStaticPath := filepath.Join(workDir, "web", "static")
		if _, errStatCwd := os.Stat(cwdStaticPath); os.IsNotExist(errStatCwd) {
			config.ErrorLogger.Fatalf("Static files directory 'web/static' also not found relative to CWD (%s at %s). Please ensure 'web/static' exists.", cwdStaticPath, workDir)
		} else if errStatCwd != nil {
			config.ErrorLogger.Fatalf("Error checking CWD-relative static files directory at %s: %v", cwdStaticPath, errStatCwd)
		} else {
			config.InfoLogger.Printf("Using CWD-relative path for static files: %s", cwdStaticPath)
			staticFilesPath = cwdStaticPath
		}
	} else if errStat != nil {
		// Another error occurred with os.Stat (e.g., permissions).
		config.ErrorLogger.Fatalf("Error checking static files directory at %s: %v", staticFilesPath, errStat)
	}

	filesDir := http.Dir(staticFilesPath)
	FileServer(r, "/", filesDir)
	config.InfoLogger.Printf("Serving static files from %s", staticFilesPath)

	// Register API routes under the "/api" prefix.
	// TODO: Update this to call the new api.Router function from the internal/api package
	r.Route("/api", api.Router) // This will need to be changed
	config.InfoLogger.Println("API routes registered under /api")
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
	config.DebugLogger.Printf("Attempting to open browser with command: %s %v", cmd, args)
	// Use exec.CommandContext for better control if needed in the future (e.g., timeouts).
	return exec.Command(cmd, args...).Start()
}
