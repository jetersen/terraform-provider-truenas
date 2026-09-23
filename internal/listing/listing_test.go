// Copyright TrueNAS 2026
// SPDX-License-Identifier: MPL-2.0

package listing

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/list"
)

type stubClient struct {
	raw json.RawMessage
	err error
}

func (s stubClient) CallRead(_ context.Context, _ string, _ ...any) (json.RawMessage, error) {
	return s.raw, s.err
}

func countResults(stream *list.ListResultsStream) int {
	n := 0
	if stream.Results == nil {
		return 0
	}
	for range stream.Results {
		n++
	}
	return n
}

func TestStreamCollection_MapsEachRow(t *testing.T) {
	c := stubClient{raw: json.RawMessage(`[{"id":1},{"id":2},{"id":3}]`)}
	var stream list.ListResultsStream
	seen := 0
	StreamCollection(context.Background(), c, "x.query", list.ListRequest{}, &stream,
		func(_ context.Context, _ list.ListRequest, raw json.RawMessage) list.ListResult {
			seen++
			return list.ListResult{}
		})
	if got := countResults(&stream); got != 3 {
		t.Errorf("results = %d, want 3", got)
	}
	if seen != 3 {
		t.Errorf("mapRow calls = %d, want 3", seen)
	}
}

func TestStreamCollection_HonorsLimit(t *testing.T) {
	c := stubClient{raw: json.RawMessage(`[{"id":1},{"id":2},{"id":3}]`)}
	var stream list.ListResultsStream
	StreamCollection(context.Background(), c, "x.query", list.ListRequest{Limit: 2}, &stream,
		func(_ context.Context, _ list.ListRequest, _ json.RawMessage) list.ListResult { return list.ListResult{} })
	if got := countResults(&stream); got != 2 {
		t.Errorf("results = %d, want 2 (limit)", got)
	}
}

func TestStreamSingleton_EmitsOne(t *testing.T) {
	c := stubClient{raw: json.RawMessage(`{"id":1}`)}
	var stream list.ListResultsStream
	StreamSingleton(context.Background(), c, "x.config", list.ListRequest{}, &stream,
		func(_ context.Context, _ list.ListRequest, _ json.RawMessage) list.ListResult { return list.ListResult{} })
	if got := countResults(&stream); got != 1 {
		t.Errorf("results = %d, want 1", got)
	}
}

func TestIdentitySchemas(t *testing.T) {
	if _, ok := IntIDIdentitySchema().Attributes["id"]; !ok {
		t.Error("int identity schema missing id")
	}
	if _, ok := StringIDIdentitySchema().Attributes["id"]; !ok {
		t.Error("string identity schema missing id")
	}
}
