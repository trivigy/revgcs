package main

import (
	"cloud.google.com/go/storage"
	"context"
	"flag"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

var bind = flag.String(
	"bind",
	"127.0.0.1:8080",
	"Listening address for incoming requests.",
)

var (
	client *storage.Client
	ctx    = context.Background()
)

func handleError(w http.ResponseWriter, err error) {
	if err != nil {
		if err == storage.ErrObjectNotExist {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
}

func header(r *http.Request, key string) (string, bool) {
	if r.Header == nil {
		return "", false
	}
	if candidate := r.Header[key]; len(candidate) > 0 {
		return candidate[0], true
	}
	return "", false
}

func handleHealth(resp http.ResponseWriter, _ *http.Request) {
	resp.WriteHeader(http.StatusOK)
}

func handleProxy(resp http.ResponseWriter, req *http.Request) {
	proc := time.Now()
	params := mux.Vars(req)
	obj := client.Bucket(params["bucket"]).Object(params["object"])
	attr, err := obj.Attrs(ctx)
	if err != nil {
		handleError(resp, err)
		return
	}

	resp.Header().Add("Content-Type", attr.ContentType)
	resp.Header().Add("Content-Language", attr.ContentLanguage)
	resp.Header().Add("Cache-Control", attr.CacheControl)
	resp.Header().Add("Content-Encoding", attr.ContentEncoding)
	resp.Header().Add("Content-Disposition", attr.ContentDisposition)
	resp.Header().Add("Content-Length", strconv.FormatInt(attr.Size, 10))

	r, err := obj.NewReader(ctx)
	if err != nil {
		handleError(resp, err)
		return
	}
	defer r.Close()
	if _, err := io.Copy(resp, r); err != nil {
		handleError(resp, err)
		return
	}

	addr := req.RemoteAddr
	if ip, found := header(req, "X-Forwarded-For"); found {
		addr = ip
	}
	log.Printf("[%s] %.3f %d %s %s",
		addr,
		time.Now().Sub(proc).Seconds(),
		http.StatusOK,
		req.Method,
		req.URL,
	)
}

func main() {
	flag.Parse()

	var err error
	client, err = storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	router := mux.NewRouter()

	router.HandleFunc("/health", handleHealth).
		Methods("GET")

	router.HandleFunc("/{bucket:[0-9a-zA-Z-_]+}/{object:.*}", handleProxy).
		Methods("GET", "HEAD")

	log.Printf("[service] listening on %s", *bind)
	if err := http.ListenAndServe(*bind, router); err != nil {
		log.Fatal(err)
	}
}
