package jellyfin

import (
	"context"
	"encoding/json"
	"time"

	"github.com/coder/websocket"

	"github.com/stanleygomes/arkflix/server/internal/auth"
)

// keepAlive é o intervalo (s) que o servidor pede no ForceKeepAlive.
const keepAlive = 60

type wsMsg struct {
	MessageType string
	MessageId   string `json:",omitempty"`
	Data        any    `json:",omitempty"`
}

// GET /socket: só responde KeepAlive como o Jellyfin; o resto é ignorado
// (não tem controle remoto nem aviso de biblioteca).
func (s *Server) socket(c *rc) {
	conn, err := websocket.Accept(c.w, c.r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	send := func(m wsMsg) error {
		m.MessageId = auth.RandomHex(16)
		raw, _ := json.Marshal(m)
		wctx, done := context.WithTimeout(ctx, 10*time.Second)
		defer done()
		return conn.Write(wctx, websocket.MessageText, raw)
	}
	if send(wsMsg{MessageType: "ForceKeepAlive", Data: keepAlive}) != nil {
		return
	}
	for {
		// cliente some sem KeepAlive por 2 intervalos: fecha
		rctx, done := context.WithTimeout(ctx, 2*keepAlive*time.Second)
		_, raw, err := conn.Read(rctx)
		done()
		if err != nil {
			return
		}
		var m wsMsg
		if json.Unmarshal(raw, &m) != nil {
			continue
		}
		if m.MessageType == "KeepAlive" {
			if send(wsMsg{MessageType: "KeepAlive"}) != nil {
				return
			}
		}
	}
}
