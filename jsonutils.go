// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jsonutils

import (
	"bytes"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
)

type JSONObject interface {
	gotypes.ISerializable

	parse(str []byte, offset int) (int, error)
	writeSource

	// String() string
	PrettyString() string
	prettyString(level int) string
	YAMLString() string
	QueryString() string
	_queryString(key string) string
	Contains(keys ...string) bool
	ContainsIgnoreCases(keys ...string) bool
	Get(keys ...string) (JSONObject, error)
	GetIgnoreCases(keys ...string) (JSONObject, error)
	GetAt(i int, keys ...string) (JSONObject, error)
	Int(keys ...string) (int64, error)
	Float(keys ...string) (float64, error)
	Bool(keys ...string) (bool, error)
	GetMap(keys ...string) (map[string]JSONObject, error)
	GetArray(keys ...string) ([]JSONObject, error)
	GetTime(keys ...string) (time.Time, error)
	GetString(keys ...string) (string, error)
	Unmarshal(obj interface{}, keys ...string) error
	Equals(obj JSONObject) bool
	unmarshalValue(val reflect.Value) error
	// IsZero() bool
	Interface() interface{}
	isCompond() bool
}

type JSONValue struct {
}

var (
	JSONNull  = &JSONValue{}
	JSONTrue  = &JSONBool{data: true}
	JSONFalse = &JSONBool{data: false}
)

type JSONDict struct {
	JSONValue
	data map[string]JSONObject
}

type JSONArray struct {
	JSONValue
	data []JSONObject
}

type JSONString struct {
	JSONValue
	data string
}

type JSONInt struct {
	JSONValue
	data int64
}

type JSONFloat struct {
	JSONValue
	data float64
}

type JSONBool struct {
	JSONValue
	data bool
}

func skipEmpty(str []byte, offset int) int {
	i := offset
	for i < len(str) {
		switch str[i] {
		case ' ', '\t', '\n', '\r':
			i++
		default:
			return i
		}
	}
	return i
}

func hexchar2num(v byte) (byte, error) {
	switch {
	case v >= '0' && v <= '9':
		return v - '0', nil
	case v >= 'a' && v <= 'f':
		return v - 'a' + 10, nil
	case v >= 'A' && v <= 'F':
		return v - 'A' + 10, nil
	default:
		return 0, ErrInvalidChar // fmt.Errorf("Illegal char %c", v)
	}
}

func hexstr2byte(str []byte) (byte, error) {
	if len(str) < 2 {
		return 0, ErrInvalidHex // fmt.Errorf("Input must be 2 hex chars")
	}
	v1, e := hexchar2num(str[0])
	if e != nil {
		return 0, e
	}
	v2, e := hexchar2num(str[1])
	if e != nil {
		return 0, e
	}
	return v1*16 + v2, nil
}

func hexstr2rune(str []byte) (rune, error) {
	if len(str) < 4 {
		return 0, ErrInvalidRune // fmt.Errorf("Input must be 4 hex chars")
	}
	v1, e := hexstr2byte(str[0:2])
	if e != nil {
		return 0, e
	}
	v2, e := hexstr2byte(str[2:4])
	if e != nil {
		return 0, e
	}
	return rune(v1)*256 + rune(v2), nil
}

func parseQuoteString(str []byte, offset int, quotec byte) (string, int, error) {
	var (
		buffer    []byte
		runebytes = make([]byte, 4)
		runen     int
		i         = offset
	)
ret:
	for i < len(str) {
		switch str[i] {
		case '\\':
			if i+1 < len(str) {
				i++
				switch str[i] {
				case 'u':
					i++
					if i+4 >= len(str) {
						return "", i, NewJSONError(str, i, "Incomplete unicode")
					}
					r, e := hexstr2rune(str[i : i+4])
					if e != nil {
						return "", i, NewJSONError(str, i, e.Error())
					}
					runen = utf8.EncodeRune(runebytes, r)
					buffer = append(buffer, runebytes[0:runen]...)
					i += 4
				case 'x':
					i++
					if i+2 >= len(str) {
						return "", i, NewJSONError(str, i, "Incomplete hex")
					}
					b, e := hexstr2byte(str[i : i+2])
					if e != nil {
						return "", i, NewJSONError(str, i, e.Error())
					}
					buffer = append(buffer, b)
					i += 2
				case 'n':
					buffer = append(buffer, '\n')
					i++
				case 'r':
					buffer = append(buffer, '\r')
					i++
				case 't':
					buffer = append(buffer, '\t')
					i++
				default:
					buffer = append(buffer, str[i])
					i++
				}
			} else {
				return "", i, NewJSONError(str, i, "Incomplete escape")
			}
		case quotec:
			i++
			break ret
		default:
			buffer = append(buffer, str[i])
			i++
		}
	}
	return string(buffer), i, nil
}

func parseString(str []byte, offset int) (string, bool, int, error) {
	var (
		i = offset
	)
	if c := str[i]; c == '"' || c == '\'' {
		r, newOfs, err := parseQuoteString(str, i+1, c)
		return r, true, newOfs, err
	}
ret2:
	for i < len(str) {
		switch str[i] {
		case ' ', ':', ',', '\t', '\n', '}', ']':
			break ret2
		default:
			i++
		}
	}
	return string(str[offset:i]), false, i, nil
}

func parseJSONValue(str []byte, offset int) (JSONObject, int, error) {
	val, quote, i, e := parseString(str, offset)
	if e != nil {
		return nil, i, errors.Wrap(e, "parseString")
	} else if quote {
		return &JSONString{data: val}, i, nil
	} else {
		lval := strings.ToLower(val)
		if len(lval) == 0 || lval == "null" || lval == "none" {
			return JSONNull, i, nil
		}
		if lval == "true" || lval == "yes" {
			return JSONTrue, i, nil
		}
		if lval == "false" || lval == "no" {
			return JSONFalse, i, nil
		}
		if ival, err := strconv.ParseInt(val, 10, 64); err == nil {
			return &JSONInt{data: ival}, i, nil
		} else if fval, err := strconv.ParseFloat(val, 64); err == nil {
			return &JSONFloat{data: fval}, i, nil
		} else {
			return &JSONString{data: val}, i, nil
		}
	}
}

func quoteString(str string) string {
	sb := &strings.Builder{}
	sb.Grow(len(str) + 2)
	sb.WriteByte('"')
	for i := 0; i < len(str); i++ {
		switch c := str[i]; c {
		case '"':
			sb.Write([]byte{'\\', '"'})
		case '\r':
			sb.Write([]byte{'\\', 'r'})
		case '\n':
			sb.Write([]byte{'\\', 'n'})
		case '\t':
			sb.Write([]byte{'\\', 't'})
		case '\\':
			sb.Write([]byte{'\\', '\\'})
		default:
			sb.WriteByte(c)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}

func jsonPrettyString(o JSONObject, level int) string {
	var buffer bytes.Buffer
	for i := 0; i < level; i++ {
		buffer.WriteString("  ")
	}
	buffer.WriteString(o.String())
	return buffer.String()
}

func (this *JSONString) PrettyString() string {
	return this.String()
}

func (this *JSONString) prettyString(level int) string {
	return jsonPrettyString(this, level)
}

func (this *JSONString) Value() string {
	return this.data
}

func (this *JSONValue) parse(str []byte, offset int) (int, error) {
	return 0, nil
}

func (this *JSONValue) PrettyString() string {
	return this.String()
}

func (this *JSONValue) prettyString(level int) string {
	return jsonPrettyString(this, level)
}

func (this *JSONInt) PrettyString() string {
	return this.String()
}

func (this *JSONInt) prettyString(level int) string {
	return jsonPrettyString(this, level)
}

func (this *JSONInt) Value() int64 {
	return this.data
}

func (this *JSONFloat) PrettyString() string {
	return this.String()
}

func (this *JSONFloat) prettyString(level int) string {
	return jsonPrettyString(this, level)
}

func (this *JSONFloat) Value() float64 {
	return this.data
}

func (this *JSONBool) PrettyString() string {
	return this.String()
}

func (this *JSONBool) prettyString(level int) string {
	return jsonPrettyString(this, level)
}

func (this *JSONBool) Value() bool {
	return this.data
}

func parseDict(str []byte, offset int) (map[string]JSONObject, int, error) {
	var dict = make(map[string]JSONObject)
	if str[offset] != '{' {
		return dict, offset, NewJSONError(str, offset, "{ not found")
	}
	var i = offset + 1
	var e error = nil
	var key string
	var stop = false
	for !stop && i < len(str) {
		i = skipEmpty(str, i)
		if i >= len(str) {
			return dict, i, NewJSONError(str, i, "Truncated")
		}
		if str[i] == '}' {
			stop = true
			i++
			continue
		}
		key, _, i, e = parseString(str, i)
		if e != nil {
			return dict, i, errors.Wrap(e, "parseString")
		}
		if i >= len(str) {
			return dict, i, NewJSONError(str, i, "Truncated")
		}
		i = skipEmpty(str, i)
		if i >= len(str) {
			return dict, i, NewJSONError(str, i, "Truncated")
		}
		if str[i] != ':' {
			return dict, i, NewJSONError(str, i, ": not found")
		}
		i++
		i = skipEmpty(str, i)
		if i >= len(str) {
			return dict, i, NewJSONError(str, i, "Truncated")
		}
		var val JSONObject = nil
		switch str[i] {
		case '[':
			val = &JSONArray{}
			i, e = val.parse(str, i)
		case '{':
			val = &JSONDict{}
			i, e = val.parse(str, i)
		default:
			val, i, e = parseJSONValue(str, i)
		}
		if e != nil {
			return dict, i, errors.Wrap(e, "parse misc")
		}
		dict[key] = val
		i = skipEmpty(str, i)
		if i >= len(str) {
			return dict, i, NewJSONError(str, i, "Truncated")
		}
		switch str[i] {
		case ',':
			i++
		case '}':
			i++
			stop = true
		default:
			return dict, i, NewJSONError(str, i, "Unexpected char")
		}
	}
	return dict, i, nil
}

func parseArray(str []byte, offset int) ([]JSONObject, int, error) {
	if str[offset] != '[' {
		return nil, offset, NewJSONError(str, offset, "[ not found")
	}
	var (
		list []JSONObject
		i    = offset + 1
		val  JSONObject
		e    error
		stop bool
	)
	for !stop && i < len(str) {
		i = skipEmpty(str, i)
		if i >= len(str) {
			return list, i, NewJSONError(str, i, "Truncated")
		}
		switch str[i] {
		case ']':
			i++
			stop = true
			continue
		case '[':
			val = &JSONArray{}
			i, e = val.parse(str, i)
		case '{':
			val = &JSONDict{}
			i, e = val.parse(str, i)
		default:
			val, i, e = parseJSONValue(str, i)
		}
		if e != nil {
			return list, i, errors.Wrap(e, "parse misc")
		}
		if i >= len(str) {
			return list, i, NewJSONError(str, i, "Truncated")
		}
		list = append(list, val)
		i = skipEmpty(str, i)
		if i >= len(str) {
			return list, i, NewJSONError(str, i, "Truncated")
		}
		switch str[i] {
		case ',':
			i++
		case ']':
			i++
			stop = true
		default:
			return list, i, NewJSONError(str, i, "Unexpected char")
		}
	}
	return list, i, nil
}

func (this *JSONDict) parse(str []byte, offset int) (int, error) {
	val, i, e := parseDict(str, offset)
	if e == nil {
		this.data = val
	}
	return i, errors.Wrap(e, "parseDict")
}

func (this *JSONDict) SortedKeys() []string {
	keys := make([]string, 0, len(this.data))
	for k := range this.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (this *JSONDict) PrettyString() string {
	return this.prettyString(0)
}

func (this *JSONDict) prettyString(level int) string {
	var buffer bytes.Buffer
	var linebuf bytes.Buffer
	for i := 0; i < level; i++ {
		linebuf.WriteString("  ")
	}
	var tab = linebuf.String()
	buffer.WriteString(tab)
	buffer.WriteByte('{')
	var idx = 0
	for _, k := range this.SortedKeys() {
		v := this.data[k]
		if idx > 0 {
			buffer.WriteString(",")
		}
		buffer.WriteByte('\n')
		buffer.WriteString(tab)
		buffer.WriteString("  ")
		buffer.WriteByte('"')
		buffer.WriteString(k)
		buffer.WriteString("\":")
		_, okdict := v.(*JSONDict)
		_, okarray := v.(*JSONArray)
		if okdict || okarray {
			buffer.WriteByte('\n')
			buffer.WriteString(v.prettyString(level + 2))
		} else {
			buffer.WriteByte(' ')
			buffer.WriteString(v.String())
		}
		idx++
	}
	if len(this.data) > 0 {
		buffer.WriteByte('\n')
		buffer.WriteString(tab)
	}
	buffer.WriteByte('}')
	return buffer.String()
}

func (this *JSONDict) Value() map[string]JSONObject {
	return this.data
}

func (this *JSONArray) parse(str []byte, offset int) (int, error) {
	val, i, e := parseArray(str, offset)
	if e == nil {
		this.data = val
	}
	return i, errors.Wrap(e, "parseArray")
}

func (this *JSONArray) PrettyString() string {
	return this.prettyString(0)
}

func (this *JSONArray) prettyString(level int) string {
	var buffer bytes.Buffer
	var linebuf bytes.Buffer
	for i := 0; i < level; i++ {
		linebuf.WriteString("  ")
	}
	var tab = linebuf.String()
	buffer.WriteString(tab)
	buffer.WriteByte('[')
	for idx, v := range this.data {
		if idx > 0 {
			buffer.WriteString(",")
		}
		buffer.WriteByte('\n')
		buffer.WriteString(v.prettyString(level + 1))
	}
	if len(this.data) > 0 {
		buffer.WriteByte('\n')
		buffer.WriteString(tab)
	}
	buffer.WriteByte(']')
	return buffer.String()
}

func (this *JSONArray) Value() []JSONObject {
	return this.data
}

func ParseString(str string) (JSONObject, error) {
	return Parse([]byte(str))
}

func Parse(str []byte) (JSONObject, error) {
	var i = 0
	i = skipEmpty(str, i)
	var val JSONObject = nil
	var e error = nil
	if i < len(str) {
		switch str[i] {
		case '{':
			val = &JSONDict{}
			i, e = val.parse(str, i)
		case '[':
			val = &JSONArray{}
			i, e = val.parse(str, i)
		default:
			val, i, e = parseJSONValue(str, i)
			// return nil, NewJSONError(str, i, "Invalid JSON string")
		}
		if e != nil {
			return nil, errors.Wrap(e, "parse misc")
		} else {
			return val, nil
		}
	} else {
		return nil, NewJSONError(str, i, "Empty string")
	}
}
