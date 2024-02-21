package rpc

import (
	"encoding/json"
	"reflect"
	"testing"
)

func Test_marshalArgs(t *testing.T) {
	type args struct {
		args []json.RawMessage
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			"no args",
			args{nil},
			[]byte("[]"),
		},

		{
			"numeric argument",
			args{args: []json.RawMessage{
				json.RawMessage("42"),
			}},
			[]byte("[42]"),
		},

		{
			"multiple various arguments",
			args{args: []json.RawMessage{
				json.RawMessage("42"),
				json.RawMessage("24"),
				json.RawMessage(`{"foo":"bar"}`),
				json.RawMessage(`null`),
			}},
			[]byte(`[42,24,{"foo":"bar"},null]`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := marshalArgs(tt.args.args...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("marshalArgs() = %s, want %s", got, tt.want)
			}
		})
	}
}
