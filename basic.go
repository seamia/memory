// github.com/seamia/memory

package memory

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
)

func (m *mapper) mapPtrIface(iVal reflect.Value, parentID nodeID, inlineable bool, isPointer bool) (nodeID, string) {
	pointee := iVal.Elem()

	if pointee.IsValid() && !pointee.IsZero() {
		return m.mapValue(pointee, parentID, inlineable) // todo: recursion?
		/*
			typ := pointee.Type().Kind()
			if typ == reflect.Ptr || typ == reflect.Interface {
				// return m.mapPtrIface(pointee, inlineable, isPointer) // todo: recursion?
			}
		*/
	}

	key := getNodeKey(iVal)

	describe("map.ptr.from", iVal)
	describe("map.ptr.to", pointee)

	// inlineable=false so an invalid parentID is fine
	pointeeNode, pointeeSummary := m.mapValue(pointee, 0, inlineable) // false

	summary := escapeString(iVal.Type().String())
	m.nodeSummaries[key] = summary

	if !pointee.IsValid() {
		// m.nodeSummaries[key] += "(" + pointeeSummary + ")"
		m.nodeSummaries[key] = pointeeSummary
		return pointeeNode, m.nodeSummaries[key]
	}

	if !inlineable {
		id := m.newBasicNode(iVal, summary)
		m.addConnection(id, portTitle, pointeeNode, "", connPointer)
		return id, summary
	}

	if !isPointer {
		warning("check this if we the type we return is correct !!!!! (%s)\n", pointee.Kind().String())
		return pointeeNode, pointeeSummary
	}

	return pointeeNode, summary // pointeeSummary
}

func (m *mapper) mapString(stringVal reflect.Value, inlineable bool) (nodeID, string) {
	// We want the output to look like a Go quoted string literal. The first
	// Quote achieves that. The second is to quote it for graphviz itself.
	quoted := stringVal.String()
	quoted = strings.Trim(quoted, "\"")
	quoted = strconv.Quote(quoted)
	/*

		todo: revisit this commented out block

		quoted = strconv.Quote(quoted)
		quoted = strconv.Quote(quoted)

		quoted = strings.ReplaceAll(strings.ReplaceAll(quoted, "<", "\\<"), ">", "\\>")

		// Lastly, quoting adds quotation-marks around the string, but it is
		// inserted into a graphviz string literal, so we have to remove those.
		quoted = quoted[1 : len(quoted)-1]
	*/
	quoted = normalizeString(quoted)

	if inlineable {
		return 0, quoted
	}
	m.nodeSummaries[getNodeKey(stringVal)] = "string"
	return m.newBasicNode(stringVal, quoted), "string"
}

func (m *mapper) mapBool(stringVal reflect.Value, inlineable bool) (nodeID, string) {
	value := fmt.Sprintf("%t", stringVal.Bool())
	if inlineable {
		return 0, value
	}
	m.nodeSummaries[getNodeKey(stringVal)] = "bool"
	return m.newBasicNode(stringVal, value), "bool"
}

func (m *mapper) mapInt(numVal reflect.Value, inlineable bool) (nodeID, string) {
	printed := strconv.Itoa(int(numVal.Int()))
	if inlineable {
		return 0, printed
	}
	m.nodeSummaries[getNodeKey(numVal)] = "int"
	return m.newBasicNode(numVal, printed), "int"
}

func (m *mapper) mapUint(numVal reflect.Value, inlineable bool) (nodeID, string) {
	printed := strconv.Itoa(int(numVal.Uint()))
	if inlineable {
		return 0, printed
	}
	m.nodeSummaries[getNodeKey(numVal)] = "uint"
	return m.newBasicNode(numVal, printed), "uint"
}

func (m *mapper) mapFunc(funcVal reflect.Value, inlineable bool) (nodeID, string) {

	uType := funcVal.Type()
	id := m.getNodeID(funcVal)
	key := getNodeKey(funcVal)
	m.nodeSummaries[key] = escapeString(uType.String())

	if inlineable || !funcVal.IsValid() || funcVal.IsZero() {
		return 0, m.nodeSummaries[key]
	}

	funcTypeName := uType.String()
	snode := createNode(id, funcTypeName, "function")

	if funcVal.IsValid() && !funcVal.IsZero() {
		ptr := funcVal.Pointer()
		fptr := runtime.FuncForPC(ptr)
		file, line := fptr.FileLine(ptr)
		name := fptr.Name()

		// name := GetFunctionName(val)
		// show("==== %v (%s:%v)", name, file, line)

		snode.addFieldInlined("name", "name", name, Info)
		snode.addFieldInlined("file", "file", file, Info)
		snode.addFieldInlined("line", "line", str(line), Info)
	}

	m.addNode(snode)
	return id, m.nodeSummaries[key]
	/*

		value := "(empty)"
		if funcVal.IsValid() && !funcVal.IsNil() && !funcVal.IsZero() {
			ptr := funcVal.Pointer()
			fptr := runtime.FuncForPC(ptr)
			// file, line := fptr.FileLine(ptr)
			name := fptr.Name()
			value = name

			// name := GetFunctionName(val)
			// show("==== %v (%s:%v)", name, file, line)
		} else {
			// show("==== none")
			value = "nil"
			return 0, value
		}

		// value := fmt.Sprintf("%t", funcVal.Bool())
		// if inlineable { return 0, value }

		m.nodeSummaries[getNodeKey(funcVal)] = "func"
		return m.newBasicNode(funcVal, value), "func"
	*/
}

func normalizeString(original string) string {
	maxAllowedStringLen := Options().MaxStringLength

	if len(original) <= maxAllowedStringLen {
		return original
	}
	half := (maxAllowedStringLen / 2) - 1

	return original[:half] + ".." + original[len(original)-half:]
}

func str(from int) string {
	return strconv.Itoa(from)
}

func (m *mapper) mapChan(chanVal reflect.Value, inlineable bool) (nodeID, string) {

	uType := chanVal.Type()
	id := m.getNodeID(chanVal)
	key := getNodeKey(chanVal)
	m.nodeSummaries[key] = escapeString(uType.String())

	if inlineable {
		return 0, m.nodeSummaries[key]
	}

	chanTypeName := uType.String()
	snode := createNode(id, chanTypeName, "channel")

	if chanVal.IsValid() && !chanVal.IsNil() && !chanVal.IsZero() {
		snode.addFieldInlined("len", "len", str(chanVal.Len()), Info)
		snode.addFieldInlined("cap", "cap", str(chanVal.Cap()), Info)
		snode.addFieldInlined("dir", "dir", uType.ChanDir().String(), Info)
	}

	m.addNode(snode)
	return id, m.nodeSummaries[key]

	/*
		value := "(empty)"
		if chanVal.IsValid() && !chanVal.IsNil() && !chanVal.IsZero() {
			ptr := chanVal.Pointer()
			fptr := runtime.FuncForPC(ptr)
			// file, line := fptr.FileLine(ptr)
			name := fptr.Name()
			value = name

			// name := GetFunctionName(val)
			// show("==== %v (%s:%v)", name, file, line)
		} else {
			// show("==== none")
			value = "nil"
			return 0, value
		}

		if chanVal.IsValid() && !chanVal.IsNil() && !chanVal.IsZero() {
			show("len: %v", chanVal.Len())
			show("cap: %v", chanVal.Cap())
			// chanVal.Type().ChanDir().String()
		}



		// value := fmt.Sprintf("%t", funcVal.Bool())
		// if inlineable { return 0, value }

		m.nodeSummaries[getNodeKey(chanVal)] = "chan"
		return m.newBasicNode(chanVal, value), "chan"
	*/
}
