// github.com/seamia/memory

package memory

import (
	"fmt"
	"io"
	"strings"
	"time"
)

var splitter = map[bool]string{
	false: "",
	true:  "|",
}

func Mrecord(w io.Writer, nodes []*cnode, connections []connection, comment string) {
	out := func(format string, arg ...interface{}) {
		fmt.Fprintf(w, format, arg...)
	}

	if !Options().SuppresHeader {
		out("/*	generated by github.com/seamia/memory\n")
		out("	based on config. settings, some of the values/connnections might be omitted\n")
		out("	config file used: %s\n", Options().LoadedFrom)
		out("	(%s) */\n", time.Now().String())
	}

	out("digraph structs {\n")
	out("\trankdir=LR;\n")

	if len(comment) > 0 {
		out("\tlabel=\"%s\"\n", comment)
		out("\ttooltip=\"%s\"\n", comment)

	}
	out("\tbgcolor=\"%s\"\n", Options().ColorBackground)

	out("\n")
	out("\tnode [\n")
	out("\t\tshape=Mrecord\n")
	out("\t\tfontname=\"%s\"\n", Options().FontName)
	out("\t\tfontsize=%s\n", Options().FontSize)
	out("\t\tfillcolor=%s\n", Options().ColorDefault)
	out("\t\tstyle=\"filled\"\n")
	out("\t];\n")

	out("\n")
	out("\t/* ------ nodes ------ */\n")
	for _, node := range nodes {
		out("\t%v\t[label=\"<%s> %v ", node.id, portTitle, node.name)
		for _, field := range node.fields {
			out("|{")
			for i, cell := range field.cells {
				prefix := splitter[i > 0]
				out("%s<%s> %s", prefix, cell.port, cell.name)
			}
			out("}")
		}
		out("\"")

		if color, defined := GetColor(node.name); defined {
			out(", fillcolor=%s, style=\"filled\"", color)
		}

		out("];\n")
	}

	out("\n")
	out("\t/* ------ connections ------ */\n")
	for _, conn := range connections {
		style := ""
		switch conn.style {
		case 1:
			style = Options().LinkPointer
		case 2:
			style = Options().LinkArray
		}

		if len(style) > 0 && !strings.HasPrefix(style, " ") {
			style = " " + style
		}

		out("\t%v:<%v>\t-> %v:%v%s;\n", conn.fromNode, conn.fromPort, conn.toNode, conn.toPort, style)
	}

	out("}\n")
}