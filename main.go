package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"runtime"
	"unsafe"

	"go-pprof-cpu-heap-example/model"

	_ "net/http/pprof"

	"github.com/mailru/easyjson"
)

func main() {
	http.HandleFunc("/foo", foo)
	http.ListenAndServe("localhost:6060", nil)
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
	ch chan *model.FooReq
}

func (p *FooReqFreeList) Get() *model.FooReq {
	select {
	case b := <-p.ch:
		return b
	default:
		fooReq := model.FooReq(make([]model.FooItem, 0, 100))
		return &fooReq
	}
}

func (p *FooReqFreeList) Put(fooReq *model.FooReq) {
	fooReqSlace := (*fooReq)[:0]
	select {
	case p.ch <- &fooReqSlace: // ok
	default: // drop
	}
}

func NewFooReqFreeList(max int) *FooReqFreeList {
	c := make(chan *model.FooReq, max)
	for i := 0; i < max; i++ {
		fooReq := model.FooReq(make([]model.FooItem, 0, 100))
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
	if err := easyjson.Unmarshal(buf.Bytes(), fooReq); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	hashes := make([]string, 0, len(*fooReq))
	var sha256Buf [sha256.Size]byte
	sha := sha256.New()
	encodedLen := base64.StdEncoding.EncodedLen(sha256.Size)
	buf.Reset()
	buf.Grow(encodedLen)
	for i := 0; i < encodedLen; i++ {
		buf.WriteByte(0)
	}
	for _, foo := range *fooReq {
		sha.Reset()
		sha.Write(stringToBytes(&foo.StrA))
		sha.Write(stringToBytes(&foo.StrB))
		base64.StdEncoding.Encode(buf.Bytes(), sha.Sum(sha256Buf[:0]))
		hashes = append(hashes, buf.String())
	}

	fooRes := model.FooRes{Hashes: hashes}

	b, err := easyjson.Marshal(fooRes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(b)
}

func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func stringToBytes(s *string) []byte {
	return *(*[]byte)(unsafe.Pointer(s))
}
