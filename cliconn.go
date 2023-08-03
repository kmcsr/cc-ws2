
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type HandlerI interface {
	DataAPI
	FsAPI

	Context()(context.Context)
	GetHost(id string)(*HostServer)
	GetHosts()([]*HostServer)
	BroadcastToClients(event string, data any, except *CliConn)
}

// This connection is only used when outside of CC
type CliConn struct {
	token  string
	req    *http.Request
	ws     *websocket.Conn
	addr   string

	handler HandlerI

	ctx    context.Context
	cancel context.CancelFunc

	askMux sync.Mutex
	askInc int
	asking map[int]chan<- any
}

func AcceptCliConn(handler HandlerI, token string, rw http.ResponseWriter, req *http.Request)(c *CliConn, err error){
	c = &CliConn{
		handler: handler,
		token: token,
		addr: req.RemoteAddr,
		asking: make(map[int]chan<- any),
	}
	c.ws, err = websocket.Accept(rw, req, nil)
	if err != nil {
		return
	}
	c.ctx, c.cancel = context.WithCancel(handler.Context())
	go func(){
		for {
			select {
			case <-time.After(10 * time.Second):
				c.send(Map{
					"type": "ping",
				})
			case <-c.ctx.Done():
				return
			}
		}
	}()
	return
}

func (c *CliConn)Addr()(string){
	return c.addr
}

func (c *CliConn)recv()(data Map, err error){
	err = wsjson.Read(c.ctx, c.ws, &data)
	return
}

func (c *CliConn)send(data Map)(err error){
	return wsjson.Write(c.ctx, c.ws, data)
}

func (c *CliConn)Reply(id int, data any)(err error){
	return c.send(Map{
		"type": "reply",
		"id": id,
		"data": data,
	})
}

func (c *CliConn)Close()(err error){
	c.cancel()
	c.ws.Close(websocket.StatusNormalClosure, "remote closed")
	return
}

func (c *CliConn)checkAndGetHost(rid int, hostid string)(host *HostServer){
	if !c.handler.CheckPerm(c.token, hostid) {
		c.Reply(rid, Map{
			"status": "error",
			"error": "Host not found or permission denied",
			"hostid": hostid,
		})
		return
	}
	if host = c.handler.GetHost(hostid); host == nil {
		c.Reply(rid, Map{
			"status": "error",
			"error": "Host not found",
			"hostid": hostid,
		})
		return
	}
	return
}

func (c *CliConn)Handle(){
	defer c.ws.Close(websocket.StatusInternalError, "500 internal error")
	for {
		data, err := c.recv()
		if err != nil {
			var cerr websocket.CloseError
			if errors.As(err, &cerr) {
				loger.Infof("[%s]: Disconnected: %v", c.addr, cerr)
			}else{
				loger.Errorf("[%s]: Error when recving data: %v", c.addr, err)
				c.ws.Close(websocket.StatusInternalError, err.Error())
			}
			return
		}
		loger.Debugf("[%s]: Recv from Cli: %v", c.addr, data)
		typ, _ := data.GetString("type")
		switch typ {
		case "reply":
			rid, _ := data.GetInt("id")
			c.onReply(rid, data["data"])
		case "user_info":
			id, _ := data.GetInt("id")
			var data UserInfo
			data, err := c.handler.GetUserInfo(c.token)
			if err != nil {
				c.Reply(id, Map{
					"status": "error",
					"error": err.Error(),
				})
				return
			}
			c.Reply(id, Map{
				"status": "ok",
				"data": data,
			})
		case "broadcast_cli":
			event, _ := data.GetString("event")
			data := data.Get("data")
			c.handler.BroadcastToClients(event, data, c)
		case "broadcast":
			hid, _ := data.GetString("host")
			if !c.handler.CheckPerm(c.token, hid) {
				break
			}
			tdata, _ := data.GetMap("data")
			if host := c.handler.GetHost(hid); host != nil {
				host.Broadcast(tdata)
			}
		case "fire_event":
			hid, _ := data.GetString("host")
			cid, _ := data.GetInt64("conn")
			tid, _ := data.GetInt("term")
			event, _ := data.GetString("event")
			args, _ := data.GetList("args")
			if !c.handler.CheckPerm(c.token, hid) {
				break
			}
			if host := c.handler.GetHost(hid); host != nil {
				if conn := host.GetConn(cid); conn != nil {
					conn.FireEventOnTerm(tid, event, args)
				}
			}
		case "list_hosts":
			id, _ := data.GetInt("id")
			nhosts := c.handler.GetHosts()
			type connMeta struct {
				Id     int64  `json:"id"`
				Addr   string `json:"addr"`
				Device string `json:"device"`
				Label  string `json:"label"`
			}
			var nilConnMeta = make([]connMeta, 0)
			type hostMeta struct {
				Id string        `json:"id"`
				Conns []connMeta `json:"conns"`
			}
			permhosts, err := c.handler.ListServers(c.token)
			if err != nil {
				c.Reply(id, Map{
					"status": "error",
					"error": err.Error(),
				})
				break
			}
			sort.Slice(nhosts, func(i, j int)(bool){ return nhosts[i].Id() < nhosts[j].Id() })
			hosts := make([]hostMeta, len(permhosts))
			for i, hid := range permhosts {
				j := sort.Search(len(nhosts), func(i int)(bool){ return nhosts[i].Id() >= hid })
				if j < len(nhosts) && nhosts[j].Id() == hid {
					h := nhosts[j]
					nconns := h.GetConns()
					conns := make([]connMeta, len(nconns))
					for i, c := range nconns {
						conns[i] = connMeta{
							Id: c.Id(),
							Addr: c.Addr(),
							Device: c.Device(),
							Label: c.Label(),
						}
					}
					hosts[i] = hostMeta{
						Id: hid,
						Conns: conns,
					}
				}else{
					hosts[i] = hostMeta{
						Id: hid,
						Conns: nilConnMeta,
					}
				}
			}
			c.Reply(id, Map{
				"status": "ok",
				"data": hosts,
			})
		case "get_host":
			id, _ := data.GetInt("id")
			hostid, _ := data.GetString("data")
			host := c.checkAndGetHost(id, hostid)
			if host == nil {
				break
			}
			type connMeta struct {
				Id     int64  `json:"id"`
				Addr   string `json:"addr"`
				Device string `json:"device"`
				Label  string `json:"label"`
			}
			var res struct {
				Id string `json:"id"`
				Conns []connMeta `json:"conns"`
			}
			res.Id = host.Id()
			conns := host.GetConns()
			res.Conns = make([]connMeta, len(conns))
			for i, c := range conns {
				res.Conns[i] = connMeta{
					Id: c.Id(),
					Addr: c.Addr(),
					Device: c.Device(),
					Label: c.Label(),
				}
			}
			c.Reply(id, Map{
				"status": "ok",
				"res": res,
			})
		case "list_terms":
			id, _ := data.GetInt("id")
			dt, _ := data.GetMap("data")
			hostid, _ := dt.GetString("host")
			connid, _ := dt.GetInt64("conn")
			host := c.checkAndGetHost(id, hostid)
			if host == nil {
				break
			}
			conn := host.GetConn(connid)
			if conn == nil {
				c.Reply(id, Map{
					"status": "error",
					"error": "Conn not found",
					"connid": connid,
				})
				break
			}
			c.Reply(id, Map{
				"status": "ok",
				"res": conn.GetTerms(),
			})
		case "get_term":
			id, _ := data.GetInt("id")
			dt, _ := data.GetMap("data")
			hostid, _ := dt.GetString("host")
			connid, _ := dt.GetInt64("conn")
			termid, _ := dt.GetInt("term")
			host := c.checkAndGetHost(id, hostid)
			if host == nil {
				break
			}
			conn := host.GetConn(connid)
			if conn == nil {
				c.Reply(id, Map{
					"status": "error",
					"error": "Conn not found",
					"connid": connid,
				})
				break
			}
			term := conn.GetTerm(termid)
			if term == nil {
				c.Reply(id, Map{
					"status": "error",
					"error": fmt.Sprintf("Term %d not found", termid),
				})
				break
			}
			c.Reply(id, Map{
				"status": "ok",
				"res": Map{
					"title": term.Title,
					"width": term.width,
					"height": term.height,
					"cursorX": term.cursorX,
					"cursorY": term.cursorY,
					"textColor": term.textColor,
					"backgroundColor": term.backgroundColor,
					"cursorBlink": term.cursorBlink,
					"palette": term.palette,
					"lines": term.lines,
				},
			})
		case "run":
			id, _ := data.GetInt("id")
			dt, _ := data.GetMap("data")
			hostid, _ := dt.GetString("host")
			connid, _ := dt.GetInt64("conn")
			program, _ := dt.GetString("prog")
			args, _ := dt.GetList("args")
			host := c.checkAndGetHost(id, hostid)
			if host == nil {
				break
			}
			conn := host.GetConn(connid)
			if conn == nil {
				c.Reply(id, Map{
					"status": "error",
					"error": fmt.Sprintf("Conn %d not found", connid),
				})
				break
			}
			_, _, err := conn.Run(program, args...)
			if err != nil {
				c.Reply(id, Map{
					"status": "failed",
					"error": err.Error(),
				})
				break
			}
			c.Reply(id, Map{
				"status": "ok",
			})
		case "exec":
			id, _ := data.GetInt("id")
			dt, _ := data.GetMap("data")
			hostid, _ := dt.GetString("host")
			connid, _ := dt.GetInt64("conn")
			codes, _ := dt.GetString("codes")
			host := c.checkAndGetHost(id, hostid)
			if host == nil {
				break
			}
			conn := host.GetConn(connid)
			if conn == nil {
				c.Reply(id, Map{
					"status": "error",
					"error": fmt.Sprintf("Conn %d not found", connid),
				})
				break
			}
			go func(){
				res, err := conn.Exec(codes)
				if err != nil {
					c.Reply(id, Map{
						"status": "failed",
						"error": err.Error(),
					})
					return
				}
				c.Reply(id, Map{
					"status": "ok",
					"res": res,
				})
			}()
		default:
			loger.Debugf("[%s]: Unknown packet type %q", c.addr, typ)
		}
	}
}

func (c *CliConn)allocAskId()(id int, resCh <-chan any){
	ch := make(chan any, 1)
	resCh = ch
	c.askMux.Lock()
	ok := true
	id = c.askInc
	for ok {
		id++
		_, ok = c.asking[id]
	}
	c.askInc = id
	c.asking[id] = ch
	c.askMux.Unlock()
	return
}

func (c *CliConn)onReply(id int, data any){
	c.askMux.Lock()
	replyCh, ok := c.asking[id]
	if ok {
		delete(c.asking, id)
	}
	c.askMux.Unlock()
	if ok {
		replyCh <- data
	}
}

func (c *CliConn)Ask(typ string, data any)(res any, err error){
	id, resCh := c.allocAskId()
	if err = c.send(Map{
		"id": id,
		"type": typ,
		"data": data,
	}); err != nil {
		return
	}
	res = <-resCh
	return
}
