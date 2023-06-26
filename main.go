
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var startTime = time.Now() // or maybe build time

type Config struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

var config *Config = loadConfig()

func loadConfig()(cfg *Config){
	const configPath = "/etc/cc_ws2/config.json"
	var data []byte
	var err error
	if data, err = os.ReadFile(configPath); err != nil {
		loger.Fatalf("Cannot read config at %s: %v", configPath, err)
	}
	cfg = new(Config)
	if err = json.Unmarshal(data, cfg); err != nil {
		loger.Fatalf("Cannot parse config at %s: %v", configPath, err)
	}
	return
}

type Handler struct{
	ctx    context.Context
	cancel context.CancelFunc

	hostMux sync.RWMutex
	hosts   map[string]*HostServer

	cliMux  sync.RWMutex
	clients map[*CliConn]struct{}
}

func NewHandler()(h *Handler){
	h = &Handler{
		hosts: make(map[string]*HostServer),
		clients: make(map[*CliConn]struct{}),
	}
	h.ctx, h.cancel = context.WithCancel(context.Background())
	return 
}

func (h *Handler)Context()(context.Context){
	return h.ctx
}

func (h *Handler)CreateHost(id string)(s *HostServer){
	h.hostMux.Lock()
	defer h.hostMux.Unlock()
	if _, ok := h.hosts[id]; !ok {
		s = NewHostServer(h.ctx, id)
		h.hosts[id] = s
	}
	return
}

func (h *Handler)GetHost(id string)(*HostServer){
	h.hostMux.RLock()
	defer h.hostMux.RUnlock()
	return h.hosts[id]
}

func (h *Handler)getOrCreateHost(id string)(host *HostServer){
	var ok bool
	h.hostMux.RLock()
	host, ok = h.hosts[id]
	h.hostMux.RUnlock()
	if !ok {
		host = NewHostServer(h.ctx, id)
		h.hostMux.Lock()
		h.hosts[id] = host
		h.hostMux.Unlock()
	}
	return
}

func (h *Handler)GetHosts()(hosts []*HostServer){
	h.hostMux.RLock()
	defer h.hostMux.RUnlock()
	hosts = make([]*HostServer, 0, len(h.hosts))
	for _, s := range h.hosts {
		hosts = append(hosts, s)
	}
	return
}

func (h *Handler)auth(authTk string)(bool){
	if authTk == "test" {
		return true
	}
	return false
}

func (h *Handler)authDaemon(authTk string, host string)(ok bool){
	if authTk == "test" {
		return true
	}
	return false
}

func (h *Handler)BroadcastToClients(typ string, data any){
	h.cliMux.RLock()
	defer h.cliMux.RUnlock()
	for c, _ := range h.clients {
		c.send(Map{
			"type": typ,
			"data": data,
		})
	}
}

func (h *Handler)onWsdEvent(host *HostServer, conn *Conn, event string, args List){
	if len(event) == 0 {
		return
	}
	if event[0] == '#' {
		event = event[1:]
		h.BroadcastToClients(event, Map{
			"host": host.Id(),
			"conn": conn.Id(),
			"args": args,
		})
		return
	}
	h.BroadcastToClients("device_event", Map{
		"host": host.Id(),
		"conn": conn.Id(),
		"event": event,
		"args": args,
	})
}

func (h *Handler)serveWsd(rw http.ResponseWriter, req *http.Request){
	remoteAddr := req.RemoteAddr
	loger.Tracef("[%s] (a daemon) connecting with: %v", remoteAddr, req.Header)

	var (
		authTk string
		remoteHost string
		err error
	)
	authTk = req.Header.Get("X-CC-Auth")
	remoteHost = req.Header.Get("X-CC-Host")
	if !h.authDaemon(authTk, remoteHost) {
		rw.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(rw, "401 Unauthorized")
		return
	}
	host := h.getOrCreateHost(remoteHost)

	conn, err := host.AcceptConn(rw, req)
	if err != nil {
		loger.Errorf("Error when accepting [%s]: %v", remoteAddr, err)
		return
	}
	conn.OnEvent = func(conn *Conn, event string, args List){
		h.onWsdEvent(host, conn, event, args)
	}
	h.BroadcastToClients("device_join", Map{
		"host": host.Id(),
		"conn": conn.Id(),
		"addr": conn.Addr(),
		"device": conn.Device(),
		"label": conn.Label(),
	})
	defer func(){
		h.BroadcastToClients("device_leave", Map{
			"host": host.Id(),
			"conn": conn.Id(),
		})
	}()
	conn.Handle()
}

func (h *Handler)serveWscli(rw http.ResponseWriter, req *http.Request){
	remoteAddr := req.RemoteAddr
	loger.Tracef("[%s] (a client) connecting with: %v", remoteAddr, req.Header)

	var (
		authTk string
		err error
	)
	que := req.URL.Query()
	authTk = que.Get("authTk")
	if !h.auth(authTk) {
		rw.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(rw, "401 Unauthorized")
		return
	}
	conn, err := AcceptCliConn(h, rw, req)
	if err != nil {
		loger.Errorf("Error when accepting cli [%s]: %v", remoteAddr, err)
		return
	}
	h.cliMux.Lock()
	h.clients[conn] = struct{}{}
	h.cliMux.Unlock()
	defer func(){
		h.cliMux.Lock()
		defer h.cliMux.Unlock()
		delete(h.clients, conn)
	}()
	conn.Handle()
}

//go:embed index.html
var mainHtml string

var debugDist http.Handler = http.StripPrefix("/main", http.FileServer(http.Dir("./vue-project/dist")))

func (h *Handler)ServeHTTP(rw http.ResponseWriter, req *http.Request){
	path := req.URL.Path
	if len(path) == 0 || path[0] != '/' {
		path = "/" + path
	}
	if path == "/wsd" {
		h.serveWsd(rw, req)
		return
	}
	if path == "/wscli" {
		h.serveWscli(rw, req)
		return
	}
	if path == "/main" || strings.HasPrefix(path, "/main/") {
		// http.ServeContent(rw, req, "main.html", startTime, strings.NewReader(mainHtml))
		debugDist.ServeHTTP(rw, req)
		return
	}
	rw.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(rw, "Path %q is not fonud", path)
}

func main(){
	handler := NewHandler()

	server := &http.Server{
		Addr: net.JoinHostPort(config.Host, strconv.Itoa(config.Port)),
		Handler: handler,
	}

	done := make(chan struct{}, 0)
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

	go func(){
		defer close(done)
		loger.Infof("Server start at %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			loger.Fatal(err)
		}
	}()

	select {
	case <-sigch:
		ctx, cancel := context.WithTimeout(context.Background(), time.Second * 3)
		server.Shutdown(ctx)
		cancel()
	case <-done:
	}
}
