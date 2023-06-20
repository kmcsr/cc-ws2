
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type ExecErr struct {
	Msg string
}

func (e *ExecErr)Error()(string){
	return "ExecErr: " + e.Msg
}

type TermNotFoundErr struct {
	TermId int
}

func (e *TermNotFoundErr)Error()(string){
	return fmt.Sprintf("Term is not found with id %d", e.TermId)
}

type ConnEventListener = func(conn *Conn, event string, args List)

type Conn struct {
	req    *http.Request
	ws     *websocket.Conn
	addr   string // as same as req.RemoteAddr
	id     int
	device string // The device's type, example are [turtle pocket computer]

	ctx    context.Context
	cancel context.CancelFunc

	askMux sync.Mutex
	askInc int
	asking map[int]chan<- any

	termMux sync.RWMutex
	terms map[int]*Term

	OnEvent ConnEventListener
	TerminateHandler func(c *Conn)(ok bool)
}

func readCCID(req *http.Request)(id int, err error){
	sCcId := req.Header.Get("X-CC-ID")
	id, err = strconv.Atoi(sCcId)
	if err != nil {
		return -1, fmt.Errorf("The value of X-CC-ID (%q) is not a vaild integer", sCcId)
	}
	if id < 0 {
		return -1, fmt.Errorf("X-CC-ID must be a non-negative integer, but got %d", id)
	}
	return
}

func AcceptConn(ctx context.Context, rw http.ResponseWriter, req *http.Request)(c *Conn, err error){
	c = &Conn{
		addr: req.RemoteAddr,
		asking: make(map[int]chan<- any),
		terms: make(map[int]*Term),
	}
	c.ws, err = websocket.Accept(rw, req, nil)
	if err != nil {
		return
	}
	c.ctx, c.cancel = context.WithCancel(ctx)
	if c.id, err = readCCID(req); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(rw, err.Error())
	}
	c.device = req.Header.Get("X-CC-Device")
	return
}

func (c *Conn)Addr()(string){
	return c.addr
}

func (c *Conn)Id()(int){
	return c.id
}

func (c *Conn)Device()(string){
	return c.device
}

func (c *Conn)Context()(context.Context){
	return c.ctx
}

func (c *Conn)onEvent(event string, args ...any){
	if c.OnEvent != nil {
		c.OnEvent(c, event, (List)(args))
	}
}

func (c *Conn)recv()(data Map, err error){
	err = wsjson.Read(c.ctx, c.ws, &data)
	return
}

func (c *Conn)send(data Map)(err error){
	return wsjson.Write(c.ctx, c.ws, data)
}

func (c *Conn)Reply(id int, data any)(err error){
	return c.send(Map{
		"type": "reply",
		"id": id,
		"data": data,
	})
}

func (c *Conn)Close()(err error){
	err = c.send(Map{ "type": "terminate" })
	c.cancel()
	c.ws.Close(websocket.StatusNormalClosure, "remote closed")
	return
}

func (c *Conn)Handle(){
	defer c.ws.Close(websocket.StatusInternalError, "500 internal error")
	defer c.cancel()
	for {
		data, err := c.recv()
		if err != nil {
			var cerr *websocket.CloseError
			if errors.As(err, &cerr) {
				loger.Infof("[%s]: Disconnected: %v", cerr)
			}else{
				loger.Errorf("[%s]: Error when recving data: %v", c.addr, err)
				c.ws.Close(websocket.StatusInternalError, err.Error())
			}
			return
		}
		loger.Debugf("[%s]: Recv: %v", c.addr, data)
		typ, _ := data.GetString("type")
		switch typ {
		case "terminated":
			loger.Infof("[%s]: Terminated", c.addr)
			c.ws.Close(websocket.StatusNormalClosure, "terminated")
			return
		case "terminate":
			if c.TerminateHandler != nil {
				if !c.TerminateHandler(c) {
					loger.Debugf("[%s]: Terminate prevented by handler", c.addr)
					continue
				}
			}
			loger.Infof("[%s]: Terminating", c.addr)
			if err = c.send(Map{ "type": "terminate" }); err != nil {
				loger.Warnf("[%s]: Error when sending terminate: %v", c.addr, err)
				c.ws.Close(websocket.StatusInternalError, err.Error())
				return
			}
			c.ws.Close(websocket.StatusNormalClosure, "terminate")
			return
		case "reply":
			rid, _ := data.GetInt("id")
			c.onReply(rid, data["data"])
		case "event":
			event, _ := data.GetString("event")
			args, _ := data.GetList("args")
			c.onEvent(event, args...)
		case "term_oper":
			rid, _ := data.GetInt("id")
			tdata, _ := data.GetMap("data")
			tid, _ := tdata.GetInt("term")
			oper, _ := tdata.GetString("oper")
			args, _ := tdata.GetList("args")
			res, err := c.onTermOper(tid, oper, args)
			if err != nil {
				c.Reply(rid, Map{
					"status": "error",
					"error": err,
				})
			}else{
				c.Reply(rid, Map{
					"status": "ok",
					"res": res,
				})
			}
		default:
			loger.Debugf("[%s]: Unknown packet type %q", c.addr, typ)
		}
	}
}

func (c *Conn)allocAskId()(id int, resCh <-chan any){
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

func (c *Conn)onReply(id int, data any){
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

func (c *Conn)Ask(typ string, data any)(res any, err error){
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

func (c *Conn)Exec(codes string)(res []any, err error){
	r, err := c.Ask("exec", codes)
	if err != nil {
		return
	}
	r0 := r.(Map)
	status := r0["status"].(string)
	if status != "ok" {
		return nil, &ExecErr{ r0["error"].(string) }
	}
	return r0["res"].([]any), nil
}

func (c *Conn)Run(program string, args ...any)(term *Term, done <-chan bool, err error){
	id, resCh := c.allocAskId()
	term = NewTerm(51, 19, program)
	c.termMux.Lock()
	c.terms[id] = term
	c.termMux.Unlock()
	if err = c.send(Map{
		"id": id,
		"type": "run",
		"data": Map{
			"prog": program,
			"args": args,
		},
	}); err != nil {
		c.termMux.Lock()
		delete(c.terms, id)
		c.termMux.Unlock()
		return
	}
	c.onEvent("#term.open", program, id, term.width, term.height)
	doneCh := make(chan bool, 1)
	done = doneCh
	go func(){
		v, _ := (<-resCh).(bool)
		doneCh <- v
		c.onEvent("#term.close", id, v)
		c.termMux.Lock()
		delete(c.terms, id)
		c.termMux.Unlock()
	}()
	return
}

func (c *Conn)GetTerm(tid int)(t *Term){
	return c.terms[tid]
}

type TermMeta struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
}

func (c *Conn)GetTerms()(terms []TermMeta){
	for id, t := range c.terms {
		terms = append(terms, TermMeta{Id: id, Title: t.Title})
	}
	return
}

func (c *Conn)onTermOper(tid int, oper string, args List)(res []any, err error){
	c.termMux.RLock()
	term, ok := c.terms[tid]
	c.termMux.RUnlock()
	if !ok {
		return nil, &TermNotFoundErr{tid}
	}
	res, err = term.Oper(oper, args)
	if err == nil {
		c.onEvent("#term.oper", tid, oper, args)
	}
	return
}

func (c *Conn)FireEventOnTerm(tid int, event string, args List)(err error){
	return c.send(Map{
		"type": "term_event",
		"term": tid,
		"event": event,
		"args": args,
	})
}
