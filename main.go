package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"

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

func foo(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var fooReq FooReq
	if err := json.Unmarshal(b, &fooReq); err != nil {
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

	b, err = json.Marshal(fooRes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(b)
}
