
package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/kmcsr/cc-ws2/plugin"
)

type Handler struct {
	DataAPI
	FsAPI

	ctx    context.Context
	cancel context.CancelFunc

	hostMux sync.RWMutex
	hosts   map[string]*HostServer

	cliMux  sync.RWMutex
	clients map[*CliConn]struct{}

	hookManager *plugin.HookManager
}

var _ HandlerI = (*Handler)(nil)

func NewHandler(dtapi DataAPI, fsapi FsAPI)(h *Handler){
	h = &Handler{
		DataAPI: dtapi,
		FsAPI: fsapi,
		hosts: make(map[string]*HostServer),
		clients: make(map[*CliConn]struct{}),
	}
	h.ctx, h.cancel = context.WithCancel(context.Background())
	var err error
	if h.hookManager, err = plugin.NewHookManager(h.ctx, h.newHookAPI); err != nil {
		loger.Panic(err) // TODO: maybe return err?
	}
	return 
}

type hookAPI struct {
	h *Handler
	hook string
}

var _ plugin.HookAPI = (*hookAPI)(nil)

func (api *hookAPI)FireEvent(ctx context.Context, hostid string, deviceid int64, data map[string]any)(err error){
	host := api.h.GetHost(hostid)
	if host == nil {
		return fmt.Errorf("Host %q is not exists", hostid)
	}
	conn := host.GetConn(deviceid)
	if conn == nil {
		return fmt.Errorf("Device %d is not connected", deviceid)
	}
	if err = conn.send((Map)(data)); err != nil {
		return
	}
	return
}

func (h *Handler)newHookAPI(hookid string)(api plugin.HookAPI, err error){
	return &hookAPI{
		h: h,
		hook: hookid,
	}, nil
}

func (h *Handler)Context()(context.Context){
	return h.ctx
}

func (h *Handler)HookManager()(*plugin.HookManager){
	return h.hookManager
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

func (h *Handler)removeHost(id string){
	h.hostMux.Lock()
	host, ok := h.hosts[id]
	if ok {
		delete(h.hosts, id)
	}
	h.hostMux.Unlock()
	if ok {
		host.Destroy()
	}
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

func (h *Handler)BroadcastToClients(event string, data any, except *CliConn){
	h.cliMux.RLock()
	defer h.cliMux.RUnlock()
	for c, _ := range h.clients {
		if c != except {
			c.send(Map{
				"type": "custom_event",
				"event": event,
				"data": data,
			})
		}
	}
}

func (h *Handler)BroadcastToClientsWithHost(hostid string, typ string, data any){
	h.cliMux.RLock()
	defer h.cliMux.RUnlock()
	for c, _ := range h.clients {
		if h.CheckPerm(c.token, hostid) {
			c.send(Map{
				"type": typ,
				"host": hostid,
				"data": data,
			})
		}
	}
}

func (h *Handler)onWsdEvent(host *HostServer, conn *Conn, event string, args List){
	if len(event) == 0 {
		return
	}
	hostid := host.Id()
	if event[0] == '#' { // internal events
		event = event[1:]
		h.BroadcastToClientsWithHost(hostid, event, Map{
			"conn": conn.Id(),
			"args": args,
		})
		return
	}
	if event[0] == '$' { // event that need to be send to hooks ($<hookid>:<event_type>)
		var hookid string
		hookid, event = splitByte(event[1:], ':')
		if hook := h.hookManager.Get(hookid); hook != nil {
			hook.OnDeviceCustomEvent(h.ctx, &plugin.DeviceCustomEvent{
				Device: &plugin.Device{
					Host: hostid,
					Id: conn.Id(),
				},
				Event: event,
				Args: args,
			})
		}
		return
	}
	h.BroadcastToClientsWithHost(hostid, "device_event", Map{
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
	if !h.AuthDaemon(authTk, remoteHost) {
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
	h.BroadcastToClientsWithHost(remoteHost, "device_join", Map{
		"conn": conn.Id(),
		"addr": conn.Addr(),
		"device": conn.Device(),
		"label": conn.Label(),
	})
	defer func(){
		h.BroadcastToClientsWithHost(remoteHost, "device_leave", Map{
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
	if !h.AuthCli(authTk) {
		rw.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(rw, "401 Unauthorized")
		return
	}
	conn, err := AcceptCliConn(h, authTk, rw, req)
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

func (h *Handler)NewServeMux()(mux *http.ServeMux){
	mux = http.NewServeMux()
	mux.Handle("/main/", webAssetsHandler)
	mux.Handle("/api/", http.StripPrefix("/api", h.newApiMux()))
	mux.HandleFunc("/wscli", h.serveWscli)
	mux.HandleFunc("/wsd", h.serveWsd)
	mux.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request){
		http.NotFound(rw, req)
	})
	return
}
