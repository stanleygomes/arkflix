// Package lists guarda as listas de cada usuário: itens de qualquer tipo
// (filme, série, episódio, ROM, álbum, faixa), em ordem. É a fonte única:
// a API do arkflix mostra a lista inteira, o Jellyfin como Playlist, o
// RomM como coleção (só as ROMs) e o Subsonic como playlist (só as faixas).
package lists

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Tabelas. Os itens apontam pro id do acervo sem FK (sobrevivem a um
// rescan que tira e põe o arquivo); entry é o id da entrada (o Jellyfin
// tira da playlist por ele, e o mesmo item pode repetir no Subsonic).
func init() {
	store.Register(store.Migration{Name: "lists/001-lists", SQL: `
CREATE TABLE lists (
  rid        INTEGER PRIMARY KEY AUTOINCREMENT, -- id numérico (coleção do RomM)
  id         TEXT NOT NULL UNIQUE,              -- 32 hex
  user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  comment    TEXT NOT NULL DEFAULT '',
  public     INTEGER NOT NULL DEFAULT 0,        -- outros usuários veem (só leitura)
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX lists_user ON lists(user_id);
CREATE TABLE lists_items (
  entry    TEXT PRIMARY KEY,
  list_id  TEXT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
  pos      INTEGER NOT NULL,
  item_id  TEXT NOT NULL,
  added_at TEXT NOT NULL
);
CREATE INDEX lists_items_list ON lists_items(list_id, pos);
CREATE INDEX lists_items_item ON lists_items(item_id);`})
}

// List é uma lista (sem os itens).
type List struct {
	RID     int64
	ID      string
	UserID  string
	Owner   string // nome do dono
	Name    string
	Comment string
	Public  bool
	Created time.Time
	Updated time.Time
}

// Entry é um item dentro da lista. Kind vazio = o item sumiu do acervo.
type Entry struct {
	ID      string // id da entrada
	ItemID  string
	Kind    string
	AddedAt time.Time
}

// Lists lê e grava as listas.
type Lists struct{ st *store.Store }

// New usa o banco do servidor.
func New(st *store.Store) *Lists { return &Lists{st: st} }

// ErrName vem quando o nome está vazio.
var ErrName = errors.New("nome vazio")

const cols = `l.rid, l.id, l.user_id, COALESCE(u.name,''), l.name, l.comment, l.public, l.created_at, l.updated_at`
const from = ` FROM lists l LEFT JOIN users u ON u.id = l.user_id `

func parseTime(s string) time.Time {
	for _, f := range []string{time.RFC3339Nano, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

func (s *Lists) query(ctx context.Context, tail string, args ...any) ([]*List, error) {
	rows, err := s.st.DB.QueryContext(ctx, `SELECT `+cols+from+tail, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*List
	for rows.Next() {
		var l List
		var created, updated string
		if err := rows.Scan(&l.RID, &l.ID, &l.UserID, &l.Owner, &l.Name, &l.Comment, &l.Public, &created, &updated); err != nil {
			return nil, err
		}
		l.Created, l.Updated = parseTime(created), parseTime(updated)
		out = append(out, &l)
	}
	return out, rows.Err()
}

func (s *Lists) one(ctx context.Context, tail string, args ...any) (*List, error) {
	ls, err := s.query(ctx, tail, args...)
	if err != nil {
		return nil, err
	}
	if len(ls) == 0 {
		return nil, store.ErrNotFound
	}
	return ls[0], nil
}

// Get busca pelo id (32 hex, aceita GUID com hífen).
func (s *Lists) Get(ctx context.Context, id string) (*List, error) {
	return s.one(ctx, `WHERE l.id = ?`, store.NormalizeID(id))
}

// ByRID busca pelo id numérico.
func (s *Lists) ByRID(ctx context.Context, rid int64) (*List, error) {
	return s.one(ctx, `WHERE l.rid = ?`, rid)
}

// Own lista as do usuário, por nome.
func (s *Lists) Own(ctx context.Context, userID string) ([]*List, error) {
	return s.query(ctx, `WHERE l.user_id = ? ORDER BY l.name COLLATE NOCASE, l.rid`, userID)
}

// Visible lista as do usuário e as públicas dos outros, por nome.
func (s *Lists) Visible(ctx context.Context, userID string) ([]*List, error) {
	return s.query(ctx, `WHERE l.user_id = ? OR l.public = 1 ORDER BY l.name COLLATE NOCASE, l.rid`, userID)
}

// CanSee: a lista é do usuário ou é pública.
func (l *List) CanSee(userID string) bool { return l.UserID == userID || l.Public }

// Create cria uma lista vazia.
func (s *Lists) Create(ctx context.Context, userID, name string) (*List, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrName
	}
	id, now := auth.RandomHex(16), store.Now()
	if _, err := s.st.DB.ExecContext(ctx, `INSERT INTO lists (id, user_id, name, created_at, updated_at) VALUES (?,?,?,?,?)`,
		id, userID, name, now, now); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Changes é o que Update troca (nil = mantém).
type Changes struct {
	Name    *string
	Comment *string
	Public  *bool
}

// Update troca nome, comentário e/ou visibilidade.
func (s *Lists) Update(ctx context.Context, id string, ch Changes) error {
	if ch.Name != nil {
		n := strings.TrimSpace(*ch.Name)
		if n == "" {
			return ErrName
		}
		ch.Name = &n
	}
	res, err := s.st.DB.ExecContext(ctx, `UPDATE lists SET name = COALESCE(?, name), comment = COALESCE(?, comment),
public = COALESCE(?, public), updated_at = ? WHERE id = ?`, ptr(ch.Name), ptr(ch.Comment), ptr(ch.Public), store.Now(), store.NormalizeID(id))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return store.ErrNotFound
	}
	return nil
}

func ptr[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}

// Delete apaga a lista (as entradas vão junto).
func (s *Lists) Delete(ctx context.Context, id string) error {
	res, err := s.st.DB.ExecContext(ctx, `DELETE FROM lists WHERE id = ?`, store.NormalizeID(id))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return store.ErrNotFound
	}
	return nil
}

type querier interface {
	QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error)
}

func entries(ctx context.Context, db querier, listID string) ([]Entry, error) {
	rows, err := db.QueryContext(ctx, `SELECT li.entry, li.item_id, COALESCE(i.kind,''), li.added_at FROM lists_items li
LEFT JOIN items i ON i.id = li.item_id WHERE li.list_id = ? ORDER BY li.pos`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		var e Entry
		var added string
		if err := rows.Scan(&e.ID, &e.ItemID, &e.Kind, &added); err != nil {
			return nil, err
		}
		e.AddedAt = parseTime(added)
		out = append(out, e)
	}
	return out, rows.Err()
}

// Entries devolve as entradas na ordem (inclusive de item que sumiu).
func (s *Lists) Entries(ctx context.Context, listID string) ([]Entry, error) {
	return entries(ctx, s.st.DB, store.NormalizeID(listID))
}

// Edit lê as entradas, aplica fn e grava a nova ordem numa transação.
// Entrada nova (ID vazio) ganha id; as que ficaram mantêm o delas.
func (s *Lists) Edit(ctx context.Context, listID string, fn func([]Entry) ([]Entry, error)) error {
	listID = store.NormalizeID(listID)
	tx, err := s.st.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM lists WHERE id = ?`, listID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return store.ErrNotFound
	}
	cur, err := entries(ctx, tx, listID)
	if err != nil {
		return err
	}
	next, err := fn(cur)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM lists_items WHERE list_id = ?`, listID); err != nil {
		return err
	}
	now := store.Now()
	for pos, e := range next {
		added := now
		if e.ID == "" {
			e.ID = auth.RandomHex(16)
		} else if !e.AddedAt.IsZero() {
			added = e.AddedAt.Format(time.RFC3339Nano)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO lists_items (entry, list_id, pos, item_id, added_at) VALUES (?,?,?,?,?)`,
			e.ID, listID, pos, store.NormalizeID(e.ItemID), added); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE lists SET updated_at = ? WHERE id = ?`, now, listID); err != nil {
		return err
	}
	return tx.Commit()
}

// Add acrescenta no fim o que ainda não está na lista (repetido é ignorado).
func (s *Lists) Add(ctx context.Context, listID string, itemIDs []string) error {
	return s.Edit(ctx, listID, func(es []Entry) ([]Entry, error) {
		for _, id := range itemIDs {
			id = store.NormalizeID(id)
			if !slices.ContainsFunc(es, func(e Entry) bool { return e.ItemID == id }) {
				es = append(es, Entry{ItemID: id})
			}
		}
		return es, nil
	})
}

// Remove tira as entradas cujo id de entrada ou de item estiver em ids.
func (s *Lists) Remove(ctx context.Context, listID string, ids []string) error {
	drop := map[string]bool{}
	for _, id := range ids {
		drop[store.NormalizeID(id)] = true
	}
	return s.Edit(ctx, listID, func(es []Entry) ([]Entry, error) {
		return slices.DeleteFunc(es, func(e Entry) bool { return drop[e.ID] || drop[e.ItemID] }), nil
	})
}

// Reorder põe primeiro os itens na ordem de itemIDs; o que não veio fica
// no fim, na ordem de antes.
func (s *Lists) Reorder(ctx context.Context, listID string, itemIDs []string) error {
	rank := map[string]int{}
	for i, id := range itemIDs {
		if _, ok := rank[store.NormalizeID(id)]; !ok {
			rank[store.NormalizeID(id)] = i
		}
	}
	return s.Edit(ctx, listID, func(es []Entry) ([]Entry, error) {
		slices.SortStableFunc(es, func(a, b Entry) int {
			ra, oka := rank[a.ItemID]
			rb, okb := rank[b.ItemID]
			switch {
			case oka && okb:
				return ra - rb
			case oka:
				return -1
			case okb:
				return 1
			}
			return 0
		})
		return es, nil
	})
}

// Move leva a entrada (id de entrada ou de item) pra posição idx.
func (s *Lists) Move(ctx context.Context, listID, entryID string, idx int) error {
	entryID = store.NormalizeID(entryID)
	return s.Edit(ctx, listID, func(es []Entry) ([]Entry, error) {
		i := slices.IndexFunc(es, func(e Entry) bool { return e.ID == entryID || e.ItemID == entryID })
		if i < 0 {
			return nil, store.ErrNotFound
		}
		e := es[i]
		es = slices.Delete(es, i, i+1)
		idx = min(max(idx, 0), len(es))
		return slices.Insert(es, idx, e), nil
	})
}

// SetKinds troca só as entradas de certos tipos (as faixas, pro Subsonic;
// as ROMs, pro RomM) pelos itemIDs, na mesma ordem e com repetição. As
// outras entradas ficam onde estão; os novos ocupam as vagas dos antigos
// e o que sobrar vai pro fim.
func (s *Lists) SetKinds(ctx context.Context, listID string, kinds []string, itemIDs []string) error {
	return s.Edit(ctx, listID, func(es []Entry) ([]Entry, error) {
		var out []Entry
		next := 0
		for _, e := range es {
			if !slices.Contains(kinds, e.Kind) {
				out = append(out, e)
				continue
			}
			if next < len(itemIDs) {
				if id := store.NormalizeID(itemIDs[next]); id != e.ItemID {
					e = Entry{ItemID: id} // entrada nova no lugar da velha
				}
				out = append(out, e)
				next++
			}
		}
		for _, id := range itemIDs[next:] {
			out = append(out, Entry{ItemID: id})
		}
		return out, nil
	})
}

// HasKinds diz se a lista tem algum item dos tipos, ou se está vazia (lista
// nova aparece em todo cliente; a de filmes não vira playlist de música).
func HasKinds(es []Entry, kinds ...string) bool {
	if len(es) == 0 {
		return true
	}
	return slices.ContainsFunc(es, func(e Entry) bool { return slices.Contains(kinds, e.Kind) })
}

// ItemIDs filtra as entradas pelos tipos (nenhum = todos os que existem).
func ItemIDs(es []Entry, kinds ...string) []string {
	var out []string
	for _, e := range es {
		if e.Kind != "" && (len(kinds) == 0 || slices.Contains(kinds, e.Kind)) {
			out = append(out, e.ItemID)
		}
	}
	return out
}
