package loginsrv

import (
	"fmt"
	"net/http"
	"sync"
)

type Server struct {
	TokenCond *sync.Cond
	Token string
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("session_token")
	if token == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "No session_token found")
		return
	}

	s.TokenCond.L.Lock()
	s.Token = token
	s.TokenCond.L.Unlock()
	s.TokenCond.Broadcast()

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "You may close this window now")
}


