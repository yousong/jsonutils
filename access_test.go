package jsonutils

import (
	"reflect"
	"testing"
)

func TestNewDict(t *testing.T) {
	dict := NewDict()
	dict.Add(NewString("1"), "a", "b")
	dict2, _ := ParseString("{\"a\": {\"b\": \"1\"}}")
	if dict.String() != dict2.String() {
		t.Errorf("Fail %s != %s", dict, dict2)
	}
	dict = NewDict()
	dict2, _ = ParseString("{}")
	if dict.String() != dict2.String() {
		t.Errorf("Fail %s != %s", dict, dict2)
	}
}

func TestNewArray(t *testing.T) {
	arr := NewArray()
	arr.Add(NewString("1"), NewInt(1), NewFloat(1.0))
	arr2, _ := ParseString("[\"1\", 1, 1.0]")
	if arr.String() != arr2.String() {
		t.Errorf("Fail %s != %s", arr, arr2)
	}
	arr = NewArray()
	arr2, _ = ParseString("[]")
	if arr.String() != arr2.String() {
		t.Errorf("Fail %s != %s", arr, arr2)
	}
}

func TestNewBool(t *testing.T) {
	type args struct {
		val bool
	}
	tests := []struct {
		name string
		args args
		want *JSONBool
	}{
		{
			name: "New-bool-true",
			args: args{true},
			want: &JSONBool{data: true},
		},
		{
			name: "New-bool-false",
			args: args{false},
			want: &JSONBool{data: false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewBool(tt.args.val); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJSONDictRemove(t *testing.T) {
	jd := JSONDict{
		data: map[string]JSONObject{
			"Hello": NewString("world"),
			"hello": NewString("world"),
			"HELLO": NewString("world"),
		},
	}
	var removed bool
	if removed = jd.Remove("HEllo"); removed {
		t.Fatalf("case sensitive remove, want false, got true")
	}
	if removed = jd.Remove("HELLO"); !removed {
		t.Fatalf("case sensitive remove, want true, got false")
	}
	if removed = jd.Remove("HELLO"); removed {
		t.Fatalf("case sensitive remove, want false, got true")
	}
	if removed = jd.RemoveIgnoreCase("hello"); !removed {
		t.Fatalf("case insensitive remove, want true, got false")
	}
	if removed = jd.RemoveIgnoreCase("hello"); removed {
		t.Fatalf("case insensitive false, want true, got true")
	}
}
