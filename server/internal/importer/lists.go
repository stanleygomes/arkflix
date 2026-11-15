package importer

import (
	"context"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/lists"
)

// ListSink recebe as listas de cada usuário (playlists do Navidrome,
// coleções do RomM) já com os ids do acervo, na ordem. Quem implementa
// deve juntar por (usuário, nome): rodar o import de novo não pode
// duplicar a lista.
type ListSink interface {
	AddList(ctx context.Context, userID, name string, itemIDs []string) error
}

// storeLists grava nas listas do servidor (internal/lists): junta por
// (usuário, nome), então rodar de novo só acrescenta o que faltar.
type storeLists struct{ l *lists.Lists }

func (s storeLists) AddList(ctx context.Context, userID, name string, itemIDs []string) error {
	own, err := s.l.Own(ctx, userID)
	if err != nil {
		return err
	}
	var target *lists.List
	for _, l := range own {
		if strings.EqualFold(l.Name, strings.TrimSpace(name)) {
			target = l
			break
		}
	}
	if target == nil {
		if target, err = s.l.Create(ctx, userID, name); err != nil {
			return err
		}
	}
	return s.l.Add(ctx, target.ID, itemIDs)
}

// list manda a lista pro destino (no dry-run só conta).
func (im *Importer) list(ctx context.Context, u userRef, name string, ids []string) error {
	im.rep.Add("listas", 1)
	im.rep.Add("itens em listas", len(ids))
	if im.Dry {
		return nil
	}
	return im.Lists.AddList(ctx, u.ID, name, ids)
}
