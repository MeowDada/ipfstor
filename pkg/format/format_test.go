package format

import (
	"fmt"
	"testing"
)

func TestBasic(t *testing.T) {
	b := basic{}

	rows := []Row{
		{
			{"Name", "Jack"},
			{"Company", "SNOW"},
			{"Phone", "8869751230"},
		},
		{
			{"Name", "Henry"},
			{"Company", "INTC"},
			{"Phone", "8869751230"},
		},
		{
			{"Name", "Tom"},
			{"Company", "AMZN"},
			{"Phone", "8869751230"},
		},
		{
			{"Name", "Dennis"},
			{"Company", "GOOG"},
			{"Phone", "8869751230777"},
		},
		{
			{"Name", "Joseph"},
			{"Company", "TSMC"},
			{"Phone", "8869751230"},
		},
	}

	data := b.Render(rows, Options{Sort: false})
	fmt.Println(string(data))
}
