package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/mrshanahan/deploy-assets/internal/util"
)

type ManifestNode struct {
	Kinds map[string]*KindNode
}

type KindNode struct {
	Items []*ItemNode
}

type ItemNode struct {
	Type       string
	Attributes map[string]*AttributeNode
}

type AttributeNode struct {
	Name              string
	MatchingValueType string
	Present           bool
	value             any
}

// We're going to (unwisely) assume this is used safely to avoid some
// code duplication.

// TODO: Transform the underlying values into correct one for return (spec. []object & []string)
// before returning so simple casts can work
func (a *AttributeNode) GetValue() any {
	if a.MatchingValueType == "string" {
		valStr := a.value.(string)
		finalVal := subVarValue(valStr)
		return finalVal
	}
	if a.MatchingValueType == "[]string" {
		valStrs := util.Map(a.value.([]any), func(x any) string { return x.(string) })
		finalValStrs := []string{}
		for _, v := range valStrs {
			finalValStrs = append(finalValStrs, subVarValue(v))
		}
		return finalValStrs
	}
	return a.value
}

var (
	envVarPatt *regexp.Regexp = regexp.MustCompile("{{ (.*?) }}")
)

func subVarValue(s string) string {
	matches := envVarPatt.FindAllStringSubmatchIndex(s, -1)
	if matches == nil {
		return s
	}
	finalVal := s
	for _, matchIndices := range matches {
		wholeStart, wholeEnd := matchIndices[0], matchIndices[1]
		subStart, subEnd := matchIndices[2], matchIndices[3]
		varName := s[subStart:subEnd]
		varVal, prs := os.LookupEnv(varName)
		if prs {
			finalVal = fmt.Sprintf("%s%s%s", finalVal[:wholeStart], varVal, finalVal[wholeEnd:])
		} else {
			finalVal = ""
		}
	}
	return finalVal
}

func ParseManifest(raw []byte) (*ManifestNode, error) {
	manifestObj := make(map[string]any)
	if err := json.Unmarshal(raw, &manifestObj); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %v", err)
	}

	manifestNode := &ManifestNode{
		Kinds: map[string]*KindNode{},
	}

	errs := []error{}
	manifestSpec := NewManifestSpec()

	kindNames := util.NewSet(util.Map(manifestSpec.Kinds, func(k ManifestKindSpec) string { return k.Name() })...)
	for k, _ := range manifestObj {
		if !kindNames.Contains(k) {
			errs = append(errs, fmt.Errorf("<root>: unrecognized top-level key '%s'", k))
		}
	}

	for _, kindSpec := range manifestSpec.Kinds {
		kindName := kindSpec.Name()
		kindJson, prs := manifestObj[kindName]
		if !prs {
			errs = append(errs, fmt.Errorf("<root>: missing required top-level key '%s'", kindName))
			continue
		}

		kindNode := &KindNode{
			Items: []*ItemNode{},
		}
		if kindSpec.IsCollection() {
			itemsJson, ok := kindJson.([]any)
			if !ok {
				actualType := reflect.TypeOf(kindJson)
				errs = append(errs, fmt.Errorf("%s: top-level entry should be array of objects, instead is %v", kindName, actualType))
				continue
			}

			for i, json := range itemsJson {
				itemJson, ok := json.(map[string]any)
				if !ok {
					actualType := reflect.TypeOf(json)
					errs = append(errs, fmt.Errorf("%s[%d]: item should be object, instead is %v", kindName, i, actualType))
					continue
				}
				jsonPath := fmt.Sprintf("%s[%d]", kindName, i)
				itemNode, err := buildItemNode(itemJson, kindSpec.ItemSpecs(), jsonPath)
				if err != nil {
					errs = append(errs, err)
				} else {
					kindNode.Items = append(kindNode.Items, itemNode)
				}
			}
		} else {
			itemJson, ok := kindJson.(map[string]any)
			if !ok {
				actualType := reflect.TypeOf(kindJson)
				errs = append(errs, fmt.Errorf("%s: top-level entry should be object, instead is %v", kindName, actualType))
				continue
			}

			jsonPath := kindName
			itemNode, err := buildItemNode(itemJson, kindSpec.ItemSpecs(), jsonPath)
			if err != nil {
				errs = append(errs, err)
			} else {
				kindNode.Items = append(kindNode.Items, itemNode)
			}
		}

		manifestNode.Kinds[kindName] = kindNode
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return manifestNode, nil
}

func buildItemNode(itemJson map[string]any, itemSpecs map[string]ManifestItemSpec, jsonPath string) (*ItemNode, error) {
	var typ string
	if err := getRequiredField(itemJson, "type", &typ); err != nil {
		return nil, fmt.Errorf("%s: %v", jsonPath, err)
	}

	itemSpec, prs := itemSpecs[typ]
	if !prs {
		return nil, fmt.Errorf("%s: unrecognized type '%s'", jsonPath, typ)
	}

	attrs := itemSpec.Attributes()
	allAttrNames := append(util.Map(attrs, func(a AttributeSpec) string { return a.Name }), "type")
	attrNames := util.NewSet(allAttrNames...)
	for k, _ := range itemJson {
		if !attrNames.Contains(k) {
			return nil, fmt.Errorf("%s: unrecognized key '%s'", jsonPath, k)
		}
	}

	itemNode := &ItemNode{
		Type:       typ,
		Attributes: make(map[string]*AttributeNode),
	}
	for _, attr := range attrs {
		var attrValue any
		prs, err := getField(itemJson, attr.Name, attr.IsRequired, &attrValue)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", jsonPath, err)
		}
		if !prs {
			attrValue = attr.DefaultValue
		}

		validValueTypes := strings.Split(attr.ValueType, "|")
		matchingType, err := checkValueType(attrValue, validValueTypes)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %v", jsonPath, attr.Name, err)
		}

		attrNode := &AttributeNode{
			Name:              attr.Name,
			MatchingValueType: matchingType,
			value:             attrValue,
			Present:           prs,
		}

		itemNode.Attributes[attr.Name] = attrNode
	}

	return itemNode, nil
}

func getRequiredField[T any](obj map[string]any, key string, value *T) error {
	if _, err := getField[T](obj, key, true, value); err != nil {
		return err
	}
	return nil
}

func getField[T any](obj map[string]any, key string, isRequired bool, value *T) (bool, error) {
	valueLiteral, prs := obj[key]
	if !prs && isRequired {
		return false, fmt.Errorf("expected item to have '%s' key", key)
	} else if !prs {
		return false, nil
	}

	result, ok := valueLiteral.(T)
	if !ok {
		expectedType := reflect.TypeFor[T]()
		actualType := reflect.TypeOf(valueLiteral)
		return false, fmt.Errorf("expected item to have type %v for '%s' key, instead is %v", expectedType, key, actualType)
	}
	*value = result
	return true, nil
}

func checkValueType(v any, types []string) (string, error) {
	validType := ""
	for _, t := range types {
		switch t {
		case "string":
			_, ok := v.(string)
			if ok && validType == "" {
				validType = t
			}
		case "int":
			_, ok := v.(int)
			if ok && validType == "" {
				validType = t
			}
		case "bool":
			_, ok := v.(bool)
			if ok && validType == "" {
				validType = t
			}
		case "object":
			_, ok := v.(map[string]any)
			if ok && validType == "" {
				validType = t
			}
		case "[]string":
			_, ok := v.([]string)
			if ok && validType == "" {
				validType = t
			}
		case "[]object":
			_, ok := v.([]map[string]any)
			if ok && validType == "" {
				validType = t
			}
		default:
			return "", fmt.Errorf("invalid attribute type: %s", t)
		}
	}
	if validType == "" {
		actualType := reflect.TypeOf(v)
		return "", fmt.Errorf("invalid value type (value: %v; accepted types: %v; actual type: %v)", v, types, actualType)
	}
	return validType, nil
}
