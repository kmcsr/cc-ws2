<script setup>
import { ref, onBeforeMount, onBeforeUnmount } from 'vue'
import { RouterView } from 'vue-router'
import Device from '../components/Device.vue'

const props = defineProps({
	token: String,
})

const plugins = ref({})

const hosts = ref([])
const connected = ref(false)

async function loadPlugin(urlpath){
	const plugin = await import(urlpath)
	if(!plugin.meta){
		throw `Plugin don't have meta data`
	}
	const pid = plugin.meta.id
	if(!pid){
		throw `Plugin must have a register id`
	}
	if(!plugin.meta.name){
		plugin.meta.name = pid
	}
	if(plugins.value[pid]){
		throw `Plugin id <${pid}> is already exists`
	}
	plugins.value[pid] = plugin
	return plugin
}

function setConnRef(ref){
	if(!ref){
		return
	}
	const { hostid, connid } = ref._.props
	const host = hosts.value.find((h) => h.id === hostid)
	if(connid){
		if(host && host.conns){
			const conn = host.conns.find((c) => c.id === connid)
			if(conn){
				conn.ref = ref
			}
		}
	}else{
		host.ref = ref
	}
}

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
	ws.askSync = function(type, data, resolve){
		let id = askIncreasement
		while(asking[id = (id + 1) & 0xffffffff]);
		asking[id] = resolve
		askIncreasement = id
		ws.send(JSON.stringify({
			type: type,
			id: id,
			data: data,
		}))
	}
	ws.ask = function(type, data){
		return new Promise((resolve) => {
			ws.askSync(type, data, resolve)
		})
	}
	ws.fireTermEvent = function(host, conn, term, event, ...args){
		ws.send(JSON.stringify({
			type: 'fire_event',
			host: host,
			conn: conn,
			term: term,
			event: event,
			args: args,
		}))
	}
	function _getConnObj(data){
		const host = hosts.value.find((h) => h.id === data.host)
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
		case 'reply': {
			const resolve = asking[event.id]
			if(resolve){
				resolve(data)
			}else{
				console.warn('Unexcept reply id', event.id)
			}
			break
		}
		case 'device_join': {
			var host = hosts.value.find((h) => h.id === data.host)
			if(host){
				const conn = host.conns.find((c) => c.id === data.conn)
				if(conn){
					console.warn('Device id already exists:', conn, 'ignore:', data)
				}else{
					host.conns.push({
						id: data.conn,
						addr: data.addr,
						device: data.device,
						label: data.label,
					host: host,
					})
				}
			}else{
				host = {
					id: data.host,
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
			const host = hosts.value.find((h) => h.id === data.host)
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
		case 'term.open': {
			const obj = _getConnObj(data)
			if(obj && obj.ref){
				obj.ref.onTermOpen(data)
			}
			break
		}
		case 'term.close': {
			const obj = _getConnObj(data)
			if(obj && obj.ref){
				obj.ref.onTermClose(data)
			}
			break
		}
		case 'term.oper': {
			const obj = _getConnObj(data)
			if(obj && obj.ref){
				obj.ref.onTermOper(data)
			}
			break
		}
		default:
			// console.debug('not handled msg:', data)
		}
	})
	ws.addEventListener('error', (event) => {
		console.error('websocket on error:', event)
	})
	return ws
}

var wsconn = null

function onWsAsk(...args){
	if(wsconn && wsconn.readyState === WebSocket.OPEN){
		wsconn.askSync(...args)
	}else{
		console.debug('Sending message to inactive websocket')
	}
}

function onFireEvent(...args){
	if(wsconn && wsconn.readyState === WebSocket.OPEN){
		wsconn.fireTermEvent(...args)
	}else{
		console.debug('Sending message to inactive websocket')
	}
}

async function reconnect(){
	if(wsconn){
		wsconn.close()
		wsconn = null
	}

	const token = props.token

	try{
		wsconn = await connectWs(token)
		connected.value = true
		console.log('Connect success!')
	}catch(e){
		connected.value = false
		console.error('Cannot connect websocket:', e)
		await alert('Cannot connect to the websocket point')
		return
	}
	const res = await wsconn.ask('list_hosts')
	if(res.status !== 'ok'){
		console.error('Cannot get hosts:', res)
	}else{
		const hsts = res.data || []
		hsts.forEach((host) => {
			host.conns.forEach((conn) => {
				conn.host = host
			})
		})
		hosts.value = hsts
	}
}

await reconnect()

onBeforeMount(() => {
	//
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
					<RouterLink :to="`${host.id}`" exact-active-class="active">
						{{host.id}}
					</RouterLink>
				</h3>
				<ul>
					<li v-for="device in host.conns"
						class="device-nav-item"
					>
						<RouterLink :to="`${host.id}/${device.id}`" exact-active-class="active">
							{{device.id}}
							<i v-if="device.label">{{device.label}}</i>
						</RouterLink>
					</li>
				</ul>
				<hr/>
			</div>
		</nav>
		<div class="device-box">
			<RouterView v-slot="{ Component }"> 
				<template v-if="Component">
					<KeepAlive>
						<component
							:is="Component"
							:key="$route.fullpath"
							:ref="setConnRef"
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
}

</style>
