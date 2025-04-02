package app

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
)

type ServeFile struct {
	contentType string
	fileName    string
}

func NewServeFile(contentType, fileName string) *ServeFile {
	return &ServeFile{
		contentType: contentType,
		fileName:    fileName,
	}
}

func (sf *ServeFile) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open(sf.fileName)
	if err != nil {
		http.Error(w, "No such file", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Add("Content-Type", sf.contentType)
	io.Copy(w, file)
}

func Run(ctx context.Context) error {
	slog.Info("starting server")

	indexHtml := NewServeFile("text/html", "/index.html")
	stylesCss := NewServeFile("text/css", "/css/style.css")
	scriptJs := NewServeFile("text/js", "/js/script.js")

	mux := http.NewServeMux()

	mux.Handle("/", indexHtml)
	mux.Handle("/index.html", indexHtml)
	mux.Handle("/css/style.css", stylesCss)
	mux.Handle("/js/script.js", scriptJs)

	s := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		s.Shutdown(ctx)
	}()

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}
