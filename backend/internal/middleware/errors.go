package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	jsonMsg, _ := json.Marshal(msg)
	fmt.Fprintf(w, `{"error":%s}`, jsonMsg)
}
