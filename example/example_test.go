/*
 * Copyright (C) distroy
 */

package example

import (
	"go/parser"
	"go/token"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/distroy/gocognit"
)

func getFileLine() (string, int) {
	_, file, line, _ := runtime.Caller(1)
	return file, line
}

func complexityFromFuncName(funcName string) int {
	var pos int
	for i := len(funcName) - 1; i >= 0; i-- {
		b := funcName[i]
		if b >= '0' && b <= '9' {
			continue
		}

		pos = i + 1
		break
	}
	str := funcName[pos:]
	n, _ := strconv.Atoi(str)
	return n
}

func analyzeFile(t testing.TB, file string) []gocognit.Stat {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatalf("parse file fail. err:%s", err.Error())
	}

	return gocognit.ComplexityStats(f, fset, nil)
}

func TestExample(t *testing.T) {
	file, _ := getFileLine()
	file = strings.ReplaceAll(file, "_test.go", ".go")

	stats := analyzeFile(t, file)
	for _, stat := range stats {
		want := complexityFromFuncName(stat.FuncName)
		if stat.Complexity == want {
			t.Logf("check func complexity succ. func:%s, complexity:%d, file:%s",
				stat.FuncName, stat.Complexity, stat.BeginPos.Filename)
		} else {
			t.Errorf("check func complexity fail. func:%s, complexity:%d, want:%d, file:%s",
				stat.FuncName, stat.Complexity, want, stat.BeginPos.Filename)
		}
	}
}
