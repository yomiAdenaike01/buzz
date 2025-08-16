package responses

import (
	"encoding/json"
	"net/http"
)

func WriteErrResponse(w http.ResponseWriter, statusCode int, response string) {
	w.WriteHeader(statusCode)
	WriteResponse(w, map[string]any{"error": response})
}
func WriteSuccessResponse(w http.ResponseWriter, response map[string]any) {
	w.WriteHeader(http.StatusOK)
	WriteResponse(w, response)
}

func WriteJSON(w http.ResponseWriter, response map[string]any) {
	w.Header().Add("Content-Type", "application/json")
	WriteResponse(w, response)
}

//	func writeErrJSON(w http.ResponseWriter, statusCode int, response []byte) {
//		w.WriteHeader(statusCode)
//		writeJSON(w, response)
//	}
func WriteResponse(w http.ResponseWriter, response map[string]any) {
	d, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}
	w.Write(d)
}
