package main

import (
	"testing"

	"github.com/ankurkotwal/MetaRefCard/metarefcard"
)

func TestMain(m *testing.M) {
	metarefcard.RunLocal()
}

func TestFS2020(t *testing.T) {
	metarefcard.RunLocal()
}
func BenchmarkFS2020(b *testing.B) {
	b.Logf("BenchmarkFS2020")
	for n := 0; n < b.N; n++ {
		metarefcard.RunLocal()
	}
}
