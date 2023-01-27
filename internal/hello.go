package internal

import "rsc.io/quote/v3"

type Helloer interface {
	Hello() string
}

type Hi struct {
	test int
}

func (*Hi) Hello() string {
	return quote.HelloV3()
}

func Proverb() string {
	return quote.Concurrency()
}
