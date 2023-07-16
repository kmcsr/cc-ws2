
package main

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Must be sync with https://tweaked.cc/module/colors.html
type Color uint16

const (
	ColorIllegal   Color = 0
	ColorWhite     Color = 1 << 0x0
	ColorOrange    Color = 1 << 0x1
	ColorMagenta   Color = 1 << 0x2
	ColorLightBlue Color = 1 << 0x3
	ColorYellow    Color = 1 << 0x4
	ColorLime      Color = 1 << 0x5
	ColorPink      Color = 1 << 0x6
	ColorGray      Color = 1 << 0x7
	ColorLightGray Color = 1 << 0x8
	ColorCyan      Color = 1 << 0x9
	ColorPurple    Color = 1 << 0xa
	ColorBlue      Color = 1 << 0xb
	ColorBrown     Color = 1 << 0xc
	ColorGreen     Color = 1 << 0xd
	ColorRed       Color = 1 << 0xe
	ColorBlack     Color = 1 << 0xf
)

func CodeToColor(c byte)(Color){
	switch c {
	case '0': return ColorWhite
	case '1': return ColorOrange
	case '2': return ColorMagenta
	case '3': return ColorLightBlue
	case '4': return ColorYellow
	case '5': return ColorLime
	case '6': return ColorPink
	case '7': return ColorGray
	case '8': return ColorLightGray
	case '9': return ColorCyan
	case 'a': return ColorPurple
	case 'b': return ColorBlue
	case 'c': return ColorBrown
	case 'd': return ColorGreen
	case 'e': return ColorRed
	case 'f': return ColorBlack
	default: return ColorIllegal
	}
}

// Must be sync with <https://tweaked.cc/module/colors.html>
var defaultPaletteColors = map[Color]int{
	ColorWhite    : 0xF0F0F0,
	ColorOrange   : 0xF2B233,
	ColorMagenta  : 0xE57FD8,
	ColorLightBlue: 0x99B2F2,
	ColorYellow   : 0xDEDE6C,
	ColorLime     : 0x7FCC19,
	ColorPink     : 0xF2B2CC,
	ColorGray     : 0x4C4C4C,
	ColorLightGray: 0x999999,
	ColorCyan     : 0x4C99B2,
	ColorPurple   : 0xB266E5,
	ColorBlue     : 0x3366CC,
	ColorBrown    : 0x7F664C,
	ColorGreen    : 0x57A64E,
	ColorRed      : 0xCC4C4C,
	ColorBlack    : 0x111111,
}

type OperNotDefinedErr struct {
	Oper string
}

func (e *OperNotDefinedErr)Error()(string){
	return fmt.Sprintf("Operation %q is not defined", e.Oper)
}

type ArgTypeErr struct {
	Index int
	Expect string
}

func (e *ArgTypeErr)Error()(string){
	return fmt.Sprintf("Expect \"%s\" for arg #%d", e.Expect, e.Index + 1)
}

type InvalidColorErr struct {
	Got Color
}

func (e *InvalidColorErr)Error()(string){
	return fmt.Sprintf("Invalid color (got %d)", e.Got)
}

var BlitLengthErr = errors.New("The argument's length must be equal")

type lineT struct {
	Text []byte
	Color []Color
	Background []Color
}

var _ json.Marshaler = lineT{}

func (l lineT)MarshalJSON()([]byte, error){
	return json.Marshal(Map{
		"text": (string)(l.Text),
		"color": l.Color,
		"background": l.Background,
	})
}

type TermEventCallback = func(t *Term, event string, args List)

type Term struct {
	Title string

	width, height int
	cursorX, cursorY int
	textColor Color
	backgroundColor Color
	lines []lineT
	palette map[Color]int
	cursorBlink bool

	OnEvent TermEventCallback
}

func NewTerm(width, height int, title string)(t *Term){
	lines := make([]lineT, height)
	for i, _ := range lines {
		lines[i] = lineT{
			Text: make([]byte, width),
			Color: make([]Color, width),
			Background: make([]Color, width),
		}
	}
	palette := make(map[Color]int, 16)
	for k, v := range defaultPaletteColors {
		palette[k] = v
	}
	t = &Term{
		Title: title,
		width: width,
		height: height,
		cursorX: 0,
		cursorY: 0,
		textColor: ColorWhite,
		backgroundColor: ColorBlack,
		lines: lines,
		cursorBlink: false,
		palette: palette,
	}
	t.clear()
	return
}

func (t *Term)clearLine(y int){
	if y < 0 || y >= t.height {
		return
	}
	l := t.lines[y]
	for i := 0; i < t.width; i++ {
		l.Text[i] = ' '
		l.Color[i] = t.textColor
		l.Background[i] = t.backgroundColor
	}
}

func (t *Term)clear(){
	for _, l := range t.lines {
		for i := 0; i < t.width; i++ {
			l.Text[i] = ' '
			l.Color[i] = t.textColor
			l.Background[i] = t.backgroundColor
		}
	}
}

func (t *Term)oper(oper string, args List)(res []any, err error){
	loger.Tracef("Oper term: %s %v", oper, args)
	switch oper {
	case "nativePaletteColour":
		fallthrough
	case "nativePaletteColor":
		color, ok := args.GetInt(0)
		if !ok {
			return nil, &ArgTypeErr{ 0, "int" }
		}
		c := (Color)(color)
		v, ok := defaultPaletteColors[c]
		if !ok {
			return nil, &InvalidColorErr{c}
		}
		return []any{v}, nil
	case "write":
		text, ok := args.GetString(0)
		if !ok {
			return nil, &ArgTypeErr{ 0, "string" }
		}
		if t.cursorY < 0 || t.cursorY >= t.height || t.cursorX >= t.width {
			return
		}
		if t.cursorX < 0 {
			text = text[-t.cursorX:]
			t.cursorX = 0
		}
		line := t.lines[t.cursorY]
		l := copy(line.Text[t.cursorX:], text)
		for i := 0; i < l; i++ {
			line.Color[t.cursorX + i] = t.textColor
			line.Background[t.cursorX + i] = t.backgroundColor
		}
		t.cursorX += l
		return
	case "scroll":
		offset, ok := args.GetInt(0)
		if !ok {
			return nil, &ArgTypeErr{ 0, "int" }
		}
		if offset == 0 {
			return
		}
		down := offset < 0
		if down {
			offset = -offset
		}
		if offset >= t.height {
			t.clear()
			return
		}
		if down {
			s := copy(t.lines, t.lines[t.height - offset:])
			for i := 0; i < s; i++ {
				t.clearLine(i)
			}
		}else{
			i := copy(t.lines[offset:], t.lines)
			for ; i < t.height; i++ {
				t.clearLine(i)
			}
		}
		return
	case "getCursorPos":
		return []any{ t.cursorX + 1, t.cursorY + 1 }, nil
	case "setCursorPos":
		x, ok := args.GetInt(0)
		if !ok {
			return nil, &ArgTypeErr{ 0, "int" }
		}
		y, ok := args.GetInt(1)
		if !ok {
			return nil, &ArgTypeErr{ 1, "int" }
		}
		t.cursorX = x - 1
		t.cursorY = y - 1
		return
	case "getCursorBlink":
		return []any{ t.cursorBlink }, nil
	case "setCursorBlink":
		blink, ok := args.GetBool(0)
		if !ok {
			return nil, &ArgTypeErr{ 0, "bool" }
		}
		t.cursorBlink = blink
		return
	case "getSize":
		return []any{ t.width, t.height }, nil
	case "clear":
		t.clear()
		return
	case "clearLine":
		y, ok := args.GetInt(0)
		if !ok {
			y = t.cursorY
		}
		t.clearLine(y)
		return
	case "getTextColour":
		fallthrough
	case "getTextColor":
		return []any{ t.textColor }, nil
	case "setTextColour":
		fallthrough
	case "setTextColor":
		color, ok := args.GetInt(0)
		if !ok {
			return nil, &ArgTypeErr{ 0, "int" }
		}
		c := (Color)(color)
		if _, ok := defaultPaletteColors[c]; !ok {
			return nil, &InvalidColorErr{c}
		}
		t.textColor = c
		return
	case "getBackgroundColour":
		fallthrough
	case "getBackgroundColor":
		return []any{ t.backgroundColor }, nil
	case "setBackgroundColour":
		fallthrough
	case "setBackgroundColor":
		color, ok := args.GetInt(0)
		if !ok {
			return nil, &ArgTypeErr{ 0, "int" }
		}
		c := (Color)(color)
		if _, ok := defaultPaletteColors[c]; !ok {
			return nil, &InvalidColorErr{c}
		}
		t.backgroundColor = c
		return
	case "isColour":
		fallthrough
	case "isColor":
		return []any{ true }, nil
	case "blit":
		text, ok := args.GetString(0)
		if !ok {
			return nil, &ArgTypeErr{ 0, "string" }
		}
		color, ok := args.GetString(1)
		if !ok {
			return nil, &ArgTypeErr{ 1, "string" }
		}
		bgColor, ok := args.GetString(2)
		if !ok {
			return nil, &ArgTypeErr{ 2, "string" }
		}
		ln := len(text)
		if ln != len(color) || ln != len(bgColor) {
			return nil, BlitLengthErr
		}
		if t.cursorY < 0 || t.cursorY >= t.height || t.cursorX >= t.width {
			return
		}
		if t.cursorX < 0 {
			text = text[-t.cursorX:]
			color = color[-t.cursorX:]
			bgColor = bgColor[-t.cursorX:]
			t.cursorX = 0
		}
		line := t.lines[t.cursorY]
		for i := 0; i < ln; i++ {
			j := t.cursorX + ln
			if j >= t.width {
				break
			}
			line.Text[j] = text[i]
			c := CodeToColor(color[i])
			if c == ColorIllegal {
				c = t.textColor
			}
			line.Color[j] = c
			c = CodeToColor(bgColor[i])
			if c == ColorIllegal {
				c = t.backgroundColor
			}
			line.Background[j] = c
		}
		return
	case "setPaletteColour":
		fallthrough
	case "setPaletteColor":
		color, ok := args.GetInt(0)
		if !ok {
			return nil, &ArgTypeErr{ 0, "color" }
		}
		c := (Color)(color)
		if _, ok := defaultPaletteColors[c]; !ok {
			return nil, &InvalidColorErr{c}
		}
		if len(args) <= 2 {
			rgb, ok := args.GetInt(1)
			if !ok {
				return nil, &ArgTypeErr{ 1, "int" }
			}
			if rgb < 0x000000 {
				rgb = 0x000000
			}
			if rgb > 0xffffff {
				rgb = 0xffffff
			}
			t.palette[c] = rgb
		}else{
			r, ok := args.GetFloat(1)
			if !ok {
				return nil, &ArgTypeErr{ 1, "float" }
			}
			g, ok := args.GetFloat(2)
			if !ok {
				return nil, &ArgTypeErr{ 2, "float" }
			}
			b, ok := args.GetFloat(3)
			if !ok {
				return nil, &ArgTypeErr{ 3, "float" }
			}
			rgb := (inRange((int)(r * 0xff), 0x100) << 16) | (inRange((int)(g * 0xff), 0x100) << 8) | (inRange((int)(b * 0xff), 0x100))
			t.palette[c] = rgb
		}
		return
	case "getPaletteColour":
		fallthrough
	case "getPaletteColor":
		color, ok := args.GetInt(0)
		if !ok {
			return nil, &ArgTypeErr{ 0, "color" }
		}
		c := (Color)(color)
		v, ok := t.palette[c]
		if !ok {
			return nil, &InvalidColorErr{c}
		}
		r := (float64)((v >> 16) & 0xff) / 0xff
		g := (float64)((v >> 8) & 0xff) / 0xff
		b := (float64)(v & 0xff) / 0xff
		return []any{r, g, b}, nil
	default:
		return nil, &OperNotDefinedErr{oper}
	}
}
