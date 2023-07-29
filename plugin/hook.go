
package plugin

import (
	"context"
	"fmt"
	"sync"

	protos "github.com/kmcsr/cc-ws2/plugin/protos"
)


type HookExistsErr struct {
	Instance *Hook
}

func (e *HookExistsErr)Error()(string){
	return fmt.Sprintf("Hook <%s> is already exists with version %s", e.Instance.Id(), e.Instance.Version())
}

type HookNotExistsErr struct {
	Id string
}

func (e *HookNotExistsErr)Error()(string){
	return fmt.Sprintf("Hook <%s> is not exists", e.Id)
}

type (
	HookMetadata = protos.HookMetadata
	Device = protos.Device
	DeviceJoinEvent = protos.DeviceJoinEvent
	DeviceLeaveEvent = protos.DeviceLeaveEvent
	DeviceEvent = protos.DeviceEvent
)

type Hook struct {
	native protos.Hook	
	metadata *HookMetadata
}

func (h *Hook)Id()(string){
	return h.metadata.GetId()
}

func (h *Hook)Version()(string){
	return h.metadata.GetVersion()
}

type HookManager struct {
	ctx    context.Context
	cancel context.CancelFunc

	native *protos.HookPlugin

	hookMux sync.RWMutex
	hooks   map[string]*Hook
}

func NewHookManager()(m *HookManager, err error){
	m = &HookManager{
		hooks: make(map[string]*Hook),
	}
	m.ctx, m.cancel = context.WithCancel(context.TODO())
	if m.native, err = protos.NewHookPlugin(m.ctx); err != nil {
		return
	}
	return
}

func (m *HookManager)Get(id string)(*Hook){
	m.hookMux.RLock()
	defer m.hookMux.RUnlock()
	return m.hooks[id]
}

func (m *HookManager)List()(hooks []*Hook){
	m.hookMux.RLock()
	defer m.hookMux.RUnlock()
	hooks = make([]*Hook, 0, len(m.hooks))
	for _, h := range m.hooks {
		hooks = append(hooks, h)
	}
	return
}

func (m *HookManager)Load(ctx context.Context, path string)(h *Hook, err error){
	h = new(Hook)
	if h.native, err = m.native.Load(ctx, path); err != nil {
		return
	}
	if h.metadata, err = h.native.Metadata(ctx, nil); err != nil {
		return
	}
	id := h.Id()
	if old := m.Get(id); old != nil {
		err = &HookExistsErr{old}
		return
	}
	loadE := &protos.HookLoadEvent{
		Reload: false,
	}
	if _, err = h.native.OnLoad(ctx, loadE); err != nil {
		return
	}
	return
}

func (m *HookManager)Unload(ctx context.Context, id string)(err error){
	h := m.Get(id)
	if h == nil {
		return &HookNotExistsErr{id}
	}
	unloadE := &protos.HookUnloadEvent{}
	_, err = h.native.OnUnload(ctx, unloadE)
	m.hookMux.Lock()
	delete(m.hooks, id)
	m.hookMux.Unlock()
	return
}

func (m *HookManager)OnDeviceJoin(event *DeviceJoinEvent){
	m.hookMux.RLock()
	defer m.hookMux.RUnlock()
	for _, h := range m.hooks {
		h.native.OnDeviceJoin(m.ctx, event)
	}
}

func (m *HookManager)OnDeviceLeave(event *DeviceLeaveEvent){
	m.hookMux.RLock()
	defer m.hookMux.RUnlock()
	for _, h := range m.hooks {
		h.native.OnDeviceLeave(m.ctx, event)
	}
}

func (m *HookManager)OnDeviceEvent(event *DeviceEvent){
	m.hookMux.RLock()
	defer m.hookMux.RUnlock()
	for _, h := range m.hooks {
		h.native.OnDeviceEvent(m.ctx, event)
	}
}
