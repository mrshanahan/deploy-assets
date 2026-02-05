package util

import (
	"fmt"
	"strings"
)

// Produces a whitespace indentation string for one level of a YAML object,
// which in our case is 4 spaces. Panics if depth < 0.
func YamlIndentString(indent int) string {
	if indent < 0 {
		panic(fmt.Sprintf("indent must be non-negative (was: %d)", indent))
	}
	return strings.Repeat(" ", indent)
}

func TabsToIndent(tabs int) int {
	if tabs < 0 {
		panic(fmt.Sprintf("tabs must be non-negative (was: %d)", tabs))
	}
	return tabs * YamlIndentSize
}

const YamlIndentSize int = 4

// func Yaml(v any) string {
// 	t := reflect.TypeOf(v)
// 	k := t.Kind()
// 	if k == reflect.Slice {
// 		vs, ok := v.([]map[string]any)
// 		if !ok {
// 			panic(fmt.Sprintf("invalid YAML input - arrays must be []map[string]any, was %v", t))
// 		}
// 		return ListYaml(vs)
// 	}
// 	if k == reflect.Map
// }

// func ListYaml(objs []map[string]any) string {

// }

// func ObjectYaml(name string, props []any, depth int) string {
// 	if depth < 0 {
// 		panic(fmt.Sprintf("depth must be non-negative (was: %d)", depth))
// 	}

// }
