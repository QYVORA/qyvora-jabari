package i18n

import (
	"fmt"
	"strings"
)

var locale = "en"

type Translations map[string]map[string]string

var translations = Translations{
	"en": {},
}

func SetLocale(l string) {
	if _, ok := translations[l]; ok {
		locale = l
	}
}

func T(key string, args ...any) string {
	table, ok := translations[locale]
	if !ok {
		table = translations["en"]
	}

	tmpl, ok := table[key]
	if !ok {
		return key
	}

	result := tmpl
	for i := 0; i < len(args); i++ {
		old := fmt.Sprintf("{%d}", i)
		new := fmt.Sprintf("%v", args[i])
		result = strings.ReplaceAll(result, old, new)
	}
	return result
}
