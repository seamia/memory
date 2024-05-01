// github.com/seamia/memory

package memory

import (
	"fmt"
	"reflect"
)

var (
	structMapping        = make(map[string]string)
	structMappingCounter int
)

func getStructTypeName(uType reflect.Type) string {
	structTypeName := uType.Name()
	if len(structTypeName) == 0 {
		structTypeName = uType.String()

		if previous, exist := structMapping[structTypeName]; exist {
			return previous
		}

		newName := fmt.Sprintf("anonymous-%v", structMappingCounter)
		structMappingCounter++

		structMapping[structTypeName] = newName
		return newName
	}
	return structTypeName
}
