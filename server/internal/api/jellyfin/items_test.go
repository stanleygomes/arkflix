package jellyfin

import (
	"context"
	"net/url"
	"slices"
	"testing"
	"time"
)

func TestViews(t *testing.T) {
	e := setup(t)
	for _, p := range []string{"/Users/" + e.userID + "/Views", "/UserViews?userId=" + e.userID} {
		l := e.list(p)
		eq(t, "views", l.TotalRecordCount, 3)
		var colls []string
		for _, v := range l.Items {
			eq(t, "type", v.Type, "CollectionFolder")
			eq(t, "folder", v.IsFolder, true)
			colls = append(colls, v.CollectionType)
		}
		eq(t, "coleções", joined(colls), "movies|tvshows|music")
	}
	// raiz do /Items sem pai nem recursão = as bibliotecas
	eq(t, "raiz", e.list("/Users/"+e.userID+"/Items").TotalRecordCount, 3)
}

func TestItemsFilters(t *testing.T) {
	e := setup(t)
	views := e.list("/UserViews").Items
	movies, shows, music := views[0].Id, views[1].Id, views[2].Id
	base := "/Users/" + e.userID + "/Items?"

	l := e.list(base + "ParentId=" + movies + "&SortBy=SortName&SortOrder=Ascending")
	eq(t, "filmes na raiz", joined(names(l.Items)), "Curta Pixel|Filme Sem Ano|O Grande Teste")
	l = e.list(base + "Recursive=true&IncludeItemTypes=Movie&SortBy=ProductionYear,SortName&SortOrder=Descending")
	eq(t, "por ano", joined(names(l.Items)), "O Grande Teste|Curta Pixel|Filme Sem Ano")
	l = e.list(base + "Recursive=true&IncludeItemTypes=Movie&SortBy=SortName&StartIndex=1&Limit=1")
	eq(t, "página", joined(names(l.Items)), "Filme Sem Ano")
	eq(t, "total", l.TotalRecordCount, 3)
	eq(t, "start", l.StartIndex, 1)
	l = e.list(base + "Recursive=true&IncludeItemTypes=Movie,Series&SearchTerm=pixel")
	eq(t, "busca", joined(names(l.Items)), "Curta Pixel")
	l = e.list(base + "Recursive=true&IncludeItemTypes=Movie,Series&SearchTerm=SERIE")
	eq(t, "busca sem acento nem caixa", joined(names(l.Items)), "Série de Teste")
	l = e.list(base + "Recursive=true&IncludeItemTypes=Movie&Years=2019,2021&SortBy=SortName")
	eq(t, "anos", joined(names(l.Items)), "Curta Pixel|O Grande Teste")
	l = e.list(base + "Recursive=true&IncludeItemTypes=Movie&NameStartsWith=f")
	eq(t, "letra", joined(names(l.Items)), "Filme Sem Ano")
	l = e.list(base + "Ids=" + e.ids.MovieBig + "," + e.ids.Ep1)
	eq(t, "ids", l.TotalRecordCount, 2)
	l = e.list(base + "Recursive=true&IncludeItemTypes=BoxSet")
	eq(t, "tipo que não temos", l.TotalRecordCount, 0)
	l = e.list(base + "Recursive=true&ParentId=" + shows + "&IncludeItemTypes=Series")
	eq(t, "séries", joined(names(l.Items)), "Série de Teste")
	l = e.list(base + "ParentId=" + e.ids.Series)
	eq(t, "filhos da série", joined(names(l.Items)), "Temporada 1|Temporada 2")
	l = e.list(base + "ParentId=" + e.ids.Series + "&Recursive=true&IncludeItemTypes=Episode")
	eq(t, "episódios", len(l.Items), 3)
	l = e.list(base + "Recursive=true&MediaTypes=Video")
	eq(t, "vídeos", len(l.Items), 6)

	// favoritos, vistos
	eq(t, "fav", e.call("POST", "/Users/"+e.userID+"/FavoriteItems/"+e.ids.MoviePixel, nil, nil), 200)
	l = e.list(base + "Recursive=true&IncludeItemTypes=Movie,Series&Filters=IsFavorite")
	eq(t, "favoritos", joined(names(l.Items)), "Curta Pixel")
	l = e.list(base + "Recursive=true&IncludeItemTypes=Movie&IsFavorite=false&SortBy=SortName")
	eq(t, "não favoritos", len(l.Items), 2)
	eq(t, "visto", e.call("POST", "/Users/"+e.userID+"/PlayedItems/"+e.ids.MovieBig, nil, nil), 200)
	l = e.list(base + "Recursive=true&IncludeItemTypes=Movie&Filters=IsPlayed")
	eq(t, "vistos", joined(names(l.Items)), "O Grande Teste")
	l = e.list(base + "Recursive=true&IncludeItemTypes=Movie&Filters=IsUnplayed&SortBy=SortName")
	eq(t, "não vistos", joined(names(l.Items)), "Curta Pixel|Filme Sem Ano")

	// música
	l = e.list(base + "Recursive=true&ParentId=" + music + "&IncludeItemTypes=MusicAlbum&Genres=rock")
	eq(t, "álbuns rock", joined(names(l.Items)), "Disco Teste")
	l = e.list(base + "ParentId=" + e.ids.Album + "&SortBy=ParentIndexNumber,IndexNumber")
	eq(t, "faixas", joined(names(l.Items)), "Bip|Bop")
	l = e.list(base + "Recursive=true&IncludeItemTypes=Audio&AlbumIds=" + e.ids.Album)
	eq(t, "AlbumIds", len(l.Items), 2)
	l = e.list(base + "Recursive=true&IncludeItemTypes=Audio&ArtistIds=" + e.ids.ArtistX)
	eq(t, "ArtistIds", joined(names(l.Items)), "Solta")
	l = e.list(base + "Recursive=true&IncludeItemTypes=MusicAlbum&AlbumArtistIds=" + e.ids.Artist)
	eq(t, "AlbumArtistIds", joined(names(l.Items)), "Disco Teste")
	l = e.list("/Artists/AlbumArtists?SortBy=SortName")
	eq(t, "artistas", joined(names(l.Items)), "Artista X|Banda Teste")
	var art dtoT
	e.get("/Artists/"+url.PathEscape("banda teste"), &art)
	eq(t, "artista por nome", art.Id, e.ids.Artist)
	var genres listT
	e.get("/MusicGenres", &genres)
	eq(t, "gêneros", joined(names(genres.Items)), "Rock")
	l = e.list(base + "Recursive=true&IncludeItemTypes=MusicAlbum&GenreIds=" + genres.Items[0].Id)
	eq(t, "GenreIds", joined(names(l.Items)), "Disco Teste")

	// jogos nunca aparecem
	eq(t, "rom", e.call("GET", "/Items/"+e.ids.RomSuper, nil, nil), 404)
	l = e.list(base + "Recursive=true")
	for _, it := range l.Items {
		if it.Type == "" {
			t.Fatalf("item sem tipo: %+v", it)
		}
	}
	eq(t, "tudo", l.TotalRecordCount, 3+1+2+3+2+2+3)
}

func TestItemDTO(t *testing.T) {
	e := setup(t)
	var m dtoT
	e.get("/Items/"+e.ids.MovieBig+"?userId="+e.userID, &m)
	eq(t, "type", m.Type, "Movie")
	eq(t, "media", m.MediaType, "Video")
	eq(t, "year", m.ProductionYear, 2021)
	eq(t, "ticks", m.RunTimeTicks, int64(20030000))
	eq(t, "server", m.ServerId, e.s.serverID)
	eq(t, "key", m.UserData.Key, m.Id)
	eq(t, "container", m.Container, "webm")
	eq(t, "premiere", m.PremiereDate, "2021-01-01T00:00:00.0000000Z")
	if m.ImageTags["Primary"] == "" || len(m.MediaSources) != 1 || len(m.MediaSources[0].MediaStreams) != 3 {
		t.Fatalf("%+v", m)
	}
	if _, err := time.Parse("2006-01-02T15:04:05.0000000Z", m.DateCreated); err != nil {
		t.Fatal(m.DateCreated)
	}
	var ep dtoT
	e.get("/Users/"+e.userID+"/Items/"+e.ids.Ep2, &ep)
	eq(t, "série", ep.SeriesName, "Série de Teste")
	eq(t, "temporada", ep.SeasonName, "Temporada 1")
	eq(t, "S", *ep.ParentIndexNumber, 1)
	eq(t, "E", *ep.IndexNumber, 2)
	eq(t, "parent", ep.ParentId, e.ids.Season1)
	if ep.SeriesPrimaryImageTag == "" {
		t.Fatal("sem SeriesPrimaryImageTag")
	}
	var sr dtoT
	e.get("/Items/"+e.ids.Series, &sr)
	eq(t, "folder", sr.IsFolder, true)
	eq(t, "child", *sr.ChildCount, 2)
	eq(t, "recursive", *sr.RecursiveItemCount, 3)
	eq(t, "unplayed", *sr.UserData.UnplayedItemCount, 3)
	var tr dtoT
	e.get("/Items/"+e.ids.TrackBop, &tr)
	eq(t, "audio", tr.Type, "Audio")
	eq(t, "album", tr.AlbumId, e.ids.Album)
	eq(t, "album artist", tr.AlbumArtists[0].Id, e.ids.Artist)
	eq(t, "artist", tr.ArtistItems[0].Name, "Banda Teste")
	if tr.AlbumPrimaryImageTag == "" || tr.ImageTags["Primary"] != "" {
		t.Fatalf("capa do álbum: %+v", tr)
	}
	var anc []dtoT
	e.get("/Items/"+e.ids.Ep1+"/Ancestors", &anc)
	eq(t, "ancestrais", joined(names(anc)), "Temporada 1|Série de Teste|Séries")
}

func TestResumeNextUpLatest(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	st := e.k.Store
	pixel, _ := st.Item(ctx, e.ids.MoviePixel)
	if _, err := st.Progress(ctx, e.userID, pixel, 1.0); err != nil {
		t.Fatal(err)
	}
	l := e.list("/Users/" + e.userID + "/Items/Resume?Limit=10&MediaTypes=Video")
	eq(t, "resume", joined(names(l.Items)), "Curta Pixel")
	eq(t, "posição", l.Items[0].UserData.PlaybackPositionTicks, int64(10_000_000))
	l = e.list("/UserItems/Resume?MediaTypes=Audio")
	eq(t, "resume áudio", l.TotalRecordCount, 0)

	eq(t, "nextup vazio", e.list("/Shows/NextUp?UserId="+e.userID).TotalRecordCount, 0)
	e.call("POST", "/UserPlayedItems/"+e.ids.Ep1, nil, nil)
	l = e.list("/Shows/NextUp?UserId=" + e.userID + "&Limit=5")
	eq(t, "nextup", joined(names(l.Items)), "O Segundo")
	e.call("POST", "/UserPlayedItems/"+e.ids.Ep2, nil, nil)
	l = e.list("/Shows/NextUp?SeriesId=" + e.ids.Series)
	eq(t, "nextup temporada 2", joined(names(l.Items)), "Recomeço")

	var latest []dtoT
	e.get("/Users/"+e.userID+"/Items/Latest?IncludeItemTypes=Series&Limit=5", &latest)
	eq(t, "latest séries", joined(names(latest)), "Série de Teste")
	e.get("/Items/Latest?IncludeItemTypes=Movie", &latest)
	eq(t, "latest filmes", len(latest), 3)
	e.get("/Items/Latest?ParentId="+viewID("music"), &latest)
	eq(t, "latest música", len(latest), 2)
	eq(t, "latest música tipo", latest[0].Type, "MusicAlbum")
}

func TestSeasonsEpisodes(t *testing.T) {
	e := setup(t)
	l := e.list("/Shows/" + e.ids.Series + "/Seasons?UserId=" + e.userID)
	eq(t, "temporadas", joined(names(l.Items)), "Temporada 1|Temporada 2")
	eq(t, "idx", *l.Items[1].IndexNumber, 2)
	l = e.list("/Shows/" + e.ids.Series + "/Episodes?SeasonId=" + e.ids.Season1)
	eq(t, "t1", joined(names(l.Items)), "O Começo|O Segundo")
	l = e.list("/Shows/" + e.ids.Series + "/Episodes?Season=2")
	eq(t, "t2", joined(names(l.Items)), "Recomeço")
	l = e.list("/Shows/" + e.ids.Series + "/Episodes?StartItemId=" + e.ids.Ep2)
	eq(t, "a partir do 2", joined(names(l.Items)), "O Segundo|Recomeço")
	l = e.list("/Shows/" + e.ids.Series + "/Episodes?StartIndex=1&Limit=1")
	eq(t, "página", joined(names(l.Items)), "O Segundo")
	eq(t, "total", l.TotalRecordCount, 3)
}

func TestMiscLists(t *testing.T) {
	e := setup(t)
	var counts map[string]int
	e.get("/Items/Counts", &counts)
	eq(t, "filmes", counts["MovieCount"], 3)
	eq(t, "músicas", counts["SongCount"], 3)
	var hints struct {
		SearchHints      []map[string]any
		TotalRecordCount int
	}
	e.get("/Search/Hints?SearchTerm=teste", &hints)
	if hints.TotalRecordCount < 3 {
		t.Fatalf("%+v", hints)
	}
	var f map[string]any
	e.get("/Items/Filters?ParentId="+viewID("movies")+"&IncludeItemTypes=Movie", &f)
	if _, ok := f["Years"]; !ok {
		t.Fatal(f)
	}
	e.get("/Items/Filters2?IncludeItemTypes=MusicAlbum", &f)
	eq(t, "similar", e.call("GET", "/Items/"+e.ids.MovieBig+"/Similar?Limit=5", nil, nil), 200)
	l := e.list("/Items/" + e.ids.TrackBip + "/InstantMix?Limit=10")
	if !slices.ContainsFunc(l.Items, func(d dtoT) bool { return d.Name == "Bop" }) {
		t.Fatal(names(l.Items))
	}
	for _, p := range []string{"/Items/" + e.ids.MovieBig + "/ThemeMedia", "/Items/" + e.ids.MovieBig + "/SpecialFeatures",
		"/Users/" + e.userID + "/Items/" + e.ids.MovieBig + "/LocalTrailers", "/Users/" + e.userID + "/Items/" + e.ids.MovieBig + "/Intros",
		"/Genres", "/Persons", "/Studios", "/Artists", "/Items/" + e.ids.MovieBig + "/Images"} {
		eq(t, p, e.call("GET", p, nil, nil), 200)
	}
}
