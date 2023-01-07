package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello/world", Hello)
	fmt.Println("Listening...")
	err := http.ListenAndServe(":8899", mux)
	if err != nil {
		fmt.Println("err...", err)
		return
	}
}

func Hello(w http.ResponseWriter, r *http.Request) {
	fmt.Println("her is Hello")
	defer r.Body.Close()

	answer := `{"status":"OK"}`
	w.Write([]byte(answer))
}
