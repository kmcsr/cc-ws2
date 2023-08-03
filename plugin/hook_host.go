
//go:build !tinygo.wasm
package plugin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	protos "github.com/kmcsr/cc-ws2/plugin/protos"
)

type (
	HookAPI interface {
		FireEvent(ctx context.Context, host string, device int64, data map[string]any)(err error)
	}
)

var ErrAPINotBind = errors.New("API can only be called after init")

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

func (h *Hook)OnDeviceCustomEvent(ctx context.Context, v *DeviceCustomEvent)(err error){
	args := make([]*protos.Any, len(v.Args))
	for i, v := range v.Args {
		if args[i], err = protos.WrapValue(v); err != nil {
			return
		}
	}
	v0 := &protos.DeviceCustomEvent{
		Device: v.Device,
		Event: v.Event,
		Args: args,
	}
	_, err = h.native.OnDeviceCustomEvent(ctx, v0)
	return
}

type HookAPIGetter = func(hookid string)(HookAPI, error)

type HookManager struct {
	ctx    context.Context
	cancel context.CancelFunc

	native    *protos.HookPlugin
	apiGetter HookAPIGetter

	hookMux sync.RWMutex
	hooks   map[string]*Hook
}

func NewHookManager(ctx context.Context, apiGetter HookAPIGetter)(m *HookManager, err error){
	m = &HookManager{
		hooks: make(map[string]*Hook),
		apiGetter: apiGetter,
	}
	m.ctx, m.cancel = context.WithCancel(ctx)
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

func (m *HookManager)unloadAll(ctx context.Context){
	for _, h := range m.hooks {
		unloadE := &protos.HookUnloadEvent{}
		h.native.OnUnload(ctx, unloadE) // ignore errors
	}
}

func (m *HookManager)LoadFromDir(ctx context.Context, path string)(errs []error){
	m.hookMux.Lock()
	defer m.hookMux.Unlock()
	return m.loadFromDir(ctx, path)
}

func (m *HookManager)loadFromDir(ctx context.Context, path string)(errs []error){
	files, err := os.ReadDir(path)
	if err != nil {
		return
	}
	m.hooks = make(map[string]*Hook, len(files))
	for _, f := range files {
		if !f.IsDir() {
			if strings.HasSuffix(f.Name(), ".wasm") {
				p := filepath.Join(path, f.Name())
				if _, er := m.load(ctx, p); er != nil {
					errs = append(errs, fmt.Errorf("Error when loading %q: %w", p, er))
				}
			}
		}
	}
	return
}

func (m *HookManager)ReloadFromDir(ctx context.Context, path string)(errs []error){
	m.hookMux.Lock()
	defer m.hookMux.Unlock()

	m.unloadAll(ctx)

	return m.loadFromDir(ctx, path)
}

type hookApiWrapper struct {
	api HookAPI
}

var _ protos.HookAPI = (*hookApiWrapper)(nil)

func (w *hookApiWrapper)FireEvent(ctx context.Context, event *protos.FireEventReq)(_ *protos.Empty, err error){
	if w.api == nil {
		err = ErrAPINotBind
		return
	}
	hostid, deviceid := event.Target.Host, event.Target.Id
	data := make(map[string]any, len(event.Data))
	for k, v := range event.Data {
		if data[k], err = v.Unwrap(); err != nil {
			return
		}
	}
	err = w.api.FireEvent(ctx, hostid, deviceid, data)
	return
}


func (m *HookManager)Load(ctx context.Context, path string)(h *Hook, err error){
	m.hookMux.Lock()
	defer m.hookMux.Unlock()
	if h, err = m.load(ctx, path); err != nil {
		return
	}
	m.hooks[h.Id()] = h
	return
}

func (m *HookManager)load(ctx context.Context, path string)(h *Hook, err error){
	h = new(Hook)
	wapi := new(hookApiWrapper)
	if h.native, err = m.native.Load(ctx, path, wapi); err != nil {
		return
	}
	if h.metadata, err = h.native.Metadata(ctx, nil); err != nil {
		return
	}
	if wapi.api, err = m.apiGetter(h.metadata.Id); err != nil {
		return
	}
	id := h.Id()
	if old := m.hooks[id]; old != nil {
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

type (
	HookError struct {
		Hook *Hook
		Origin error
	}
	HookErrorList []*HookError
)

var _ error = (*HookError)(nil)
var _ error = (HookErrorList)(nil)

func (e *HookError)Unwrap()(error){ return e.Origin }
func (e *HookError)Error()(string){
	return fmt.Sprintf("Hook %s(v%s) error: %s", e.Hook.Id(), e.Hook.Version(), e.Origin.Error())
}

func (e HookErrorList)Unwrap()(errs []error){
	errs = make([]error, len(e))
	for i, err := range e {
		errs[i] = err
	}
	return 
}

func (e HookErrorList)Error()(string){
	if len(e) == 0 {
		return ""
	}
	if len(e) == 1 {
		return e[0].Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Hook errors: (total %d)\n", len(e))
	for _, err := range e {
		b.WriteString("\t")
		b.WriteString(err.Error())
	}
	return b.String()
}

func (m *HookManager)ForEach(cb func(*Hook)(error))(errs HookErrorList){
	m.hookMux.RLock()
	defer m.hookMux.RUnlock()
	var err error
	for _, h := range m.hooks {
		if err = cb(h); err != nil {
			errs = append(errs, &HookError{h, err})
		}
	}
	return
}

func (m *HookManager)OnDeviceJoin(event *DeviceJoinEvent)(err error){
	return m.ForEach(func(h *Hook)(err error){
		_, err = h.native.OnDeviceJoin(m.ctx, event)
		return
	})
}

func (m *HookManager)OnDeviceLeave(event *DeviceLeaveEvent)(err error){
	return m.ForEach(func(h *Hook)(err error){
		_, err = h.native.OnDeviceLeave(m.ctx, event)
		return
	})
}

func (m *HookManager)OnDeviceEvent(event *DeviceEvent)(err error){
	args := make([]*protos.Any, len(event.Args))
	for i, v := range event.Args {
		if args[i], err = protos.WrapValue(v); err != nil {
			return
		}
	}
	event0 := &protos.DeviceEvent{
		Device: event.Device,
		Event: event.Event,
		Args: args,
	}

	return m.ForEach(func(h *Hook)(err error){
		_, err = h.native.OnDeviceEvent(m.ctx, event0)
		return
	})
}
