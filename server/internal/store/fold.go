package store

import (
	"database/sql/driver"
	"strings"
	"unicode"

	"modernc.org/sqlite"
)

// Fold deixa o texto comparável em busca: minúsculo e sem acento
// ("Série" e "serie" batem). No SQL vira a função fold().
func Fold(s string) string {
	return strings.Map(func(r rune) rune {
		r = unicode.ToLower(r)
		if b, ok := accents[r]; ok {
			return b
		}
		return r
	}, s)
}

var accents = func() map[rune]rune {
	m := map[rune]rune{}
	for base, list := range map[rune]string{
		'a': "áàâãäåā", 'e': "éèêëē", 'i': "íìîïī", 'o': "óòôõöøō", 'u': "úùûüū",
		'c': "ç", 'n': "ñ", 'y': "ýÿ",
	} {
		for _, r := range list {
			m[r] = base
		}
	}
	return m
}()

func init() {
	sqlite.MustRegisterDeterministicScalarFunction("fold", 1, func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
		s, _ := args[0].(string)
		return Fold(s), nil
	})
}
