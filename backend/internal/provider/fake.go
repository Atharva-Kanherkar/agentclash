package provider

import "context"

type FakeClient struct {
	Response Response
	Err      error
	Requests []Request
}

func (f *FakeClient) InvokeModel(_ context.Context, request Request) (Response, error) {
	f.Requests = append(f.Requests, request)
	if f.Err != nil {
		return Response{}, f.Err
	}
	return f.Response, nil
}
