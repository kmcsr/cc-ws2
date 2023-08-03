
package plugin

import (
	protos "github.com/kmcsr/cc-ws2/plugin/protos"
)

type (
	Empty = protos.Empty
	HookMetadata = protos.HookMetadata
	HookLoadEvent = protos.HookLoadEvent
	HookUnloadEvent = protos.HookUnloadEvent

	Device = protos.Device
	DeviceJoinEvent = protos.DeviceJoinEvent
	DeviceLeaveEvent = protos.DeviceLeaveEvent
	DeviceEvent struct {
		Device *Device
		Event  string
		Args   []any
	}
	DeviceCustomEvent struct {
		Device *Device
		Event  string
		Args   []any
	}
)
