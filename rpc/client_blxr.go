package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

func marshalArgs(args ...json.RawMessage) []byte {
	parts := make([][]byte, 0, len(args))
	for _, arg := range args {
		parts = append(parts, arg)
	}

	data := [][]byte{
		{'['},
		bytes.Join(parts, []byte{','}),
		{']'},
	}

	return bytes.Join(data, []byte{})
}

// Forward arguments to a remote procedure as they come from sentry API handler.
func (c *Client) Forward(ctx context.Context, result interface{}, method string, args ...json.RawMessage) error {
	if result != nil && reflect.TypeOf(result).Kind() != reflect.Ptr {
		return fmt.Errorf("call result parameter must be pointer or nil interface: %v", result)
	}

	msg := &jsonrpcMessage{
		Version: vsn,
		ID:      c.nextID(),
		Method:  method,
		Params:  marshalArgs(args...),
	}

	op := &requestOp{
		ids:  []json.RawMessage{msg.ID},
		resp: make(chan []*jsonrpcMessage, 1),
	}

	var err error
	if c.isHTTP {
		err = c.sendHTTP(ctx, op, msg)
	} else {
		err = c.send(ctx, op, msg)
	}
	if err != nil {
		return err
	}

	// dispatch has accepted the request and will close the channel when it quits.
	batchresp, err := op.wait(ctx, c)
	if err != nil {
		return err
	}
	resp := batchresp[0]
	switch {
	case resp.Error != nil:
		return resp.Error
	case len(resp.Result) == 0:
		return ErrNoResult
	default:
		if result == nil {
			return nil
		}
		return json.Unmarshal(resp.Result, result)
	}
}
