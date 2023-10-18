// package main is the entry point of the program
package main

// import fmt package to print to console
// import net/http package to create http web server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// main function is the entry point of the program

func main() {

	// chi router
	router := chi.NewRouter()
	// A good base middleware stack
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	// router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/", mainRouter)

	http.ListenAndServe("localhost:5000", router)
}

func mainRouter(r chi.Router){

	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "public"))
	FileServer(r, "/", filesDir)


	r.Route("/api", apiRouter)
}


func FileServer(r chi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}

