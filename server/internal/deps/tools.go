//go:build tools

package deps

// dependências já baixadas pros pacotes que vão usar (scanner, imagens,
// websocket do Jellyfin); some quando forem importadas de verdade
import (
	_ "github.com/coder/websocket"
	_ "github.com/dhowden/tag"
	_ "golang.org/x/image/draw"
)
