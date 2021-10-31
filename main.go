package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"sync"

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

var bufPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 1024*1024))
	},
}

func foo(w http.ResponseWriter, r *http.Request) {
	buf := bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufPool.Put(buf)
	}()
	_, err := io.Copy(buf, r.Body)
	r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var fooReq FooReq
	if err := json.Unmarshal(buf.Bytes(), &fooReq); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var hashes []string
	for _, foo := range fooReq {
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
