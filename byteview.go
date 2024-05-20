package main

import "time"

// ByteView 用于缓存值的结构 ByteView
type ByteView struct {
	b      []byte
	s      string
	expire time.Time
}

func (v ByteView) Len() int {
	return len(v.b)
}

func (v ByteView) String() string {
	return string(v.b)
}

// ByteSlice 返回一个拷贝的字节切片, 防止缓存值被外部程序修改
func (v ByteView) ByteSlice() []byte {
	if v.b != nil {
		return cloneBytes(v.b)
	}
	return []byte(v.s)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
