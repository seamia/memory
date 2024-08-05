// github.com/seamia/memory

package memory

import (
	"fmt"
	"io"
	"time"
)

/*
	var splitter = map[bool]string{
		false: "",
		true:  "|",
	}
*/
func mTable(w io.Writer, nodes []*cnode, connections []connection, props info, comment string) {
	out := func(format string, arg ...interface{}) {
		fmt.Fprintf(w, format+"\n", arg...)
	}

	if !Options().SuppresHeader {
		out("/*	generated by github.com/seamia/memory")
		out("	based on config file settings, some of the values/connnections might be omitted")
		out("	config file used: %s", Options().LoadedFrom)
		out("	(%s) */", time.Now().String())
	}

	out("digraph \"seamia/memory\" {")
	out("\trankdir=LR;")

	if len(comment) > 0 {
		out("\tlabel=\"%s\"", comment)
		out("\ttooltip=\"%s\"", comment)

	}
	out("\tbgcolor=\"%s\"", Options().ColorBackground)

	out("")
	out("\tnode [")

	out("\t\tfontname=\"%s\"", Options().FontName)
	out("\t\tfontsize=%s", Options().FontSize)
	out("\t\tfillcolor=%s", Options().ColorDefault)
	out("\t\tstyle=\"filled\"")
	out("\t];")

	out("")
	out("\t/* ------ nodes ------ */")
	for _, node := range nodes {
		node.write(w)
	}

	out("")
	out("\t/* ------ connections ------ */")
	for _, conn := range connections {
		conn.write(w)
	}

	if !Options().SuppresInfo {
		out("")
		out("\t/* ------ info ------ */")
		props.write(w)
	}

	out("}")
}
