// github.com/seamia/memory

package memory

import (
	"fmt"
	"io"
	"reflect"
	"strings"
)

type cell struct {
	port string
	name string
	kind CellType
}

func (c *cell) write(w io.Writer) {

	out := func(format string, arg ...interface{}) {
		fmt.Fprintf(w, format, arg...)
	}

	// name := htmlize(c.name)
	// out("<TD BGCOLOR=\"%s\" PORT=\"%s\" ALIGN=\"%s\" TITLE=\"%s\"><i>%s</i></TD>", c.bgcolor, c.port, c.align, name, name)

	render(out, c.kind, c.name, c.port)
}

type field struct {
	cells []cell
}

func (f *field) write(w io.Writer) {

	out := func(format string, arg ...interface{}) {
		fmt.Fprintf(w, format, arg...)
	}

	out("<TR>")
	for _, entry := range f.cells {
		entry.write(w)
	}
	out("</TR>")
}

type cnode struct {
	id      nodeID
	name    string
	tooltip string
	fields  []field
}

func createNode(id nodeID, name string, tooltip string) *cnode {
	node := cnode{
		id:      id,
		name:    name,
		tooltip: tooltip,
	}

	trace("create.node: %s [%v]\n", name, id)
	return &node
}

func (s *cnode) addFieldInlined(port string, name, summary string, kind CellType) {
	s.fields = append(s.fields, field{
		cells: []cell{
			cell{
				port: port,
				name: name,
				kind: Key,
			},
			cell{
				name: summary,
				kind: kind,
			},
		},
	})
}

func (s *cnode) addField(port string, name string, kind CellType) {
	s.fields = append(s.fields, field{
		cells: []cell{
			cell{
				port: port,
				name: name,
				kind: kind,
			},
		},
	})
}

func (s *cnode) addFields(port1 string, name1 string, port2 string, name2 string) {
	s.fields = append(s.fields, field{
		cells: []cell{
			cell{
				port: port1,
				name: name1,
				kind: Key,
			},
			cell{
				port: port2,
				name: name2,
				kind: Value,
			},
		},
	})
}

func (s *cnode) addCells(args ...cell) {
	var cells []cell
	for _, one := range args {
		cells = append(cells, one)
	}
	if len(cells) > 0 {
		s.fields = append(s.fields, field{
			cells: cells,
		})
	} else {
		warning("the imput was empty?")
	}
}

func (s *cnode) colspan() int {
	span := 1
	for _, entry := range s.fields {
		if len(entry.cells) > span {
			span = len(entry.cells)
		}
	}
	return span
}

func (s *cnode) write(w io.Writer) {
	out := func(format string, arg ...interface{}) {
		fmt.Fprintf(w, format, arg...)
	}

	// 		Node_Ja_128	[shape=plaintext tooltip="*" label=<*>];
	out("\t%v	[shape=plaintext tooltip=\"%s\" label=<", s.id.getName(), s.tooltip)

	table := getProperties(Frame)
	out("<TABLE BORDER=\"%s\" CELLBORDER=\"%s\" CELLSPACING=\"%s\" BGCOLOR=\"%s\">",
		table["border"], table["cellborder"], table["cellspacing"], table["bgcolor"])

	header := getProperties(Header)
	header = customize(header, s.tooltip)
	out("<TR><TD COLSPAN=\"%v\" PORT=\"%s\" BGCOLOR=\"%s\" ALIGN=\"%s\">%s</TD></TR>",
		s.colspan(), portTitle,
		header["bgcolor"], header["align"],
		formattedText(Header, s.name))

	for _, entry := range s.fields {
		entry.write(w)
	}
	out("</TABLE>")

	out(">];\n")
}

func (m *mapper) addConnection(fromNode nodeID, port string, toNode nodeID, tooltip string, style connectionStyle) {
	if toNode == 0 {
		report("toNode is zero...\n")
		return
	}

	m.connections = append(m.connections, connection{
		fromNode: fromNode,
		fromPort: port,
		toNode:   toNode,
		toPort:   portTitle,
		tooltip:  tooltip,
		style:    style,
	})
}

func (m *mapper) addNode(node *cnode) {
	m.nodes = append(m.nodes, node)
}

func (m *mapper) write(w io.Writer) {
	m.optimize()
	// Mrecord(w, m.nodes, m.connections, m.comment)
	mTable(w, m.nodes, m.connections, m.comment)
}

func (m *mapper) isRoot(what reflect.Value) bool {
	if m.currentRoot.UnsafeAddr() == what.UnsafeAddr() {
		if m.currentRoot.Type().String() == what.Type().String() {
			return true
		}
	}
	return false
}
func (m *mapper) Nil() string {
	if value, found := m.nodeSummaries[nilKey]; found {
		return value
	}

	return "nil"
}

func (m *mapper) optimize() {

	if Options().CollapsePointerNodes || Options().CollapseSingleSliceNodes {

		direct := make(map[nodeID][]nodeID)
		reverse := make(map[nodeID][]nodeID)

		for _, conn := range m.connections {
			direct[conn.fromNode] = append(direct[conn.fromNode], conn.toNode)
			reverse[conn.toNode] = append(reverse[conn.toNode], conn.fromNode)
		}

		access := make(map[nodeID]*cnode)
		for _, node := range m.nodes {
			access[node.id] = node
		}

		singleUse := make([]nodeID, 0)
		remap := make(map[nodeID]nodeID)

		for _, node := range m.nodes {
			from := node.id

			if len(direct[from]) == 1 {
				to := direct[from][0]
				// if len(reverse[to]) == 1 {
				if len(node.fields) <= 1 { // == 0
					parts := strings.Split(node.name, ".")
					suffix := parts[len(parts)-1]

					toName := access[to].name
					if toName == node.name || toName == suffix {

						singleUse = append(singleUse, from)
						remap[from] = to
					}
				}
				// }
			}
		}

		var connections []connection
		for _, conn := range m.connections {
			from := conn.fromNode
			to := conn.toNode

			if newTo, exists := remap[to]; exists {
				conn.toNode = newTo

				if len(access[to].fields) == 0 {
					conn.style = 1
				} else {
					conn.style = 2
				}

			} else if _, found := remap[from]; found {
				continue
			}

			connections = append(connections, conn)
		}
		m.connections = connections

		for _, id := range singleUse {
			delete(access, id)
		}

		var nodes []*cnode
		for _, value := range access {
			nodes = append(nodes, value)
		}
		m.nodes = nodes
	}
}

func (node nodeID) getName() string {
	return fmt.Sprintf("Node_Ja_%v", node)
}
