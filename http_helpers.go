package main

import "net/http"

func allowMethod(
	writer http.ResponseWriter,
	request *http.Request,
	method string,
) bool {
	if request.Method == method {
		return true
	}

	http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
	return false
}
