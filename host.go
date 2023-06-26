
package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

type HostServer struct {
	id string

	ctx    context.Context
	cancel context.CancelFunc

	connMux sync.RWMutex
	conns   map[int]*Conn
}

func NewHostServer(ctx context.Context, id string)(s *HostServer){
	ctx0, cancel := context.WithCancel(ctx)
	return &HostServer{
		ctx: ctx0,
		cancel: cancel,
		id: id,
		conns: make(map[int]*Conn),
	}
}

func (s *HostServer)Id()(string){
	return s.id
}

func (s *HostServer)AcceptConn(rw http.ResponseWriter, req *http.Request)(conn *Conn, err error){
	var ccId int
	if ccId, err = readCCID(req); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(rw, err.Error())
	}
	s.connMux.Lock()
	if _, ok := s.conns[ccId]; ok {
		s.connMux.Unlock()
		err = fmt.Errorf("Device ID %d is already connected", ccId)
		rw.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(rw, err.Error())
		return
	}
	s.conns[ccId] = nil // take the slot first
	s.connMux.Unlock()

	conn, err = AcceptConn(s.ctx, rw, req)
	s.connMux.Lock()
	s.conns[conn.id] = conn
	s.connMux.Unlock()
	go func(){
		select {
		case <-conn.Context().Done():
			s.connMux.Lock()
			delete(s.conns, conn.id)
			s.connMux.Unlock()
		case <-s.ctx.Done():
		}
	}()
	go conn.Run("shell")
	return
}

func (s *HostServer)GetConn(id int)(*Conn){
	s.connMux.RLock()
	defer s.connMux.RUnlock()
	return s.conns[id]
}

func (s *HostServer)GetConns()(conns []*Conn){
	s.connMux.RLock()
	defer s.connMux.RUnlock()
	conns = make([]*Conn, 0, len(s.conns))
	for _, c := range s.conns {
		conns = append(conns, c)
	}
	return
}

func (s *HostServer)GetConnCount()(n int){
	s.connMux.RLock()
	defer s.connMux.RUnlock()
	return len(s.conns)
}
