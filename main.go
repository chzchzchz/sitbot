package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"
)

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	log.Println(os.Args)
	jsonb, err := ioutil.ReadFile(os.Args[1])
	panicErr(err)
	p, err := UnmarshalProfile(jsonb)
	panicErr(err)
	b, err := NewBot(context.TODO(), *p)
	panicErr(err)
	<-b.Done()
	b.Close()
}
