package main

import "st"
import "testing"
import __regexp__ "regexp"

var tests = []testing.InternalTest{
	{"st.TestLookUp", st.TestLookUp},
	{"st.TestAddSymbol", st.TestAddSymbol},
	{"st.TestReplaceSymbol", st.TestReplaceSymbol},
	{"st.TestRemoveSymbol", st.TestRemoveSymbol},
	{"st.TestLookUpPointerType", st.TestLookUpPointerType},
	{"st.TestGetBaseType", st.TestGetBaseType},
}
var benchmarks = []testing.InternalBenchmark{ //
}

func main() {
	testing.Main(__regexp__.MatchString, tests)
	testing.RunBenchmarks(__regexp__.MatchString, benchmarks)
}
