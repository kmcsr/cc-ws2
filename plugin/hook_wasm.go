
//go:build tinygo.wasm
package plugin

import (
	"context"
	"fmt"

	"github.com/kmcsr/cc-ws2/plugin/protos"
)

type HookPlugin interface {
	OnLoad(context.Context, *HookLoadEvent)(error)
	OnUnload(context.Context, *HookUnloadEvent)
	OnDeviceJoin(context.Context, *DeviceJoinEvent)
	OnDeviceLeave(context.Context, *DeviceLeaveEvent)
	OnDeviceEvent(context.Context, *DeviceEvent)
	OnDeviceCustomEvent(context.Context, *DeviceCustomEvent)
}

type EmptyHook struct{}

func (EmptyHook)OnDeviceJoin(context.Context, *DeviceJoinEvent){}
func (EmptyHook)OnDeviceLeave(context.Context, *DeviceLeaveEvent){}
func (EmptyHook)OnDeviceEvent(context.Context, *DeviceEvent){}
func (EmptyHook)OnDeviceCustomEvent(context.Context, *DeviceCustomEvent){}

type hookWrapper struct {
	meta *HookMetadata
	p HookPlugin
}

var _ protos.Hook = hookWrapper{}

func RegisterHook(meta *HookMetadata, h HookPlugin){
	protos.RegisterHook(hookWrapper{meta, h})
}

func (w hookWrapper)Metadata(ctx context.Context, v *Empty)(res *HookMetadata, err error){
	return w.meta, nil
}

func (w hookWrapper)OnLoad(ctx context.Context, v *HookLoadEvent)(res *Empty, err error){
	err = w.p.OnLoad(ctx, v)
	return
}

func (w hookWrapper)OnUnload(ctx context.Context, v *HookUnloadEvent)(res *Empty, err error){
	w.p.OnUnload(ctx, v)
	return
}

func (w hookWrapper)OnDeviceJoin(ctx context.Context, v *DeviceJoinEvent)(res *Empty, err error){
	w.p.OnDeviceJoin(ctx, v)
	return
}

func (w hookWrapper)OnDeviceLeave(ctx context.Context, v *DeviceLeaveEvent)(res *Empty, err error){
	w.p.OnDeviceLeave(ctx, v)
	return
}

func (w hookWrapper)OnDeviceEvent(ctx context.Context, v *protos.DeviceEvent)(res *Empty, err error){
	args := make([]any, len(v.Args))
	for i, v := range v.Args {
		if args[i], err = v.Unwrap(); err != nil {
			return nil, fmt.Errorf("Error when parsing arg %d: %w", i, err)
		}
	}
	var v0 = &DeviceEvent{
		Device: v.Device,
		Event: v.Event,
		Args: args,
	}
	w.p.OnDeviceEvent(ctx, v0)
	return
}

func (w hookWrapper)OnDeviceCustomEvent(ctx context.Context, v *protos.DeviceCustomEvent)(res *Empty, err error){
	args := make([]any, len(v.Args))
	for i, v := range v.Args {
		if args[i], err = v.Unwrap(); err != nil {
			return nil, fmt.Errorf("Error when parsing arg %d: %w", i, err)
		}
	}
	var v0 = &DeviceCustomEvent{
		Device: v.Device,
		Event: v.Event,
		Args: args,
	}
	w.p.OnDeviceCustomEvent(ctx, v0)
	return
}
