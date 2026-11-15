package importer

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/auth"
	"github.com/stanleygomes/arkflix/server/internal/store"
)

// Jellyfin importa do jellyfin.db (10.11+/12.x: o library.db foi pro
// mesmo arquivo, com BaseItems e UserData em tabelas do EF Core):
//
//   - Users: nome, admin (Permissions Kind 0), desativado (Kind 2) fica de
//     fora; a senha (PBKDF2-SHA512) vai como hash legado e vira bcrypt no
//     1º login.
//   - UserData por item, casado pelo BaseItems.Path: assistido, vezes,
//     onde parou (ticks / 10^7), última vez, favorito, nota (0-10 → 0-5).
func (im *Importer) Jellyfin(ctx context.Context, file string) error {
	db, err := openSQLite(file)
	if err != nil {
		return err
	}
	defer db.Close()
	for _, t := range []string{"Users", "Permissions", "UserData", "BaseItems"} {
		if !hasTable(ctx, db, t) {
			return errors.New("jellyfin.db sem a tabela " + t + ": precisa do Jellyfin 10.11 ou mais novo (antes os itens ficavam no library.db)")
		}
	}
	if err := im.begin(ctx, "jellyfin"); err != nil {
		return err
	}
	users, err := im.jellyfinUsers(ctx, db)
	if err != nil {
		return err
	}
	if err := im.jellyfinUserData(ctx, db, users); err != nil {
		return err
	}
	return im.finish(ctx)
}

func (im *Importer) jellyfinUsers(ctx context.Context, db *sql.DB) (map[string]userRef, error) {
	rows, err := db.QueryContext(ctx, `SELECT u.Id, u.Username, COALESCE(u.Password,''),
  COALESCE((SELECT p.Value FROM Permissions p WHERE p.UserId = u.Id AND p.Kind = 0), 0),
  COALESCE((SELECT p.Value FROM Permissions p WHERE p.UserId = u.Id AND p.Kind = 2), 0)
FROM Users u ORDER BY u.Username`)
	if err != nil {
		return nil, err
	}
	type row struct {
		id, name, pass  string
		admin, disabled bool
	}
	var list []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.name, &r.pass, &r.admin, &r.disabled); err != nil {
			rows.Close()
			return nil, err
		}
		list = append(list, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := map[string]userRef{}
	for _, r := range list {
		if r.disabled {
			im.rep.Add("usuários desativados (pulados)", 1)
			continue
		}
		u, err := im.user(ctx, auth.Imported{Name: r.name, Admin: r.admin, Hash: r.pass})
		if err != nil {
			return nil, err
		}
		out[strings.ToUpper(r.id)] = u
	}
	return out, nil
}

func (im *Importer) jellyfinUserData(ctx context.Context, db *sql.DB, users map[string]userRef) error {
	rows, err := db.QueryContext(ctx, `SELECT d.UserId, b.Path, d.Played, d.PlayCount, d.PlaybackPositionTicks, d.LastPlayedDate,
  d.IsFavorite, d.Rating
FROM UserData d JOIN BaseItems b ON b.Id = d.ItemId
WHERE b.Path IS NOT NULL AND b.Path <> ''`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var uid, p string
		var played, fav bool
		var count int
		var ticks int64
		var last any
		var rating sql.NullFloat64
		if err := rows.Scan(&uid, &p, &played, &count, &ticks, &last, &fav, &rating); err != nil {
			return err
		}
		u, ok := users[strings.ToUpper(uid)]
		if !ok {
			im.rep.Add("estado de usuário não importado", 1)
			continue
		}
		mapped := im.Maps.Apply(p)
		it := im.idx.byPath[mapped]
		if it == nil {
			im.rep.Miss("jellyfin", mapped)
			continue
		}
		im.rep.Add("itens casados", 1)
		d := store.UserData{Played: played, PlayCount: count, Position: float64(ticks) / 1e7, LastPlayed: toTime(last), Favorite: fav}
		if rating.Valid && rating.Float64 > 0 {
			d.Rating = min(5, int(math.Round(rating.Float64/2)))
		}
		im.addUserData(u.ID, it.Kind, it, d)
	}
	return rows.Err()
}
