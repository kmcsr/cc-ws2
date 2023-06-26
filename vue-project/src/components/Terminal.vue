<script setup>
import { ref, computed, onBeforeMount } from 'vue'
import { toHexColor, mouseBtnToCC, keyCodeToCC } from '../utils'

const props = defineProps({
	hostid: String,
	connid: Number,
	termid: Number,
})

const emit = defineEmits(['ask', 'fire-event'])

const closed = ref(false)
const width = ref(0)
const height = ref(0)
const lines = ref([])
const textColor = ref(0)
const backgroundColor = ref(0)
const palette = ref({})
const cursorBlink = ref(false)
const cursorX = ref(0)
const cursorY = ref(0)
const shouldCursorBlink = computed(() => 
	(cursorBlink.value && 0 <= cursorX.value && cursorX.value < width.value && 0 <= cursorY.value && cursorY.value < height.value)
)
const termEles = ref([])
const cursorTarget = computed(() => {
	if(!shouldCursorBlink.value){
		return null
	}
	if(cursorY.value < termEles.value.length){
		const line = termEles.value[cursorY.value]
		if(cursorX.value < line.children.length){
			return line.children[cursorX.value]
		}
	}
	return null
})

function askWs(type, data){
	return new Promise((resolve) => {
		emit('ask', type, data, resolve)
	})
}

onBeforeMount(async () => {
	const res = await askWs('get_term', {
		host: props.hostid,
		conn: props.connid,
		term: props.termid,
	})
	if(res.status !== 'ok'){
		console.error('Cannot get term data:', res)
		return
	}
	const termData = res.res;
	({
		width: width.value,
		height: height.value,
		lines: lines.value,
		textColor: textColor.value,
		backgroundColor: backgroundColor.value,
		palette: palette.value,
		cursorBlink: cursorBlink.value,
		cursorX: cursorX.value,
		cursorY: cursorY.value,
	} = termData)
})

function getPaletteColor(code){
	const num = palette.value[code]
	return toHexColor(num)
}

function fireEvent(event, ...args){
	if(typeof event !== 'string'){
		throw 'Event type must be a string'
	}
	emit('fire-event', props.hostid, props.connid, props.termid, event, ...args)
}

function onKeydown(event){
	if(event.metaKey){
		return
	}
	event.preventDefault()
	const keyCode = keyCodeToCC(event.code)
	if(keyCode){
		const press = event.repeat
		fireEvent('key', keyCode, press)
		if(event.key.length === 1){ // most likely inputed a char
			fireEvent('char', event.key)
		}
	}
}

function onKeyup(event){
	const keyCode = keyCodeToCC(event.code)
	if(keyCode){
		fireEvent('key_up', keyCode)
	}
}

function onMousedown(event, x, y){
	let btn = mouseBtnToCC(event.button)
	if(btn){
		fireEvent('mouse_click', btn, x + 1, y + 1)
	}
}

function onMouseup(event, x, y){
	let btn = mouseBtnToCC(event.button)
	if(btn){
		fireEvent('mouse_up', btn, x + 1, y + 1)
	}
}

function onMousewheel(event, x, y){
	if(event.deltaX){
		fireEvent('mouse_scroll', event.deltaX < 0 ?-1 :1, x, y)
	}
}

//:export event
function onTermClose(data){
	closed.value = true
}

//:export event
function onTermOper(data){
	const args = data.args
	const type = args[1]
	switch(type){
	case 'write': {
		let [text] = args[2]
		if(cursorY.value < 0 || cursorY.value >= height.value || cursorX.value >= width.value){
			return
		}
		if(cursorX.value < 0){
			text = text.substr(-cursorX.value)
			cursorX.value = 0
		}
		const line = lines.value[cursorY.value]
		let l = width.value - cursorX.value
		if(text.length < l){
			l = text.length
		}else{
			text = text.substr(0, l)
		}
		line.text = line.text.substring(0, cursorX.value) + text + line.text.substr(cursorX.value + l)
		for(let i = 0; i < l; i++){
			let j = cursorX.value + i
			line.color[j] = textColor.value
			line.background[j] = backgroundColor.value
		}
		cursorX.value += l
		break
	}
	case 'blit': {
		let [text, color, bgColor] = args[2]
		if(cursorY.value < 0 || cursorY.value >= height.value || cursorX.value >= width.value){
			return
		}
		if(cursorX.value < 0){
			text = text.substr(-cursorX.value)
			color = color.slice(-cursorX.value)
			bgColor = bgColor.slice(-cursorX.value)
			cursorX.value = 0
		}
		const line = lines.value[cursorY.value]
		const lineE = termBox.children[cursorY.value]
		let l = width.value - cursorX.value
		if(text.length < l){
			l = text.length
		}else{
			text = text.substr(0, l)
		}
		line.text = line.text.substring(0, cursorX.value) + text + line.text.substr(cursorX.value + l)
		for(let i = 0; i < l; i++){
			let j = cursorX.value + i
			line.color[j] = color[i]
			line.background[j] = bgColor[i]
		}
		cursorX.value += l
		break
	}
	case 'setCursorPos': {
		let [x, y] = args[2]
		cursorX.value = x - 1
		cursorY.value = y - 1
		break
	}
	case 'setCursorBlink': {
		let [blink] = args[2]
		cursorBlink.value = blink
		break
	}
	case 'setTextColour':
	case 'setTextColor': {
		let [c] = args[2]
		textColor.value = c
		break
	}
	case 'setBackgroundColour':
	case 'setBackgroundColor': {
		let [c] = args[2]
		backgroundColor.value = c
		break
	}
	case 'scroll': {
		let [offset] = args[2]
		if(!offset){
			break
		}
		let down = offset < 0
		if(down){
			offset = -offset
		}
		if(offset < height.value){
			const emptyLine = {
				text: ' '.repeat(width.value),
				color: new Array(width.value).fill(textColor.value),
				background: new Array(width.value).fill(backgroundColor.value),
			}
			if(down){
				lines.value.pop()
				lines.value.unshift(emptyLine)
			}else{
				lines.value.shift()
				lines.value.push(emptyLine)
			}
			break
		}
	}
	case 'clear': {
		for(let y = 0; y < height.value; y++){
			const line = lines.value[y]
			line.text = ' '.repeat(width.value)
			for(let x = 0; x < width.value; x++){
				line.color[x] = textColor.value
				line.background[x] = backgroundColor.value
			}
		}
		break
	}
	case 'clearLine': {
		let [y] = args[2]
		if(y < 0 || y >= height.value){
			break
		}
		const line = lines.value[y]
		line.text = ' '.repeat(width.value)
		for(let x = 0; x < width.value; x++){
			line.color[x] = textColor.value
			line.background[x] = backgroundColor.value
		}
		break
	}
	case 'setPaletteColour':
	case 'setPaletteColor': {
		let [color, r] = args[2]
		palette.value[color] = r
		break
	}
	default:
		// do nothing
	}
}

defineExpose({
	onTermClose,
	onTermOper,
})

</script>

<template>
	<div>
		<div class="term" tabindex="0"
			@contextmenu.prevent
			@keydown="(event) => onKeydown(event)"
			@keyup.prevent="(event) => onKeyup(event)"
		>
			<div ref="termEles" v-for="(line, y) in lines" :key="y">
				<span v-for="(ch, x) in line.text" :key="x"
					@mousedown="(event) => onMousedown(event, x, y)"
					@mouseup="(event) => onMouseup(event, x, y)"
					@mousewheel.prevent="(event) => onMousewheel(event, x, y)"
					:style="{
						'color': getPaletteColor(line.color[x]),
						'background-color': getPaletteColor(line.background[x])
					}">
					{{ch}}
				</span>
			</div>
		</div>
		<Teleport v-if="cursorBlink && cursorTarget" :to="cursorTarget">
			<span class="term-cursor" :style="{'color': getPaletteColor(textColor)}">_</span>
		</Teleport>
	</div>
</template>

<style scoped>

@keyframes flash {
	0%   { opacity: 0; }
	50%  { opacity: 1; }
	100% { opacity: 0; }
}

.term {
	display: inline-flex;
	flex-direction: column;
	padding: 1px;
	border: 0.5rem ridge #ffff99;
	background: lightgray;
	font-family: monospace;
	user-select: none;
}

.term>div {
	display: inline-flex;
	flex-direction: row;
}

.term>div>span, .term-cursor {
	width: 10px;
	height: 15px;
	padding: 1px;
	font-size: 14px;
	line-height: 12px;
}

.term-cursor {
	float: left;
	padding: 0;
	font-family: monospace;
	animation: flash 0.9s infinite;
	user-select: none;
}

</style>
