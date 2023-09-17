package requestresponsestruct

import "net/http"

type Request struct {
	Scheme     string              `bson:"scheme"`
	Method     string              `bson:"method"`
	Host       string              `bson:"host"`
	Path       string              `bson:"path"`
	GetParams  map[string][]string `bson:"get_params"`
	Headers    http.Header         `bson:"headers"`
	Cookies    []http.Cookie       `bson:"cookies"`
	PostParams map[string][]string `bson:"post_params"`
}

type Response struct {
	Code    int         `bson:"code"`
	Message string      `bson:"message"`
	Headers http.Header `bson:"headers"`
	Body    string      `bson:"body"`
}
