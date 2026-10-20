package main

import (
	"strings"
	"testing"
)

func TestDBMLProjectionKeepsCompositeForeignKeys(t *testing.T) {
	s := schema{
		Columns: []column{
			{Table:"accounts",Name:"id",Type:"bigint",NotNull:true,Primary:true},
			{Table:"accounts",Name:"created_at",Type:"timestamp with time zone",NotNull:true},
			{Table:"children",Name:"account_id",Type:"bigint",NotNull:true},
			{Table:"children",Name:"child_id",Type:"bigint",NotNull:true},
		},
		References: []reference{{
			Name:"children_account_fkey",FromTable:"children",ToTable:"accounts",
			FromColumns:[]string{"account_id","child_id"},ToColumns:[]string{"id","id"},
		}},
		Indexes: []definition{{Table:"children",Name:"one_open_uq",SQL:"CREATE UNIQUE INDEX one_open_uq ON children (account_id) WHERE ended_at IS NULL"}},
	}
	d := string(renderDBML(s))
	for _, expected := range []string{
		"Table accounts {", "id bigint [pk, not null]", "created_at timestamptz [not null]",
		"Ref: {children.account_id, children.child_id} > {accounts.id, accounts.id}",
	} {
		if !strings.Contains(d,expected) { t.Fatalf("DBML missing %q:\n%s",expected,d) }
	}
	if strings.Contains(d,"CREATE UNIQUE INDEX") { t.Fatal("DBML must not misrepresent partial-index DDL") }
	md:=string(renderDetails(s))
	if !strings.Contains(md,"WHERE ended_at IS NULL") {
		t.Fatal("partial unique predicate must remain visible in schema details")
	}
}

func TestDBMLNameAndTypeSafety(t *testing.T) {
	for input, expected := range map[string]string{
		"bigint":"bigint",
		"character varying(20)":"varchar(20)",
		"timestamp with time zone":"timestamptz",
		"double precision":"float8",
	}{
		if got:=dbmlType(input);got!=expected { t.Fatalf("%q: got %q, want %q",input,got,expected) }
	}
	if got:=name("hello world");got!="\"hello world\"" {t.Fatalf("quoted identifier: %q",got)}
	if got:=name("organizations");got!="organizations" {t.Fatalf("simple identifier: %q",got)}
}
