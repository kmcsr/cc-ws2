
import { createApp, ref } from 'https://unpkg.com/vue@3/dist/vue.esm-browser.prod.js'

export function onHostLoad(context, host){
	console.log(`Focusing host ${host.id}`)
	const element = context.allocHTMLBlock()
	createApp({
		template: hostUITemplate,
		data(){
			return {
				host: host,
			}
		},
	}).mount(element)
}

export function onDeviceLoad(context, device){
	console.log(`Focusing device ${device.id}(${device.device}) at ${device.host.id}`)
	const element = context.allocHTMLBlock()
	const app = createApp({
		template: `<h3>Hello!!!</h3>`,
		data(){
			return {
				host: host,
			}
		},
		methods: {
			onEvent({event, args}){
				console.log('Device event:', event, 'with', args)
			},
		},
	}).mount(element)
	context.addEventListener(app.onEvent)
}

const hostUITemplate = `
	<div>Host is {{host}}</div>
`
