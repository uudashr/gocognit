[![GoDoc](https://godoc.org/github.com/uudashr/gocognit?status.svg)](https://godoc.org/github.com/uudashr/gocognit)
# Gocognit
Gocognit calculates cognitive complexities of functions in Go source code. A measurement of how hard does the code is intuitively to understand.


## Rules

The cognitive complexity of a function is calculated according to the
following rules:

### Increments
There is an increment for each of the following:
1. `if`, `else if`, `else`
2. `switch`, `select`
3. `for`
4. `goto` LABEL, `break` LABEL, `continue` LABEL
5. sequence of binary logical operators
6. each method in a recursion cycle

### Nesting level
The following structures increment the nesting level:
1. `if`, `else if`, `else`
2. `switch`, `select`
3. `for`
4. function literal or lambda

### Nesting increments
The following structures receive a nesting increment commensurate with their nested depth inside nesting structures:
1. `if`
2. `switch`, `select`
3. `for`

## Installation
```
$ go get github.com/uudashr/gocognit/cmd/gocognit
```

## Usage

```
$ gocognit
Calculate cognitive complexities of Go functions.
Usage:
        gocognitive [flags] <Go file or directory> ...
Flags:
        -over N   show functions with complexity > N only and
                  return exit code 1 if the set is non-empty
        -top N    show the top N most complex functions only
        -avg      show the average complexity over all functions,
                  not depending on whether -over or -top are set
The output fields for each line are:
<complexity> <package> <function> <file:row:column>
```

Examples:

```
$ gocognit .
$ gocognit main.go
$ gocognit -top 10 src/
$ gocognit -over 25 docker
$ gocognit -avg .
```

The output fields for each line are:
```
<complexity> <package> <function> <file:row:column>
```

## Related project
- [Gocyclo](https://github.com/fzipp/gocyclo) where the code are based on.
- [Cognitive Complexity: A new way of measuring understandability](https://www.sonarsource.com/docs/CognitiveComplexity.pdf) white paper by G. Ann Campbell