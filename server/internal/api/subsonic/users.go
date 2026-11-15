package subsonic

import (
	"strings"

	"github.com/stanleygomes/arkflix/server/internal/auth"
)

// userResp: tocar, baixar e scrobble liberados; admin conforme o usuário.
func userResp(u *auth.User) user {
	return user{Username: u.Name, ScrobblingEnabled: true, AdminRole: u.Admin, DownloadRole: true,
		PlaylistRole: true, CoverArtRole: u.Admin, StreamRole: true, Folder: []int{folderID}}
}

// getUser: o próprio usuário; admin pode ver qualquer um.
func (a *API) getUser(c *call) (*resp, error) {
	name, err := c.p.need("username")
	if err != nil {
		return nil, err
	}
	u := c.user
	if !strings.EqualFold(name, c.user.Name) {
		if !c.user.Admin {
			return nil, fail(errNotAllowed, "")
		}
		if u, err = a.d.Auth.UserByName(c.ctx, name); err != nil {
			return nil, fail(errNotFound, "User not found")
		}
	}
	res := a.newResp()
	r := userResp(u)
	res.User = &r
	return res, nil
}

// getUsers é só pra admin.
func (a *API) getUsers(c *call) (*resp, error) {
	if !c.user.Admin {
		return nil, fail(errNotAllowed, "")
	}
	all, err := a.d.Auth.Users(c.ctx)
	if err != nil {
		return nil, err
	}
	out := &users{User: []user{}}
	for _, u := range all {
		out.User = append(out.User, userResp(u))
	}
	res := a.newResp()
	res.Users = out
	return res, nil
}
