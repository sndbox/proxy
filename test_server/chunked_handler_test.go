package main

import (
	"testing"
)

func TestSizeToInt(t *testing.T) {
	check := func(s string, expect int) {
		actual, err := sizeToInt(s)
		if err != nil {
			t.Error(err)
		}
		if actual != expect {
			t.Errorf("got %d, want %d\n", actual, expect)
		}
	}
	check("30", 30)
	check("100k", 100*1000)
	check("6m", 6*1000*1000)

	checkErr := func(s string) {
		_, err := sizeToInt(s)
		if err == nil {
			t.Errorf("%s is invalid, but no error reported", s)
		}
	}
	checkErr("")
	checkErr("1h")
	checkErr("a")
}
