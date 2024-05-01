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

		/*
			fld := structVal.Field(index)

			if !fld.CanAddr() {
				// TODO: when does this happen? Can we work around it?
				warning("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA: %s", fld.String())
				//continue
			}
			anonymous := uType.Field(index).Anonymous
			fieldName := uType.Field(index).Name
			fieldType := uType.Field(index).Type.Name()

			if len(fieldType) == 0 {
				fieldType = uType.Field(index).Type.String()
			}

			fldIsEmpty := isEmpty(fld)
			if fldIsEmpty {
				report("field %s is EMPTY\n", fld.Type().Kind().String())
			} else {
				report("field %s is NOT EMPTY\n", fld.Type().Kind().String())
			}

			_ = anonymous
			_ = fieldType
			structRef := getStructRef(index)

			warning("struct: %s; fld: [%s], anonymous: %v; fieldType: %v\n", structTypeName, fieldName, anonymous, fieldType)

			switch skipField("struct", structTypeName, fieldName) {
			case doNotSkip:
				break
			case ignoreCompletely:
				continue
			case ignoreValue:
				snode.addFieldInlined(structRef, fieldName, ignoredValue, Blank)
				continue
			}

			if fld.CanAddr() {
				fld = reflect.NewAt(fld.Type(), unsafe.Pointer(fld.UnsafeAddr())).Elem()
			} else {
				//??????????????????
			}
			fieldID, summary := m.mapValue(fld, id, isInlinableValue(fld))

			// if fld was inlined (id == 0) then print summary, else just the name and a link to the actual
			if fieldID == 0 {
				value, isnil := interpretValueType(summary, fieldType, fld)

				if !isnil || showStructNilFields {
					snode.addFieldInlined(structRef, fieldName, value, valueCellType(isnil))
				} else {
					warning("not showing fld [%s] cause it is nil", fieldName)
				}
			} else {
				// snode.addField(structRef, fieldName, Key)

				if showZeroFields || !fldIsEmpty {
					outgoing := getStructOutgoing(index)
					snode.addCells(cell{structRef, fieldName, Key}, cell{outgoing, fieldType, Type})

					m.addConnection(id, outgoing, fieldID, fieldName+"", kind2style(fld.Type().Kind()))
				} else {
					warning("not showing fld [%s] cause it is empty", fieldName)
				}
			}
		*/
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
		/*
			value := sliceVal.Index(index)
			fldIsEmpty := isEmpty(value)
			if value.CanAddr() {
				value = reflect.NewAt(value.Type(), unsafe.Pointer(value.UnsafeAddr())).Elem()
			}

			indexID, summary := m.mapValue(value, sliceID, isInlinableValue(value))
			if indexID == 0 {

				// field was inlined so print summary
				if discardNilEntriesInSlices && m.isNil(summary) {
					// we are discarding this entry
					discardedEntries++
				} else {
					// snode.addFields(getSliceRef(sliceID, index), str(index), getValueRef(sliceID, index), summary)
					snode.addCells(
						cell{getSliceRef(sliceID, index), str(index), Key},
						cell{getValueRef(sliceID, index), summary, Value})
				}
			} else {
				if showZeroFields || !fldIsEmpty {
				}

				// need pointer to value
				snode.addField(getSliceRef(sliceID, index), str(index), Key)
				m.addConnection(sourceID, getSliceRef(sliceID, index), indexID, str(index), kind2style(value.Type().Kind())) // connArray
			}
		*/
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
		/*
			keyID, keySummary := m.mapValue(mapKey, id, true)
			valueID, valueSummary := m.mapValue(mapVal.MapIndex(mapKey), id, true)

			// style := kind2style(mapVal.MapIndex(mapKey).Kind())
			style := value2style(mapVal.MapIndex(mapKey))

			switch skipField("map", keySummary, "") {
			case doNotSkip:
				break
			case ignoreCompletely:
				continue
			case ignoreValue:
				valueSummary = ignoredValue
			}

			// snode.addFields(getKeyRef(mapID, index), keySummary, getValueRef(mapID, index), valueSummary)
			snode.addCells(
				cell{getKeyRef(mapID, index), keySummary, Key},
				cell{getValueRef(mapID, index), valueSummary, Type})

			if keyID != 0 {
				m.addConnection(id, getKeyRef(mapID, index), keyID, "", style)
			}
			if valueID != 0 {
				m.addConnection(id, getValueRef(mapID, index), valueID, "", style)
			}
		*/
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

/*
func describe(val reflect.Value) {
	if a := val.Addr(); !a.IsNil() {
	}

	// if a := val.assignTo(context string, dst *abi.Type, target unsafe.Pointer) Value {
	if a := val.Bool() bool {

	}
}
		if a := val.Bytes() []byte {
		if a := val.bytesSlow() []byte {
		if a := val.Call(in []Value) []Value {
		if a := val.call(op string, in []Value) []Value {
		if a := val.CallSlice(in []Value) []Value {
		if a := val.CanAddr() bool {
		if a := val.CanComplex() bool {
		if a := val.CanConvert(t Type) bool {
		if a := val.CanFloat() bool {
		if a := val.CanInt() bool {
		if a := val.CanInterface() bool {
		if a := val.CanSet() bool {
		if a := val.CanUint() bool {
		if a := val.Cap() int {
		if a := val.capNonSlice() int {
		if a := val.Clear() {
		if a := val.Close() {
		if a := val.Comparable() bool {
		if a := val.Complex() complex128 {
		if a := val.Convert(t Type) Value {
		if a := val.Elem() Value {
		if a := val.Equal(u Value) bool {
		if a := val.extendSlice(n int) Value {
		if a := val.Field(i int) Value {
		if a := val.FieldByIndex(index []int) Value {
		if a := val.FieldByIndexErr(index []int) (Value, error) {
		if a := val.FieldByName(name string) Value {
		if a := val.FieldByNameFunc(match func(string) bool) Value {
		if a := val.Float() float64 {
		if a := val.grow(n int) {
		if a := val.Grow(n int) {
		if a := val.Index(i int) Value {
		if a := val.Int() int64 {
		if a := val.Interface() (i any) {
		if a := val.InterfaceData() [2]uintptr {
		if a := val.IsNil() bool {
		if a := val.IsValid() bool {
		if a := val.IsZero() bool {
		if a := val.Kind() Kind {
		if a := val.Len() int {
		if a := val.lenNonSlice() int {
		if a := val.MapIndex(key Value) Value {
		if a := val.MapKeys() []Value {
		if a := val.MapRange() *MapIter {
		if a := val.Method(i int) Value {
		if a := val.MethodByName(name string) Value {
		if a := val.NumField() int {
		if a := val.NumMethod() int {
		if a := val.OverflowComplex(x complex128) bool {
		if a := val.OverflowFloat(x float64) bool {
		if a := val.OverflowInt(x int64) bool {
		if a := val.OverflowUint(x uint64) bool {
		if a := val.panicNotBool() {
		if a := val.Pointer() uintptr {
		if a := val.pointer() unsafe.Pointer {
		if a := val.Recv() (x Value, ok bool) {
		if a := val.recv(nb bool) (val Value, ok bool) {
		if a := val.runes() []rune {
		if a := val.Send(x Value) {
		if a := val.send(x Value, nb bool) (selected bool) {
		if a := val.Set(x Value) {
		if a := val.SetBool(x bool) {
		if a := val.SetBytes(x []byte) {
		if a := val.SetCap(n int) {
		if a := val.SetComplex(x complex128) {
		if a := val.SetFloat(x float64) {
		if a := val.SetInt(x int64) {
		if a := val.SetIterKey(iter *MapIter) {
		if a := val.SetIterValue(iter *MapIter) {
		if a := val.SetLen(n int) {
		if a := val.SetMapIndex(key, elem Value) {
		if a := val.SetPointer(x unsafe.Pointer) {
		if a := val.setRunes(x []rune) {
		if a := val.SetString(x string) {
		if a := val.SetUint(x uint64) {
		if a := val.SetZero() {
		if a := val.Slice(i, j int) Value {
		if a := val.Slice3(i, j, k int) Value {
		if a := val.String() string {
		if a := val.stringNonString() string {
		if a := val.TryRecv() (x Value, ok bool) {
		if a := val.TrySend(x Value) bool {
		if a := val.typ() *abi.Type {
		if a := val.Type() Type {
		if a := val.typeSlow() Type {
		if a := val.Uint() uint64 {
		if a := val.UnsafeAddr() uintptr {
		if a := val.UnsafePointer() unsafe.Pointer {
}
*/

/*

func kk(val reflect.Value) {

	switch val.Kind() {
	case reflect.Invalid:
	case reflect.Bool
	case reflect.Int
	case reflect.Int8
	case reflect.Int16
	case reflect.Int32
	case reflect.Int64
	case reflect.Uint
	case reflect.Uint8
	case reflect.Uint16
	case reflect.Uint32
	case reflect.Uint64
	case reflect.Uintptr
	case reflect.Float32
	case reflect.Float64
	case reflect.Complex64
	case reflect.Complex128
	case reflect.Array
	case reflect.Chan
	case reflect.Func
	case reflect.Interface
	case reflect.Map
	case reflect.Pointer
	case reflect.Slice
	case reflect.String
	case reflect.Struct
	case reflect.UnsafePointer
	}
}


*/

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
