package main

import (
	"bytes"
	"context"
	"errors"
	"strings"

	"github.com/minio/minio-go/v7"
)

// s3Store keeps files under one prefix, which also becomes part of the address
// the app is given.
type s3Store struct {
	client *minio.Client
	bucket string
	prefix string
}

func prefixOf(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "/")
	if value == "" {
		return ""
	}
	return value + "/"
}

func (s *s3Store) Put(ctx context.Context, name string, data []byte) error {
	_, err := s.client.PutObject(ctx, s.bucket, s.prefix+name, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: "application/octet-stream", CacheControl: "no-cache"})
	return err
}

func (s *s3Store) List(ctx context.Context) ([]string, error) {
	var names []string
	for object := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: s.prefix}) {
		if object.Err != nil {
			return nil, errors.New("listing failed")
		}
		name := strings.TrimPrefix(object.Key, s.prefix)
		if !strings.Contains(name, "/") {
			names = append(names, name)
		}
	}
	return names, nil
}

func (s *s3Store) Delete(ctx context.Context, name string) error {
	return s.client.RemoveObject(ctx, s.bucket, s.prefix+name, minio.RemoveObjectOptions{})
}
