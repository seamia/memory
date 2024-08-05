// github.com/seamia/memory

package memory

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/user"
	"path"
	"sync"
	"syscall"
)

const (
	optionsFileName = "seamia.memory.options"

	optionAllowExternalResolver = true
	optionAllowStringResolver   = true
	optionAllowMetadata         = true
)

type Settings struct {
	MaxStringLength          int                          `json:"maxStringLength"`
	MaxSliceLength           int                          `json:"maxSliceLength"`
	MaxMapEntries            int                          `json:"maxMapEntries"`
	Discard                  map[string]int               `json:"discard"`
	Substitute               map[string]map[string]string `json:"substitute"`
	Colors                   interface{}                  `json:"colors"`
	SuppresHeader            bool                         `json:"suppresHeader"`
	SuppresInfo              bool                         `json:"suppresInfo"`
	CollapsePointerNodes     bool                         `json:"collapsePointerNodes"`
	CollapseSingleSliceNodes bool                         `json:"collapseSingleSliceNodes"`
	ColorBackground          string                       `json:"colorBackground"` // transparent
	ColorDefault             string                       `json:"colorDefault"`    // whitesmoke
	FontName                 string                       `json:"fontName"`
	FontSize                 string                       `json:"fontSize"`
	LinkPointer              string                       `json:"link.pointer"`
	LinkArray                string                       `json:"link.array"`
	PropsData                interface{}                  `json:"properties"`
	Props                    map[string]map[string]string `json:"-"`
	Connectors               map[string]map[string]string `json:"connectors"`

	LoadedFrom string `json:"-"`
}

var (
	settings = Settings{
		MaxStringLength:          64,
		MaxSliceLength:           100,
		MaxMapEntries:            32,
		CollapsePointerNodes:     false, // true,
		CollapseSingleSliceNodes: false, // true,
		ColorBackground:          "transparent",
		ColorDefault:             "whitesmoke",
		FontName:                 "Cascadia Code",
		FontSize:                 "10",
	}

	guard          sync.Mutex
	customColors   = make(map[string]string)
	settingsLoaded bool
)

func Options() *Settings {
	if !settingsLoaded {
		guard.Lock()
		defer guard.Unlock()

		if !settingsLoaded {
			settingsLoaded = true

			trace("loading settings")
			loadedFrom := "./" + optionsFileName
			data, err := os.ReadFile(loadedFrom)
			if err != nil {
				warning("failed to open file (%s): %v", loadedFrom, err)
				loadedFrom = homeDir(optionsFileName)
				data, err = os.ReadFile(loadedFrom)
			}

			if err == nil {
				if err := json.Unmarshal(data, &settings); err != nil {
					warning("error while loading config file (%v)", err)
				} else {
					settings.LoadedFrom = loadedFrom
				}
			} else {
				if perr, found := err.(*fs.PathError); found && perr.Err == syscall.ERROR_FILE_NOT_FOUND {
					// it is okay to have config file missing --> do not report this fact
				} else {
					warning("error while reading config file (%v)", err)
				}
			}

			settings.ColorBackground = correctColor(settings.ColorBackground)
			settings.ColorDefault = correctColor(settings.ColorDefault)

			loadColors()

			applyProperties(loadProps())
			applyConnectors(settings.Connectors)
		}
	}
	return &settings
}

/*
the "colors" entry of the config file can be of these three types:
1. string - name of the file containing color definitions
2. []string - names of the files containing color definitions to be combined
3. map[string]string - actual color definitions
*/
func loadColors() {
	input := settings.Colors
	if input == nil {
		return
	}
	trace("loading custom colors")

	var list []string
	switch actual := input.(type) {
	case string:
		list = append(list, actual)

	case []interface{}:
		for _, entry := range actual {
			if txt, converts := entry.(string); converts {
				list = append(list, txt)
			}
		}
	case map[string]interface{}:
		for k, v := range actual {
			if txt, converts := v.(string); converts {
				customColors[k] = txt
			}
		}
		return

	default:
		warning("unrecognized format of Colors section of the config file (%v)", actual)
		return
	}

	for _, entry := range list {
		trace("processing color file (%s)", entry)
		data, err := os.ReadFile(entry)
		if err == nil {
			var loaded map[string]string
			if err := json.Unmarshal(data, &loaded); err == nil {
				for k, v := range loaded {
					customColors[k] = correctColor(v)
				}
			} else {
				warning("error (%v) while processing config file (%v)", err, entry)
			}
		} else {
			warning("error (%v) while loading config file (%v)", err, entry)
		}
	}
}

func GetColor(name string) (string, bool) {
	if len(name) == 0 || len(customColors) == 0 {
		return "", false
	}
	result, found := customColors[name]

	if found {
		trace("found custom color for (%s): %s", name, result)
	}

	return result, found
}

func homeDir(name string) string {
	if current, err := user.Current(); err == nil {
		return path.Join(current.HomeDir, name)
	}
	return name
}

func loadProps() map[string]map[string]string {

	result := map[string]map[string]string{}
	input := settings.PropsData
	if input == nil {
		return result
	}
	trace("loading custom props")

	var list []string
	switch actual := input.(type) {
	case string:
		// a single external config
		list = append(list, actual)

	case []interface{}:
		// multiple external configs
		for _, entry := range actual {
			if txt, converts := entry.(string); converts {
				list = append(list, txt)
			}
		}
	case map[string]interface{}:
		// inline definition
		for k, v := range actual {
			if len(result[k]) == 0 {
				result[k] = make(map[string]string)
			}
			if values, converts := v.(map[string]interface{}); converts {
				for key, value := range values {
					if txt, converts := value.(string); converts {
						result[k][key] = correctColor(txt)
					}
				}
			}
		}
		return result

	default:
		warning("unrecognized format of props section of the config file (%v)", actual)
		return result
	}

	for _, entry := range list {
		trace("processing props file (%s)", entry)
		data, err := os.ReadFile(entry)
		if err == nil {
			var loaded map[string]map[string]string
			if err := json.Unmarshal(data, &loaded); err == nil {

				for k, values := range loaded {
					if len(result[k]) == 0 {
						result[k] = make(map[string]string)
					}
					for key, value := range values {
						result[k][key] = correctColor(value)
					}
				}
			} else {
				warning("error (%v) while processing config file (%v)", err, entry)
			}
		} else {
			warning("error (%v) while loading config file (%v)", err, entry)
		}
	}
	return result
}
