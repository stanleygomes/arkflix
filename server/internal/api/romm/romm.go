// Package romm fala a API do RomM (https://romm.app) em cima do acervo
// único: login OAuth2/basic/client token, consoles, ROMs, download (arquivo
// com Range ou zip na hora), BIOS, capas e os saves/states/screenshots de
// cada usuário. Sem provedores de metadados, sem upload de ROM, sem admin:
// o que é do RomM e não existe aqui volta null ou lista vazia, no formato
// que os clientes (app arkflix, Argosy, muOS, Playnite...) esperam.
package romm

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/deps"
)

// Version é a versão do RomM que o servidor diz ser (os clientes checam
// recurso pela versão: client tokens precisam de 4.4+).
const Version = "4.8.0"

type api struct {
	d     *deps.Deps
	basic *basicCache
}

// New monta o handler da API (/api/* e /assets/romm/*).
func New(d *deps.Deps) http.Handler {
	a := &api{d: d, basic: newBasicCache()}
	mux := http.NewServeMux()
	a.routes(mux)
	return mux
}

// handlerFunc recebe quem chamou (nil em rota pública sem login).
type handlerFunc func(w http.ResponseWriter, r *http.Request, c *caller)

// handle registra uma rota. scopes nil = pública; vazio = só precisa login.
func (a *api) handle(mux *http.ServeMux, pattern string, scopes []string, fn handlerFunc) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		c := a.authenticate(r)
		if scopes != nil {
			if c == nil {
				w.Header().Set("WWW-Authenticate", "Bearer")
				fail(w, http.StatusUnauthorized, "Not authenticated")
				return
			}
			for _, s := range scopes {
				if !c.scopes[s] {
					fail(w, http.StatusForbidden, "Forbidden")
					return
				}
			}
		}
		fn(w, r, c)
	})
}

func (a *api) routes(mux *http.ServeMux) {
	pub := []string(nil)
	need := func(s ...string) []string { return append([]string{}, s...) }

	// login
	a.handle(mux, "POST /api/token", pub, a.token)
	a.handle(mux, "POST /api/login", pub, a.login)
	a.handle(mux, "POST /api/logout", pub, a.logout)
	a.handle(mux, "POST /api/client-tokens", need(scopeMeWrite), a.createClientToken)
	a.handle(mux, "GET /api/client-tokens", need(scopeMeRead), a.listClientTokens)
	a.handle(mux, "GET /api/client-tokens/all", need(scopeUsersRead), a.listAllClientTokens)
	a.handle(mux, "DELETE /api/client-tokens/{id}", need(), a.deleteClientToken)
	a.handle(mux, "DELETE /api/client-tokens/{id}/admin", need(scopeUsersWrite), a.adminDeleteClientToken)
	a.handle(mux, "PUT /api/client-tokens/{id}/regenerate", need(scopeMeWrite), a.regenerateClientToken)

	// sistema
	a.handle(mux, "GET /api/heartbeat", pub, a.heartbeat)
	a.handle(mux, "GET /api/config", pub, a.config)
	a.handle(mux, "GET /api/stats", need(scopeRomsRead), a.stats)
	a.handle(mux, "GET /api/tasks", need(scopeTasksRun), a.tasks)
	a.handle(mux, "GET /api/tasks/status", need(scopeTasksRun), a.tasksStatus)
	a.handle(mux, "GET /api/tasks/{task_id}", need(scopeTasksRun), a.taskByID)
	a.handle(mux, "POST /api/tasks/run/{task_name}", need(scopeTasksRun), a.runTask)
	a.handle(mux, "POST /api/tasks/scan", need(scopeTasksRun), a.runScan)

	// usuários
	a.handle(mux, "GET /api/users/me", need(scopeMeRead), a.me)
	a.handle(mux, "GET /api/users", need(scopeUsersRead), a.users)
	a.handle(mux, "GET /api/users/identifiers", need(scopeUsersRead), a.userIDs)
	a.handle(mux, "GET /api/users/{id}", need(scopeUsersRead), a.user)

	// consoles e BIOS
	a.handle(mux, "GET /api/platforms", need(scopePlatformsRead), a.platforms)
	a.handle(mux, "GET /api/platforms/supported", need(scopePlatformsRead), a.platforms)
	a.handle(mux, "GET /api/platforms/filesystem", need(scopePlatformsRead), a.platforms)
	a.handle(mux, "GET /api/platforms/identifiers", need(scopePlatformsRead), a.platformIDs)
	a.handle(mux, "GET /api/platforms/{id}", need(scopePlatformsRead), a.platform)
	a.handle(mux, "GET /api/firmware", need(scopeFirmwareRead), a.firmwareList)
	a.handle(mux, "GET /api/firmware/identifiers", need(scopeFirmwareRead), a.firmwareIDs)
	a.handle(mux, "GET /api/firmware/{id}", need(scopeFirmwareRead), a.firmware)
	a.handle(mux, "GET /api/firmware/{id}/content/{file_name}", need(scopeFirmwareRead), a.firmwareContent)

	// ROMs
	a.handle(mux, "GET /api/roms", need(scopeRomsRead), a.roms)
	a.handle(mux, "GET /api/roms/identifiers", need(scopeRomsRead), a.romIDs)
	a.handle(mux, "GET /api/roms/filters", need(scopeRomsRead), a.romFilters)
	a.handle(mux, "GET /api/roms/download", need(scopeRomsRead), a.bulkDownload)
	a.handle(mux, "GET /api/roms/{id}", need(scopeRomsRead), a.rom)
	a.handle(mux, "GET /api/roms/{id}/simple", need(scopeRomsRead), a.romSimple)
	a.handle(mux, "GET /api/roms/{id}/content/{file_name}", need(scopeRomsRead), a.romContent)
	a.handle(mux, "GET /api/roms/{id}/files", need(scopeRomsRead), a.romFile)
	a.handle(mux, "GET /api/roms/{id}/files/content/{file_name}", need(scopeRomsRead), a.romFileContent)
	a.handle(mux, "PUT /api/roms/{id}/props", need(scopeRomsUserWrite), a.props)

	// capas (o RomM serve pelo nginx, sem login)
	a.handle(mux, "GET /assets/romm/resources/roms/{platform_id}/{rom_id}/cover/{size}", pub, a.cover)

	// saves, states e screenshots do usuário
	for _, k := range kinds {
		a.assetRoutes(mux, k)
	}

	// coleções = listas (internal/lists); virtuais e inteligentes não tem
	a.handle(mux, "GET /api/collections", need(scopeCollectionsRead), a.collections)
	a.handle(mux, "GET /api/collections/identifiers", need(scopeCollectionsRead), a.collectionIDs)
	a.handle(mux, "POST /api/collections", need(scopeCollectionsWrite), a.addCollection)
	a.handle(mux, "PUT /api/collections/{id}", need(scopeCollectionsWrite), a.updateCollection)
	a.handle(mux, "DELETE /api/collections/{id}", need(scopeCollectionsWrite), a.deleteCollection)
	a.handle(mux, "POST /api/collections/{id}/roms", need(scopeCollectionsWrite), a.collectionRoms(true))
	a.handle(mux, "DELETE /api/collections/{id}/roms", need(scopeCollectionsWrite), a.collectionRoms(false))
	a.handle(mux, "GET /api/collections/virtual", need(scopeCollectionsRead), emptyList)
	a.handle(mux, "GET /api/collections/virtual/identifiers", need(scopeCollectionsRead), emptyList)
	a.handle(mux, "GET /api/collections/smart", need(scopeCollectionsRead), emptyList)
	a.handle(mux, "GET /api/collections/smart/identifiers", need(scopeCollectionsRead), emptyList)
	a.handle(mux, "GET /api/collections/{id}", need(scopeCollectionsRead), a.getCollection)
	a.handle(mux, "GET /api/collections/virtual/{id}", need(scopeCollectionsRead), collectionNotFound)
	a.handle(mux, "GET /api/collections/smart/{id}", need(scopeCollectionsRead), collectionNotFound)

	// o resto: 404 no formato do FastAPI
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fail(w, http.StatusNotFound, "Not Found") })
}

func emptyList(w http.ResponseWriter, r *http.Request, c *caller) {
	writeJSON(w, http.StatusOK, []any{})
}

func collectionNotFound(w http.ResponseWriter, r *http.Request, c *caller) {
	fail(w, http.StatusNotFound, "Collection not found")
}

// writeJSON manda v como JSON.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Debug("romm: json", "err", err)
	}
}

// fail responde erro como o FastAPI: {"detail": "..."}.
func fail(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}

// internal loga e responde 500.
func internal(w http.ResponseWriter, err error) {
	slog.Error("romm", "err", err)
	fail(w, http.StatusInternalServerError, "Internal server error")
}

// pathID lê um id numérico do caminho (422 se não for número, como o FastAPI).
func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, "Invalid "+name)
		return 0, false
	}
	return id, true
}

// queryBool lê ?x=true/false/1/0 (ausente = nil).
func queryBool(r *http.Request, name string) (*bool, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return nil, nil
	}
	switch strings.ToLower(v) {
	case "true", "1", "yes", "on":
		t := true
		return &t, nil
	case "false", "0", "no", "off":
		f := false
		return &f, nil
	}
	return nil, errors.New("Invalid boolean: " + name)
}

// queryInts junta ?x=1&x=2 e ?x=1,2.
func queryInts(r *http.Request, names ...string) ([]int64, error) {
	var out []int64
	for _, name := range names {
		for _, v := range r.URL.Query()[name] {
			for p := range strings.SplitSeq(v, ",") {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				n, err := strconv.ParseInt(p, 10, 64)
				if err != nil {
					return nil, errors.New("Invalid " + name)
				}
				out = append(out, n)
			}
		}
	}
	return out, nil
}

// queryStrings junta ?x=a&x=b (sem separar por vírgula: nome pode ter).
func queryStrings(r *http.Request, name string) []string {
	var out []string
	for _, v := range r.URL.Query()[name] {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
