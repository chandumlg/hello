package main

import (
	"fmt"
	"time"

	. "github.com/chandumlg/hello/internal"
)

func main() {
	testHi := &Hi{}
	fmt.Println(testHi.Hello())

	time.Sleep(30 * time.Minute)
}
