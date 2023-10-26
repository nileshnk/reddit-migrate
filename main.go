package main

import (
	"fmt"
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
	listener, errListen := net.Listen("tcp",ADDR)
	if errListen != nil {
		fmt.Println("Could not start the application. Check if port are available!")
	} else {
		var urlAddr string
		if strings.Split(ADDR, ":")[0] != ""{
			urlAddr = fmt.Sprintf("http://%s", ADDR)
		}else {
			urlAddr = fmt.Sprintf("http://localhost%s", ADDR)
		}
		openInBrowser(urlAddr)
		fmt.Printf("Application is running on %s 🚀\n", urlAddr)
	}

	errServe := http.Serve(listener, router)
	if errServe != nil {
		fmt.Println("Error while serving the application:", errServe)
	}
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

// function which opens a browser with provided URL
func openInBrowser(url string) error {
    var cmd string
    var args []string

    switch runtime.GOOS {
    case "windows":
        cmd = "cmd"
        args = []string{"/c", "start"}
    case "darwin":
        cmd = "open"
    default: // "linux", "freebsd", "openbsd", "netbsd"
        cmd = "xdg-open"
    }
    args = append(args, url)
    return exec.Command(cmd, args...).Start()
}