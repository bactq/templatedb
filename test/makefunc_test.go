package test

import (
	"testing"

	"github.com/tianxinzizhen/templatedb"
)

type MTest struct {
	Select func(int, int, int, int)
}

func TestMakefunc(t *testing.T) {
	dest := &MTest{}
	err := templatedb.InitMakeFunc(dest)
	if err != nil {
		t.Error(err)
	}
}
