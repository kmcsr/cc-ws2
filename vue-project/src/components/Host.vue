<script setup>
import { ref, onUpdated } from 'vue'
import HNode from './HNode.vue'

const props = defineProps({
	hostid: String,
})

class Context{
	constructor(hostid){
		this.hostid = hostid
		this.extNodes = ref([])
		this._loadedPlugins = {}
	}
	loadPlugin(plugin, deviceObj){ // TODO: remove deviceObj arg
		if(plugin.meta.id in this._loadedPlugins){
			return false
		}
		this._loadedPlugins[plugin.meta.id] = plugin.onHostLoad(this, deviceObj)
		return true
	}
	allocHTMLBlock(options){
		const ele = document.createElement('div')
		this.extNodes.push({node: ele, ...options})
		return ele
	}
}

const context = ref(new Context(props.hostid))

function getContext(){
	return context.value
}

onUpdated(()=>{
	if(props.hostid !== context.value.hostid){
		context.value = new Context(props.hostid)
	}
})

defineExpose({
	props,
	getContext,
})

</script>

<template>
	<div>
		<h2>{{hostid}}</h2>
		<hr/>
		<h3>Hooks</h3>
		<hr/>
		<div class="hooks-box">
			TODO
		</div>

		<div class="extention-box">
			<HNode v-for="ele in context.extNodes" :node="ele.node" :styles="ele.styles"/>
		</div>
	</div>
</template>

<style scoped>
	
</style>