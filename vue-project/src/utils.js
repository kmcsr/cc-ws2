
export function toHexColor(num){
	let s = Number.parseInt(num).toString(16)
	if(s === 'NaN'){
		return null
	}
	if(s.length > 6){
		throw {msg: 'Color number is too big', code: code, color: num}
	}
	s = '#' + '0'.repeat(6 - s.length) + s
	return s
}

export function mouseBtnToCC(btn){
	switch(btn){
	case 0: return 1 // left button
	case 1: return 3 // middle button
	case 2: return 2 // right button
	default: return null
	}
}

export function keyCodeToCC(code){
	if(!code){
		return null
	}
	let lcode = code.toLowerCase()
	if(lcode.length === 1 && 'a' <= lcode && lcode <= 'z'){
		return lcode
	}
	if(lcode.startsWith('key') && lcode.length === 4){
		let rcode = lcode.substr(3, 1)
		if('a' <= rcode && rcode <= 'z'){
			return rcode
		}
	}
	if(lcode.startsWith('numpad')){
		return 'numPad' + code.substr(6)
	}
	if(lcode.startsWith('arrow')){
		return lcode.substr(5)
	}
	if(lcode.startsWith('f')){ // function keys F<number>
		let n = Number.parseInt(lcode.substr(1))
		if('f' + n === lcode){
			return lcode
		}
	}
	switch(lcode){
	case 'pause':
	case 'home':
	case 'end':
	case 'insert':
	case 'delete':
	case 'semicolon':
	case 'comma':
	case 'minus':
	case 'period':
	case 'slash':
	case 'backspace':
	case 'tab':
	case 'enter':
	case 'backslash':
	case 'space':
		return lcode
	case 'equal': return 'equals'
	case 'quote': return 'apostrophe'
	case 'pageup': return 'pageUp'
	case 'pagedown': return 'pageDown'
	case 'printscreen': return 'printScreen'
	case 'shiftleft': return 'leftShift'
	case 'shiftright': return 'rightShift'
	case 'ctrleft':
	case 'controlleft': return 'leftCtrl'
	case 'ctrlright':
	case 'controlright': return 'rightCtrl'
	case 'altleft': return 'leftAlt'
	case 'altright': return 'rightAlt'
	case 'capslock': return 'capsLock'
	case 'numlock': return 'numLock'
	case 'scrolllock': return 'scrollLock'
	case 'backquote': return 'grave'
	case 'digit0': return 'zero'
	case 'digit1': return 'one'
	case 'digit2': return 'two'
	case 'digit3': return 'three'
	case 'digit4': return 'four'
	case 'digit5': return 'five'
	case 'digit6': return 'six'
	case 'digit7': return 'seven'
	case 'digit8': return 'eight'
	case 'digit9': return 'nine'
	case 'bracketleft': return 'leftBracket'
	case 'bracketright': return 'rightBracket'
	case 'contextmenu': return 'menu'
	}
	return code.substr(0, 1).toLowerCase() + code
}
