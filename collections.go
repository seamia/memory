// github.com/seamia/memory

package memory

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"unsafe"
)

const (
	inlinable                 = true
	ignoredValue              = "***"
	discardNilEntriesInSlices = true // todo: make this configurable
)

func (m *mapper) mapStruct(structVal reflect.Value) (nodeID, string) {

	uType := structVal.Type()
	id := m.getNodeID(structVal)
	key := getNodeKey(structVal)
	m.nodeSummaries[key] = escapeString(uType.String())

	structTypeName := getStructTypeName(uType)
	snode := createNode(id, structTypeName, "struct: "+m.nodeSummaries[key])

	if structTypeName == "Object" {
		debug()
	}

	for index := 0; index < uType.NumField(); index++ {
		fld := structVal.Field(index)

		if structTypeName == "Object" && uType.Field(index).Name == "Decl" {
			debug()
		}

		m.unified(snode, fld, uType.Field(index).Type, uType.Field(index).Name, index)
	}

	if showZeroFields || !isEmpty(structVal) || m.isRoot(structVal) {
		m.addNode(snode)
	}
	return id, m.nodeSummaries[key]
}

func (m *mapper) unified(snode *cnode, fld reflect.Value, typ reflect.Type, fieldName string, index int) {

	if !fld.CanAddr() {
		// TODO: when does this happen? Can we work around it?
		// warning("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA: %s", fld.String())
		//continue
	}
	//anonymous := uType.Field(index).Anonymous
	// fieldName := uType.Field(index).Name
	fieldType := getTypeName(typ) // typ.Name()

	fldIsEmpty := isEmpty(fld)
	if fldIsEmpty {
		trace("field %s is EMPTY\n", fld.Type().Kind().String())
	} else {
		trace("field %s is NOT EMPTY\n", fld.Type().Kind().String())
	}

	structRef := getStructRef(index)

	// warning("struct: %s; fld: [%s], anonymous: %v; fieldType: %v\n", structTypeName, fieldName, anonymous, fieldType)

	/*
		switch skipField("struct", structTypeName, fieldName) {
		case doNotSkip:
			break
		case ignoreCompletely:
			continue
		case ignoreValue:
			snode.addFieldInlined(structRef, fieldName, ignoredValue, Blank)
			continue
		}
	*/

	if fld.CanAddr() {
		fld = reflect.NewAt(fld.Type(), unsafe.Pointer(fld.UnsafeAddr())).Elem()
	} else {
		//??????????????????
	}
	fieldID, summary := m.mapValue(fld, snode.id, isInlinableValue(fld))

	// if fld was inlined (id == 0) then print summary, else just the name and a link to the actual
	if fieldID == 0 {
		value, cell, isnil := m.interpretValueType(summary, fieldType, fld)
		if cell == Default {
			cell = valueCellType(isnil)
		}

		if !isnil || showStructNilFields {
			snode.addFieldInlined(structRef, fieldName, value, cell)
		} else {
			warning("not showing fld [%s] cause it is nil", fieldName)
		}
	} else {
		// snode.addField(structRef, fieldName, Key)

		if showZeroFields || !fldIsEmpty {
			outgoing := getStructOutgoing(index)
			snode.addCells(cell{structRef, fieldName, Key}, cell{outgoing, fieldType, Type})

			m.addConnection(snode.id, outgoing, fieldID, fieldName+"", kind2style(fld.Type().Kind()))
		} else {
			warning("not showing fld [%s] cause it is empty", fieldName)
		}
	}
}

func (m *mapper) mapSlice(sliceVal reflect.Value, parentID nodeID, inlineable bool) (nodeID, string) {
	sliceID := m.getNodeID(sliceVal)
	key := getNodeKey(sliceVal)
	sliceType := escapeString(sliceVal.Type().String())
	m.nodeSummaries[key] = sliceType

	if sliceVal.Len() == 0 {
		m.nodeSummaries[key] = sliceType + "\\{\\}"

		if inlineable {
			return 0, m.nodeSummaries[key]
		}

		return m.newBasicNode(sliceVal, m.nodeSummaries[key]), sliceType
	}

	// inlinableType := isInlinableType(sliceVal.Type())
	snode := createNode(sliceID, sliceType, "[]")

	// sourceID is the nodeID that links will start from
	// if inlined then these come from the parent
	// if not inlined then these come from this node
	sourceID := sliceID
	if inlineable && sliceVal.Len() <= m.inlineableItemLimit {
		//		sourceID = parentID
	}

	length, totalLength := sliceVal.Len(), sliceVal.Len()
	if length > Options().MaxSliceLength {
		length = Options().MaxSliceLength
	}

	discardedEntries := 0

	for index := 0; index < length; index++ {
		value := sliceVal.Index(index)
		typ := value.Type()

		m.unified(snode, value, typ, str(index), index)

		_ = sourceID
	}

	if discardedEntries > 0 {
		snode.addField(getSliceRef(sliceID, totalLength-1), fmt.Sprintf("+ %d nil entries", discardedEntries), Footer)
	}

	if totalLength != length {
		snode.addField(getSliceRef(sliceID, totalLength-1), fmt.Sprintf("%d more ...", (totalLength-length)), Footer)
	}

	if showZeroFields || !isEmpty(sliceVal) || m.isRoot(sliceVal) {
		m.addNode(snode)
	}
	return sliceID, m.nodeSummaries[key]
}

func (m *mapper) mapMap(mapVal reflect.Value, parentID nodeID, inlineable bool) (nodeID, string) {
	// create a string type while escaping graphviz special characters
	mapType := escapeString(mapVal.Type().String())

	nodeKey := getNodeKey(mapVal)

	if mapVal.Len() == 0 {
		m.nodeSummaries[nodeKey] = mapType + "\\{\\}"

		if inlineable {
			return 0, m.nodeSummaries[nodeKey]
		}

		return m.newBasicNode(mapVal, m.nodeSummaries[nodeKey]), mapType
	}

	mapID := m.getNodeID(mapVal)
	var id nodeID
	if inlineable && mapVal.Len() <= m.inlineableItemLimit {
		m.nodeSummaries[nodeKey] = mapType
		id = parentID
	} else {
		id = mapID
	}

	snode := createNode(id, mapType, "map")

	for index, mapKey := range mapVal.MapKeys() { // []Value

		if index > Options().MaxMapEntries {
			break
		}

		_, keySummary := m.mapValue(mapKey, id, true)

		value := mapVal.MapIndex(mapKey)
		m.unified(snode, value, value.Type(), keySummary, index)
	}

	if showZeroFields || !isEmpty(mapVal) || m.isRoot(mapVal) {
		m.addNode(snode)
	}
	return id, m.nodeSummaries[nodeKey]
}

const (
	formatIndex = "%di%d"
	formatKey   = "%dk%d"
	formatValue = "%dv%d"
	portTitle   = "name"
)

func getStructRef(index int) string {
	return fmt.Sprintf("f%d", index)
}

func getStructOutgoing(index int) string {
	return fmt.Sprintf("o%d", index)
}

func getSliceRef(sliceID nodeID, index int) string {
	return fmt.Sprintf(formatIndex, sliceID, index)
}

func getKeyRef(sliceID nodeID, index int) string {
	return fmt.Sprintf(formatKey, sliceID, index)
}

func getValueRef(sliceID nodeID, index int) string {
	return fmt.Sprintf(formatValue, sliceID, index)
}

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (m *mapper) explore(val reflect.Value) {
	uType := val.Type()
	id := m.getNodeID(val)
	key := getNodeKey(val)

	// val.IsNil()

	switch val.Kind() {
	case reflect.Func:
		show("--------------------------------- func:")
		show("type: %s", uType.Name())
		show("string: %s", val.String())
		/*
			if val.IsNil() {
				show("nil")
			}
			if val.IsZero() {
				show("zero")
			}
			if val.IsValid() {
				show("valid")
			}
			if val.CanAddr() {
				show("CanAddr")
			}
			if val.CanInterface() {
				show("CanInterface")
			}
		*/
		if val.IsValid() && !val.IsNil() && !val.IsZero() {
			ptr := val.Pointer()
			fptr := runtime.FuncForPC(ptr)
			file, line := fptr.FileLine(ptr)
			name := fptr.Name()

			// name := GetFunctionName(val)
			show("==== %v (%s:%v)", name, file, line)
		} else {
			show("==== none")
		}

	case reflect.Chan:
		show("--------------------------------- chan:")
		if val.IsValid() && !val.IsNil() && !val.IsZero() {
			show("len: %v", val.Len())
			show("cap: %v", val.Cap())
		}

	case reflect.UnsafePointer:
		show("--------------------------------- UnsafePointer:")
	default:
	}

	report("==== [%v] [%v] [%v] [%v]\n", val, uType, id, key)
}

func show(format string, args ...interface{}) {
	report(format+"\n", args...)
}

var (
	replacements = map[string]string{
		"<": "&lt;",
		">": "&gt;",
		/*
			"\"": "&quot;",
			"&":  "&amp;",
			"¢":  "&cent;",
			"©":  "&copy;",
			"®":  "&reg;",
			"£":  "&#163;",
			"¥":  "&#165;",
			"€":  "&euro;",
			":":  "&colon;",
		*/
	}
	titleReplacements = map[string]string{

		"\"": "&quot;",
		"<":  "&lt;",
		">":  "&gt;",
		// "&":  "&amp;",
		"¢": "&cent;",
		"©": "&copy;",
		"®": "&reg;",
		"£": "&#163;",
		"¥": "&#165;",
		"€": "&euro;",
		// ":":  "&colon;",
	}
)

func txt2title(txt string) string {
	txt = strings.Trim(txt, "\"")
	txt = strings.ReplaceAll(txt, "&", "&amp;")

	for from, to := range titleReplacements {
		txt = strings.ReplaceAll(txt, from, to)
	}

	return txt
}

func htmlize(txt string) string {

	if strings.Contains(txt, "adaptSimpleTransport") {
		// runtime.Breakpoint()
	}
	txt = strings.ReplaceAll(txt, "&", "&amp;")

	for from, to := range replacements {
		txt = strings.ReplaceAll(txt, from, to)
	}

	return txt
}

var escaper = strings.NewReplacer(
	"{", "\\{",
	"}", "\\}",
	"\"", "\\\"",
	">", "\\>",
	"<", "\\<",
)

func escapeString(s string) string {
	// return escaper.Replace(s)
	return s
}
