
package main

import (
	"net/http"
)

func (h *Handler)newApiMux()(mux *http.ServeMux){
	mux = http.NewServeMux()
	mux.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request){
		writeJson(rw, http.StatusNotFound, Map{
			"status": "error",
			"error": "404 not found",
			"path": req.URL.Path,
		})
	})
	mux.HandleFunc("/create_token", func(rw http.ResponseWriter, req *http.Request){
		token := req.Header.Get("Authorization")
		if !h.CheckRootToken(token) {
			writeUnauth(rw)
			return
		}
		tk, err := h.NewCliToken(nil) // TODO: exp time
		if err != nil {
			writeInternalError(rw, err)
			return
		}
		writeJson(rw, http.StatusOK, Map{
			"status": "ok",
			"token": tk,
		})
	})
	mux.HandleFunc("/remove_token", func(rw http.ResponseWriter, req *http.Request){
		rtToken := req.Header.Get("Authorization")
		if !h.CheckRootToken(rtToken) {
			writeUnauth(rw)
			return
		}
		token := req.FormValue("token")
		err := h.RemoveCliToken(token)
		if err != nil {
			writeInternalError(rw, err)
			return
		}
		writeJson(rw, http.StatusOK, Map{
			"status": "ok",
		})
	})
	mux.HandleFunc("/create_daemon_token", func(rw http.ResponseWriter, req *http.Request){
		token := req.Header.Get("Authorization")
		if !h.CheckRootToken(token) {
			writeUnauth(rw)
			return
		}
		values := req.URL.Query()
		server := values.Get("server")
		tk, err := h.NewDaemonToken(server, nil) // TODO: exp time
		if err != nil {
			writeInternalError(rw, err)
			return
		}
		writeJson(rw, http.StatusOK, Map{
			"status": "ok",
			"token": tk,
		})
	})
	mux.HandleFunc("/remove_daemon_token", func(rw http.ResponseWriter, req *http.Request){
		rtToken := req.Header.Get("Authorization")
		if !h.CheckRootToken(rtToken) {
			writeUnauth(rw)
			return
		}
		token := req.FormValue("token")
		err := h.RemoveDaemonToken(token)
		if err != nil {
			writeInternalError(rw, err)
			return
		}
		writeJson(rw, http.StatusOK, Map{
			"status": "ok",
		})
	})
	mux.HandleFunc("/create_server", func(rw http.ResponseWriter, req *http.Request){
		token := req.Header.Get("Authorization")
		if !h.CheckRootToken(token) {
			writeUnauth(rw)
			return
		}
		sid := req.FormValue("id")
		err := h.CreateServer(sid)
		if err != nil {
			writeInternalError(rw, err)
			return
		}
		writeJson(rw, http.StatusOK, Map{
			"status": "ok",
		})
	})
	mux.HandleFunc("/remove_server", func(rw http.ResponseWriter, req *http.Request){
		token := req.Header.Get("Authorization")
		if !h.CheckRootToken(token) {
			writeUnauth(rw)
			return
		}
		sid := req.FormValue("id")
		err := h.RemoveServer(sid)
		if err != nil {
			writeInternalError(rw, err)
			return
		}
		h.removeHost(sid)
		writeJson(rw, http.StatusOK, Map{
			"status": "ok",
		})
	})
	mux.HandleFunc("/tokens", func(rw http.ResponseWriter, req *http.Request){
		token := req.Header.Get("Authorization")
		if !h.CheckRootToken(token) {
			writeUnauth(rw)
			return
		}
		tokens, err := h.ListTokens()
		if err != nil {
			writeInternalError(rw, err)
			return
		}
		writeJson(rw, http.StatusOK, Map{
			"status": "ok",
			"data": tokens,
		})
	})
	mux.HandleFunc("/daemon_tokens", func(rw http.ResponseWriter, req *http.Request){
		token := req.Header.Get("Authorization")
		if !h.CheckRootToken(token) {
			writeUnauth(rw)
			return
		}
		tokens, err := h.ListDaemonTokens()
		if err != nil {
			writeInternalError(rw, err)
			return
		}
		writeJson(rw, http.StatusOK, Map{
			"status": "ok",
			"data": tokens,
		})
	})
	mux.HandleFunc("/perm_root", func(rw http.ResponseWriter, req *http.Request){
		rtToken := req.Header.Get("Authorization")
		if !h.CheckRootToken(rtToken) {
			writeUnauth(rw)
			return
		}
		values := req.URL.Query()
		token := values.Get("token")
		if !h.AuthCli(token) {
			writeJson(rw, http.StatusOK, Map{
				"status": "error",
				"error": "Token not exists",
			})
			return
		}
		switch req.Method {
		case "GET":
			writeJson(rw, http.StatusOK, Map{
				"status": "ok",
				"value": h.CheckRootToken(token),
			})
		case "POST":
			var (
				value bool
				err error
			)
			if err = readJsonBody(req, &value); err != nil {
				writeJson(rw, http.StatusBadRequest, Map{
					"status": "error",
					"error": err.Error(),
				})
				return
			}
			if err = h.SetRoot(token, value); err != nil {
				writeInternalError(rw, err)
				return
			}
			writeJson(rw, http.StatusOK, Map{
				"status": "ok",
			})
		default:
			rw.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/perm_server", func(rw http.ResponseWriter, req *http.Request){
		rtToken := req.Header.Get("Authorization")
		if !h.CheckRootToken(rtToken) {
			writeUnauth(rw)
			return
		}
		values := req.URL.Query()
		token := values.Get("token")
		id := values.Get("id")
		switch req.Method {
		case "GET":
			writeJson(rw, http.StatusOK, Map{
				"status": "ok",
				"value": h.CheckPerm(token, id),
			})
		case "POST":
			var (
				value bool
				err error
			)
			if err = readJsonBody(req, &value); err != nil {
				writeJson(rw, http.StatusBadRequest, Map{
					"status": "error",
					"error": err.Error(),
				})
				return
			}
			if err = h.SetPerm(token, id, value); err != nil {
				writeInternalError(rw, err)
				return
			}
			writeJson(rw, http.StatusOK, Map{
				"status": "ok",
			})
		default:
			rw.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/perm_servers", func(rw http.ResponseWriter, req *http.Request){
		values := req.URL.Query()
		token := values.Get("token")
		servers, err := h.ListServers(token)
		if err != nil {
			writeInternalError(rw, err)
			return
		}
		writeJson(rw, http.StatusOK, Map{
			"status": "ok",
			"data": servers,
		})
	})
	return
}

func writeUnauth(rw http.ResponseWriter)(error){
	return writeJson(rw, http.StatusUnauthorized, Map{
		"status": "error",
		"error": "permission denied",
	})
}

func writeInternalError(rw http.ResponseWriter, err error)(error){
	return writeJson(rw, http.StatusInternalServerError, Map{
		"status": "error",
		"error": err.Error(),
	})
}
