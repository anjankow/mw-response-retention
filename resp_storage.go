package main

import (
	"context"
	"sync"
)

type RespStorage struct {
	storage sync.Map
}

func (s *RespStorage) Store(ctx context.Context, key string, response RetainedResponse) error {
	s.storage.Store(key, response)
	return nil
}

func (s *RespStorage) Retrieve(ctx context.Context, key string) (RetainedResponse, error) {
	resp, ok := s.storage.Load(key)
	if !ok {
		return RetainedResponse{}, ErrNotRetained
	}

	return resp.(RetainedResponse), nil
}
