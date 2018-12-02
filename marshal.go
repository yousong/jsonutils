package jsonutils

/**
jsonutils.Marshal

Convert any object to JSONObject

*/

import (
	"fmt"
	"reflect"
	"time"

	"strings"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
)

func marshalSlice(val reflect.Value, info *jsonMarshalInfo) JSONObject {
	if val.Len() == 0 && info != nil && info.omitEmpty {
		return nil
	}
	objs := make([]JSONObject, val.Len())
	for i := 0; i < val.Len(); i += 1 {
		objs[i] = marshalValue(val.Index(i), nil)
	}
	arr := NewArray(objs...)
	if info != nil && info.forceString {
		return NewString(arr.String())
	} else {
		return arr
	}
}

func marshalMap(val reflect.Value, info *jsonMarshalInfo) JSONObject {
	keys := val.MapKeys()
	if len(keys) == 0 && info != nil && info.omitEmpty {
		return nil
	}
	objPairs := make([]JSONPair, 0)
	for i := 0; i < len(keys); i += 1 {
		key := keys[i]
		val := marshalValue(val.MapIndex(key), nil)
		if val != JSONNull {
			objPairs = append(objPairs, JSONPair{key: fmt.Sprintf("%s", key), val: val})
		}
	}
	dict := NewDict(objPairs...)
	if info != nil && info.forceString {
		return NewString(dict.String())
	} else {
		return dict
	}
}

func marshalStruct(val reflect.Value, info *jsonMarshalInfo) JSONObject {
	objPairs := struct2JSONPairs(val)
	if len(objPairs) == 0 && info != nil && info.omitEmpty {
		return nil
	}
	dict := NewDict(objPairs...)
	if info != nil && info.forceString {
		return NewString(dict.String())
	} else {
		return dict
	}
}

type jsonMarshalInfo struct {
	ignore      bool
	omitEmpty   bool
	name        string
	forceString bool
}

func parseJsonMarshalInfo(fieldTag reflect.StructTag) jsonMarshalInfo {
	info := jsonMarshalInfo{}
	info.omitEmpty = true

	tags := utils.TagMap(fieldTag)
	if val, ok := tags["json"]; ok {
		keys := strings.Split(val, ",")
		if len(keys) > 0 {
			if keys[0] == "-" {
				if len(keys) > 1 {
					info.name = keys[0]
				} else {
					info.ignore = true
				}
			} else {
				info.name = keys[0]
			}
		}
		if len(keys) > 1 {
			for _, k := range keys[1:] {
				switch k {
				case "omitempty":
					info.omitEmpty = true
				case "string":
					info.forceString = true
				case "allowempty":
					info.omitEmpty = false
				}
			}
		}
	}
	if val, ok := tags["name"]; ok {
		info.name = val
	}
	return info
}

func struct2JSONPairs(val reflect.Value) []JSONPair {
	structType := val.Type()
	objPairs := make([]JSONPair, 0)
	for i := 0; i < structType.NumField(); i += 1 {
		fieldType := structType.Field(i)
		if !gotypes.IsFieldExportable(fieldType.Name) { // unexportable field, ignore
			continue
		}
		if fieldType.Anonymous {
			nextVal := val.Field(i)
			switch fieldType.Type.Kind() {
			case reflect.Struct: // embbed struct
				nextVal = val.Field(i)
			case reflect.Interface: // embbed interface
			CHECKINTERFACE:
				for {
					log.Debugf("%s %#v %#v", nextVal.Type(), nextVal.Type().Kind(), reflect.Struct)
					switch nextVal.Type().Kind() {
					case reflect.Interface:
						log.Debugf("anonymous interface, do elem...")
						nextVal = nextVal.Elem()
					case reflect.Ptr:
						log.Debugf("pointer interface, do indirect...")
						nextVal = reflect.Indirect(nextVal)
					case reflect.Struct:
						log.Debugf("anonuymous struct, break ...")
						break CHECKINTERFACE
					default:
						log.Warningf("embeded interface point to a non struct data %s", nextVal.Type())
						break CHECKINTERFACE
					}
				}
			default:
				log.Warningf("unsupport anonymous embeded type %s", fieldType.Type.Name())
				continue
			}
			log.Debugf("%#v", nextVal)
			newPairs := struct2JSONPairs(nextVal)
			objPairs = append(objPairs, newPairs...)
		} else {
			jsonInfo := parseJsonMarshalInfo(fieldType.Tag)

			if jsonInfo.ignore {
				continue
			}
			key := jsonInfo.name
			if len(key) == 0 {
				key = utils.CamelSplit(fieldType.Name, "_")
			}
			val := marshalValue(val.Field(i), &jsonInfo)
			if val != nil && val != JSONNull {
				objPair := JSONPair{key: key, val: val}
				objPairs = append(objPairs, objPair)
			}
		}
	}
	return objPairs
}

func marshalInt64(val int64, info *jsonMarshalInfo) JSONObject {
	if val == 0 && info != nil && info.omitEmpty {
		return nil
	} else if info != nil && info.forceString {
		return NewString(fmt.Sprintf("%d", val))
	} else {
		return NewInt(val)
	}
}

func marshalFloat64(val float64, info *jsonMarshalInfo) JSONObject {
	if val == 0.0 && info != nil && info.omitEmpty {
		return nil
	} else if info != nil && info.forceString {
		return NewString(fmt.Sprintf("%f", val))
	} else {
		return NewFloat(val)
	}
}

func marshalBoolean(val bool, info *jsonMarshalInfo) JSONObject {
	if !val && info != nil && info.omitEmpty {
		return nil
	} else if info != nil && info.forceString {
		return NewString(fmt.Sprintf("%v", val))
	} else {
		if val {
			return JSONTrue
		} else {
			return JSONFalse
		}
	}
}

func marshalTristate(val tristate.TriState, info *jsonMarshalInfo) JSONObject {
	if val.IsTrue() {
		return JSONTrue
	} else if val.IsFalse() {
		return JSONFalse
	} else {
		if info != nil && info.omitEmpty {
			return nil
		} else {
			return JSONNull
		}
	}
}

func marshalString(val string, info *jsonMarshalInfo) JSONObject {
	if len(val) == 0 && info != nil && info.omitEmpty {
		return nil
	} else {
		return NewString(val)
	}
}

func marshalTime(val time.Time, info *jsonMarshalInfo) *JSONString {
	if val.IsZero() {
		if info != nil && info.omitEmpty {
			return nil
		}
		return NewString("")
	} else {
		return NewString(timeutils.FullIsoTime(val))
	}
}

func Marshal(obj interface{}) JSONObject {
	if obj == nil {
		return JSONNull
	}
	objValue := reflect.Indirect(reflect.ValueOf(obj))
	return marshalValue(objValue, nil)
}

func marshalValue(objValue reflect.Value, info *jsonMarshalInfo) JSONObject {
	switch objValue.Type() {
	case JSONDictPtrType, JSONArrayPtrType, JSONBoolPtrType, JSONIntPtrType, JSONFloatPtrType, JSONStringPtrType, JSONObjectType:
		if objValue.IsNil() {
			return nil
		}
		return objValue.Interface().(JSONObject)
	case JSONDictType:
		json, ok := objValue.Interface().(JSONDict)
		if ok {
			if len(json.data) == 0 && info != nil && info.omitEmpty {
				return nil
			} else {
				return &json
			}
		} else {
			return nil
		}
	case JSONArrayType:
		json, ok := objValue.Interface().(JSONArray)
		if ok {
			if len(json.data) == 0 && info != nil && info.omitEmpty {
				return nil
			} else {
				return &json
			}
		} else {
			return nil
		}
	case JSONBoolType:
		json, ok := objValue.Interface().(JSONBool)
		if ok {
			if !json.data && info != nil && info.omitEmpty {
				return nil
			} else {
				return &json
			}
		} else {
			return nil
		}
	case JSONIntType:
		json, ok := objValue.Interface().(JSONInt)
		if ok {
			if json.data == 0 && info != nil && info.omitEmpty {
				return nil
			} else {
				return &json
			}
		} else {
			return nil
		}
	case JSONFloatType:
		json, ok := objValue.Interface().(JSONFloat)
		if ok {
			if json.data == 0.0 && info != nil && info.omitEmpty {
				return nil
			} else {
				return &json
			}
		} else {
			return nil
		}
	case JSONStringType:
		json, ok := objValue.Interface().(JSONString)
		if ok {
			if len(json.data) == 0 && info != nil && info.omitEmpty {
				return nil
			} else {
				return &json
			}
		} else {
			return nil
		}
	case tristate.TriStateType:
		tri, ok := objValue.Interface().(tristate.TriState)
		if ok {
			return marshalTristate(tri, info)
		} else {
			return nil
		}
	}
	switch objValue.Kind() {
	case reflect.Slice, reflect.Array:
		return marshalSlice(objValue, info)
	case reflect.Struct:
		if objValue.Type() == gotypes.TimeType {
			return marshalTime(objValue.Interface().(time.Time), info)
		} else {
			return marshalStruct(objValue, info)
		}
	case reflect.Map:
		return marshalMap(objValue, info)
	case reflect.String:
		strValue := objValue.Convert(gotypes.StringType)
		return marshalString(strValue.Interface().(string), info)
	case reflect.Bool:
		return marshalBoolean(objValue.Interface().(bool), info)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		intValue := objValue.Convert(gotypes.Int64Type)
		return marshalInt64(intValue.Interface().(int64), info)
	case reflect.Float32, reflect.Float64:
		floatValue := objValue.Convert(gotypes.Float64Type)
		return marshalFloat64(floatValue.Interface().(float64), info)
	case reflect.Interface, reflect.Ptr:
		if objValue.IsNil() {
			return nil
		}
		return marshalValue(objValue.Elem(), info)
	default:
		log.Errorf("unsupport object %s %s", objValue.Type(), objValue.Interface())
		return JSONNull
	}
}
