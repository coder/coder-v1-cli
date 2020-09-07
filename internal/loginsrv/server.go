package loginsrv

import (
	"fmt"
	"net/http"
)

// Server waits for the login callback to send the session token.
type Server struct {
	TokenChan chan<- string
}

func (srv *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	token := req.URL.Query().Get("session_token")
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprintf(w, "No session_token found.\n") // Best effort.
		return
	}

	select {
	case <-ctx.Done():
		// Client disconnect. Nothing to do.
	case srv.TokenChan <- token:
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "You may close this window now.\n") // Best effort.
	}
}
