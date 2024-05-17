package main

import (
	"errors"
	"log"
	"sync"
)

// 接口型函数 :定义一个函数类型 F，并且实现接口 A 的方法，然后在这个方法中调用自己。
//这是 Go 语言中将其他函数（参数返回值定义与 F 一致）转换为接口 A 的常用技巧。

// Getter  这样设计的好处是适合复杂的场景，将封装后的结构体作为参数传递
// 例如在数据库操作中，会有很多中间状态需要保持，比如超时重连、加锁等等
type Getter interface {
	Get(key string) ([]byte, error)
}

// GetterFunc 的作用就是简化操作，让普通函数也可以实现Get接口
type GetterFunc func(key string) ([]byte, error)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

// Group Group模块是对外提供服务接⼝的部分，其要实现对缓存的增删查⽅法
type Group struct {
	name      string
	getter    Getter
	mainCache cache
	peers     PeerPicker
}

var (
	mu    sync.RWMutex
	group = make(map[string]*Group)
)

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	mu.Lock()
	defer mu.Unlock()
	if getter == nil {
		panic("error getter  can not nil")
	}
	g := &Group{
		name:      name,
		mainCache: cache{cacheBytes: cacheBytes},
		getter:    getter,
	}
	group[name] = g
	return g
}

func GetGroup(name string) *Group {
	mu.RLocker()
	if g, ok := group[name]; ok {
		return g
	}
	mu.RUnlock()
	return nil
}

// Get 优先查询在本地cache获取，如果获取不到，则去getter中获取原数据
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, errors.New("key is required")
	}
	if bytesView, ok := g.mainCache.get(key); ok {
		log.Println("[GroupCache] hit")
		return bytesView, nil
	}
	return g.load(key)
}

// 加载当地的原数据
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

func (g *Group) populateCache(key string, bytesView ByteView) {
	g.mainCache.add(key, bytesView, 0)
}

// RegisterPeers registers a PeerPicker for choosing remote peer
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

func (g *Group) load(key string) (value ByteView, err error) {
	if g.peers != nil {
		if peer, ok := g.peers.PickPeer(key); ok {
			if value, err = g.getFromPeer(peer, key); err == nil {
				return value, nil
			}
			log.Println("[GeeCache] Failed to get from peer", err)
		}
	}

	return g.getLocally(key)
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}
