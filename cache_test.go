package tcache

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"reflect"
	"strconv"
	"testing"
)

func TestNew(t *testing.T) {
	var err error
	var cache *Cache

	// create a simple cache without prewarming
	cache, err = New(getMd5Value, nil)
	if err != nil {
		t.Fatalf("error: %v, should not have returned an error", err)
	}
	eq := reflect.DeepEqual(
		cache.GetAll(),
		map[string]interface{}{},
	)
	if !eq {
		t.Fatalf("values: %v, want an empty map", cache.GetAll())
	}

	// test creating a prewarmed cache
	cache, err = New(getMd5Value, &preWarm)
	if err != nil {
		t.Fatalf("error: %v, should not have returned an error", err)
	}
	eq = reflect.DeepEqual(
		cache.GetAll(),
		preWarmMap,
	)
	if !eq {
		t.Fatalf("values: %v, want %v", cache.GetAll(), preWarmMap)
	}

	// test creating a prewarmed cache that fails to init
	testErr := errors.New("error")
	var preWarmErr = func() (errMap map[string]interface{}, err error) {
		err = testErr
		return
	}
	cache, err = New(getMd5Value, &preWarmErr)
	if err != testErr {
		t.Fatalf("error: %v, want %v", testErr)
	}
}

func TestGet(t *testing.T) {

}

func TestClear(t *testing.T) {
	var err error
	var cache *Cache

	// test clearing a non initiated cache
	cache = &Cache{}
	err = cache.Clear()
	if err != ErrNotInitialized {
		t.Fatalf("error: %v, want ErrNotInitialized", err)
	}

	// test clearing an initialized cache
	cache, _ = New(getMd5Value, &preWarm)
	err = cache.Clear()
	if err != nil {
		t.Fatalf("error: %v, want nil", err)
	}
	eq := reflect.DeepEqual(
		cache.GetAll(),
		map[string]interface{}{})
	if !eq {
		t.Fatalf("values: %v, expected empty map", cache.GetAll())
	}
}

func TestGetAll(t *testing.T) {
	var err error
	var cache *Cache

	// create a simple md5 cache
	cache, err = New(getMd5Value, nil)
	if err != nil {
		t.Fatalf("error: %v, should not have returned an error", err)
	}

	// make sure it returns an empty map
	if cache.GetAll() == nil {
		t.Fatal("should have returned an empty map")
	}
	if len(cache.GetAll()) != 0 {
		t.Fatalf("length: %d, want 0")
	}

	// fetch 2 values and get the map
	cache.Get("2")
	cache.Get("3")
	eq := reflect.DeepEqual(
		cache.GetAll(),
		map[string]interface{}{
			"2": computeMD5("2"),
			"3": computeMD5("3"),
		})
	if !eq {
		t.Fatal("maps should be equal")
	}

	// test a prewarmed cache
	cache, err = New(getMd5Value, &preWarm)
	if err != nil {
		t.Fatalf("error: %v, should not have returned an error", err)
	}
	eq = reflect.DeepEqual(
		cache.GetAll(),
		preWarmMap,
	)
	if !eq {
		t.Fatalf("values: %v, want %v", cache.GetAll(), preWarmMap)
	}

	// test getting a non initiated cache
	cache = &Cache{}
	if cache.GetAll() != nil {
		t.Fatalf("values: %v, wanted nil")
	}
}

func create1To10MD5Map() map[string]interface{} {
	items := make(map[string]interface{})
	for i := 1; i <= 10; i++ {
		key := strconv.Itoa(i)
		items[key] = computeMD5(key)
	}
	return items
}

func computeMD5(key string) string {
	md5Sum := md5.Sum([]byte(key))
	return hex.EncodeToString(md5Sum[:])
}

func getMd5Value(key string) (value interface{}, err error) {
	return computeMD5(key), nil
}

var preWarm = func() (map[string]interface{}, error) {
	return create1To10MD5Map(), nil
}

var preWarmMap = map[string]interface{}{
	"8":  "c9f0f895fb98ab9159f51fd0297e236d",
	"10": "d3d9446802a44259755d38e6d163e820",
	"1":  "c4ca4238a0b923820dcc509a6f75849b",
	"3":  "eccbc87e4b5ce2fe28308fd9f2a7baf3",
	"4":  "a87ff679a2f3e71d9181a67b7542122c",
	"5":  "e4da3b7fbbce2345d7772b0674a318d5",
	"6":  "1679091c5a880faf6fb5e6087eb1b2dc",
	"7":  "8f14e45fceea167a5a36dedd4bea2543",
	"9":  "45c48cce2e2d7fbdea1afc51c7c6ad26",
	"2":  "c81e728d9d4c2f636f067f89cc14862c",
}
