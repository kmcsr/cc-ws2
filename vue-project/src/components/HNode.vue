<script setup>
import { ref, onMounted } from 'vue'

const props = defineProps({
	node: Object, // HTMLElement
	styles: Array, // []String
})

const box = ref(null)

onMounted(() => {
	const shadow = box.value.attachShadow({ mode: 'open' })
	if(props.styles){
		for(let css of props.styles){
			let style = document.createElement('style')
			style.textContent = css
			shadow.appendChild(style)
		}
	}
	shadow.appendChild(props.node)
})

</script>

<template>
	<div ref="box"></div>
</template>
