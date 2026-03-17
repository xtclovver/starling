package handler

import (
	"encoding/json"
	"net/http"

	"github.com/usedcvnt/microtwitter/api-gateway/internal/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type response struct {
	Data  any    `json:"data"`
	Error *errBody `json:"error"`
}

type errBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response{Data: data})
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response{Error: &errBody{Code: code, Message: msg}})
}

func handleGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	switch st.Code() {
	case codes.NotFound:
		writeError(w, http.StatusNotFound, st.Message())
	case codes.AlreadyExists:
		writeError(w, http.StatusConflict, st.Message())
	case codes.InvalidArgument:
		writeError(w, http.StatusBadRequest, st.Message())
	case codes.PermissionDenied:
		writeError(w, http.StatusForbidden, st.Message())
	case codes.Unauthenticated:
		writeError(w, http.StatusUnauthorized, st.Message())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func getUserID(r *http.Request) string {
	id, _ := r.Context().Value(middleware.UserIDKey).(string)
	return id
}
