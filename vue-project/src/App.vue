<script setup>
import { ref, onMounted } from 'vue'
import Device from './components/Device.vue'

const hosts = ref([])
const selected = ref(null)

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
					})
				}
			}else{
				hosts.value.push({
					id: data.host,
					conns: [{
						id: data.conn,
						addr: data.addr,
						device: data.device,
						label: data.label,
					}]
				})
			}
			break
		}
		case 'device_leave': {
			const host = hosts.value.find((h) => h.id === data.host)
			if(host){
				const i = host.conns.findIndex((c) => c.id === data.conn)
				if(i >= 0){
					host.conns.splice(i, 1)
				}
			}
			break
		}
		case 'term.open': {
			const obj = _getConnObj(data)
			if(obj){
				obj.ref.onTermOpen(data)
			}
			break
		}
		case 'term.close': {
			const obj = _getConnObj(data)
			if(obj){
				obj.ref.onTermClose(data)
			}
			break
		}
		case 'term.oper': {
			const obj = _getConnObj(data)
			if(obj){
				obj.ref.onTermOper(data)
			}
			break
		}
		default:
			console.debug('not handled msg:', data)
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

async function reauth(){
	if(wsconn){
		wsconn.close()
		wsconn = null
	}

	const token = prompt('Please input your token:')

	if(token){
		try{
			wsconn = await connectWs(token)
			console.log('Connect success!')
		}catch(e){
			console.error('Cannot connect websocket:', e)
			return
		}
		hosts.value = (await wsconn.ask('list_hosts')) || []
	}
}

onMounted(() => {
	reauth()
})

function switchDevice(hostid, deviceid){
	const selecting = JSON.stringify([hostid, deviceid])
	if(selected.value && selected.value.encoded === selecting){
		return
	}
	selected.value = {
		host: hostid,
		device: deviceid,
		encoded: selecting,
	}
}

</script>

<template>
	<main class="main">
		<nav class="device-nav">
			<h2>Devices</h2>
			<hr class="device-nav-hr" />
			<div v-for="host in hosts">
				<h3>{{host.id}}</h3>
				<ul>
					<li v-for="device in host.conns"
						class="device-nav-item"
						:class="(selected && selected.host === host.id && selected.device === device.id) ?'selected' :''"
						tabindex="0"
						@click="switchDevice(host.id, device.id)"
					>
						{{device.id}}
					</li>
				</ul>
				<hr/>
			</div>
		</nav>
		<div class="device-box">
			<div v-if="selected">
				<KeepAlive>
					<Device :ref="(ref) => {
							hosts.find((h) => h.id === selected.host).conns.find((c) => c.id === selected.device).ref = ref
						}"
						:hostid="selected.host" :connid="selected.device" :key="selected.encoded"
						v-on:ask="onWsAsk"
						v-on:fire-event="onFireEvent"
					/>
				</KeepAlive>
			</div>
			<div v-else>
				<i>Please select a device</i>
			</div>
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

.device-nav-hr {
	margin-bottom: 1rem;
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
	transition: all 0.3s ease-out;
}

.device-nav-item.selected {
	background-color: #ddd;
	color: #444;
	cursor: default;
}

.device-nav-item:not(.selected):hover {
	background-color: #ddd;
	color: #444;
	height: 1.7rem;
	line-height: 1.7rem;
}

.device-box {
	padding: 1rem;
	width: calc(100% - 13rem);
}

</style>
