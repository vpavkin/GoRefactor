package st

import (
	"testing"
	"go/ast"
)

func TestLookUp(t *testing.T) {

	s := NewSymbolTable(nil)
	vsym := &VariableSymbol{Obj: &ast.Object{Name: "vs"}}
	s.Table.Push(vsym)
	if r, ok := s.LookUp("vs", ""); !ok || r != vsym {
		t.Fatalf("LookUp failed1")
	}
	s.Table.Push(&VariableSymbol{Obj: &ast.Object{Name: "aaa"}})
	s.Table.Insert(0, &VariableSymbol{Obj: &ast.Object{Name: "vs"}})

	if r, ok := s.LookUp("vs", ""); !ok || r != vsym {
		t.Fatalf("LookUp failed2")
	}

	ss := NewSymbolTable(nil)
	vvsym := &VariableSymbol{Obj: &ast.Object{Name: "vvs"}}
	ss.Table.Push(vvsym)
	s.AddOpenedScope(ss)

	if r, ok := s.LookUp("vvs", ""); !ok || r != vvsym {
		t.Fatalf("LookUp in OpenedScope failed")
	}
}

func TestAddSymbol(t *testing.T) {
	s := NewSymbolTable(nil)
	vsym := &VariableSymbol{Obj: &ast.Object{Name: "vs"}}

	s.AddSymbol(vsym)
	if r, ok := s.LookUp("vs", ""); !ok || r != vsym {
		t.Fatalf("AddSymbol failed")
	}

}
func TestReplaceSymbol(t *testing.T) {
	s := NewSymbolTable(nil)
	vsym := &VariableSymbol{Obj: &ast.Object{Name: "vs"}}
	s.AddSymbol(vsym)
	vvsym := &VariableSymbol{Obj: &ast.Object{Name: "vvs"}}
	s.AddSymbol(vvsym)
	aaa := &VariableSymbol{Obj: &ast.Object{Name: "aaa"}}
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
	vsym := &VariableSymbol{Obj: &ast.Object{Name: "vs"}}
	s.AddSymbol(vsym)
	vvsym := &VariableSymbol{Obj: &ast.Object{Name: "vvs"}}
	s.AddSymbol(vvsym)
	aaa := &VariableSymbol{Obj: &ast.Object{Name: "aaa"}}
	s.AddSymbol(aaa)
	s.RemoveSymbol("aaa")

	if r, _ := s.LookUp("aaa", ""); r == aaa {
		t.Fatalf("RemoveSymbol failed1")
	}
}

func TestLookUpPointerType(t *testing.T) {
	s := NewSymbolTable(nil)
	tsym := &ArrayTypeSymbol{TypeSymbol: &TypeSymbol{Obj: &ast.Object{Name: "ts"}}}
	s.AddSymbol(tsym)
	ptsym := &PointerTypeSymbol{BaseType: tsym}
	s.AddSymbol(ptsym)
	pptsym := &PointerTypeSymbol{BaseType: ptsym}
	s.AddSymbol(pptsym)
	if r, ok := s.LookUpPointerType("ts", 1); !ok || r != ptsym {
		t.Fatalf("LookUpPT failed1")
	}
	if r, ok := s.LookUpPointerType("ts", 2); !ok || r != pptsym {
		t.Fatalf("LookUpPT failed1")
	}

	ss := NewSymbolTable(nil)
	pppsym := &PointerTypeSymbol{BaseType: pptsym}
	ss.Table.Push(pppsym)
	s.AddOpenedScope(ss)

	if r, ok := s.LookUpPointerType("ts", 3); !ok || r != pppsym {
		t.Fatalf("LookUpPointerType in OpenedScope failed ")
	}
}

func TestGetBaseType(t *testing.T) {

	tsym := &ArrayTypeSymbol{TypeSymbol: &TypeSymbol{Obj: &ast.Object{Name: "ts"}}}
	ptsym := &PointerTypeSymbol{BaseType: tsym}
	pptsym := &PointerTypeSymbol{BaseType: ptsym}

	if r, cyc := GetBaseType(tsym); cyc || r != tsym {
		t.Fatalf("GetBaseType failed1")
	}
	if r, cyc := GetBaseType(ptsym); cyc || r != tsym {
		t.Fatalf("GetBaseType failed2")
	}
	if r, cyc := GetBaseType(pptsym); cyc || r != tsym {
		t.Fatalf("GetBaseType failed3")
	}
	p := &PointerTypeSymbol{}
	asym := &AliasTypeSymbol{&TypeSymbol{Obj: &ast.Object{Name: "all"}}, p}
	p.BaseType = asym

	if _, cyc := GetBaseType(p); !cyc {
		t.Fatalf("GetBaseType failed4")
	}
	if _, cyc := GetBaseType(asym); !cyc {
		t.Fatalf("GetBaseType failed5")
	}

}
