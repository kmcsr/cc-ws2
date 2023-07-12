<script setup>
import { ref } from 'vue'
import { RouterLink, RouterView } from 'vue-router'
import VueCookie from 'vue-cookies'

const token = ref(VueCookie.get('_token'))

function reconnect(){ // TODO: better prompt
	let tk = prompt('Input the connect token:')
	if(confirm('Save this token for auto login?')){
		VueCookie.set('_token', tk, '30d')
	}
	token.value = tk
}

if(!token.value){
	console.debug('Using saved token')
	reconnect()
}

</script>

<template>
	<header id="header">
		<button @click="reconnect">Reconnect</button>
		<nav id="header-nav">
			<RouterLink to="/">Dashboard</RouterLink>
			<RouterLink to="/admins">Admin</RouterLink>
		</nav>
	</header>
	<div id="body">
		<RouterView :token="token" :key="token"/>
	</div>
	<footer id="footer">
	</footer>
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
	padding: 0 0.5rem;
}

#body {
	height: calc(100% - 3rem);
}

</style>
