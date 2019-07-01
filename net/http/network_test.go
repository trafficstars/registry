package http

import (
	"testing"
)

func Test_listOfLocalAddresses(t *testing.T) {
	_, err := listOfLocalAddresses()
	if err != nil {
		t.Error(err)
	}
}
