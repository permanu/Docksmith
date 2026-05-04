package main

import "net/http"

func main() {
	http.ListenAndServe(":4290", nil)
}
