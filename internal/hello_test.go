package internal

import (
	"testing"
)

func TestHello(t *testing.T) {
	want := "Hello, world."
	testHi := &Hi{}
	if got := testHi.Hello(); got != want {
		t.Errorf("Hello() = %q, want %q", got, want)
	}
}

func TestProverb(t *testing.T) {
	want := "Concurrency is not parallelism."
	if got := Proverb(); got != want {
		t.Errorf("Proverb() = %q, want %q", got, want)
	}
}
