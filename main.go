package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"runtime"

	_ "net/http/pprof"
)

func main() {
	http.HandleFunc("/foo", foo)
	http.ListenAndServe("localhost:6060", nil)
}

type FooItem struct {
	StrA string `json:"srt_a"`
	StrB string `json:"str_b"`
}

type FooReq []FooItem

type FooRes struct {
	Hashes []string `json:"hashes"`
}

type BufFreeList struct {
	ch chan *bytes.Buffer
}

func (p *BufFreeList) Get() *bytes.Buffer {
	select {
	case b := <-p.ch:
		return b
	default:
		return bytes.NewBuffer(make([]byte, 0, 1024*1024))
	}
}

func (p *BufFreeList) Put(b *bytes.Buffer) {
	select {
	case p.ch <- b: // ok
	default: // drop
	}
}

func NewBufFreeList(max int) *BufFreeList {
	c := make(chan *bytes.Buffer, max)
	for i := 0; i < max; i++ {
		c <- bytes.NewBuffer(make([]byte, 0, 1024*1024))
	}
	return &BufFreeList{ch: c}
}

var bufFreeList = NewBufFreeList(runtime.NumCPU())

type FooReqFreeList struct {
	ch chan *FooReq
}

func (p *FooReqFreeList) Get() *FooReq {
	select {
	case b := <-p.ch:
		return b
	default:
		fooReq := FooReq(make([]FooItem, 0, 100))
		return &fooReq
	}
}

func (p *FooReqFreeList) Put(fooReq *FooReq) {
	fooReqSlace := (*fooReq)[:0]
	select {
	case p.ch <- &fooReqSlace: // ok
	default: // drop
	}
}

func NewFooReqFreeList(max int) *FooReqFreeList {
	c := make(chan *FooReq, max)
	for i := 0; i < max; i++ {
		fooReq := FooReq(make([]FooItem, 0, 100))
		c <- &fooReq
	}
	return &FooReqFreeList{ch: c}
}

var fooReqFreeList = NewFooReqFreeList(runtime.NumCPU())

func foo(w http.ResponseWriter, r *http.Request) {
	buf := bufFreeList.Get()
	defer func() {
		buf.Reset()
		bufFreeList.Put(buf)
	}()
	_, err := io.Copy(buf, r.Body)
	r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	fooReq := fooReqFreeList.Get()
	defer func() {
		fooReqFreeList.Put(fooReq)
	}()
	if err := json.Unmarshal(buf.Bytes(), fooReq); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var hashes []string
	for _, foo := range *fooReq {
		sha := sha256.New()
		sha.Write([]byte(foo.StrA))
		sha.Write([]byte(foo.StrB))
		hashes = append(hashes, base64.StdEncoding.EncodeToString(sha.Sum(nil)))
	}
	fooRes := FooRes{Hashes: hashes}

	b, err := json.Marshal(fooRes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(b)
}
