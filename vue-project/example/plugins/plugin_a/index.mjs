
import { createApp, ref } from 'https://unpkg.com/vue@3/dist/vue.esm-browser.prod.js'

export function onDeviceJoin(device){
	console.log(`Device ${device.id}(${device.device}) at ${device.host.id} joined`)
}

export function onDeviceLeave(device){
	console.log(`Device ${device.id}(${device.device}) at ${device.host.id} leaved`)
}

export function onHostFocused(host, context){
	console.log(`Focusing host ${host.id}`)
	const element = context.allocHTMLBlock()
	createApp({
		data(){
			return {
				host: host,
				context: context,
			}
		},
		template: hostUITemplate,
	}).mount(element)
}

export function onDeviceFocused(device, context){
	console.log(`Focusing device ${device.id}(${device.device}) at ${device.host.id}`)
}

const hostUITemplate = `
	<div>Host is {{host}}</div>
`
