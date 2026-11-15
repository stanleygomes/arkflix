package subsonic

import (
	"context"
	"database/sql"
	"slices"
	"strconv"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// plRow é uma playlist do banco (sem as faixas).
type plRow struct {
	id, userID, owner, name, comment string
	public                           bool
	created, changed                 string
}

const plSelect = `SELECT p.id, p.user_id, COALESCE(us.name,''), p.name, p.comment, p.public, p.created_at, p.changed_at
FROM subsonic_playlists p LEFT JOIN users us ON us.id = p.user_id `

func (a *API) plRows(ctx context.Context, tail string, args ...any) ([]plRow, error) {
	rows, err := a.d.Store.DB.QueryContext(ctx, plSelect+tail, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []plRow
	for rows.Next() {
		var p plRow
		if err := rows.Scan(&p.id, &p.userID, &p.owner, &p.name, &p.comment, &p.public, &p.created, &p.changed); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// visible: a playlist é do usuário ou é pública.
func (a *API) plLoad(c *call, id string) (plRow, error) {
	ps, err := a.plRows(c.ctx, `WHERE p.id = ? AND (p.user_id = ? OR p.public = 1)`, cleanID(id), c.user.ID)
	if err != nil {
		return plRow{}, err
	}
	if len(ps) == 0 {
		return plRow{}, fail(errNotFound, "playlist not found")
	}
	return ps[0], nil
}

func (a *API) plItemIDs(ctx context.Context, id string) ([]string, error) {
	rows, err := a.d.Store.DB.QueryContext(ctx, `SELECT item_id FROM subsonic_playlist_items WHERE playlist_id = ? ORDER BY pos`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (a *API) plBuild(c *call, p plRow, ts []*track) playlist {
	pl := playlist{ID: p.id, Name: p.name, Comment: p.comment, Public: p.public, Owner: p.owner,
		Created: ms(parseT(p.created)), Changed: ms(parseT(p.changed)), SongCount: len(ts),
		Readonly: p.userID != c.user.ID}
	var dur float64
	for _, t := range ts {
		dur += t.Duration
		if pl.CoverArt == "" {
			pl.CoverArt = t.cover()
		}
	}
	pl.Duration = int(dur)
	return pl
}

func (a *API) getPlaylists(c *call) (*resp, error) {
	ps, err := a.plRows(c.ctx, `WHERE p.user_id = ? OR p.public = 1 ORDER BY p.name COLLATE NOCASE`, c.user.ID)
	if err != nil {
		return nil, err
	}
	out := &playlists{}
	for _, p := range ps {
		ids, err := a.plItemIDs(c.ctx, p.id)
		if err != nil {
			return nil, err
		}
		ts, err := a.tracksByIDs(c.ctx, c.user.ID, ids)
		if err != nil {
			return nil, err
		}
		out.Playlist = append(out.Playlist, a.plBuild(c, p, ts))
	}
	res := a.newResp()
	res.Playlists = out
	return res, nil
}

func (a *API) playlistResp(c *call, id string) (*resp, error) {
	p, err := a.plLoad(c, id)
	if err != nil {
		return nil, err
	}
	ids, err := a.plItemIDs(c.ctx, p.id)
	if err != nil {
		return nil, err
	}
	ts, err := a.tracksByIDs(c.ctx, c.user.ID, ids)
	if err != nil {
		return nil, err
	}
	res := a.newResp()
	res.Playlist = &playlistFull{playlist: a.plBuild(c, p, ts), Entry: trackChildren(ts)}
	return res, nil
}

func (a *API) getPlaylist(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	return a.playlistResp(c, id)
}

// own carrega a playlist e exige que seja do usuário (senão 50).
func (a *API) own(c *call, id string) (plRow, error) {
	p, err := a.plLoad(c, id)
	if err != nil {
		return p, err
	}
	if p.userID != c.user.ID {
		return p, fail(errNotAllowed, "")
	}
	return p, nil
}

func (a *API) plWrite(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := a.d.Store.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func setItems(ctx context.Context, tx *sql.Tx, plID string, ids []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM subsonic_playlist_items WHERE playlist_id = ?`, plID); err != nil {
		return err
	}
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, `INSERT INTO subsonic_playlist_items (playlist_id, pos, item_id) VALUES (?,?,?)`, plID, i, cleanID(id)); err != nil {
			return err
		}
	}
	_, err := tx.ExecContext(ctx, `UPDATE subsonic_playlists SET changed_at = ? WHERE id = ?`, store.Now(), plID)
	return err
}

// createPlaylist cria; com playlistId troca as faixas da existente.
func (a *API) createPlaylist(c *call) (*resp, error) {
	plID, name, songIDs := c.p.str("playlistId"), c.p.str("name"), c.p.strs("songId")
	if plID == "" && name == "" {
		return nil, fail(errMissingParam, "required parameter name is missing")
	}
	if plID != "" {
		p, err := a.own(c, plID)
		if err != nil {
			return nil, err
		}
		err = a.plWrite(c.ctx, func(tx *sql.Tx) error {
			if name != "" {
				if _, err := tx.ExecContext(c.ctx, `UPDATE subsonic_playlists SET name = ? WHERE id = ?`, name, p.id); err != nil {
					return err
				}
			}
			return setItems(c.ctx, tx, p.id, songIDs)
		})
		if err != nil {
			return nil, err
		}
		return a.playlistResp(c, p.id)
	}
	id := auth.RandomHex(16)
	err := a.plWrite(c.ctx, func(tx *sql.Tx) error {
		now := store.Now()
		if _, err := tx.ExecContext(c.ctx, `INSERT INTO subsonic_playlists (id, user_id, name, created_at, changed_at) VALUES (?,?,?,?,?)`,
			id, c.user.ID, name, now, now); err != nil {
			return err
		}
		return setItems(c.ctx, tx, id, songIDs)
	})
	if err != nil {
		return nil, err
	}
	return a.playlistResp(c, id)
}

// updatePlaylist: name, comment, public, tira por índice e acrescenta no fim.
func (a *API) updatePlaylist(c *call) (*resp, error) {
	plID, err := c.p.need("playlistId")
	if err != nil {
		return nil, err
	}
	p, err := a.own(c, plID)
	if err != nil {
		return nil, err
	}
	ids, err := a.plItemIDs(c.ctx, p.id)
	if err != nil {
		return nil, err
	}
	var drop []int
	for _, s := range c.p.strs("songIndexToRemove") {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 && n < len(ids) {
			drop = append(drop, n)
		}
	}
	var keep []string
	for i, id := range ids {
		if !slices.Contains(drop, i) {
			keep = append(keep, id)
		}
	}
	keep = append(keep, c.p.strs("songIdToAdd")...)
	err = a.plWrite(c.ctx, func(tx *sql.Tx) error {
		if c.p.has("name") {
			if _, err := tx.ExecContext(c.ctx, `UPDATE subsonic_playlists SET name = ? WHERE id = ?`, c.p.str("name"), p.id); err != nil {
				return err
			}
		}
		if c.p.has("comment") {
			if _, err := tx.ExecContext(c.ctx, `UPDATE subsonic_playlists SET comment = ? WHERE id = ?`, c.p.str("comment"), p.id); err != nil {
				return err
			}
		}
		if c.p.has("public") {
			if _, err := tx.ExecContext(c.ctx, `UPDATE subsonic_playlists SET public = ? WHERE id = ?`, c.p.bool("public", false), p.id); err != nil {
				return err
			}
		}
		return setItems(c.ctx, tx, p.id, keep)
	})
	if err != nil {
		return nil, err
	}
	return a.newResp(), nil
}

func (a *API) deletePlaylist(c *call) (*resp, error) {
	id, err := c.p.need("id")
	if err != nil {
		return nil, err
	}
	p, err := a.own(c, id)
	if err != nil {
		return nil, err
	}
	if _, err := a.d.Store.DB.ExecContext(c.ctx, `DELETE FROM subsonic_playlists WHERE id = ?`, p.id); err != nil {
		return nil, err
	}
	return a.newResp(), nil
}
