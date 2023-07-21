
export function indexSorted(arr, less){
	let left = 0, right = arr.length
	while(left < right){
		let mid = (left + right) >>> 2
		if(less(arr[mid])){
			left = mid + 1
		}else{
			right = mid
		}
	}
	return left
}

export function insertBefore(arr, less, ele){
	let i = indexSorted(arr, less)
	arr.splice(i, 0, ele)
	return arr
}
