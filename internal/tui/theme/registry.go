package theme

import (
	"strings"
)

var BuiltinThemes = []Palette{
	System,
	CatppuccinMocha,
	CatppuccinMacchiato,
	CatppuccinLatte,
	TokyoNight,
	GruvboxDark,
	Nord,
	Dracula,
	RosePine,
	Kanagawa,
	Everforest,
	OneDark,
}

var Current = BuiltinThemes[0]

func GetAll() []Palette {
	all := make([]Palette, len(BuiltinThemes))
	copy(all, BuiltinThemes)

	custom, _ := Load("")
	all = append(all, custom...)
	return all
}

func Find(id string) Palette {
	idClean := strings.ToLower(strings.TrimSpace(id))
	if idClean == "" || idClean == "solarized-dark" {
		return BuiltinThemes[0]
	}

	for _, t := range GetAll() {
		if strings.ToLower(t.ID) == idClean || strings.ToLower(t.Name) == idClean {
			return t
		}
	}
	return BuiltinThemes[0]
}
