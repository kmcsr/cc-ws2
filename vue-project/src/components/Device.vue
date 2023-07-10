<script setup>
import { ref, computed, onBeforeMount } from 'vue'
import Terminal from './Terminal.vue'

const props = defineProps({
	hostid: String,
	connid: Number,
})

const emit = defineEmits(['ask', 'fire-event'])

const terms = ref([])
const selectedTermIndex = ref(null)
const selectedTermId = computed(() => selectedTermIndex.value === null ?null :terms.value[selectedTermIndex.value].id)

function askWs(type, data){
	return new Promise((resolve) => {
		emit('ask', type, data, resolve)
	})
}

onBeforeMount(async () => {
	const res = await askWs('list_terms', {
		host: props.hostid,
		conn: props.connid,
	})
	if(res.status !== 'ok'){
		console.error('Cannot get term list:', res)
		return
	}
	const rterms = res.res;
	terms.value = rterms.map((o) => {
		o.running = true
		return o
	})
})

function switchTerm(i){
	selectedTermIndex.value = i
}

function closeTerm(i){
	const termid = terms.value[i].id
	terms.value.splice(i, 1)
	if(selectedTermIndex.value >= i && --selectedTermIndex.value < 0){
		selectedTermIndex.value = terms.value.length > 0 ?0 :null
	}
	emit('fire-event', props.hostid, props.connid, termid, 'terminate')
}

async function onNewTerm(){
	var program = prompt('Program:')
	if(!program){
		return
	}
	var arg = prompt('Arg:')
	if(arg === null){
		return
	}
	const res = await askWs('run', {
		host: props.hostid,
		conn: props.connid,
		prog: program,
		args: [arg],
	})
	if(res.status !== 'ok'){
		console.error('Cannot start new program:', res)
		alert('Err: ' + res.error)
		return
	}
	console.debug('Successed to start program:', res)
	selectedTermIndex.value = null
	return
}

//:export event
function onDeviceLeave(){
	terms.value.forEach((o) => {
		o.running = false
	})
}

//:export event
function onTermOpen(data){
	const [title, id] = data.args
	terms.value.push({
		id: id,
		title: title,
		running: true,
	})
	if(selectedTermIndex.value === null){
		selectedTermIndex.value = terms.value.length - 1
	}
}

//:export event
function onTermClose(data){
	const [id, successed] = data.args
	// const index = terms.value.findIndex((e) => e.running && e.id === id)
	const term = terms.value.find((e) => e.running && e.id === id)
	if(term){
		term.running = false
		if(term.ref){
			term.ref.onTermClose(data)
		}
	}else{
		console.warn('Unknown termid:', id)
	}
}

//:export event
function onTermOper(data){
	const args = data.args
	const id = args[0]
	const term = terms.value.find((e) => e.running && e.id === id)
	if(term){
		if(term.ref){
			term.ref.onTermOper(data)
		}else{
			console.debug('Instance of term', id, 'is not defined')
		}
	}else{
		console.debug('Activing term', id, 'not found')
	}
}

defineExpose({
	onDeviceLeave,
	onTermOpen,
	onTermClose,
	onTermOper,
})

</script>

<template>
	<div>
		<h2>{{hostid}} <i style="font-size: 1rem; font-weight: 400;">ID: {{connid}}</i></h2>
		<hr style="margin-bottom: 1rem;" />
		<h3>Terminals</h3>
		<hr/>
		<nav class="term-nav-box">
			<TransitionGroup tag="div" class="term-nav" name="term-nav">
				<button v-for="(term, i) in terms" :key="term"
					:class="selectedTermIndex === i ?'selected' :''"
					@click.self="switchTerm(i)"
					:title="term.title"
				>
					{{term.title}}
					<button class="term-close-btn" @click="closeTerm(i)"
						title="Close this terminal">X</button>
				</button>
			</TransitionGroup>
			<button class="term-new-btn" @click="onNewTerm">
				<b>New +</b>
			</button>
		</nav>
		<div class="term-box">
			<div v-if="selectedTermIndex !== null">
				<KeepAlive>
					<Terminal :ref="(ref) => {
							const term = terms[selectedTermIndex]
							if(term){
								term.ref = ref
							}
						}"
						:hostid="hostid" :connid="connid" :termid="selectedTermId" :key="terms[selectedTermIndex]"
						v-on:ask="(...args) => emit('ask', ...args)"
						v-on:fire-event="(...args) => emit('fire-event', ...args)"
					/>
				</KeepAlive>
			</div>
			<div v-else>
				<i>Please select or create a terminal</i>
			</div>
		</div>
	</div>
</template>

<style scoped>

.term-nav-box {
	display: flex;
	flex-direction: row;
	justify-content: space-between;
	height: 2rem;
	background: lightgray;
	font-family: monospace;
}

.term-nav {
	display: flex;
	flex-direction: row;
	align-items: center;
	width: 100%;
	height: 100%;
	overflow: auto hidden;
}

.term-nav>button, .term-new-btn {
	display: inline-flex;
	flex-direction: row;
	align-items: center;
	justify-content: space-between;
	height: 100%;
	padding: 0 0.3rem 0 0.5rem;
	border: none;
	background: transparent;
	color: #fff;
	white-space: nowrap;
	cursor: pointer;
	user-select: none;
	transition: all 0.5s ease;
}

.term-nav>button {
	border-right: #eee 1px solid;
	max-width: 50%;
}

.term-nav>button.selected {
	background: #000d;
}

.term-nav>button:hover {
	transform: scale(1.1);
}

.term-nav-move,
.term-nav-enter-active,
.term-nav-leave-active {
	transition: all 0.5s ease;
}

.term-nav-enter-from,
.term-nav-leave-to {
	opacity: 0;
	max-width: 0 !important;
	transform: translateY(-10px);
}

.term-close-btn {
	height: 50%;
	padding: 0;
	margin-left: 0.5rem;
	border: none;
	background: transparent;
	color: #eee;
	cursor: pointer;
	user-select: none;
}

.term-close-btn:hover {
	color: red;
}

.term-new-btn {
	float: right;
	border-left: #eee 1px solid;
}

.term-box {
	margin-top: 0.5rem;
	margin-left: 0.3rem;
}

</style>
