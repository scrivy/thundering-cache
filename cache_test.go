package tcache

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"math/rand"
	"reflect"
	"strconv"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	// create a simple cache without prewarming
	cache, err := New(getMd5Value, nil)
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
	cache, _ := New(getMd5Value, nil)
	valueInterface, err := cache.Get("2")
	if err != nil {
		t.Fatalf("error: %v, want nil", err)
	}
	value, ok := valueInterface.(string)
	if !ok {
		t.Fatal("type assertion failed")
	}
	twoMd5 := "c81e728d9d4c2f636f067f89cc14862c"
	if value != twoMd5 {
		t.Fatalf("value: %s, want %s", value, twoMd5)
	}

	// create a prewarmed cache and start slamming it, run wit -race
	cache, _ = New(getMd5Value, &preWarm)
	wg := &sync.WaitGroup{}
	slam1To10ALot(cache, wg)
	for k, vi := range preWarmMap {
		valueInterface, err := cache.Get(k)
		if err != nil {
			t.Fatalf("error: %v, want nil", err)
		}
		value, ok := valueInterface.(string)
		if !ok {
			t.Fatal("type assertion failed")
		}
		v := vi.(string)
		if value != v {
			t.Fatalf("value: %s, want %s", value, v)
		}
	}
	for i := 1; i < 1000; i++ {
		key := strconv.Itoa(i)
		shouldBe := computeMD5(key)
		if value, _ := cache.Get(key); value.(string) != shouldBe {
			t.Fatalf("key %v, value %v, want %v", key, value, shouldBe)
		}
	}
	wg.Wait()
}

func TestGetAll(t *testing.T) {
	// create a simple md5 cache
	cache, _ := New(getMd5Value, nil)

	// make sure it returns an empty map
	if cache.GetAll() == nil {
		t.Fatal("should have returned an empty map")
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
	cache, _ = New(getMd5Value, &preWarm)
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

func TestClear(t *testing.T) {
	// test clearing an initialized cache
	cache, _ := New(getMd5Value, &preWarm)
	eq := reflect.DeepEqual(
		cache.GetAll(),
		preWarmMap,
	)
	if !eq {
		t.Fatalf("values: %v, want %v", cache.GetAll(), preWarmMap)
	}
	cache.Clear()
	eq = reflect.DeepEqual(
		cache.GetAll(),
		map[string]interface{}{})
	if !eq {
		t.Fatalf("values: %v, expected empty map", cache.GetAll())
	}
}

// TODO
func TestUpdate(t *testing.T) {

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

var (
	preWarm = func() (map[string]interface{}, error) {
		return create1To10MD5Map(), nil
	}
	preWarmMap = map[string]interface{}{
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
)

func checkKey(key string, value string) bool {
	return computeMD5(key) == value
}

func slam1To10ALot(cache *Cache, wg *sync.WaitGroup) {
	for i := 0; i < 8000; i++ {
		wg.Add(1)
		go func(cache *Cache, wg *sync.WaitGroup) {
			for i := 0; i < 10; i++ {
				cache.Get(strconv.Itoa(rand.Intn(9) + 1))
			}
			wg.Done()
		}(cache, wg)
	}
}
