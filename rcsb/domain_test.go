package rcsb

import (
	"testing"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "rcsb" {
		t.Errorf("Scheme = %q, want rcsb", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "rcsb" {
		t.Errorf("Identity.Binary = %q, want rcsb", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	typ, id, err := Domain{}.Classify("4HHB")
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}
	if typ != "entry" {
		t.Errorf("type = %q, want entry", typ)
	}
	if id != "4HHB" {
		t.Errorf("id = %q, want 4HHB", id)
	}
}

func TestClassifyLowerCase(t *testing.T) {
	typ, id, err := Domain{}.Classify("4hhb")
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}
	if id != "4HHB" {
		t.Errorf("id = %q, want 4HHB (uppercased)", id)
	}
	_ = typ
}

func TestClassifyInvalid(t *testing.T) {
	_, _, err := Domain{}.Classify("too-long-id")
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("entry", "4HHB")
	if err != nil {
		t.Fatalf("Locate error: %v", err)
	}
	if got != "https://www.rcsb.org/structure/4HHB" {
		t.Errorf("Locate = %q, want https://www.rcsb.org/structure/4HHB", got)
	}
}
