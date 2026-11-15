package library

import (
	"maps"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// MetaKey é a chave de Extra onde o pacote metadata anota o que veio da
// internet: {"source", "fetched", "missed", "fields": [...], "keys": [...]}.
// fields são colunas do item (overview, genres, year, premiere, image,
// backdrop, logo, name); keys são chaves de Extra (tmdb_id, rating...).
const MetaKey = "meta"

// keepMeta mantém num rescan o que veio da internet: campo que o disco
// deixou vazio fica com o valor anterior; campo que o disco passou a ter
// (nfo, imagem ao lado, tag) ganha e sai da lista. Sem isso, arquivo que
// muda perderia os metadados.
func keepMeta(old, it *store.Item) {
	if old == nil {
		return
	}
	m, ok := old.Extra[MetaKey].(map[string]any)
	if !ok {
		return
	}
	m = maps.Clone(m)
	kept := []any{}
	for _, f := range strs(m["fields"]) {
		if restore(f, old, it) {
			kept = append(kept, f)
		}
	}
	m["fields"] = kept
	if it.Extra == nil {
		it.Extra = map[string]any{}
	}
	for _, k := range strs(m["keys"]) {
		if v, ok := old.Extra[k]; ok {
			if _, has := it.Extra[k]; !has {
				it.Extra[k] = v
			}
		}
	}
	it.Extra[MetaKey] = m
}

// restore copia o campo f do item antigo se o disco não trouxe nada.
func restore(f string, old, it *store.Item) bool {
	str := func(dst *string, src string) bool {
		if *dst != "" {
			return false
		}
		*dst = src
		return true
	}
	switch f {
	case "name":
		// só o nome genérico de episódio ("Episódio 3") é trocado
		if it.Kind != store.KindEpisode || it.Name != EpisodeName(it.Index) {
			return false
		}
		it.Name = old.Name
		return true
	case "overview":
		return str(&it.Overview, old.Overview)
	case "premiere":
		return str(&it.Premiere, old.Premiere)
	case "image":
		return str(&it.Image, old.Image)
	case "backdrop":
		return str(&it.Backdrop, old.Backdrop)
	case "logo":
		return str(&it.Logo, old.Logo)
	case "year":
		if it.Year != 0 {
			return false
		}
		it.Year = old.Year
		return true
	case "genres":
		if len(it.Genres) > 0 {
			return false
		}
		it.Genres = old.Genres
		return true
	}
	return false
}

func strs(v any) []string {
	var out []string
	switch l := v.(type) {
	case []any:
		for _, x := range l {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
	case []string:
		out = l
	}
	return out
}
