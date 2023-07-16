<script setup>

defineProps({
	size: [String, Number],
	top: String,
	front: String,
	right: String,
})

</script>

<template>
	<div class="box" :style="{ '--size': size }">
		<div class="cube">
			<div class="face top" :style="{ 'background-image': `${top}` }"></div>
			<div class="face front" :style="{ 'background-image': `${front}` }"></div>
			<div class="face right" :style="{ 'background-image': `${right}` }"></div>
		</div>
	</div>
</template>

<style scoped>

.box {
	--rsize: calc(var(--size) / 5 * 3);
	--hrsize: calc(var(--rsize) / 2);
	display: inline-block;
	width: var(--rsize);
	height: var(--rsize);
	margin: calc((var(--size) - var(--rsize)) / 2);
	perspective: none;
}

.cube {
	position: relative;
	width: 100%;
  height: 100%;
	transform-style: preserve-3d;
	transform: rotateX(45deg) rotateY(45deg);
}

.face {
  position: absolute;
  width: 100%;
  height: 100%;
  background-size: 100% 100%;
  image-rendering: pixelated;
}

.top {
  transform: rotateX(90deg) translateZ(var(--hrsize));
}

.right {
	transform: rotateY(90deg) translateZ(var(--hrsize));
}

.front {
	transform: rotateX(0deg) translateZ(calc(var(--hrsize) * -1));
}

</style>