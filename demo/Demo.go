package main

import (
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/verzth/jumper"
	"net/http"
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", index).Methods("GET")

	err := http.ListenAndServe(":9999", handlers.CORS(
		handlers.AllowedHeaders([]string{"Content-Type","Authorization"}),
		handlers.AllowedMethods([]string{http.MethodGet}),
		handlers.AllowedOrigins([]string{"*"}),
	)(r))
	if err != nil {
		panic(err)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
	var req = jumper.PlugRequest(r, w)
	var res = jumper.PlugResponse(w)

	vn := req.GetMap("list")["obj"]
	fmt.Println(vn.(map[string]interface{})["id"].([]interface{})[0])

	res.ReplySuccess("0000000", "SSSSSS", "Success", nil)
}