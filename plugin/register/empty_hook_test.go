
//go:build tinygo.wasm
package register_test

import (
	"context"

	. "github.com/kmcsr/cc-ws2/plugin/register"
)

type EmptyHookImpl struct {
	EmptyHook
}

var _ HookPlugin = (*EmptyHookImpl)(nil)

func (*EmptyHookImpl)OnLoad(context.Context, *HookLoadEvent)(_ error){ return }
func (*EmptyHookImpl)OnUnload(context.Context, *HookUnloadEvent){ return }
