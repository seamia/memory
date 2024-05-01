// github.com/seamia/memory

package memory

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

type ignoreResponse int

const (
	doNotSkip        ignoreResponse = 0
	ignoreCompletely ignoreResponse = 1
	ignoreValue      ignoreResponse = 2

	showHexForLargeInts = true
	showTypeForInts     = true
	showStructNilFields = true
	showZeroFields      = true
)

func skipField(kind, collection, field string) ignoreResponse {

	if len(Options().Discard) != 0 {

		collection = strings.Trim(collection, " \"\\")

		full := kind + ":" + collection + "." + field
		if value, found := Options().Discard[full]; found {
			switch value {
			case 0:
				return doNotSkip
			case 1:
				return ignoreCompletely
			case 2:
				return ignoreValue
			default:
				warning("unrecognized value (%v) for key (%v)", value, full)
			}
		}
	}
	return doNotSkip
}

func (m *mapper) interpretValueType(val, typ string, value reflect.Value) (string, CellType, bool) {

	isNil := false
	if len(Options().Substitute) != 0 {
		if replace, found := Options().Substitute[typ]; found && len(replace) != 0 {
			if change, exists := replace[val]; exists {
				return change, Default, isNil
			}
		}
	}

	if value.IsZero() {
		isNil = true
		if value.CanInt() || value.CanUint() {
			return "0", Blank, isNil
		} else if value.CanFloat() || value.CanComplex() {
			return "0.0", Blank, isNil
		}

		switch value.Kind() {
		case reflect.String:
			return "\"\"", Blank, isNil
		case reflect.Pointer, reflect.Interface:
			return "nil", Blank, isNil // todo: pull the actual string from the mapper
		case reflect.Bool:
			return "false", Blank, isNil
		case reflect.Slice:
			return "[]", Blank, isNil
		}

		warning("unhandled (zero) kind: %s\n", value.Kind().String())

		return "nil", Blank, isNil // todo: pull the actual string from the mapper
	}

	if typ == "string" {
		return val, Default, isNil
	}

	if optionAllowExternalResolver {
		if txt, can := m.resolve(value); can {
			return txt, ExternalResolver, isNil
		}
	}

	if optionAllowStringResolver {
		if txt, can := stringResolver(value); can {
			return txt, StringResolver, isNil
		}
	}

	if showHexForLargeInts {
		if txt, can := canUseHex(value); can {
			return txt, Default, isNil
		}
	}

	if showTypeForInts {
		val += " (" + typ + ")"
	}
	return val, Default, isNil
}

func (m *mapper) resolve(value reflect.Value) (string, bool) {
	for _, resolver := range m.resolvers {
		if txt, yes := resolver(value); yes {
			return txt, true
		}
	}

	return "", false
}

func canUseHex(value reflect.Value) (string, bool) {
	if value.CanUint() {
		v := value.Uint()
		if v > 16 {
			return fmt.Sprintf("%v (0x%x)", v, v), true
		}
	} else if value.CanInt() {
		v := value.Int()
		if v > 16 {
			return fmt.Sprintf("%v (0x%x)", v, v), true
		}
	}
	return "", false
}

func warning0(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Warning: "+format+"\n", args...)
}
