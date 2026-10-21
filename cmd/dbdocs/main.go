// Command dbdocs projects the schema installed by PostgreSQL migrations.
// It neither applies SQL nor treats conceptual business DBML as executable DDL.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

type column struct {
	Table, Name, Type string
	NotNull, Primary  bool
}
type reference struct {
	Name, FromTable, ToTable string
	FromColumns, ToColumns   []string
}
type definition struct {
	Table, Name, SQL string
}
type schema struct {
	Columns                                   []column
	References                                []reference
	Constraints, Indexes, Triggers, Functions []definition
}

func main() {
	out := flag.String("out", "", "directory for generated documentation")
	dbmlFile := flag.String("dbml", "implemented.dbml", "DBML output filename (basename only)")
	flag.Parse()
	dsn := os.Getenv("DB_DOCS_DATABASE_URL")
	if *out == "" || dsn == "" || filepath.Base(*dbmlFile) != *dbmlFile || *dbmlFile == "." || *dbmlFile == "" {
		fmt.Fprintln(os.Stderr, "set DB_DOCS_DATABASE_URL and -out (read-only schema projection)")
		os.Exit(2)
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dbdocs: unable to connect to PostgreSQL")
		os.Exit(1)
	}
	defer conn.Close(ctx)
	doc, err := inspect(ctx, conn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dbdocs: schema inspection failed:", err)
		os.Exit(1)
	}
	if len(doc.Columns) == 0 {
		fmt.Fprintln(os.Stderr, "dbdocs: no implemented tables; run migrations first")
		os.Exit(1)
	}
	if err := os.MkdirAll(*out, 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for file, data := range map[string][]byte{
		*dbmlFile:         renderDBML(doc),
		"constraints.md":   renderDetails(doc),
	} {
		if err := os.WriteFile(filepath.Join(*out, file), data, 0644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	fmt.Printf("Generated implemented schema documentation (%d columns)\n", len(doc.Columns))
}

const columnsSQL = `
SELECT c.relname, a.attname, pg_catalog.format_type(a.atttypid, a.atttypmod),
       a.attnotnull,
       EXISTS (SELECT 1 FROM pg_constraint k
               WHERE k.conrelid=c.oid AND k.contype='p' AND a.attnum=ANY(k.conkey))
FROM pg_class c
JOIN pg_namespace n ON n.oid=c.relnamespace
JOIN pg_attribute a ON a.attrelid=c.oid AND a.attnum>0 AND NOT a.attisdropped
WHERE n.nspname='public' AND c.relkind IN ('r','p') AND c.relname <> 'schema_migrations'
ORDER BY c.relname, a.attnum
`

const referencesSQL = `
SELECT src.relname, dst.relname, k.conname, sa.attname, da.attname
FROM pg_constraint k
JOIN pg_class src ON src.oid=k.conrelid
JOIN pg_namespace n ON n.oid=src.relnamespace
JOIN pg_class dst ON dst.oid=k.confrelid
JOIN LATERAL unnest(k.conkey, k.confkey)
     WITH ORDINALITY AS attrs(source_att, target_att, ord) ON true
JOIN pg_attribute sa ON sa.attrelid=src.oid AND sa.attnum=attrs.source_att
JOIN pg_attribute da ON da.attrelid=dst.oid AND da.attnum=attrs.target_att
WHERE n.nspname='public' AND k.contype='f'
ORDER BY src.relname, k.conname, attrs.ord
`

func inspect(ctx context.Context, conn *pgx.Conn) (schema, error) {
	var s schema
	rows, err := conn.Query(ctx, columnsSQL)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var c column
		if err := rows.Scan(&c.Table, &c.Name, &c.Type, &c.NotNull, &c.Primary); err != nil {
			rows.Close()
			return s, err
		}
		s.Columns = append(s.Columns, c)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return s, err
	}
	rows.Close()

	rows, err = conn.Query(ctx, referencesSQL)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var from, to, name, fc, tc string
		if err := rows.Scan(&from, &to, &name, &fc, &tc); err != nil {
			rows.Close()
			return s, err
		}
		n := len(s.References)
		if n == 0 || s.References[n-1].Name != name || s.References[n-1].FromTable != from {
			s.References = append(s.References, reference{Name: name, FromTable: from, ToTable: to})
			n++
		}
		s.References[n-1].FromColumns = append(s.References[n-1].FromColumns, fc)
		s.References[n-1].ToColumns = append(s.References[n-1].ToColumns, tc)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return s, err
	}
	rows.Close()

	queries := []struct {
		sql    string
		target *[]definition
	}{
		{`SELECT rel.relname, k.conname, pg_get_constraintdef(k.oid, true)
FROM pg_constraint k JOIN pg_class rel ON rel.oid=k.conrelid
JOIN pg_namespace n ON n.oid=rel.relnamespace
WHERE n.nspname='public' AND rel.relname <> 'schema_migrations'
ORDER BY rel.relname, k.conname`, &s.Constraints},
		{`SELECT tablename, indexname, indexdef FROM pg_indexes
WHERE schemaname='public' AND tablename <> 'schema_migrations'
ORDER BY tablename, indexname`, &s.Indexes},
		{`SELECT rel.relname, t.tgname, pg_get_triggerdef(t.oid, true)
FROM pg_trigger t JOIN pg_class rel ON rel.oid=t.tgrelid
JOIN pg_namespace n ON n.oid=rel.relnamespace
WHERE n.nspname='public' AND NOT t.tgisinternal
ORDER BY rel.relname, t.tgname`, &s.Triggers},
		{`SELECT 'public', p.proname, pg_get_functiondef(p.oid)
FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
WHERE n.nspname='public' AND p.prokind='f'
ORDER BY p.proname, pg_get_function_identity_arguments(p.oid)`, &s.Functions},
	}
	for _, q := range queries {
		rows, err := conn.Query(ctx, q.sql)
		if err != nil {
			return s, err
		}
		for rows.Next() {
			var item definition
			if err := rows.Scan(&item.Table, &item.Name, &item.SQL); err != nil {
				rows.Close()
				return s, err
			}
			*q.target = append(*q.target, item)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return s, err
		}
		rows.Close()
	}
	return s, nil
}

var simpleName = regexp.MustCompile(`^[a-z_][a-z_0-9]*$`)

func name(v string) string {
	if simpleName.MatchString(v) {
		return v
	}
	return "\"" + strings.ReplaceAll(v, "\"", "\"\"") + "\""
}

func dbmlType(t string) string {
	t = strings.Replace(t, "timestamp with time zone", "timestamptz", 1)
	t = strings.Replace(t, "timestamp without time zone", "timestamp", 1)
	t = strings.Replace(t, "time with time zone", "timetz", 1)
	t = strings.Replace(t, "time without time zone", "time", 1)
	t = strings.Replace(t, "character varying", "varchar", 1)
	t = strings.Replace(t, "double precision", "float8", 1)
	t = strings.Replace(t, "bit varying", "varbit", 1)
	return strings.ReplaceAll(t, " ", "_")
}

func renderDBML(s schema) []byte {
	var out bytes.Buffer
	out.WriteString("// GENERATED from migrated PostgreSQL public schema; DO NOT EDIT.\n")
	out.WriteString("// Migrations, not this diagram, are the executable source of truth.\n")
	out.WriteString("// SQL indexes, constraints, functions and triggers: constraints.md and schema.sql.\n")
	out.WriteString("Project bdspro_implemented {\n  database_type: 'PostgreSQL'\n}\n\n")
	last := ""
	for _, c := range s.Columns {
		if c.Table != last {
			if last != "" {
				out.WriteString("}\n\n")
			}
			fmt.Fprintf(&out, "Table %s {\n", name(c.Table))
			last = c.Table
		}
		opts := []string{}
		if c.Primary {
			opts = append(opts, "pk")
		}
		if c.NotNull {
			opts = append(opts, "not null")
		}
		fmt.Fprintf(&out, "  %s %s", name(c.Name), dbmlType(c.Type))
		if len(opts) > 0 {
			fmt.Fprintf(&out, " [%s]", strings.Join(opts, ", "))
		}
		out.WriteByte('\n')
	}
	if last != "" {
		out.WriteString("}\n\n")
	}
	for _, r := range s.References {
		l, rr := make([]string, len(r.FromColumns)), make([]string, len(r.ToColumns))
		for i := range l {
			l[i] = name(r.FromTable) + "." + name(r.FromColumns[i])
			rr[i] = name(r.ToTable) + "." + name(r.ToColumns[i])
		}
		if len(l) == 1 {
			fmt.Fprintf(&out, "Ref: %s > %s\n", l[0], rr[0])
		} else {
			fmt.Fprintf(&out, "Ref: {%s} > {%s}\n", strings.Join(l, ", "), strings.Join(rr, ", "))
		}
	}
	return out.Bytes()
}

func renderDetails(s schema) []byte {
	var out bytes.Buffer
	out.WriteString("# Implemented PostgreSQL schema details\n\n")
	out.WriteString("Generated from a migrated database; this document is read-only.\n")
	out.WriteString("See `implemented.dbml` for tables/FKs and `schema.sql` for exact DDL.\n")
	for _, group := range []struct {
		title string
		items []definition
	}{
		{"Constraints", s.Constraints},
		{"Indexes (including partial and expression indexes)", s.Indexes},
		{"Triggers", s.Triggers},
		{"Functions", s.Functions},
	} {
		fmt.Fprintf(&out, "\n## %s\n", group.title)
		for _, d := range group.items {
			fmt.Fprintf(&out, "\n### %s.%s\n\n```sql\n%s\n```\n", d.Table, d.Name, d.SQL)
		}
	}
	return out.Bytes()
}
