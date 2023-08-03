
//go:build tinygo.wasm
package plugin_test

import (
	"context"

	. "github.com/kmcsr/cc-ws2/plugin"
)

type EmptyHookImpl struct {
	EmptyHook
}

var _ HookPlugin = (*EmptyHookImpl)(nil)

func (*EmptyHookImpl)OnLoad(context.Context, *HookLoadEvent)(_ error){ return }
func (*EmptyHookImpl)OnUnload(context.Context, *HookUnloadEvent){ return }
