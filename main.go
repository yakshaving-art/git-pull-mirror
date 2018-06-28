package main

import (
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Requested on URI: %s\n", r.RequestURI)
		log.Printf("Request: %#v", r)

		reader := r.Body
		defer reader.Close()
		b, err := ioutil.ReadAll(reader)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		body := string(b)
		log.Printf("Body: %#v", body)

		w.WriteHeader(http.StatusOK)
	})

	log.Println("Listening on 127.0.0.1:9092")
	if err := http.ListenAndServe("127.0.0.1:9092", nil); err != nil {
		log.Fatalf("Failed to start http server: %s", err)
	}
}
