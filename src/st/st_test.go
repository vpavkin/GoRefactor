package st

import (
	"testing"

)

func TestLookUp(t *testing.T) {

	s := NewSymbolTable(nil)
	vsym := MakeVariable("vs",nil,nil,false)
	s.Table.Push(vsym)
	if r, ok := s.LookUp("vs", ""); !ok || r != vsym {
		t.Fatalf("LookUp failed1")
	}
	s.Table.Push(MakeVariable("aaa",nil,nil,false))
	s.Table.Insert(0, MakeVariable("vs",nil,nil,false))

	if r, ok := s.LookUp("vs", ""); !ok || r != vsym {
		t.Fatalf("LookUp failed2")
	}

	ss := NewSymbolTable(nil)
	vvsym := MakeVariable("vvs",nil,nil,false)
	ss.Table.Push(vvsym)
	s.AddOpenedScope(ss)

	if r, ok := s.LookUp("vvs", ""); !ok || r != vvsym {
		t.Fatalf("LookUp in OpenedScope failed")
	}

	sss := NewSymbolTable(nil)
	vvvsym := MakeVariable("vvvs",nil,nil,false)
	sss.Table.Push(vvvsym)
	ss.AddOpenedScope(sss)
	if r, ok := s.LookUp("vvvs", ""); !ok || r != vvvsym {
		t.Fatalf("LookUp in OpenedScope failed")
	}
}

func TestAddSymbol(t *testing.T) {
	s := NewSymbolTable(nil)
	vsym := MakeVariable("vs",nil,nil,false)

	s.AddSymbol(vsym)
	if r, ok := s.LookUp("vs", ""); !ok || r != vsym {
		t.Fatalf("AddSymbol failed")
	}

}
func TestReplaceSymbol(t *testing.T) {
	s := NewSymbolTable(nil)
	vsym := MakeVariable("vs",nil,nil,false)
	s.AddSymbol(vsym)
	vvsym := MakeVariable("vvs",nil,nil,false)
	s.AddSymbol(vvsym)
	aaa := MakeVariable("aaa",nil,nil,false)
	s.ReplaceSymbol("vvs", aaa)

	if r, ok := s.LookUp("vs", ""); !ok || r != vsym {
		t.Fatalf("ReplaceSymbol failed1")
	}
	if _, ok := s.LookUp("vvs", ""); ok {
		t.Fatalf("ReplaceSymbol failed2")
	}
	if r, ok := s.LookUp("aaa", ""); !ok || r != aaa {
		t.Fatalf("ReplaceSymbol failed3")
	}
}

func TestRemoveSymbol(t *testing.T) {

	s := NewSymbolTable(nil)
	vsym := MakeVariable("vs",nil,nil,false)
	s.AddSymbol(vsym)
	vvsym :=MakeVariable("vvs",nil,nil,false)
	s.AddSymbol(vvsym)
	aaa :=MakeVariable("aaa",nil,nil,false)
	s.AddSymbol(aaa)
	s.RemoveSymbol("aaa")

	if r, _ := s.LookUp("aaa", ""); r == aaa {
		t.Fatalf("RemoveSymbol failed1")
	}
}

func TestLookUpPointerType(t *testing.T) {
	s := NewSymbolTable(nil)
	tsym := MakeArrayType("ts",nil,nil,0)
	s.AddSymbol(tsym)
	ptsym := MakePointerType(nil, tsym)
	s.AddSymbol(ptsym)
	pptsym := MakePointerType(nil, ptsym)
	s.AddSymbol(pptsym)
	if r, ok := s.LookUpPointerType("ts", 1); !ok || r != ptsym {
		t.Fatalf("LookUpPT failed1")
	}
	if r, ok := s.LookUpPointerType("ts", 2); !ok || r != pptsym {
		t.Fatalf("LookUpPT failed1")
	}

	ss := NewSymbolTable(nil)
	pppsym := MakePointerType(nil, pptsym)
	ss.Table.Push(pppsym)
	s.AddOpenedScope(ss)

	if r, ok := s.LookUpPointerType("ts", 3); !ok || r != pppsym {
		t.Fatalf("LookUpPointerType in OpenedScope failed ")
	}
}

func TestGetBaseType(t *testing.T) {

	tsym := MakeArrayType("ts",nil,nil,0)
	ptsym := MakePointerType(nil, tsym)
	pptsym := MakePointerType(nil, ptsym)

	if r, cyc := GetBaseType(tsym); cyc || r != tsym {
		t.Fatalf("GetBaseType failed1")
	}
	if r, cyc := GetBaseType(ptsym); cyc || r != tsym {
		t.Fatalf("GetBaseType failed2")
	}
	if r, cyc := GetBaseType(pptsym); cyc || r != tsym {
		t.Fatalf("GetBaseType failed3")
	}
	p := MakePointerType(nil, nil)
	asym := MakeAliasType("all",nil, p)
	p.BaseType = asym

	if _, cyc := GetBaseType(p); !cyc {
		t.Fatalf("GetBaseType failed4")
	}
	if _, cyc := GetBaseType(asym); !cyc {
		t.Fatalf("GetBaseType failed5")
	}

}
