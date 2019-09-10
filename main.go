package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"

	_ "github.com/lib/pq" // Postgres

	"github.com/iancoleman/strcase"

	"github.com/jmoiron/sqlx"
)

type colType struct {
	Column   string
	GoColumn string
	GoType   string
}

type tblColType struct {
	Table  string `db:"table_name"`
	Column string `db:"column_name"`
	Type   string `db:"data_type"`
}

var registeredTbls map[string][]colType

var (
	name string
	pass string
	port string
	host string
	user string
	path string
)

const query string = `
select table_name, column_name, data_type
from information_schema."columns"
where table_schema = 'public'`

const tmpl string = `
package models

{{ range $k, $v := . }}
// {{ $k }} model automatically generated
type {{ $k }} struct { {{ range $v }}
    {{ .GoColumn }} {{ .GoType }} ` + "`db:\"{{ .Column }}\" json:\"{{ .Column }}\"`" + `{{ end }}
}
{{ end }}`

func main() {

	flag.StringVar(&name, "dbname", "", "database name")
	flag.StringVar(&pass, "password", "", "database password")
	flag.StringVar(&host, "host", "", "database host")
	flag.StringVar(&port, "port", "", "database port")
	flag.StringVar(&user, "user", "", "database user")
	flag.StringVar(&path, "path", "", "path where models are going to be generated")
	flag.Parse()

	connectionString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, pass, name)
	fmt.Printf("Connection string: %s\n", connectionString)
	conn, err := sqlx.Connect("postgres", connectionString)
	if err != nil {
		panic(err)
	}

	result, err := conn.Queryx(query)
	if err != nil {
		panic(err)
	}

	registeredTbls = make(map[string][]colType)
	for result.Next() {
		var res tblColType
		result.StructScan(&res)

		var goType string
		switch res.Type {
		case "char":
			fallthrough
		case "varchar":
			fallthrough
		case "text":
			fallthrough
		case "character varying":
			goType = "*string"
			break

		case "smallint":
			goType = "*int16"
			break
		case "int":
			goType = "*int32"
			break
		case "bigint":
			fallthrough
		case "integer":
			goType = "*int64"
			break
		case "float":
			fallthrough
		case "real":
			fallthrough
		case "numeric":
			goType = "*float64"
			break

		case "date":
			fallthrough
		case "time":
			fallthrough
		case "timestamp":
			fallthrough
		case "timestampz":
			fallthrough
		case "interval":
			goType = "*time.Time"
			break

		default:
			goType = "interface{}"
		}

		threatedResult := colType{
			GoType:   goType,
			GoColumn: strcase.ToCamel(res.Column),
			Column:   res.Column,
		}

		goTable := strcase.ToCamel(res.Table)
		registeredTbls[goTable] = append(registeredTbls[goTable], threatedResult)
	}

	t, err := template.New("models").Parse(tmpl)
	if err != nil {
		panic(err)
	}

	file, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	t.Execute(file, registeredTbls)
}
