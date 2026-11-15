package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// UserData é o estado de um usuário num item. Compartilhado entre as APIs:
// estrela no Subsonic = favorito no Jellyfin; onde parou vale pra todos.
type UserData struct {
	Played     bool
	PlayCount  int
	Position   float64 // segundos
	LastPlayed time.Time
	Favorite   bool
	FavoriteAt time.Time
	Rating     int
}

// UserData lê o estado (zero se nunca mexeu).
func (s *Store) UserData(ctx context.Context, userID, itemID string) (UserData, error) {
	var d UserData
	var last, fav sql.NullString
	err := s.DB.QueryRowContext(ctx, `SELECT played, play_count, position, last_played, favorite, favorite_at, rating
FROM userdata WHERE user_id = ? AND item_id = ?`, userID, NormalizeID(itemID)).
		Scan(&d.Played, &d.PlayCount, &d.Position, &last, &d.Favorite, &fav, &d.Rating)
	if errors.Is(err, sql.ErrNoRows) {
		return d, nil
	}
	if last.Valid {
		d.LastPlayed = parseTime(last.String)
	}
	if fav.Valid {
		d.FavoriteAt = parseTime(fav.String)
	}
	return d, err
}

// UserDataMany lê o estado de vários itens de uma vez (ausente = zero).
func (s *Store) UserDataMany(ctx context.Context, userID string, ids []string) (map[string]UserData, error) {
	out := make(map[string]UserData, len(ids))
	for _, id := range ids {
		d, err := s.UserData(ctx, userID, id)
		if err != nil {
			return nil, err
		}
		out[id] = d
	}
	return out, nil
}

// SetUserData grava o estado inteiro.
func (s *Store) SetUserData(ctx context.Context, userID, itemID string, d UserData) error {
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO userdata (user_id, item_id, played, play_count, position, last_played, favorite, favorite_at, rating)
VALUES (?,?,?,?,?,?,?,?,?)
ON CONFLICT(user_id, item_id) DO UPDATE SET played=excluded.played, play_count=excluded.play_count,
  position=excluded.position, last_played=excluded.last_played, favorite=excluded.favorite,
  favorite_at=excluded.favorite_at, rating=excluded.rating`,
		userID, NormalizeID(itemID), d.Played, d.PlayCount, d.Position, timeOrNil(d.LastPlayed), d.Favorite, timeOrNil(d.FavoriteAt), d.Rating)
	return err
}

// UpdateUserData lê, aplica fn e grava.
func (s *Store) UpdateUserData(ctx context.Context, userID, itemID string, fn func(*UserData)) (UserData, error) {
	d, err := s.UserData(ctx, userID, itemID)
	if err != nil {
		return d, err
	}
	fn(&d)
	return d, s.SetUserData(ctx, userID, itemID, d)
}

// SetFavorite marca/desmarca favorito.
func (s *Store) SetFavorite(ctx context.Context, userID, itemID string, on bool) (UserData, error) {
	return s.UpdateUserData(ctx, userID, itemID, func(d *UserData) {
		d.Favorite = on
		if on {
			d.FavoriteAt = time.Now().UTC()
		} else {
			d.FavoriteAt = time.Time{}
		}
	})
}

// SetPlayed marca assistido/ouvido (zera a posição) ou tira a marca.
func (s *Store) SetPlayed(ctx context.Context, userID, itemID string, played bool, at time.Time) (UserData, error) {
	return s.UpdateUserData(ctx, userID, itemID, func(d *UserData) {
		d.Played = played
		d.Position = 0
		if played {
			d.PlayCount++
			if at.IsZero() {
				at = time.Now()
			}
			d.LastPlayed = at.UTC()
		}
	})
}

// Progress grava onde parou. Perto do fim (>= 90%) vira assistido e a
// posição zera, como o Jellyfin faz; no começo (< 5%) não guarda posição.
func (s *Store) Progress(ctx context.Context, userID string, it *Item, pos float64) (UserData, error) {
	return s.UpdateUserData(ctx, userID, it.ID, func(d *UserData) {
		d.LastPlayed = time.Now().UTC()
		switch {
		case it.Duration > 0 && pos >= it.Duration*0.9:
			if !d.Played || d.Position > 0 {
				d.PlayCount++
			}
			d.Played, d.Position = true, 0
		case it.Duration > 0 && pos < it.Duration*0.05:
			d.Position = 0
		default:
			d.Position = pos
		}
	})
}

func timeOrNil(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}
