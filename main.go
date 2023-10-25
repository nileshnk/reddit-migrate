package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {

	// chi router
	router := chi.NewRouter()

	router.Use(middleware.Logger)

	router.Route("/", mainRouter)

	args := os.Args;
	// Priority 1
	ADDR := os.Getenv("GO_ADDR")
	// Priority 2
	for _, a := range args {
		str := strings.Split(a, "=");
		if(str[0]=="--addr"){
			ADDR = str[1]
		}
	}
	
	if(ADDR == "") {
		// Priority 3
		ADDR = "localhost:5005"
	}
	http.ListenAndServe(ADDR, router)
}

func mainRouter(r chi.Router){

	// serving static files
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

