package importer

import (
	"context"
	"fmt"
	"io"
)

// ListSink recebe as listas de cada usuário (playlists do Navidrome,
// coleções do RomM) já com os ids do acervo, na ordem. Quem implementa
// deve juntar por (usuário, nome): rodar o import de novo não pode
// duplicar a lista.
type ListSink interface {
	AddList(ctx context.Context, userID, name string, itemIDs []string) error
}

// pendingLists só anota as listas: o destino (internal/lists) ainda não
// existe neste ramo.
type pendingLists struct{ w io.Writer }

func (p pendingLists) AddList(_ context.Context, userID, name string, itemIDs []string) error {
	fmt.Fprintf(p.w, "lista pendente (sem destino ainda): usuário %s, %q, %d itens\n", userID, name, len(itemIDs))
	return nil
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
