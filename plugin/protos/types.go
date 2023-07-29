
// this file implemented wrap/unwrap between go types and proto Any type
package protos

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"unsafe"
)

type UnsupportTypeErr struct {
	Type reflect.Type
}

func (e *UnsupportTypeErr)Error()(string){
	return fmt.Sprintf("Unsupport type %s (%s)", e.Type.String(), e.Type.Kind().String())
}

type UnknownTypeErr struct {
	Type string
}

func (e *UnknownTypeErr)Error()(string){
	return fmt.Sprintf("Unknown type %s", e.Type)
}

var ErrIncorrectFormat = errors.New("Incorrect format")

var boolType = reflect.TypeOf((bool)(false))
var intType = reflect.TypeOf((int)(0))
var int8Type = reflect.TypeOf((int8)(0))
var int16Type = reflect.TypeOf((int16)(0))
var int32Type = reflect.TypeOf((int32)(0))
var int64Type = reflect.TypeOf((int64)(0))
var uintType = reflect.TypeOf((uint)(0))
var uint8Type = reflect.TypeOf((uint8)(0))
var uint16Type = reflect.TypeOf((uint16)(0))
var uint32Type = reflect.TypeOf((uint32)(0))
var uint64Type = reflect.TypeOf((uint64)(0))
var uintptrType = reflect.TypeOf((uintptr)(0))
var float32Type = reflect.TypeOf((float32)(0))
var float64Type = reflect.TypeOf((float64)(0))
var complex64Type = reflect.TypeOf((complex64)(0))
var complex128Type = reflect.TypeOf((complex128)(0))
var stringType = reflect.TypeOf((string)(""))
var anyType = reflect.TypeOf((*any)(nil)).Elem()

var typeRegisterMap = map[string]reflect.Type{}
var typeToStringMap = map[reflect.Type]string{}

func RegisterType(v any){
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Pointer && t.PkgPath() == "" {
		t = t.Elem()
	}
	name := t.PkgPath()
	if len(name) == 0 {
		panic("Type is not exported")
	}
	name += "." + t.Name()
	typeRegisterMap[name] = t
	buf := bytes.NewBuffer(nil)
	if err := reflectTypeToString(t, buf); err != nil {
		panic(err)
	}
	typeToStringMap[t] = buf.String()
}

func getTypeRegisteredName(t reflect.Type)(name string, ok bool){
	if t.Kind() == reflect.Pointer && t.PkgPath() == "" {
		t = t.Elem()
	}
	if name = t.PkgPath(); name != "" {
		name += "." + t.Name()
		if _, ok = typeRegisterMap[name]; !ok {
			return "", false
		}
		return
	}
	switch t.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128, reflect.String:
		return "", true // builtin type, or alias of builtin type
	}
	return "", false
}

func reflectTypeToString(t reflect.Type, buf *bytes.Buffer)(err error){
	if tstr, ok := typeToStringMap[t]; ok {
		// use cached string
		buf.WriteString(tstr)
		return
	}
	switch t.Kind() {
	case reflect.Bool:       buf.WriteString("B");    return
	case reflect.Int:        buf.WriteString("i");    return
	case reflect.Int8:       buf.WriteString("i8");   return
	case reflect.Int16:      buf.WriteString("i16");  return
	case reflect.Int32:      buf.WriteString("i32");  return
	case reflect.Int64:      buf.WriteString("i64");  return
	case reflect.Uint:       buf.WriteString("u");    return
	case reflect.Uint8:      buf.WriteString("u8");   return
	case reflect.Uint16:     buf.WriteString("u16");  return
	case reflect.Uint32:     buf.WriteString("u32");  return
	case reflect.Uint64:     buf.WriteString("u64");  return
	case reflect.Uintptr:    buf.WriteString("uptr"); return
	case reflect.Float32:    buf.WriteString("f32");  return
	case reflect.Float64:    buf.WriteString("f64");  return
	case reflect.Complex64:  buf.WriteString("c64");  return
	case reflect.Complex128: buf.WriteString("c128"); return
	case reflect.String:     buf.WriteString("str");  return
	case reflect.Pointer:
		buf.WriteByte('*')
		return reflectTypeToString(t.Elem(), buf)
	case reflect.Slice:
		buf.WriteString("[]")
		return reflectTypeToString(t.Elem(), buf)
	case reflect.Array:
		buf.WriteByte('[')
		buf.WriteString(strconv.Itoa(t.Len()))
		buf.WriteByte(']')
		return reflectTypeToString(t.Elem(), buf)
	case reflect.Map:
		buf.WriteString("map[")
		if err = reflectTypeToString(t.Key(), buf); err != nil {
			return
		}
		buf.WriteByte(']')
		return reflectTypeToString(t.Elem(), buf)
	case reflect.Struct:
		buf.WriteString("struct{")
		var flag bool = false
		for i, l := 0, t.NumField(); i < l; i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			if flag {
				buf.WriteByte(';')
			}else{
				flag = true
			}
			buf.WriteString(f.Name)
			buf.WriteByte(' ')
			if err = reflectTypeToString(f.Type, buf); err != nil {
				return
			}
		}
		buf.WriteByte('}')
		return
	case reflect.Interface:
		if t.NumMethod() == 0 { // only support `interface{}` or `any`
			buf.WriteString("any")
			return
		}
		return &UnsupportTypeErr{t}
	default:
		return &UnsupportTypeErr{t}
	}
}

type typeScanner struct {
	r io.ByteScanner

	peekLit string
	peekTk int // 0: invaild; 1: literal; 2: other
}

func newTypeScanner(r io.ByteScanner)(s *typeScanner){
	return &typeScanner{
		r: r,
	}
}

func isLiteralByte(b byte)(bool){
	return ('0' <= b && b <= '9') || ('A' <= b && b <= 'Z') || ('a' <= b && b <= 'z')
}

func (s *typeScanner)parseNext()(lit string, tk int, err error){
	b, err := s.r.ReadByte()
	if err != nil {
		return
	}
	bts := []byte{b}
	if !isLiteralByte(b) {
		return (string)(bts), 2, nil
	}
	for {
		b, err = s.r.ReadByte()
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			return
		}
		if !isLiteralByte(b) {
			if err = s.r.UnreadByte(); err != nil {
				return
			}
			break
		}
		bts = append(bts, b)
	}
	return (string)(bts), 1, nil
}

func (s *typeScanner)next()(lit string, tk int, err error){
	if s.peekTk != 0 {
		lit, tk = s.peekLit, s.peekTk
		s.peekTk = 0
		return
	}
	return s.parseNext()
}

func (s *typeScanner)peek()(lit string, tk int, err error){
	if s.peekTk == 0 {
		if s.peekLit, s.peekTk, err = s.parseNext(); err != nil {
			s.peekTk = 0
			return
		}
	}
	lit, tk = s.peekLit, s.peekTk
	return
}

func scanToReflectType(r *typeScanner)(t reflect.Type, err error){
	lit, tk, err := r.next()
	if err != nil {
		return
	}
	if tk == 1 {
		switch lit {
		case "B":    return boolType, nil
		case "i":    return intType, nil
		case "i8":   return int8Type, nil
		case "i16":  return int16Type, nil
		case "i32":  return int32Type, nil
		case "i64":  return int64Type, nil
		case "u":    return uintType, nil
		case "u8":   return uint8Type, nil
		case "u16":  return uint16Type, nil
		case "u32":  return uint32Type, nil
		case "u64":  return uint64Type, nil
		case "uptr": return uintptrType, nil
		case "f32":  return float32Type, nil
		case "f64":  return float64Type, nil
		case "c64":  return complex64Type, nil
		case "c128": return complex128Type, nil
		case "str":  return stringType, nil
		case "any":  return anyType, nil
		case "map":
			var key, val reflect.Type
			if lit, tk, err = r.next(); err != nil {
				return
			}
			if lit != "[" {
				return nil, ErrIncorrectFormat
			}
			if key, err = scanToReflectType(r); err != nil {
				return
			}
			if lit, tk, err = r.next(); err != nil {
				return
			}
			if lit != "]" {
				return nil, ErrIncorrectFormat
			}
			if val, err = scanToReflectType(r); err != nil {
				return
			}
			t = reflect.MapOf(key, val)
			return
		case "struct":
			panic("TODO: support struct")
		}
	}else{
		if lit == "*" {
			if t, err = scanToReflectType(r); err != nil {
				return
			}
			t = reflect.PointerTo(t)
			return
		}else if lit == "[" {
			if lit, tk, err = r.next(); err != nil {
				return
			}
			if tk == 1 { // scanned a number which means it's an array type
				var l int
				if l, err = strconv.Atoi(lit); err != nil {
					return
				}
				if lit, tk, err = r.next(); err != nil {
					return
				}
				if lit != "]" {
					return nil, ErrIncorrectFormat
				}
				if t, err = scanToReflectType(r); err != nil {
					return
				}
				t = reflect.ArrayOf(l, t)
			}else{ // if it's a slice
				if lit != "]" {
					return nil, ErrIncorrectFormat
				}
				if t, err = scanToReflectType(r); err != nil {
					return
				}
				t = reflect.SliceOf(t)
			}
			return
		}
	}
	return nil, &UnknownTypeErr{lit}
}

func writeUint32(buf *bytes.Buffer, v uint32){
	buf.Write([]byte{
		(byte)((v >> 24) & 0xff),
		(byte)((v >> 16) & 0xff),
		(byte)((v >> 8) & 0xff),
		(byte)(v & 0xff),
	})
}

func readUint32(buf *bytes.Reader)(v uint32, err error){
	var b [4]byte
	if _, err = io.ReadFull(buf, b[:]); err != nil {
		return
	}
	v =
		((uint32)(b[0]) << 24) |
		((uint32)(b[1]) << 16) |
		((uint32)(b[2]) << 8) |
		(uint32)(b[3])
	return
}

func writeUint64(buf *bytes.Buffer, v uint64){
	buf.Write([]byte{
		(byte)((v >> 56) & 0xff),
		(byte)((v >> 48) & 0xff),
		(byte)((v >> 40) & 0xff),
		(byte)((v >> 32) & 0xff),
		(byte)((v >> 24) & 0xff),
		(byte)((v >> 16) & 0xff),
		(byte)((v >> 8) & 0xff),
		(byte)(v & 0xff),
	})
}

func readUint64(buf *bytes.Reader)(v uint64, err error){
	var b [8]byte
	if _, err = io.ReadFull(buf, b[:]); err != nil {
		return
	}
	v =
		((uint64)(b[0]) << 56) |
		((uint64)(b[1]) << 48) |
		((uint64)(b[2]) << 40) |
		((uint64)(b[3]) << 32) |
		((uint64)(b[4]) << 24) |
		((uint64)(b[5]) << 16) |
		((uint64)(b[6]) << 8) |
		(uint64)(b[7])
	return
}

func writeFloat32(buf *bytes.Buffer, v float32){
	writeUint32(buf, *((*uint32)((unsafe.Pointer)(&v))))
}

func readFloat32(buf *bytes.Reader)(v float32, err error){
	v0, err := readUint32(buf)
	if err != nil {
		return
	}
	return *((*float32)((unsafe.Pointer)(&v0))), nil
}

func writeFloat64(buf *bytes.Buffer, v float64){
	writeUint64(buf, *((*uint64)((unsafe.Pointer)(&v))))
}

func readFloat64(buf *bytes.Reader)(v float64, err error){
	v0, err := readUint64(buf)
	if err != nil {
		return
	}
	return *((*float64)((unsafe.Pointer)(&v0))), nil
}

func writeString(buf *bytes.Buffer, v string){
	writeUint64(buf, (uint64)(len(v)))
	buf.WriteString(v)
}

func readString(buf *bytes.Reader)(v string, err error){
	l, err := readUint64(buf)
	if err != nil {
		return
	}
	bts := make([]byte, l)
	if _, err = io.ReadFull(buf, bts); err != nil {
		return
	}
	v = (string)(bts)
	return
}

func writeReflectValue(value reflect.Value, buf *bytes.Buffer)(err error){
	t := value.Type()
	switch t.Kind() {
	case reflect.Bool:
		if value.Bool() {
			buf.WriteByte(0x01)
		}else{
			buf.WriteByte(0x00)
		}
		return
	case reflect.Int8:
		v := value.Int()
		buf.WriteByte((byte)(v))
		return
	case reflect.Int16, reflect.Int32:
		v := value.Int()
		writeUint32(buf, (uint32)(v))
		return
	case reflect.Int64, reflect.Int:
		v := value.Int()
		writeUint64(buf, (uint64)(v))
		return
	case reflect.Uint8:
		v := value.Uint()
		buf.WriteByte((byte)(v))
		return
	case reflect.Uint16, reflect.Uint32:
		v := value.Uint()
		writeUint32(buf, (uint32)(v))
		return
	case reflect.Uint64, reflect.Uint, reflect.Uintptr:
		v := value.Uint()
		writeUint64(buf, v)
		return
	case reflect.Float32:
		v := value.Float()
		writeFloat32(buf, (float32)(v))
		return
	case reflect.Float64:
		v := value.Float()
		writeFloat64(buf, v)
		return
	case reflect.Complex64:
		v := (complex64)(value.Complex())
		writeFloat32(buf, real(v))
		writeFloat32(buf, imag(v))
		return
	case reflect.Complex128:
		v := value.Complex()
		writeFloat64(buf, real(v))
		writeFloat64(buf, imag(v))
		return
	case reflect.String:
		v := value.String()
		writeString(buf, v)
		return
	case reflect.Pointer:
		e := value.Elem()
		if value.IsNil() {
			buf.WriteByte(0x00)
			return
		}
		buf.WriteByte(0x01)
		return writeReflectValue(e, buf)
	case reflect.Slice:
		writeUint64(buf, (uint64)(value.Len()))
		if t.Elem().Kind() == reflect.Uint8 {
			buf.Write(value.Bytes())
			return
		}
		fallthrough
	case reflect.Array:
		l := value.Len()
		for i := 0; i < l; i++ {
			if err = writeReflectValue(value.Index(i), buf); err != nil {
				return
			}
		}
		return
	case reflect.Map:
		l := value.Len()
		writeUint64(buf, (uint64)(l))
		r := value.MapRange()
		i := 0
		for r.Next() {
			i++
			if err = writeReflectValue(r.Key(), buf); err != nil {
				return
			}
			if err = writeReflectValue(r.Value(), buf); err != nil {
				return
			}
		}
		if i != l {
			panic("Map changed during serialize")
		}
		return
	case reflect.Struct:
		for i, l := 0, t.NumField(); i < l; i++ {
			if !t.Field(i).IsExported() {
				continue
			}
			if err = writeReflectValue(value.Field(i), buf); err != nil {
				return
			}
		}
		return
	case reflect.Interface:
		if t.NumMethod() == 0 {
			var bts []byte
			if bts, err = encodeValue(value.Interface()); err != nil {
				return
			}
			writeUint64(buf, (uint64)(len(bts)))
			buf.Write(bts)
			return
		}
		return &UnsupportTypeErr{t}
	default:
		return &UnsupportTypeErr{t}
	}
}

func readToReflectValue(value reflect.Value, buf *bytes.Reader)(err error){
	t := value.Type()
	switch t.Kind() {
	case reflect.Bool:
		var v byte
		if v, err = buf.ReadByte(); err != nil {
			return
		}
		value.SetBool(v != 0x00)
		return
	case reflect.Int8:
		var v byte
		if v, err = buf.ReadByte(); err != nil {
			return
		}
		value.SetInt((int64)(v))
		return
	case reflect.Int16, reflect.Int32:
		var v uint32
		if v, err = readUint32(buf); err != nil {
			return
		}
		value.SetInt((int64)(v))
		return
	case reflect.Int64, reflect.Int:
		var v uint64
		if v, err = readUint64(buf); err != nil {
			return
		}
		value.SetInt((int64)(v))
		return
	case reflect.Uint8:
		var v byte
		if v, err = buf.ReadByte(); err != nil {
			return
		}
		value.SetUint((uint64)(v))
		return
	case reflect.Uint16, reflect.Uint32:
		var v uint32
		if v, err = readUint32(buf); err != nil {
			return
		}
		value.SetUint((uint64)(v))
		return
	case reflect.Uint64, reflect.Uint, reflect.Uintptr:
		var v uint64
		if v, err = readUint64(buf); err != nil {
			return
		}
		value.SetUint(v)
		return
	case reflect.Float32:
		var v float32
		if v, err = readFloat32(buf); err != nil {
			return
		}
		value.SetFloat((float64)(v))
		return
	case reflect.Float64:
		var v float64
		if v, err = readFloat64(buf); err != nil {
			return
		}
		value.SetFloat(v)
		return
	case reflect.Complex64:
		var rl, im float32
		if rl, err = readFloat32(buf); err != nil {
			return
		}
		if im, err = readFloat32(buf); err != nil {
			return
		}
		value.SetComplex((complex128)(complex(rl, im)))
		return
	case reflect.Complex128:
		var rl, im float64
		if rl, err = readFloat64(buf); err != nil {
			return
		}
		if im, err = readFloat64(buf); err != nil {
			return
		}
		value.SetComplex(complex(rl, im))
		return
	case reflect.String:
		var v string
		if v, err = readString(buf); err != nil {
			return
		}
		value.SetString(v)
		return
	case reflect.Pointer:
		var b byte
		if b, err = buf.ReadByte(); err != nil {
			return
		}
		if b != 0x00 {
			e := reflect.New(t.Elem())
			value.Set(e)
			return readToReflectValue(e.Elem(), buf)
		}
		return
	case reflect.Slice:
		var l uint64
		if l, err = readUint64(buf); err != nil {
			return
		}
		for i := 0; i < (int)(l); i++ {
			if err = readToReflectValue(value.Index(i), buf); err != nil {
				return
			}
		}
	case reflect.Array:
		l := value.Len()
		for i := 0; i < l; i++ {
			if err = readToReflectValue(value.Index(i), buf); err != nil {
				return
			}
		}
		return
	case reflect.Map:
		var l uint64
		if l, err = readUint64(buf); err != nil {
			return
		}
		value.Set(reflect.MakeMapWithSize(t, (int)(l)))
		for i := 0; i < (int)(l); i++ {
			k, v := reflect.New(t.Key()).Elem(), reflect.New(t.Elem()).Elem()
			if err = readToReflectValue(k, buf); err != nil {
				return
			}
			if err = readToReflectValue(v, buf); err != nil {
				return
			}
			value.SetMapIndex(k, v)
		}
		return
	case reflect.Struct:
		for i, l := 0, t.NumField(); i < l; i++ {
			if !t.Field(i).IsExported() {
				continue
			}
			if err = readToReflectValue(value.Field(i), buf); err != nil {
				return
			}
		}
		return
	case reflect.Interface:
		if t.NumMethod() == 0 {
			var l uint64
			if l, err = readUint64(buf); err != nil {
				return
			}
			bts := make([]byte, l)
			if _, err = io.ReadFull(buf, bts); err != nil {
				return
			}
			var v any
			if v, err = decodeValue(bts); err != nil {
				return
			}
			if v != nil {
				value.Set(reflect.ValueOf(v))
			}
			return
		}
		return &UnsupportTypeErr{t}
	default:
		return &UnsupportTypeErr{t}
	}
	return
}

func encodeValue(value any)(bts []byte, err error){
	if value == nil { // Zero length byte array represent the value is nil
		return
	}
	rv := reflect.ValueOf(value)
	rt := rv.Type()
	tname, ok := getTypeRegisteredName(rt)
	if !ok {
		return nil, &UnsupportTypeErr{rt}
	}
	buf := bytes.NewBuffer(nil)
	buf.WriteString(tname)
	buf.WriteByte(':')
	if err = reflectTypeToString(rt, buf); err != nil {
		return
	}
	buf.WriteByte(0x00)
	if err = writeReflectValue(rv, buf); err != nil {
		return
	}
	bts = buf.Bytes()
	return
}

func decodeValue(bts []byte)(value any, err error){
	if len(bts) == 0 {
		return
	}
	i := bytes.IndexByte(bts, 0x00)
	if i == -1 {
		return nil, ErrIncorrectFormat
	}
	typd, data := bts[:i], bts[i + 1:]
	i = bytes.IndexByte(typd, ':')
	if i == -1 {
		return nil, ErrIncorrectFormat
	}
	tname, typd := (string)(typd[:i]), typd[i + 1:]
	if len(typd) == 0 {
		return nil, ErrIncorrectFormat
	}
	rt, ok := typeRegisterMap[tname]
	if ok {
		if typd[0] == '*' && rt.Kind() != reflect.Pointer {
			rt = reflect.PointerTo(rt)
		}
	}else if rt, err = scanToReflectType(newTypeScanner(bytes.NewReader(typd))); err != nil {
		return
	}
	rv := reflect.New(rt).Elem()
	buf := bytes.NewReader(data)
	if err = readToReflectValue(rv, buf); err != nil {
		return
	}
	value = rv.Interface()
	return
}

func WrapValue(value any)(v *Any, err error){
	bts, err := encodeValue(value)
	if err != nil {
		return
	}
	return &Any{
		Value: bts,
	}, nil
}

func (v *Any)Unwrap()(value any, err error){
	return decodeValue(v.Value)
}
