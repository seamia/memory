// github.com/seamia/memory

package memory

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
)

//var spewer = &spew.ConfigState{
//	Indent:                  "  ",
//	SortKeys:                true, // maps should be spewed in a deterministic order
//	DisablePointerAddresses: true, // don't spew the addresses of pointers
//	DisableCapacities:       true, // don't spew capacities of collections
//	SpewKeys:                true, // if unable to sort map keys then spew keys to strings and sort those
//	MaxDepth:                1,
//}

type (
	nodeKey         string
	nodeID          int
	connectionStyle int

	connection struct {
		fromNode nodeID
		fromPort string

		toNode nodeID
		toPort string

		tooltip string
		style   connectionStyle
	}

	CustomResolver    = func(value reflect.Value) (string, bool)
	CustomInformation = func() map[string]string
)

const (
	connDefault connectionStyle = iota
	connPointer
	connArray
	connInner
)

var (
	nilKey         nodeKey = "nil0"
	tmpFileCounter int32
)

type mapper struct {
	writer              io.Writer
	nodeIDs             map[nodeKey]nodeID
	nodeSummaries       map[nodeKey]string
	inlineableItemLimit int

	nodes       []*cnode
	connections []connection
	properties  info

	comment string

	knownEntries map[uintptr]reflect.Value
	currentRoot  reflect.Value

	resolvers []CustomResolver
}

// Map prints the given datastructure using the default config
func Map(w io.Writer, is ...interface{}) {
	defaultConfig().Map(w, is...)
}

// Map prints out a Graphviz digraph of the given datastructure to the given io.Writer
func (c *Config) Map(w io.Writer, is ...interface{}) {

	trace("==================[%v]==[%v]===\n", w, is) // todo: remove this mask of "later" bug

	var comment string
	lenis := len(is)
	if lenis > 1 {
		if txt, converts := is[lenis-1].(string); converts {
			comment = strings.ReplaceAll(txt, "\"", "\\")
			// is = is[:lenis-1]
		}
	}

	comment = strings.ReplaceAll(comment, "\\", "/")

	if w == nil {
		current := atomic.AddInt32(&tmpFileCounter, 1)
		fileName := fmt.Sprintf("./memory-%v.dot", current)
		f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			warning("failed to create file (%s), due to: %v", fileName, err)
			return
		}
		w = f
		defer func() {
			trace("closing file: %v", fileName)
			f.Close()
		}()
	}

	m := &mapper{
		w,
		map[nodeKey]nodeID{nilKey: 0},
		map[nodeKey]string{nilKey: "nil"},
		2,
		nil,
		nil,
		info{},
		comment,
		map[uintptr]reflect.Value{},
		reflect.Value{},
		nil,
	}

	var iVals []reflect.Value
	for index, i := range is {
		// fmt.Printf("type of %v: %T\n", index, i)
		_ = index

		if txt, ok := i.(string); ok {
			// this is a comment - ignore it
			if len(m.comment) == 0 {
				m.comment = txt
			}
		} else if reolver, ok := i.(CustomResolver); ok {
			m.add(reolver)
		} else if inform, ok := i.(CustomInformation); ok {
			for key, value := range inform() {
				m.addInfo(key, value)
			}
		} else if inform, ok := i.(func() map[string]string); ok {
			for key, value := range inform() {
				m.addInfo(key, value)
			}
		} else {
			iVal := reflect.ValueOf(i)
			if !iVal.CanAddr() {
				if iVal.Kind() != reflect.Pointer && iVal.Kind() != reflect.Interface {
					fmt.Fprint(w, "error: cannot map unaddressable value")
					return
				}

				iVal = iVal.Elem()
			}
			iVals = append(iVals, iVal)
		}
	}

	// fmt.Fprintln(w, "digraph structs {")
	// fmt.Fprintln(w, "  node [shape=Mrecord];")
	for _, iVal := range iVals {
		m.currentRoot = iVal
		m.mapValue(iVal, 0, false)
	}
	m.currentRoot = reflect.Value{}
	// fmt.Fprintln(w, "}")
	m.write(w)
}

// for values that aren't addressable keep an incrementing counter instead
var keyCounter int

func getNodeKey(val reflect.Value) nodeKey {
	if val.CanAddr() {
		// return nodeKey(fmt.Sprint(val.Kind()) + fmt.Sprint(val.UnsafeAddr()))
		txt := fmt.Sprintf("%s:%x:%s", val.Kind(), val.UnsafeAddr(), val.Type().Name())
		return nodeKey(txt)
	}

	// reverse order of type and "address" to prevent (incredibly unlikely) collisions

	if val.Kind() == reflect.Pointer {
		if !val.IsNil() {
			got := getNodeKey(val.Elem())
			trace("pointer to: %v\n", got)
		}
	}

	// *.Pointer returns v's value as a uintptr.
	// It panics if v's Kind is not [Chan], [Func], [Map], [Pointer], [Slice], or [UnsafePointer].
	switch val.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		if val.IsValid() && !val.IsZero() {
			ptr := val.Pointer()
			if ptr != 0 {
				txt := fmt.Sprintf("%s@%x", val.Kind().String(), ptr)
				return nodeKey(txt)
			}
		}
	}

	keyCounter++
	return nodeKey(fmt.Sprintf("%v=%s", keyCounter, val.Kind()))
}

func (m *mapper) getNodeID(iVal reflect.Value) nodeID {
	// have to key on kind and address because a struct and its first element have the same UnsafeAddr()
	key := getNodeKey(iVal)
	if id, ok := m.nodeIDs[key]; !ok {
		id = nodeID(len(m.nodeIDs))
		m.nodeIDs[key] = id
		return id
	} else {
		return id
	}
}

func (m *mapper) add(resolver CustomResolver) {
	m.resolvers = append(m.resolvers, resolver)
}

func (m *mapper) newBasicNode(iVal reflect.Value, text string) nodeID {
	id := m.getNodeID(iVal)
	m.addNode(createNode(id, text, iVal.Kind().String()))
	// fmt.Fprintf(m.writer, "  %d [label=\"<name> %s\"];\n", id, text)
	return id
}

func (m *mapper) known(iVal reflect.Value) bool {

	if !iVal.CanAddr() {
		report("--------------------------------------- can't addr: %s (%v)\n", iVal.Kind().String(), iVal.IsZero())
		return false
	}

	var ptr uintptr
	switch iVal.Kind() {
	case reflect.Func, reflect.Chan, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		ptr = iVal.Pointer() // works only with: [Chan], [Func], [Map], [Pointer], [Slice], or [UnsafePointer].

	case reflect.Struct:
		report("------ struct: %s\n", iVal.Type().Name())
		for index := 0; index < iVal.Type().NumField(); index++ {
			field := iVal.Field(index)
			if !field.CanAddr() {

			} else {
				// uptr := field.UnsafePointer()
				// report("                   field: %s, addr: %v\n", field.Type().String(), uptr)
			}
		}

	default:
		addr := iVal.Addr()
		ptr = addr.Pointer()
	}

	if iVal.CanAddr() {
		/*
			addr := iVal.Addr()
			addr.Pointer()
			ptr = iVal.Pointer()
		*/
	} else {
		ptr = iVal.Pointer()
	}

	report("---------------------------------%s\t(0x%x)\n", iVal.Kind().String(), ptr)

	if ptr == 0 || iVal.IsZero() {
		return false
	}

	if _, found := m.knownEntries[ptr]; found {
		return true
	}

	m.knownEntries[ptr] = iVal
	return false
}

func (m *mapper) mapValue(iVal reflect.Value, parentID nodeID, inlineable bool) (nodeID, string) {
	if !iVal.IsValid() {
		// zero value => probably result of nil pointer
		return m.nodeIDs[nilKey], m.nodeSummaries[nilKey]
	}

	key := getNodeKey(iVal)
	describe("map.value", iVal)

	// m.known(iVal)

	const reserved = "(reserved)"
	if summary, ok := m.nodeSummaries[key]; ok {
		// already seen this address so no need to map again
		if summary != reserved {
			return m.nodeIDs[key], summary
		} else {
			debug()
		}
	} else {
		// to deal with "references to itself" let's "reserve" the spot here
		m.nodeSummaries[key] = reserved

		defer func() {
			if m.nodeSummaries[key] == reserved {
				warning("node [%s] was not updated\n", key)
			}
		}()
	}

	switch iVal.Kind() {
	// Indirections
	case reflect.Ptr, reflect.Interface:
		return m.mapPtrIface(iVal, parentID, inlineable, iVal.Kind() == reflect.Ptr)

	// Collections
	case reflect.Struct:
		return m.mapStruct(iVal)
	case reflect.Slice, reflect.Array:
		return m.mapSlice(iVal, parentID, inlineable)
	case reflect.Map:
		return m.mapMap(iVal, parentID, inlineable)

	// Simple types
	case reflect.Bool:
		return m.mapBool(iVal, inlineable)
	case reflect.String:
		return m.mapString(iVal, inlineable)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		return m.mapInt(iVal, inlineable)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		return m.mapUint(iVal, inlineable)

	case reflect.Func:
		return m.mapFunc(iVal, iVal.IsZero()) // inlineable

	case reflect.Chan:
		return m.mapChan(iVal, iVal.IsZero()) // inlineable

	case reflect.Uintptr:
		return m.mapUint(iVal, inlineable)

	// If we've missed anything then just fmt.Sprint it
	default:
		report("unhandled: %v\n", iVal.Kind().String())
		m.explore(iVal)

		return m.newBasicNode(iVal, fmt.Sprint(iVal.Interface())), iVal.Kind().String()
	}
}

func (m *mapper) isNil(what string) bool {
	if value, found := m.nodeSummaries[nilKey]; found {
		if what == value {
			return true
		}
	}
	return false
}

func (m *mapper) addInfo(key string, format string, args ...interface{}) {
	m.properties.add(key, format, args...)
}

func (conn *connection) write(w io.Writer) {
	out := func(format string, arg ...interface{}) {
		fmt.Fprintf(w, format, arg...)
	}

	// [weight=1, penwidth=3 color="#9ACEEB" tooltip="text"];
	styles := []string{}
	toPort := conn.toPort

	add := func(what string, args ...interface{}) {
		styles = append(styles, fmt.Sprintf(what, args...))
	}
	port := func(to string) {
		toPort += ":" + to
	}

	for prop, value := range connectorProperties[conn.style] {
		switch prop {
		case "port":
			port(value)
		case "color":
			add("%s=\"%s\"", prop, value)
		default:
			add("%s=%s", prop, value)
		}
	}

	/*
		switch conn.style {
		case connDefault:
			add("color=\"%s\"", "black")
			port("w")
		case connPointer:
			add("color=\"%s\"", "red")
			port("w") // "n"
		case connArray:
			add("color=\"%s\"", "blue")
		case connInner:
			add("color=\"%s\"", "green")
			add("weight=3")
			add("penwidth=3")
			port("w")
		}
	*/

	// style = Options().LinkPointer
	// Options().LinkArray

	if len(conn.tooltip) > 0 {
		if tooltip := strings.Trim(conn.tooltip, " \t\"\r\n"); len(tooltip) > 0 {
			add("tooltip=\"%s\"", tooltip)
		}
	}

	if optionAllowMetadata {
		add("id=\"%s;%s;\"", conn.fromNode.getName(), conn.toNode.getName())
	}

	style := ""
	if len(styles) > 0 {
		style = " [" + strings.Join(styles, " ") + "]"
	}

	out("\t%v:<%v>:e\t-> %v:%v%s;\n", conn.fromNode.getName(), conn.fromPort, conn.toNode.getName(), toPort, style)

}

func kind2style(from reflect.Kind) connectionStyle {
	switch from {
	case reflect.Pointer, reflect.Interface:
		return connPointer
	case reflect.Slice, reflect.Array:
		return connArray
	case reflect.Struct:
		return connInner
	}
	return connDefault
}

func value2style(iVal reflect.Value) connectionStyle {

	switch iVal.Kind() {
	case reflect.Ptr:
		return connPointer
	case reflect.Interface:
		return value2style(iVal.Elem()) // todo: recursion?
	case reflect.Struct:
		return connInner

	case reflect.Slice, reflect.Array:
		return connArray
	}
	return connDefault
}

func stringResolver(value reflect.Value) (string, bool) {
	// value.CanConvert()
	if str := value.MethodByName("String"); str.IsValid() {
		if back := str.Call([]reflect.Value{}); len(back) == 1 && back[0].IsValid() {
			return back[0].String(), true
		}
	}

	return "", false
}
