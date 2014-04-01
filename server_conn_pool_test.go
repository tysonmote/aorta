package main

import (
	"sync"
	"testing"
)

func TestServerConnPool(t *testing.T) {
	pool := NewServerConnPool()
	serverConn := pool.Get("cool.com", "1234", "pw")
	if serverConn.host != "cool.com:1234" {
		t.Errorf("incorrect host for ServerConn: %s", serverConn.host)
	}
	if serverConn.auth != "pw" {
		t.Errorf("incorrect auth for ServerConn: %s", serverConn.auth)
	}

	serverConn2 := pool.Get("cool.com", "1234", "pw")
	if serverConn2 != serverConn {
		t.Errorf("subsequent Get for same server didn't return same ServerConn: %#v", serverConn2)
	}

	serverConn3 := pool.Get("cool.com", "1234", "other")
	if serverConn3 == serverConn {
		t.Errorf("different auth should return different ServerConn, but didn't")
	}
}

func BenchmarkServerConnPool_1(b *testing.B) {
	pool := NewServerConnPool()
	for i := 0; i < b.N; i++ {
		pool.Get("cool.com", "1234", "pw")
	}
}

func BenchmarkServerConnPool_10(b *testing.B) {
	pool := NewServerConnPool()
	servers := [][]string{
		[]string{"cool0.com", "1234", "pw"},
		[]string{"cool1.com", "1234", "pw"},
		[]string{"cool2.com", "1234", "pw"},
		[]string{"cool3.com", "1234", "pw"},
		[]string{"cool4.com", "1234", "pw"},
		[]string{"cool5.com", "1234", "pw"},
		[]string{"cool6.com", "1234", "pw"},
		[]string{"cool7.com", "1234", "pw"},
		[]string{"cool8.com", "1234", "pw"},
		[]string{"cool9.com", "1234", "pw"},
	}
	var deets []string
	for i := 0; i < b.N; i++ {
		deets = servers[i%len(servers)]
		pool.Get(deets[0], deets[1], deets[2])
	}
}

func BenchmarkServerConnPool_2_GoRoutines(b *testing.B) {
	wg := sync.WaitGroup{}
	pool := NewServerConnPool()
	for i := 0; i < b.N; i++ {
		wg.Add(2)
		go func() {
			pool.Get("cool.com", "1234", "pw")
			wg.Done()
		}()
		go func() {
			pool.Get("cool.com", "1234", "pw")
			wg.Done()
		}()
	}

	wg.Wait()
}