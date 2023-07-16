<script setup>
import { ref, onBeforeMount } from 'vue'
import axios from 'axios'

const props = defineProps({
	token: String,
})

const installedPlugins = ref([])
const pendingPluginList = ref(null)
const outsidePluginUrl = ref('')

async function refreshInstalledPlugins(){
	const pluginList = (await axios.get(`/api/cli_plugin`, {
		headers: {
			'Authorization': props.token,
		}
	})).data.data
	installedPlugins.value = pluginList
}

async function getPluginList(){
	const pluginList = (await axios.get(`/api/web_plugin`)).data.data
	return pluginList
}

async function openPluginList(){
	const pluginList = await getPluginList()
	pendingPluginList.value = pluginList
}

async function addPlugin(pluginid, version){
	try{
		const res = (await axios.post(`/api/cli_plugin`, JSON.stringify({
			id: pluginid,
			version: version,
		}), {
			headers: {
				'Authorization': props.token,
			}
		})).data
		if(res.status !== 'ok'){
			alert('Cannot install plugin: ' + e.error)
			return false
		}
	}catch(e){
		alert('Cannot install plugin: ' +
			(e.response.data.error || JSON.stringify(e.response.data)))
		return false
	}
	await Promise.all([refreshInstalledPlugins(), alert('Successed')])
	return true
}

async function removePlugin(pluginid){
	try{
		const res = (await axios.delete(`/api/cli_plugin`, {
			headers: {
				'Authorization': props.token,
			},
			params: {
				plugin: pluginid,
			},
		})).data
		if(res.status !== 'ok'){
			alert('Cannot remove plugin: ' + e.error)
			return false
		}
	}catch(e){
		alert('Cannot remove plugin: ' +
			(e.response.data.error || JSON.stringify(e.response.data)))
		return false
	}
	await Promise.all([refreshInstalledPlugins(), alert('Successed')])
	return true
}

onBeforeMount(async () => {
	await refreshInstalledPlugins()
})

</script>

<template>
	<main class="main">
		<div>
			<h2>Installed plugins</h2>
			<button @click="openPluginList">Search for plugin +</button>
			<table class="plugins-table">
				<thead>
					<th>Plugin</th>
					<th>Version</th>
					<th>Operation</th>
				</thead>
				<tbody>
					<tr v-for="plugin in installedPlugins">
						<td>{{plugin.id}}</td>
						<td>{{plugin.version}}</td>
						<td>
							<button @click="removePlugin(plugin.id, plugin.version)">Remove</button>
						</td>
					</tr>
				</tbody>
			</table>
			<Teleport v-if="pendingPluginList !== null" to="body">
				<div class="plugin-list-box-bg">
					<div class="plugin-list-box">
						<button @click="pendingPluginList = null">Close</button>
						<div>
							<input type="text" v-model="outsidePluginUrl"/>
							<button @click="addPlugin(outsidePluginUrl, 'outside') && (outsidePluginUrl = '')">Add outside plugin</button>
						</div>
						<table class="plugins-table">
							<thead>
								<th>Plugin ID</th>
								<th>Name</th>
								<th>Version</th>
								<th>Author</th>
								<th>Description</th>
								<th>Operation</th>
							</thead>
							<tbody>
								<tr v-for="plugin in pendingPluginList">
									<td>{{plugin.id}}</td>
									<td>{{plugin.name}}</td>
									<td>{{plugin.version}}</td>
									<td>{{plugin.author}}</td>
									<td>{{plugin.desc}}</td>
									<td>
										<button @click="addPlugin(plugin.id, plugin.version)">Add</button>
									</td>
								</tr>
							</tbody>
						</table>
					</div>
				</div>
			</Teleport>
		</div>
	</main>
</template>

<style scoped>

.main {
	padding: 1rem;
}

.plugins-table,
.plugins-table th,
.plugins-table td {
	padding: 0.2rem 0.3rem;
	border: 1px #0008 solid;
	border-collapse: collapse;
	white-space: nowrap;
}

.plugin-list-box-bg {
	position: fixed;
	top: 0;
	left: 0;
	z-index: 10;
	width: 100vw;
	height: 100vh;
	background-color: #0005;
}

.plugin-list-box {
	position: absolute;
	top: 50%;
	left: 50%;
	transform: translate(-50%, -50%);
	width: 80%;
	height: 80%;
	padding: 1.5rem;
	border-radius: 1rem;
	background-color: #fff;
	box-shadow: #0008 0 0 1rem;
	overflow: auto;
}

.plugin-list-box>* {
	margin-bottom: 0.5rem;
}

</style>