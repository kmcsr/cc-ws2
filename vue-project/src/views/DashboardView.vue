<script setup>
import { ref, watch, provide, onMounted, onBeforeUnmount, onErrorCaptured } from 'vue'
import { onBeforeRouteUpdate, RouterView } from 'vue-router'
import axios from 'axios'
import Cube3D from '../components/Cube3D.vue'
import { insertBefore } from '../sort.js'
import { Lock } from '../lock.js'

const props = defineProps({
	token: String,
})

const userinfo = ref(null)
provide('userinfo', userinfo)

const loadError = ref(null)
const plugins = ref({})

const hosts = ref([])
const connected = ref(false)
const routerRef = ref(null)

onBeforeRouteUpdate(() => {
	loadError.value = null
})

onErrorCaptured((err) => {
	if(err.error){
		loadError.value = err.error
	}else{
		loadError.value = String(err)
	}
})

async function connectWs(token){
	const ws = new WebSocket(`${window.location.origin.replace('http', 'ws')}/wscli?authTk=${encodeURIComponent(token)}`)
	await new Promise((resolve, reject) => {
		ws.addEventListener('open', (event) => {
			resolve(event)
			ws.removeEventListener('error', reject)
		})
		ws.addEventListener('error', reject)
	})
	var askIncreasement = 0
	var asking = {}
	ws.send_native = ws.send
	ws.send = function(msg){
		return ws.send_native(JSON.stringify(msg))
	}
	ws.askSync = function(type, data, resolve){
		let id = askIncreasement
		while(asking[id = (id + 1) & 0xffffffff]);
		asking[id] = resolve
		askIncreasement = id
		ws.send({
			type: type,
			id: id,
			data: data,
		})
	}
	ws.ask = function(type, data){
		return new Promise((resolve) => {
			ws.askSync(type, data, resolve)
		})
	}
	ws.fireTermEvent = function(host, conn, term, event, ...args){
		ws.send({
			type: 'fire_event',
			host: host,
			conn: conn,
			term: term,
			event: event,
			args: args,
		})
	}
	ws.addEventListener('message', (event0) => {
		const event = JSON.parse(event0.data)
		const data = event.data
		switch(event.type){
		case 'ping': {
			// TODO
			break
		}
		case 'reply': {
			const resolve = asking[event.id]
			if(resolve){
				resolve(data)
			}else{
				console.warn('Unexcept reply id', event.id)
			}
			break
		}
		}
	})
	ws.addEventListener('close', (event) => {
		console.error('websocket closed:', event)
	})
	ws.addEventListener('error', (event) => {
		console.error('websocket on error:', event)
	})
	return ws
}

var wsconn = null

function initWs(ws){
	function _getConnObj(hostid, data){
		const host = hosts.value.find((h) => h.id === hostid)
		if(host){
			const conn = host.conns.find((c) => c.id === data.conn)
			return conn
		}
		return null
	}
	ws.addEventListener('message', (event0) => {
		const event = JSON.parse(event0.data)
		const data = event.data
		switch(event.type){
		case 'device_join': {
			const hostid = event.host
			var host = hosts.value.find((h) => h.id === hostid)
			if(host){
				const conn = host.conns.find((c) => c.id === data.conn)
				if(conn){
					console.warn('Device id already exists:', conn, 'ignore:', data)
				}else{
					const id = data.conn
					insertBefore(host.conns, (c) => c.id <= id, {
						host: host,
						id: id,
						addr: data.addr,
						device: data.device,
						label: data.label,
					})
				}
			}else{
				host = {
					id: hostid,
				}
				host.conns = [{
					id: data.conn,
					addr: data.addr,
					device: data.device,
					label: data.label,
					host: host,
				}]
				hosts.value.push(host)
			}
			break
		}
		case 'device_leave': {
			const hostid = event.host
			const host = hosts.value.find((h) => h.id === hostid)
			if(host){
				const i = host.conns.findIndex((c) => c.id === data.conn)
				if(i >= 0){
					const conn = host.conns[i]
					if(conn.ref){
						conn.ref.onDeviceLeave()
					}
					host.conns.splice(i, 1)
				}
			}
			break
		}
		case 'device_event': {
			const obj = _getConnObj(event.host, data)
			if(obj && obj.ref){
				obj.ref.onEvent(data)
			}
			break
		}
		case 'term.open': {
			const obj = _getConnObj(event.host, data)
			if(obj && obj.ref){
				obj.ref.onTermOpen(data)
			}
			break
		}
		case 'term.close': {
			const obj = _getConnObj(event.host, data)
			if(obj && obj.ref){
				obj.ref.onTermClose(data)
			}
			break
		}
		case 'term.oper': {
			const obj = _getConnObj(event.host, data)
			if(obj && obj.ref){
				obj.ref.onTermOper(data)
			}
			break
		}
		case 'custom_event': {
			const eventTyp = event.event
			onCustomEvent(eventTyp, data)
			break
		}
		}
	})
	return ws
}

const wsconnLock = new Lock()

async function reconnect(){
	await wsconnLock.lock()
	try{
		connected.value = false
		if(wsconn){
			wsconn.close()
			wsconn = null
		}

		const token = props.token

		try{
			wsconn = initWs(await connectWs(token))
			let res = await wsconn.ask('user_info')
			if(res.status !== 'ok'){
				throw res
			}
			userinfo.value = res.data
			connected.value = true
			console.log('Connect success!')
		}catch(e){
			if(wsconn){
				wsconn.close()
				wsconn = null
			}
			console.error('Cannot connect websocket:', e)
			await alert('Cannot connect to the websocket point')
			return
		}
		const res = await wsconn.ask('list_hosts')
		if(res.status !== 'ok'){
			hosts.value = []
			console.error('Cannot get hosts:', res)
		}else{
			const hsts = res.data || []
			hsts.forEach((host) => {
				host.conns.sort((a, b) => a.id - b.id).forEach((conn) => {
					conn.host = host
				})
			})
			hosts.value = hsts
		}
	}finally{
		wsconnLock.unlock()
	}
}

async function provideWebsocket(){
	await wsconnLock.waitForUnlock()
	if(!wsconn || wsconn.readyState !== WebSocket.OPEN){
		console.warn('Websocket is inactive, reconnecting...')
		await reconnect()
	}
	return wsconn
}

async function onWsAsk(...args){
	await provideWebsocket()
	wsconn.askSync(...args)
}

async function onFireEvent(...args){
	await provideWebsocket()
	wsconn.fireTermEvent(...args)
}

function onCustomEvent(event, data){
	let i = event.indexOf(':')
	if(i > 0){
		const pluginid = event.substring(0, i)
		event = event.substring(i + 1)
		const plugin = plugins.value[pluginid]
		if(plugin && plugin.onevent){
			plugin.onevent(event, data)
		}
	}
}

class PluginAPI{
	constructor(plugin){
		this.plugin = plugin
		this._pluginId = plugin.meta.id
	}
	get pluginId(){
		return this._pluginId
	}
	get user(){
		return userinfo.value
	}
	async broadcast(event, data){
		await provideWebsocket()
		return wsconn.send({
			type: 'broadcast_cli',
			event: this.pluginId + ':' + event,
			data: data,
		})
	}
}

async function loadPlugin(pluginid, version){
	if(version === 'outside'){
		console.log(`loading outside plugin at ${pluginid}`)
		return loadPluginByUrl(pluginid)
	}
	console.log(`loading plugin ${pluginid} v${version}`)
	const urlpath = `/api/web_plugin/${pluginid}@${version}`
	return loadPluginByUrl(urlpath)
}

async function loadPluginByUrl(urlpath){
	if(urlpath.substr(-1) === '/'){
		urlpath = urlpath.substring(0, urlpath.length - 1)
	}
	const meta = (await axios.get(`${urlpath}/meta.json`)).data
	const pid = meta.id
	if(!pid){
		throw `Plugin must have a register id`
	}
	if(!meta.name){
		meta.name = pid
	}
	const pluginM = await import(`${urlpath}/index.mjs`)
	const plugin = {}
	for(let key of Object.keys(pluginM)){
		plugin[key] = pluginM[key]
	}
	plugin.meta = meta
	if(plugins.value[pid]){
		throw `Plugin id <${pid}> is already exists`
	}
	plugins.value[pid] = plugin

	if(plugin.onload){
		plugin.onload(new PluginAPI(plugin))
	}

	const ref = routerRef.value
	if(ref){
		const props = ref.props || ref._.props
		const { hostid, connid } = props
		const ctx = ref.getContext()
		const host = hosts.value.find((h) => h.id === hostid)
		if(connid){
			const conn = host.conns.find((c) => c.id === connid)
			if(conn){
				ctx.loadPlugin(plugin, conn)
			}
		}else{
			ctx.loadPlugin(plugin, host)
		}
	}
	return plugin
}

async function loadPlugins(){
	const pluginList = (await axios.get(`/api/cli_plugin`, {
		headers: {
			'Authorization': props.token,
		}
	})).data.data
	for(const plugin of pluginList){ // have to keep the plugin order
		await loadPlugin(plugin.id, plugin.version).catch((err) => {
			console.error(`Couldn't load plugin ${plugin.id}:`, err)
		})
	}
}

function forEachPlugin(cb){
	return Object.values(plugins.value).forEach(cb)
}

var lastFocus = ''

watch(routerRef, (ref) => {
	if(!ref || loadError.value){
		return
	}
	const props = ref.props
	const propstr = JSON.stringify(props)
	const { hostid, connid } = props
	const ctx = ref.getContext()
	const host = hosts.value.find((h) => h.id === hostid)
	if(!host){
		return
	}
	if(connid){ // focused on device
		if(host.conns){
			const conn = host.conns.find((c) => c.id === connid)
			if(!conn || conn.ref === ref || lastFocus === propstr){
				return
			}
			conn.ref = ref
			lastFocus = propstr
			forEachPlugin((plugin) => {
				ctx.loadPlugin(plugin, conn)
			})
		}
	}else{ // focused on host
		if(host.ref === ref && lastFocus === propstr){
			return
		}
		host.ref = ref
		lastFocus = propstr
		forEachPlugin((plugin) => {
			ctx.loadPlugin(plugin, host)
		})
	}
})

await reconnect()

onMounted(async () => {
	await loadPlugins()
})

onBeforeUnmount(() => {
	if(wsconn){
		wsconn.close()
		wsconn = null
	}
})

</script>

<template>
	<main class="main">
		<nav class="device-nav">
			<h2>Devices</h2>
			<hr/>
			<div v-for="host in hosts">
				<h3>
					<RouterLink :to="`/dashboard/${host.id}`" exact-active-class="active">
						{{host.id}}
					</RouterLink>
				</h3>
				<ul>
					<li v-for="device in host.conns"
						class="device-nav-item"
					>
						<RouterLink :to="`/dashboard/${host.id}/${device.id}`" exact-active-class="active">
							{{device.id}}
							<i v-if="device.label">{{device.label}}</i>
						</RouterLink>
					</li>
				</ul>
				<hr/>
			</div>
		</nav>
		<div class="device-box">
			<div v-if="loadError">
				<i><b>Error: {{loadError}}</b></i>
			</div>
			<RouterView v-slot="{ Component }"> 
				<template v-if="Component">
					<KeepAlive>
						<component
							:is="Component"
							:key="$route.fullPath"
							ref="routerRef"
							v-on:ask="onWsAsk"
							v-on:fire-event="onFireEvent"
							>
						</component>
					</KeepAlive>
				</template>
				<div v-else-if="connected">
					<i>Please select a device</i>
				</div>
				<div v-else>
					<b><i>Please reconnect</i></b>
				</div>
			</RouterView>
		</div>
	</main>
</template>

<style scoped>

.main {
	display: flex;
	flex-direction: row;
	height: 100%;
}

.device-nav {
	width: 13rem;
	height: 100%;
	padding: 0.8rem;
	background: gray;
	color: #f0f0f0;
}

.device-nav>div>ul {
	list-style-type: none;
	padding-inline-start: 0.8rem;
}

.device-nav-item {
	height: 1.5rem;
	line-height: 1.5rem;
	padding-left: 0.2rem;
	cursor: pointer;
	user-select: none;
}

.device-nav a {
	display: block;
	color: #f0f0f0;
	text-decoration: none;
	transition: all 0.3s ease-out;
}

.device-nav a.active {
	background-color: #ddd;
	color: #444;
	cursor: default;
}

.device-nav a:not(.active):hover {
	background-color: #ddd;
	color: #444;
	height: 1.8rem;
	line-height: 1.8rem;
}

.device-box {
	padding: 1rem;
	width: calc(100% - 13rem);
	overflow: auto;
}

</style>
