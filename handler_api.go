
package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
		if tokens == nil {
			tokens = make([]Token, 0)
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
		if tokens == nil {
			tokens = make([]DaemonToken, 0)
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
		if servers == nil {
			servers = make([]string, 0)
		}
		writeJson(rw, http.StatusOK, Map{
			"status": "ok",
			"data": servers,
		})
	})
	mux.HandleFunc("/server_plugins", func(rw http.ResponseWriter, req *http.Request){
		token := req.Header.Get("Authorization")
		if !h.AuthCli(token) {
			writeUnauth(rw)
			return
		}
		values := req.URL.Query()
		server := values.Get("server")
		scripts, err := h.ListServerWebScripts(server)
		if err != nil {
			writeInternalError(rw, err)
			return
		}
		if scripts == nil {
			scripts = make([]WebScriptId, 0)
		}
		writeJson(rw, http.StatusOK, Map{
			"status": "ok",
			"data": scripts,
		})
	})
	mux.HandleFunc("/web_plugin", func(rw http.ResponseWriter, req *http.Request){
		var err error
		
		values := req.URL.Query()
		switch req.Method {
		case "POST": // Update plugin metadata
			token := req.Header.Get("Authorization")
			if !h.CheckRootToken(token) {
				writeUnauth(rw)
				return
			}
			oper := values.Get("oper")
			switch strings.ToLower(oper) {
			case "create":
				var meta WebScriptMeta
				if err = readJsonBody(req, &meta); err != nil {
					writeJson(rw, http.StatusBadRequest, Map{
						"status": "error",
						"error": err.Error(),
					})
					return
				}
				if err = h.CreatePlugin(meta); err != nil {
					writeInternalError(rw, err)
					return
				}
				writeJson(rw, http.StatusOK, Map{
					"status": "ok",
				})
			case "delete":
				var pid WebScriptId
				if err = readJsonBody(req, &pid); err != nil {
					writeJson(rw, http.StatusBadRequest, Map{
						"status": "error",
						"error": err.Error(),
					})
					return
				}
				if err = h.DeletePlugin(pid); err != nil {
					writeInternalError(rw, err)
					return
				}
				writeJson(rw, http.StatusOK, Map{
					"status": "ok",
				})
			}
		default:
			rw.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	mux.Handle("/web_plugin/", http.StripPrefix("/web_plugin",
		(http.HandlerFunc)(func(rw http.ResponseWriter, req *http.Request){
		var err error

		pluginId, path := splitByte(strings.TrimPrefix(req.URL.Path, "/"), '/')
		if filepath.IsAbs(path) {
			writeJson(rw, http.StatusBadRequest, Map{
				"status": "error",
				"error": "Abs path passed",
			})
			return
		}
		pluginId, version := splitByte(pluginId, '@')
		id := WebScriptId{Id: pluginId, Version: version}

		values := req.URL.Query()
		switch req.Method {
		case "GET": // Read plugin file
			list := values.Has("dir")
			if list {
				files, err := h.ListPluginFiles(id, path)
				if err != nil {
					if errors.Is(err, PluginNotExistsErr) {
						writeJson(rw, http.StatusNotFound, Map{
							"status": "error",
							"error": "Plugin not found",
							"plugin": pluginId,
							"version": version,
						})
						return
					}
					if errors.Is(err, os.ErrNotExist) {
						writeJson(rw, http.StatusNotFound, Map{
							"status": "error",
							"error": "Path is not exists",
							"path": path,
						})
						return
					}
					writeInternalError(rw, err)
					return
				}
				writeJson(rw, http.StatusOK, Map{
					"status": "ok",
					"data": files,
				})
			}else{
				r, modTime, err := h.GetPluginFile(id, path)
				if err != nil {
					if errors.Is(err, PluginNotExistsErr) {
						rw.WriteHeader(http.StatusNotFound)
						fmt.Fprintf(rw, "Plugin %s(%s) not found", pluginId, version)
						return
					}
					if errors.Is(err, os.ErrNotExist) {
						rw.WriteHeader(http.StatusNotFound)
						fmt.Fprintf(rw, "File %q not found", path)
						return
					}
					rw.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(rw, "Error: %v; %q", err, path)
					return
				}
				defer r.Close()
				_, name := splitByteR(path, '/')
				http.ServeContent(rw, req, name, modTime, r)
			}
		case "PUT": // Write plugin file
			defer req.Body.Close()
			token := req.Header.Get("Authorization")
			if !h.CheckRootToken(token) {
				writeUnauth(rw)
				return
			}
			if err = h.PutPluginFile(id, path, req.Body); err != nil {
				if errors.Is(err, PluginNotExistsErr) {
					writeJson(rw, http.StatusNotFound, Map{
						"status": "error",
						"error": "Plugin not found",
						"plugin": pluginId,
						"version": version,
					})
					return
				}
				writeInternalError(rw, err)
				return
			}
			writeJson(rw, http.StatusOK, Map{
				"status": "ok",
			})
		case "DELETE": // Remove plugin file
			token := req.Header.Get("Authorization")
			if !h.CheckRootToken(token) {
				writeUnauth(rw)
				return
			}
			if err = h.DelPluginFile(id, path); err != nil {
				if errors.Is(err, PluginNotExistsErr) {
					writeJson(rw, http.StatusNotFound, Map{
						"status": "error",
						"error": "Plugin not found",
						"plugin": pluginId,
						"version": version,
					})
					return
				}
				writeInternalError(rw, err)
				return
			}
			writeJson(rw, http.StatusOK, Map{
				"status": "ok",
			})
		default:
			rw.WriteHeader(http.StatusMethodNotAllowed)
		}
	})))
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
