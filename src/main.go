package main

import (
	"flag"
	"fmt"
	"net/http"

	controller "./controller"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/docgen"
	"github.com/go-chi/render"
)

var routes = flag.Bool("routes", false, "Generate router documentation")

func main() {
	flag.Parse()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		routesData := fmt.Sprintln(docgen.JSONRoutesDoc(r))
		w.Write([]byte(routesData))
	})

	r.Get("/ping", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("pong"))
	})

	r.Route("/vms", func(r chi.Router) {
		r.Get("/", controller.listVMs)
		r.Post("/create", controller.CreateVMs)
		// r.Route("/{vm}", func(r chi.Router) {
		// 	r.Get("/stop", VmStopByquery)
		// 	r.Get("/start", VmStartByQuery)
		// 	r.Delete("/delete", VmDeleteByQuery)
		// 	r.Put("/resize".VmResizeByQuery)
		// 	r.Post("/snapshot", VmSpanshotByQuery)
		// })
		// r.Route("/find", func(r chi.Router) {
		// 	r.Get("/{user}", findVmByUser)
		// 	r.Get("/{user}/{hostname}", findbyUserHostname)
		// 	r.Get("/{user}/{hostname}/{id}", findbyUserHostnameID)
		// })
	})

	// Passing -routes to the program will generate docs for the above
	// router definition. See the `routes.json` file in this folder for
	// the output.
	if *routes {
		// fmt.Println(docgen.JSONRoutesDoc(r))
		fmt.Println(docgen.MarkdownRoutesDoc(r, docgen.MarkdownOpts{
			ProjectPath: "github.com/Shashwatsh/box-api",
			Intro:       "Welcome to the github.com/Shashwatsh/sb-api generated docs.",
		}))
		return
	}

	http.ListenAndServe(":3333", r)
}
