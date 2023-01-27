package main

import (
	"fmt"

	. "github.com/chandumlg/hello/internal"
)

func main() {
	testHi := &Hi{}
	fmt.Println(testHi.Hello())
}
