
package main

type Map map[string]any

func (m Map)Get(k string)(any){
	return m[k]
}

func (m Map)GetBool(k string)(v bool, ok bool){
	v, ok = m[k].(bool)
	return
}

func (m Map)GetInt(k string)(v int, ok bool){
	if v, ok = m[k].(int); !ok {
		var v0 float64
		if v0, ok = m[k].(float64); ok {
			v = (int)(v0)
		}
	}
	return
}

func (m Map)GetFloat(k string)(v float64, ok bool){
	v, ok = m[k].(float64)
	return
}

func (m Map)GetString(k string)(v string, ok bool){
	v, ok = m[k].(string)
	return
}

func (m Map)GetList(k string)(v List, ok bool){
	var v0 []any
	if v0, ok = m[k].([]any); ok {
		v = (List)(v0)
	}
	return
}

func (m Map)GetMap(k string)(v Map, ok bool){
	var v0 map[string]any
	if v0, ok = m[k].(map[string]any); ok {
		v = (Map)(v0)
	}
	return
}
type List []any

func (l List)Get(i int)(any){
	if i >= len(l) {
		return nil
	}
	return l[i]
}

func (l List)GetBool(i int)(v bool, ok bool){
	if i >= len(l) {
		return
	}
	v, ok = l[i].(bool)
	return
}

func (l List)GetInt(i int)(v int, ok bool){
	if i >= len(l) {
		return
	}
	if v, ok = l[i].(int); !ok {
		var v0 float64
		if v0, ok = l[i].(float64); ok {
			v = (int)(v0)
		}
	}
	return
}

func (l List)GetFloat(i int)(v float64, ok bool){
	if i >= len(l) {
		return
	}
	v, ok = l[i].(float64)
	return
}

func (l List)GetString(i int)(v string, ok bool){
	if i >= len(l) {
		return
	}
	v, ok = l[i].(string)
	return
}

func (l List)GetList(i int)(v List, ok bool){
	if i >= len(l) {
		return
	}
	var v0 []any
	if v0, ok = l[i].([]any); ok {
		v = (List)(v0)
	}
	return
}

func (l List)GetMap(i int)(v Map, ok bool){
	if i >= len(l) {
		return
	}
	var v0 map[string]any
	if v0, ok = l[i].(map[string]any); ok {
		v = (Map)(v0)
	}
	return
}


func inRange(n, max int)(int){
	n %= max
	if n < 0 {
		return max + n
	}
	return n
}
