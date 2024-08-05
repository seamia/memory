// github.com/seamia/memory

package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type (
	CellType int
	Writef   func(format string, arg ...interface{})
	m2s      = map[string]string
)

const (
	Header CellType = iota
	Footer
	Key
	Value
	Blank
	Info
	Type
	Frame
	Default
	ExternalResolver
	StringResolver
	InfoFrame
	InfoHeader
	InfoKey
	InfoValue

	background = "bgcolor"
	alignment  = "align"
	text       = "text"
)

func valueCellType(isNil bool) CellType {
	if isNil {
		return Blank
	}
	return Value
}

func render(out Writef, kind CellType, text string, port string) {
	name := htmlize(text)
	title := txt2title(text)
	props := getProperties(kind)

	if len(port) > 0 {
		port = "PORT=\"" + port + "\" "
	}

	out("<TD BGCOLOR=\"%s\" %sALIGN=\"%s\" TITLE=\"%s\">%s</TD>",
		props[background], port, props[alignment], title, formattedText(kind, name))

}

func getProperty(kind CellType, key string) string {
	if properties := getProperties(kind); len(properties) > 0 {
		if value, found := properties[key]; found {
			return value
		}
	}
	return ""
}

func getProperties(kind CellType) m2s {
	if properties, found := cellTypeProperties[kind]; found {
		return copyMap(properties)
	}

	result := m2s{
		background: "#a6cee3",
		alignment:  "right",
	}
	return result
	/*
		switch kind {
		case Header:
			result[background] = "cornsilk"
			result[alignment] = "right"
		case Footer:
			result[background] = "#cecece"
			result[alignment] = "left"
		case Key:
			result[background] = "#a6cee3"
			result[alignment] = "right"
		case Value:
			result[background] = "#e3a6ce"
			result[alignment] = "left"
		case Blank:
			result[background] = "#ffffff"
			result[alignment] = "right"
		case Info:
			result[background] = "#cccccc"
			result[alignment] = "left"
		case Type:
			result[background] = "bisque"
			result[alignment] = "right"
		case Frame:
			result[background] = "#fffaf0"
			result["border"] = "1"
			result["cellborder"] = "0"
			result["cellspacing"] = "0"
		case ExternalResolver:
			result[background] = "burlywood" // "chocolate"
		case StringResolver:
			result[background] = "darkseagreen" // "darkgoldenrod1"

		default:
			warning("unhandled CellType (%v)", kind)
			result[background] = "#0000ff"
		}
		return result
	*/
}

func customize(original m2s, name string) m2s {
	if color, found := GetColor(name); found {
		original[background] = color
	}

	/*
		switch name {
		case "struct: ast.FuncType":
			original[background] = "olive"

		case "struct: ast.GoStmt":
			original[background] = "firebrick1"

		case "struct: ast.CallExpr":
			original[background] = "fuchsia"

		case "struct: ast.DeferStmt":
			original[background] = "darkorange"
		}
	*/
	return original
}

func formattedText(kind CellType, origin string) string {

	switch getProperty(kind, text) {
	case "bold":
		return "<b>" + origin + "</b>"
	case "italic":
		return "<i>" + origin + "</i>"
	case "underline":
		return "<u>" + origin + "</u>"
	}
	return origin
}

var (
	cellTypeName = map[CellType]string{
		Header:           "header",
		Footer:           "footer",
		Key:              "key",
		Value:            "value",
		Blank:            "blank",
		Info:             "info",
		Type:             "type",
		Frame:            "frame",
		Default:          "default",
		ExternalResolver: "resolver.external",
		StringResolver:   "resolver.internal",
		InfoFrame:        "info.frame",
		InfoHeader:       "info.header",
		InfoKey:          "info.key",
		InfoValue:        "info.value",
	}

	cellTypeProperties = map[CellType]m2s{
		Default: m2s{
			background: "#a6cee3",
			alignment:  "right",
		},

		Header: m2s{
			background: "cornsilk",
			alignment:  "right",
		},
		Footer: m2s{
			background: "#cecece",
			alignment:  "left",
		},
		Key: m2s{
			background: "#a6cee3",
			alignment:  "right",
		},
		Value: m2s{
			background: "#e3a6ce",
			alignment:  "left",
		},
		Blank: m2s{
			background: "#ffffff",
			alignment:  "right",
		},
		Info: m2s{
			background: "#cccccc",
			alignment:  "left",
		},
		Type: m2s{
			background: "bisque",
			alignment:  "right",
		},
		Frame: m2s{
			background:    "#fffaf0",
			alignment:     "left",
			"border":      "1",
			"cellborder":  "0",
			"cellspacing": "0",
		},
		ExternalResolver: m2s{
			alignment:  "left",
			background: "burlywood", // "chocolate"
		},
		StringResolver: m2s{
			alignment:  "left",
			background: "darkseagreen", // "darkgoldenrod1"
		},

		InfoFrame: m2s{
			background:    "transparent",
			"border":      "0",
			"cellborder":  "0",
			"cellspacing": "0",
			"fontsize":    "7",
		},
		InfoHeader: m2s{
			alignment:  "left",
			background: "gray100",
		},
		InfoKey: m2s{
			alignment:  "right",
			background: "ghostwhite",
		},
		InfoValue: m2s{
			alignment:  "left",
			background: "floralwhite",
		},
	}

	connectorProperties = map[connectionStyle]m2s{
		connDefault: m2s{
			"color": "black",
			"port":  "w",
		},
		connPointer: m2s{
			"color": "red",
			"port":  "w",
		},
		connArray: m2s{
			"color": "blue",
		},
		connInner: m2s{
			"color":    "green",
			"port":     "w",
			"weight":   "3",
			"penwidth": "3",
		},
	}
)

/*

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

*/

func (ct CellType) String() string {
	if name, found := cellTypeName[ct]; found {
		return name
	}
	warning("there is no name defined for cell.type (%v)", int(ct))
	return fmt.Sprintf("#%d", int(ct))
}

func string2CellType(txt string) (CellType, bool) {
	txt = strings.ToLower(txt)
	for ct, name := range cellTypeName {
		if name == txt {
			return ct, true
		}
	}
	return Default, false
}

func loadProperties() {
	for _, name := range []string{"./" + optionsFileName, "~/" + optionsFileName} {
		loadPropertiesFrom(name)
	}
}

func loadPropertiesFrom(name string) {
	raw, err := os.ReadFile(name)
	if err != nil {
		return
	}
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return
	}
}

func correctColor(original string) string {
	switch original {
	case "none", "clear", "empty", "":
		return "transparent"
	}
	return original
}

func applyProperties(from map[string]m2s) {
	for key, value := range from {
		if ct, found := string2CellType(key); found {
			if len(cellTypeProperties[ct]) == 0 {
				cellTypeProperties[ct] = make(m2s)
			}
			for attribute, color := range value {
				cellTypeProperties[ct][attribute] = correctColor(color)
			}
		}
	}
}

func string2connectionStyle(name string) (connectionStyle, bool) {
	switch strings.ToLower(name) {
	case "default":
		return connDefault, true
	case "pointer":
		return connPointer, true
	case "array":
		return connArray, true
	case "inner":
		return connInner, true
	}
	return connDefault, false
}

func applyConnectors(from map[string]m2s) {
	for key, value := range from {
		if cs, found := string2connectionStyle(key); found {
			if len(connectorProperties[cs]) == 0 {
				connectorProperties[cs] = make(m2s)
			}
			for attribute, v := range value {
				if attribute == "port" {
					switch strings.ToLower(v) {
					case "w", "west", "left":
						v = "w"
					case "n", "north", "up":
						v = "n"
					case "e", "east", "right":
						v = "e"
					case "":
						v = ""
					default:
						continue
					}
				}
				connectorProperties[cs][attribute] = v
			}
		}
	}
}
