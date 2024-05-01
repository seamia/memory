// github.com/seamia/memory

package memory

type (
	CellType int
	Writef   func(format string, arg ...interface{})
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

	background = "bgcolor"
	alignment  = "align"
)

func valueCellType(isNil bool) CellType {
	if isNil {
		return Blank
	}
	return Value
}

func render(out Writef, kind CellType, text string, port string) {
	// name := text // $$$$$$$$$$$$$$$$$
	name := htmlize(text)
	title := txt2title(text)
	props := getProperties(kind)

	if len(port) > 0 {
		port = "PORT=\"" + port + "\" "
	}

	out("<TD BGCOLOR=\"%s\" %sALIGN=\"%s\" TITLE=\"%s\"><i>%s</i></TD>",
		props[background], port, props[alignment], title, name)

}

func getProperties(kind CellType) map[string]string {
	result := map[string]string{
		background: "#a6cee3",
		alignment:  "right",
	}

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
}

func customize(original map[string]string, name string) map[string]string {
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
	return original

}

func formattedText(kind CellType, origin string) string {
	switch kind {
	case Header:
		return "<b>" + origin + "</b>"
	}
	return origin
}
