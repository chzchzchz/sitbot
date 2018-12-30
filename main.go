package main

import (
	"log"
	"os"
)

func main() {
	laddr := "localhost:9991"
	if len(os.Args) > 1 {
		laddr = os.Args[1]
	}
	os.Setenv("SITBOT_URL", "http://"+laddr)
	log.Println("serving bot on", laddr)
	if err := ServeHttp(NewGang(), laddr); err != nil {
		panic(err)
	}
}
