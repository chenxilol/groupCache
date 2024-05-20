package main

import (
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	loadCounts := make(map[string]int, len(db))
	gee := NewGroup("scores", 2<<10, 0, GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				if _, ok := loadCounts[key]; !ok {
					loadCounts[key] = 0
				}
				loadCounts[key] += 1
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	for k, v := range db {
		if view, err := gee.Get(k); err != nil || view.String() != v {
			t.Fatal("failed to get value of Tom")
		} // load from callback function
		if _, err := gee.Get(k); err != nil || loadCounts[k] > 1 {
			t.Fatalf("cache %s miss", k)
		} // cache hit
	}

	if view, err := gee.Get("unknown"); err == nil {
		t.Fatalf("the value of unknow should be empty, but %s got", view)
	}
}

func TestName(t *testing.T) {
	ticker := time.Tick(200 * time.Millisecond)
	for {
		select {
		case <-ticker:
			get, err := http.Get("http://localhost:8001/groupCache/scores/Tom")
			if err != nil {
				log.Println("err", err)
				return
			}
			fmt.Println(get.Body)
		}
	}
}
