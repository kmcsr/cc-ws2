<script setup>
import { ref, watch } from 'vue'
import CloseSvg from 'vue-material-design-icons/Close.vue'

var closeCb = null

const ALERT_ID = 1
const CONFRIM_ID = 2
const PROMPT_ID = 3

const closeBtn = ref(null)
const doneBtn = ref(null)
const textareaRef = ref(null)
const show = ref(0)
const message = ref(null)
const inputs = ref(null)

function onKeydown(event){
	if(event.key === 'Escape'){
		if(closeBtn.value){
			closeBtn.value.click()
			event.stopImmediatePropagation()
		}
	}
}

function beforeActive(){
	window.addEventListener('keydown', onKeydown, { capture: true })
}

watch(doneBtn, (value) => {
	if(value && show.value !== PROMPT_ID){
		value.focus()
	}
})

watch(textareaRef, (value) => {
	if(value){
		value.focus()
	}
})

function close(...args){
	window.removeEventListener('keydown', onKeydown)
	if(closeCb){
		closeCb(...args)
		closeCb = null
	}
	show.value = 0
	message.value = null
}

function alert(msg){
	if(closeCb){
		console.debug('WARN: old alert was not closed')
		const oldCloseCb = closeCb
		return new Promise((resolve) => {
			closeCb = resolve
		}).then((...args) => {
			oldCloseCb(...args)
			return alert(msg)
		})
	}
	return new Promise((resolve) => {
		beforeActive()
		closeCb = resolve
		show.value = ALERT_ID
		message.value = typeof msg === 'undefined' ?'' :String(msg)
	})
}

function confirm(msg){
	if(closeCb){
		console.debug('WARN: old alert was not closed')
		const oldCloseCb = closeCb
		return new Promise((resolve) => {
			closeCb = resolve
		}).then((...args) => {
			oldCloseCb(...args)
			return confirm(msg)
		})
	}
	return new Promise((resolve) => {
		beforeActive()
		closeCb = resolve
		show.value = CONFRIM_ID
		message.value = typeof msg === 'undefined' ?'' :String(msg)
	})
}

function prompt(msg){
	if(closeCb){
		console.debug('WARN: old alert was not closed')
		const oldCloseCb = closeCb
		return new Promise((resolve) => {
			closeCb = resolve
		}).then((...args) => {
			oldCloseCb(...args)
			return prompt(msg)
		})
	}
	return new Promise((resolve) => {
		beforeActive()
		closeCb = resolve
		show.value = PROMPT_ID
		message.value = typeof msg === 'undefined' ?'Prompt' :String(msg)
	})
}

defineExpose({
	alert,
	confirm,
	prompt,
})

</script>

<template>
	<Teleport to="body">
		<Transition name="alert">
			<div v-if="show" class="background">
				<dialog class="box">
					<template v-if="show === ALERT_ID">
						<h3 class="alert-title">
							<span class="alert-close" ref="closeBtn" @click="close()">
								<CloseSvg size="1.5rem"/>
							</span>
							Alert
						</h3>
						<p>{{message}}</p>
						<div class="alert-btns">
							<button ref="doneBtn" @click="close(true)">OK</button>
						</div>
					</template>
					<template v-else-if="show === CONFRIM_ID">
						<h3 class="alert-title">
							<span class="alert-close" ref="closeBtn" @click="close(false)">
								<CloseSvg size="1.5rem"/>
							</span>
							Confirm
						</h3>
						<p>{{message}}</p>
						<div class="alert-btns">
							<button @click="close(false)">Cancel</button>
							<button ref="doneBtn" @click="close(true)">OK</button>
						</div>
					</template>
					<template v-else-if="show === PROMPT_ID">
						<h3 class="alert-title">
							<span class="alert-close" ref="closeBtn" @click="close(null)">
								<CloseSvg size="1.5rem"/>
							</span>
							{{message}}
						</h3>
						<textarea class="prompt-input" v-model="inputs"
							ref="textareaRef"
							@keydown.enter.exact.prevent="close(inputs)"></textarea>
						<div class="alert-btns">
							<button @click="close(null)">Cancel</button>
							<button ref="doneBtn" @click="close(inputs)">Done</button>
						</div>
					</template>
				</dialog>
			</div>
		</Transition>
	</Teleport>
</template>

<style scoped>

.alert-enter-active,
.alert-leave-active {
  transition: opacity 0.1s;
}

.alert-enter-from,
.alert-leave-to {
  opacity: 0;
}

.background {
	--alert-cover-color: #0005;
	position: fixed;
	top: 0;
	left: 0;
	width: 100vw;
	height: 100vh;
	background-color: var(--alert-cover-color);
}

.box {
	display: block;
	position: absolute;
	top: 50%;
	left: 50%;
	transform: translate(-50%, -50%);
	min-width: 20rem;
	max-width: 100%;
	min-height: 10rem;
	max-height: 100%;
	padding: 0.7rem;
	border: none;
	border-radius: 0.5rem;
	box-shadow: #000 0 0 1rem;
	background: #fff;
}

.alert-title {
	margin-bottom: 0.5rem;
	border-bottom: var(--color-text) 1px solid;
	color: #00bd7e;
}

.alert-close {
	float: right;
	height: 1.5rem;
	color: var(--color-text);
	cursor: pointer;
	transition: all ease 0.3s;
}

.alert-close:hover {
	color: #f00;
	transform: rotate(90deg);
}

.prompt-input {
	display: block;
	width: 100%;
	min-width: 100%;
	max-width: 100%;
	min-height: 4rem;
	max-height: 60vh;
}

</style>