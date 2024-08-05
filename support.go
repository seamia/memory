// github.com/seamia/memory

package memory

import (
	"reflect"
)

func isInlinableValue(what reflect.Value) bool {
	if !what.IsZero() {
		t := what.Type()

		switch t.Kind() {
		case reflect.Interface:
			return isInlinableValue(what.Elem())

		case reflect.Pointer, reflect.Chan, reflect.Func, reflect.Struct, reflect.Map:
			return false
		case reflect.Array, reflect.Slice:
			return what.Len() == 0
		}
	} else {
		return inlinable
	}

	trace("[%s] is inlinable", what.Type().Kind())
	return inlinable
}

func isInlinableType(what reflect.Type) bool {
	switch what.Kind() {
	case reflect.Pointer, reflect.Chan, reflect.Func, reflect.Interface, reflect.Struct, reflect.Map:
		return false
	case reflect.Array, reflect.Slice:
		return false
	}
	return inlinable
}

func isEmpty(what reflect.Value) bool {
	return isEmptyGuarged(what, 0)
}

func isEmptyGuarged(what reflect.Value, guard int) bool {

	if guard > 100 {
		return false
	}

	if !what.CanAddr() {
		trace("non-addr %s\n", what.Type().Kind().String())
	}

	t := what.Type()
	if !what.IsZero() {
		trace("non-zero %s\n", t.Kind().String())
		switch t.Kind() {
		case reflect.Pointer, reflect.Interface:
			// todo: remember this element to avoid endless recursion
			return isEmptyGuarged(what.Elem(), guard+1)
		case reflect.Struct:

			for index := 0; index < t.NumField(); index++ {
				fld := what.Field(index)
				// todo: recursion ?
				if !isEmptyGuarged(fld, guard+1) {
					return false
				}
			}
			return true

		case reflect.String:
			return len(what.String()) == 0

		case reflect.Chan, reflect.Func:
			return false

		case reflect.Array, reflect.Slice, reflect.Map:
			return what.Len() == 0

		default:
			if what.CanInt() {
				return what.Int() == 0
			}
			if what.CanUint() {
				return what.Uint() == 0
			}

			report("unhandled %s\n", t.Kind().String())
			return false
		}
	} else {

		if t.Kind() == reflect.Struct {
			for index := 0; index < t.NumField(); index++ {
				fld := what.Field(index)
				// todo: recursion ?
				if !isEmptyGuarged(fld, guard+1) {
					return false
				}
			}
			return true
		} else if what.CanInt() || what.CanUint() {
			return what.IsZero()
		} else if t.Kind() == reflect.String {
			return len(what.String()) == 0
		} else if t.Kind() == reflect.Slice {
			return what.Len() == 0
		} else if t.Kind() == reflect.Pointer {
			return true
		}

		trace("zero: %s\n", t.Kind().String())
		return true
	}
	return false
}

var (
	typeNameMapping = map[string]string{
		"interface {}": "any",
	}
)

func getTypeName(typ reflect.Type) string {
	name := typ.Name()

	if len(name) == 0 {
		name = typ.String()
	}

	if correction, found := typeNameMapping[name]; found {
		name = correction
	}

	return name
}

func describe(text string, iVal reflect.Value) {
	if !iVal.IsValid() {
		trace("non valid value => probably result of nil pointer [%v]\n", iVal)
		return
	}

	key := getNodeKey(iVal)
	report("\t%s: ========= [%s] [%v] [%v] [key: %v]\n", text, iVal.Type().Kind().String(), iVal, iVal.String(), key)
}

func copyMap(sourceMap m2s) m2s {
	if sourceMap == nil {
		return nil
	}

	// Create the target map
	targetMap := make(m2s, len(sourceMap))

	// Copy from the original map to the target map
	for key, value := range sourceMap {
		targetMap[key] = value
	}
	return targetMap
}

func debug() {

}
