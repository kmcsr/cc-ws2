<script setup>
import { onMounted, ref, watch } from 'vue'
import axios from 'axios'

const props = defineProps({
	token: String,
})

const isroot = ref(false)

const servers = ref([])
const tokens = ref([])
const daemonTokens = ref([])

async function createServer(){
	const sid = prompt('server id:')
	if(!sid){
		return
	}
	try{
		const res = await axios.get(`/api/create_server`, {
			params: {
				id: sid,
			},
			headers: {
				'Authorization': props.token,
			}
		})
		if(res.data.status !== 'ok'){
			throw res
		}
	}catch(e){
		alert('Cannot create server')
		throw e
	}
	await refreshAll()
}

async function deleteServer(server){
	try{
		const res = await axios.get(`/api/remove_server`, {
			params: {
				id: server,
			},
			headers: {
				'Authorization': props.token,
			}
		})
		if(res.data.status !== 'ok'){
			throw res
		}
	}catch(e){
		alert('Cannot delete server')
		throw e
	}
	await refreshAll()
}

async function createCliToken(){
	try{
		const res = await axios.get(`/api/create_token`, {
			headers: {
				'Authorization': props.token,
			}
		})
		if(res.data.status !== 'ok'){
			throw res
		}
		const token = res.data.token
	}catch(e){
		alert('Cannot create server')
		throw e
	}
	alert('Created')
	await refreshTokens()
}

async function removeToken(token){
	const res = await axios.get(`/api/remove_token`, {
		params: {
			token: token,
		},
		headers: {
			'Authorization': props.token,
		}
	})
	if(res.data.status !== 'ok'){
		throw res
	}
	await refreshTokens()
}

async function createDaemonToken(){
	const sid = prompt('server id:')
	if(!sid){
		return
	}
	if(servers.value.indexOf(sid) < 0){
		alert('Server not exists')
		return
	}
	try{
		const res = await axios.get(`/api/create_daemon_token`, {
			params: {
				server: sid,
			},
			headers: {
				'Authorization': props.token,
			}
		})
		if(res.data.status !== 'ok'){
			throw res
		}
		const token = res.data.token
	}catch(e){
		alert('Cannot create server')
		throw e
	}
	await refreshDaemonTokens()
}

async function removeDaemonToken(token){
	const res = await axios.get(`/api/remove_daemon_token`, {
		params: {
			token: token,
		},
		headers: {
			'Authorization': props.token,
		}
	})
	if(res.data.status !== 'ok'){
		throw res
	}
	await refreshDaemonTokens()
}

async function getPermServers(token){
	const res = await axios.get(`/api/perm_servers`, {
		params: {
			token: token,
		}
	})
	if(res.data.status !== 'ok'){
		throw res
	}
	return res.data.data || []
}

async function refreshServers(){
	servers.value = await getPermServers(props.token)
}

async function refreshTokens(){
	const res = await axios.get(`/api/tokens`, {
		headers: {
			'Authorization': props.token,
		}
	})
	if(res.data.status !== 'ok'){
		throw res
	}
	const tks = res.data.data || []
	for(const t of tks){
		t.perm_servers = t.root ?[] :await getPermServers(t.token)
	}
	tokens.value = tks
}

async function refreshDaemonTokens(){
	const res = await axios.get(`/api/daemon_tokens`, {
		headers: {
			'Authorization': props.token,
		}
	})
	if(res.data.status !== 'ok'){
		throw res
	}
	const tks = res.data.data || []
	daemonTokens.value = tks
}

function refreshAll(){
	return Promise.all([refreshServers(), refreshTokens(), refreshDaemonTokens()])
}

async function copyText(text){
	await navigator.clipboard.writeText(text)
}

async function onTokenRootChange(token, value){
	const res = await axios.post(`/api/perm_root`, JSON.stringify(value), {
		params: {
			token: token,
		},
		headers: {
			'Authorization': props.token,
		}
	})
	if(res.data.status !== 'ok'){
		throw res
	}
	await refreshTokens()
}

async function onTokenServerPermChange(token, server, value){
	const res = await axios.post(`/api/perm_server`, JSON.stringify(value), {
		params: {
			token: token,
			id: server,
		},
		headers: {
			'Authorization': props.token,
		}
	})
	if(res.data.status !== 'ok'){
		throw res
	}
	await refreshTokens()
}

onMounted(async () => {
	try{
		await refreshAll()
	}catch(e){
		alert('permission denied')
		throw e
	}
	isroot.value = true
})

</script>

<template>
	<main class="main">
		<div v-if="isroot">
			<h1>Admin dashboard</h1>
			<hr/>
			<h2>Servers</h2>
			<hr/>
			<h4>Total: {{servers.length}}</h4>
			<ul>
				<li>
					<button @click.passive="createServer">
						<b>Add New +</b>
					</button>
				</li>
				<li v-for="svr in servers" :key="svr">
					{{svr}} <button @click.passive="deleteServer(svr)">-</button>
				</li>
			</ul>
			<h2>Cli Tokens</h2>
			<hr/>
			<h4>Total: {{tokens.length}}</h4>
			<div class="token-table-box">
				<table class="token-table">
					<thead>
						<tr>
							<th>Token</th>
							<th>Root</th>
							<th v-for="svr in servers" :key="svr">{{svr}}</th>
						</tr>
					</thead>
					<tbody>
						<tr>
							<td>
								<button @click.passive="createCliToken">
									Create New +
								</button>
							</td>
						</tr>
						<tr v-for="tk in tokens" :key="tk.token">
							<td class="token-token-id"
								@click.passive="copyText(tk.token)">
								<span>{{tk.token}}</span>
								<button :disabled="tk.token === token" @click.passive="removeToken(tk.token)">-</button>
							</td>
							<td>
								<input type="checkbox"
									:disabled="tk.token === token" :checked="tk.root"
									@change.passive="onTokenRootChange(tk.token, $event.target.checked)" />
							</td>
							<td v-for="svr in servers" :key="svr">
								<input type="checkbox"
									:disabled="tk.root" :checked="tk.root || tk.perm_servers.indexOf(svr) >= 0"
									@change.passive="onTokenServerPermChange(tk.token, svr, $event.target.checked)" />
							</td>
						</tr>
					</tbody>
				</table>
			</div>
			<h2>Daemon Tokens</h2>
			<hr/>
			<h4>Total: {{daemonTokens.length}}</h4>
			<div class="token-table-box">
				<table class="token-table">
					<thead>
						<tr>
							<th>Token</th>
							<th>Server</th>
						</tr>
					</thead>
					<tbody>
						<tr>
							<td>
								<button @click.passive="createDaemonToken">
									Create New +
								</button>
							</td>
						</tr>
						<tr v-for="tk in daemonTokens" :key="tk.token">
							<td class="token-token-id"
								@click.passive="copyText(tk.token)">
								<span>{{tk.token}}</span>
								<button @click.passive="removeDaemonToken(tk.token)">-</button>
							</td>
							<td>
								{{tk.server}}
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>
		<div v-else>
			<b><i>Permission denied</i></b>
		</div>
	</main>
</template>

<style scoped>

.main {
	padding: 1rem;
}

hr {
	margin-bottom: 0.5rem;
}

.token-table-box {
	width: 100%;
	overflow: auto;
}

.token-table, .token-table th, .token-table td {
	border: #000 1px solid;
}

.token-table {
/*border-collapse: collapse;*/
}

.token-table td {
	text-align: center;
	line-height: 100%;
}

.token-token-id>span {
	display: inline-block;
	max-width: 7rem;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
	user-select: none;
	cursor: pointer;
}

</style>