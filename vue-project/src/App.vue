<script setup>
import { ref, onMounted } from 'vue'
import { RouterLink, RouterView } from 'vue-router'
import VueCookie from 'vue-cookies'
import Alerter from './components/Alerter.vue'

const mounted = ref(false)
const alerter = ref(null)

const token = ref(VueCookie.get('_token'))

async function relog(){ // TODO: better prompt
	let tk = await prompt('Input the connect token:')
	if(!tk){
		return
	}
	if(await confirm('Save this token for auto login?')){
		VueCookie.set('_token', tk, '30d')
	}
	token.value = tk
}

window.sleep = function(td){
	return new Promise((re) => setTimeout(re, td))
}

onMounted(async () => {
	({alert: window.alert, confirm: window.confirm, prompt: window.prompt, alertHint: window.alertHint} = alerter.value)
	try{
		if(token.value){
			console.debug('Using saved token')
			VueCookie.set('_token', token.value, '30d')
		}else{
			await relog()
		}
	}finally{
		mounted.value = true
	}
})

</script>

<template>
	<header id="header">
		<button @click="relog">Relog</button>
		<nav id="header-nav">
			<RouterLink class="green" to="/dashboard">Dashboard</RouterLink>
			<RouterLink class="green" to="/settings">Settings</RouterLink>
			<RouterLink class="green" to="/admins">Admin</RouterLink>
		</nav>
	</header>
	<div id="body">
		<RouterView v-slot="{ Component }"> 
			<!-- https://vuejs.org/guide/built-ins/suspense.html#combining-with-other-components -->
			<template v-if="mounted && Component">
				<KeepAlive :include="['DashboardView']">
					<Suspense>
						<component :is="Component"
							:token="token" :key="token">
						</component>
						<template #fallback>
							Loading...
						</template>
					</Suspense>
				</KeepAlive>
			</template>
		</RouterView>
	</div>
	<footer id="footer">
	</footer>
	<Alerter ref="alerter" :secrets="token?[token.substr(4)]:[]" :key="token"/>
</template>

<style scoped>
#header {
	display: flex;
	flex-direction: row;
	align-items: center;
	height: 3rem;
	padding: 0 0.5rem;
	box-shadow: #000a 0 1px 0.3rem;
}

#header-nav {
	display: flex;
	flex-direction: row;
	margin-left: 0.5rem;
}

#header-nav>a {
	display: inline-block;
	padding: 0 0.5rem;
}

#body {
	height: calc(100% - 3rem);
}

</style>
