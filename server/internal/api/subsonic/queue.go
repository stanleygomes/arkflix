package subsonic

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/stanleygomes/arkflix/server/internal/store"
)

// queue é a fila salva de um usuário.
type queue struct {
	ids       []string
	current   int
	position  int64
	changed   time.Time
	changedBy string
}

func (a *API) loadQueue(c *call) (queue, error) {
	var q queue
	var raw, changed string
	err := a.d.Store.DB.QueryRowContext(c.ctx, `SELECT items, current, position, changed_at, changed_by FROM subsonic_playqueue WHERE user_id = ?`,
		c.user.ID).Scan(&raw, &q.current, &q.position, &changed, &q.changedBy)
	if errors.Is(err, sql.ErrNoRows) {
		return q, nil
	}
	if err != nil {
		return q, err
	}
	json.Unmarshal([]byte(raw), &q.ids)
	q.changed = parseT(changed)
	return q, nil
}

func (a *API) saveQueue(c *call, ids []string, current int, position int64) error {
	clean := make([]string, len(ids))
	for i, id := range ids {
		clean[i] = cleanID(id)
	}
	raw, _ := json.Marshal(clean)
	_, err := a.d.Store.DB.ExecContext(c.ctx, `INSERT INTO subsonic_playqueue (user_id, items, current, position, changed_at, changed_by)
VALUES (?,?,?,?,?,?) ON CONFLICT(user_id) DO UPDATE SET items=excluded.items, current=excluded.current,
position=excluded.position, changed_at=excluded.changed_at, changed_by=excluded.changed_by`,
		c.user.ID, string(raw), current, position, store.Now(), c.client)
	return err
}

// queueEntries resolve as faixas; o índice atual acompanha as que sumiram.
func (a *API) queueEntries(c *call, q queue) ([]child, int, error) {
	ts, err := a.tracksByIDs(c.ctx, c.user.ID, q.ids)
	if err != nil {
		return nil, 0, err
	}
	have := map[string]bool{}
	for _, t := range ts {
		have[t.ID] = true
	}
	cur := 0
	for i := 0; i < q.current && i < len(q.ids); i++ {
		if have[q.ids[i]] {
			cur++
		}
	}
	return trackChildren(ts), cur, nil
}

func (a *API) getPlayQueue(c *call) (*resp, error) {
	q, err := a.loadQueue(c)
	if err != nil {
		return nil, err
	}
	pq := &playQueue{Username: c.user.Name}
	if len(q.ids) > 0 {
		entries, cur, err := a.queueEntries(c, q)
		if err != nil {
			return nil, err
		}
		pq.Entry, pq.Position, pq.Changed, pq.ChangedBy = entries, q.position, ms(q.changed), q.changedBy
		if cur < len(entries) {
			pq.Current = entries[cur].ID
		}
	}
	res := a.newResp()
	res.PlayQueue = pq
	return res, nil
}

func (a *API) savePlayQueue(c *call) (*resp, error) {
	ids := c.p.strs("id")
	current := cleanID(c.p.str("current"))
	cur := 0
	for i, id := range ids {
		if cleanID(id) == current {
			cur = i
			break
		}
	}
	if err := a.saveQueue(c, ids, cur, int64(c.p.int("position", 0))); err != nil {
		return nil, err
	}
	return a.newResp(), nil
}

func (a *API) getPlayQueueByIndex(c *call) (*resp, error) {
	q, err := a.loadQueue(c)
	if err != nil {
		return nil, err
	}
	pq := &playQueueByIndex{Username: c.user.Name}
	if len(q.ids) > 0 {
		entries, cur, err := a.queueEntries(c, q)
		if err != nil {
			return nil, err
		}
		pq.Entry, pq.Position, pq.Changed, pq.ChangedBy = entries, q.position, ms(q.changed), q.changedBy
		if len(entries) > 0 {
			cur = min(cur, len(entries)-1)
			pq.CurrentIndex = &cur
		}
	}
	res := a.newResp()
	res.PlayQueueByIndex = pq
	return res, nil
}

func (a *API) savePlayQueueByIndex(c *call) (*resp, error) {
	ids := c.p.strs("id")
	cur := 0
	if len(ids) > 0 {
		n, err := strconv.Atoi(c.p.str("currentIndex"))
		if err != nil || n < 0 || n >= len(ids) {
			return nil, fail(errMissingParam, "missing parameter index, err: %v", err)
		}
		cur = n
	}
	if err := a.saveQueue(c, ids, cur, int64(c.p.int("position", 0))); err != nil {
		return nil, err
	}
	return a.newResp(), nil
}

// Marcadores: linha própria (comentário, datas, posição em ms) e a posição
// também vai pro userdata, pro Jellyfin continuar de onde parou.

func (a *API) getBookmarks(c *call) (*resp, error) {
	rows, err := a.d.Store.DB.QueryContext(c.ctx, `SELECT item_id, position, comment, created_at, changed_at
FROM subsonic_bookmarks WHERE user_id = ? ORDER BY changed_at DESC`, c.user.ID)
	if err != nil {
		return nil, err
	}
	type bm struct {
		id, comment, created, changed string
		pos                           int64
	}
	var list []bm
	for rows.Next() {
		var b bm
		if err := rows.Scan(&b.id, &b.pos, &b.comment, &b.created, &b.changed); err != nil {
			rows.Close()
			return nil, err
		}
		list = append(list, b)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	ids := make([]string, len(list))
	for i, b := range list {
		ids[i] = b.id
	}
	ts, err := a.tracksByIDs(c.ctx, c.user.ID, ids)
	if err != nil {
		return nil, err
	}
	byID := map[string]*track{}
	for _, t := range ts {
		byID[t.ID] = t
	}
	out := &bookmarks{}
	for _, b := range list {
		t := byID[b.id]
		if t == nil {
			continue
		}
		out.Bookmark = append(out.Bookmark, bookmark{Entry: t.child(), Position: b.pos, Username: c.user.Name,
			Comment: b.comment, Created: ms(parseT(b.created)), Changed: ms(parseT(b.changed))})
	}
	res := a.newResp()
	res.Bookmarks = out
	return res, nil
}

func (a *API) createBookmark(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	if a.kindOf(c.ctx, id) != "track" {
		return nil, fail(errNotFound, "Song not found")
	}
	id = cleanID(id)
	pos := int64(c.p.int("position", 0))
	now := store.Now()
	_, err = a.d.Store.DB.ExecContext(c.ctx, `INSERT INTO subsonic_bookmarks (user_id, item_id, position, comment, created_at, changed_at)
VALUES (?,?,?,?,?,?) ON CONFLICT(user_id, item_id) DO UPDATE SET position=excluded.position, comment=excluded.comment,
changed_at=excluded.changed_at`, c.user.ID, id, pos, c.p.str("comment"), now, now)
	if err != nil {
		return nil, err
	}
	_, err = a.d.Store.UpdateUserData(c.ctx, c.user.ID, id, func(d *store.UserData) {
		d.Position = float64(pos) / 1000
		d.LastPlayed = time.Now().UTC()
	})
	if err != nil {
		return nil, err
	}
	return a.newResp(), nil
}

func (a *API) deleteBookmark(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	id = cleanID(id)
	res, err := a.d.Store.DB.ExecContext(c.ctx, `DELETE FROM subsonic_bookmarks WHERE user_id = ? AND item_id = ?`, c.user.ID, id)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		if _, err := a.d.Store.UpdateUserData(c.ctx, c.user.ID, id, func(d *store.UserData) { d.Position = 0 }); err != nil {
			return nil, err
		}
	}
	return a.newResp(), nil
}
