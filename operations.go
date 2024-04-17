package fscache

import (
	"errors"
	"time"
)

var (
	errKeyNotFound = errors.New("key was not found")
)

func (ch Cache) Set(key string, value interface{}, duration ...time.Time) error {
	fs := make(map[string]interface{})
	fs[key] = value
	ch.Fscache = append(ch.Fscache, fs)

	return nil
}

func (ch Cache) Get(key string) (interface{}, error) {
	for _, cache := range ch.Fscache {
		if val, ok := cache[key]; ok {
			return val, nil
		}
	}

	return "", errKeyNotFound
}

func (ch Cache) Del(key string) error {
	for index, cache := range ch.Fscache {
		if _, ok := cache[key]; ok {
			ch.Fscache = append(ch.Fscache[:index], ch.Fscache[index+1:]...)
			return nil
		}
	}

	return errKeyNotFound
}
func (ch Cache) Clear() error
func (ch Cache) Size() int
func (ch Cache) MemSize() int
