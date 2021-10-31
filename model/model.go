package model

//easyjson:json
type FooItem struct {
	StrA string `json:"srt_a"`
	StrB string `json:"str_b"`
}

//easyjson:json
type FooReq []FooItem

//easyjson:json
type FooRes struct {
	Hashes []string `json:"hashes"`
}
