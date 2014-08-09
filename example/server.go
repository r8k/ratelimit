package main

import "fmt"
import "net"
import "net/http"
import "github.com/gorilla/mux"
import "github.com/r8k/ratelimit"

var (
	store *ratelimit.RedisLimiter
	err   error
)

func redis_ping(w http.ResponseWriter, r *http.Request) {
	Limit, err := store.Get("client_ip")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		panic(err)
	} else {
		w.WriteHeader(http.StatusOK)
		fmt.Println(*Limit)
		fmt.Fprintf(w, "Hello World!")
	}
}

func main() {
	store, err = ratelimit.Init(&net.TCPAddr{Port: 6379})
	if err != nil {
		panic(err)
	}
	defer store.Close()

	mux := mux.NewRouter()
	mux.HandleFunc("/", redis_ping)

	http.ListenAndServe(":8080", mux)
}
