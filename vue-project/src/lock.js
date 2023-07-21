// A simple lock impl by promise with async api
// by zyxkad@gmail.com

export class Lock{
	constructor(){
		this._locking = null
		this._unlocker = null
	}
	get locking(){
		return !!this._locking
	}
	tryLock(){
		if(this._locking){
			return false
		}
		this._locking = new Promise((resolve) => {
			this._unlocker = resolve
		})
		return true
	}
	async lock(){
		while(this._locking){
			await this._locking
		}
		this._locking = new Promise((resolve) => {
			this._unlocker = resolve
		})
	}
	async waitForUnlock(){
		while(this._locking){
			await this._locking
		}
	}
	unlock(){
		if(this._unlocker){
			this._unlocker()
			this._locking = null
			this._unlocker = null
		}
	}
}
