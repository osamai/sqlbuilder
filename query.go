package sqlbuilder

import (
	"reflect"
	"strconv"
	"strings"
)

// Query describes an sql query.
type Query struct {
	str    *strings.Builder
	args   []interface{}
	table  string
	driver string
}

// NewQuery returns new Query with table.
func NewQuery(table string) *Query {
	return &Query{
		str:    &strings.Builder{},
		table:  table,
		driver: "pg",
	}
}

// SetDriver sets driver field to the given value.
// SetDriver panics if driver is not supported.
func (q *Query) SetDriver(driver string) {
	switch d := strings.ToLower(driver); d {
	case "pg", "postgres", "postgresql":
		q.driver = "pg"
	case "mysql":
		q.driver = d
	default:
		panic("sqlbuilder.SetDriver: unsupported driver: " + driver)
	}
}

// Reset resets query string and arguments.
func (q *Query) Reset() {
	q.str.Reset()
	q.args = nil
}

// String returns query string.
func (q *Query) String() string {
	return q.str.String()
}

// Args returns query arguments.
func (q *Query) Args() []interface{} {
	return q.args
}

// Table returns table name.
func (q *Query) Table() string {
	return q.table
}

// SetTable sets table field and calls Reset.
func (q *Query) SetTable(table string) {
	q.Reset()
	q.table = table
}

func (q *Query) addColumns(columns ...string) {
	for i, c := range columns {
		q.str.WriteString(c)
		if i != len(columns)-1 {
			q.str.WriteByte(',')
		}
	}
}

func (q *Query) addArg(arg interface{}) {
	q.args = append(q.args, arg)
	switch q.driver {
	case "pg":
		q.str.WriteString("$" + strconv.Itoa(len(q.args)))
	case "mysql":
		q.str.WriteByte('?')
	}
}

// Statement returns Statement instance from query.
func (q *Query) Statement() *Statement {
	return &Statement{q}
}

// Select returns sql select statement.
func (q *Query) Select(columns ...string) *Statement {
	q.Reset()
	q.str.WriteString("SELECT ")
	if columns != nil {
		q.addColumns(columns...)
	} else {
		q.str.WriteByte('*')
	}
	q.str.WriteString(" FROM " + q.table)
	return q.Statement()
}

// Insert returns sql insert statement.
func (q *Query) Insert(columns []string, values ...interface{}) *Statement {
	q.Reset()
	q.str.WriteString("INSERT INTO " + q.table + " (")
	q.addColumns(columns...)
	q.str.WriteString(") VALUES (")

	v := reflect.ValueOf(values[0])
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		for i, vs := range values {
			if i != 0 {
				v = reflect.ValueOf(vs)
				if v.Kind() == reflect.Ptr {
					v = v.Elem()
				}
				q.str.WriteByte('(')
			}

			for j := 0; j < v.Len(); j++ {
				q.addArg(v.Index(j).Interface())
				if j != v.Len()-1 {
					q.str.WriteByte(',')
				}
			}
			q.str.WriteByte(')')
			if i != len(values)-1 {
				q.str.WriteByte(',')
			}
		}
		return q.Statement()
	}

	for i, vs := range values {
		q.addArg(vs)
		if i != len(values)-1 {
			q.str.WriteByte(',')
		}
	}
	q.str.WriteByte(')')
	return q.Statement()
}

// Update returns sql update statement.
// data type can be string or map[string]interface{}.
// args is only used if data is a string.
func (q *Query) Update(data interface{}, args ...interface{}) *Statement {
	q.Reset()
	q.str.WriteString("UPDATE " + q.table + " SET ")

	switch d := data.(type) {
	case string:
		q.Raw(d, args...)
	case map[string]interface{}:
		i := len(d) - 1
		for k, v := range d {
			q.str.WriteString(k + "=")
			q.addArg(v)
			if i != 0 {
				q.str.WriteByte(',')
			}
			i--
		}
	default:
		panic("sqlbuilder.Update: unexpected data type")
	}

	return q.Statement()
}

// Delete returns sql delete statement.
func (q *Query) Delete() *Statement {
	q.Reset()
	q.str.WriteString("DELETE FROM " + q.table)
	return q.Statement()
}

// Raw wirtes raw string to query and appends args to query arguments.
func (q *Query) Raw(str string, args ...interface{}) *Query {
	if q.driver == "pg" {
		idx := strings.IndexByte(str, '?')
		if idx != -1 {
			var i, last int
			for idx != -1 && i < len(args) {
				q.str.WriteString(str[last : last+idx])
				q.args = append(q.args, args[i])
				q.str.WriteByte('$')
				q.str.WriteString(strconv.Itoa(len(q.args)))

				i++
				last += idx + 1
				idx = strings.IndexByte(str[last:], '?')
			}
			if len(str) > last {
				q.str.WriteString(str[last:])
			}
			return q
		}
	}

	q.str.WriteString(str)
	if args != nil {
		q.args = append(q.args, args...)
	}
	return q
}

// RawByte writes byte to query.
func (q *Query) RawByte(b byte) *Query {
	q.str.WriteByte(b)
	return q
}
