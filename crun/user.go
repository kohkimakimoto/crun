package crun

import (
	"os/user"
	"strconv"
)

func LookupUserStruct(id string) (*user.User, error) {
	var u *user.User

	if _, err := strconv.Atoi(id); err == nil {
		u, err = user.LookupId(id)
		if err != nil {
			return nil, err
		}
	} else {
		u, err = user.Lookup(id)
		if err != nil {
			return nil, err
		}
	}

	return u, nil
}

func LookupGroup(id string) (int, error) {
	var g *user.Group

	if _, err := strconv.Atoi(id); err == nil {
		g, err = user.LookupGroupId(id)
		if err != nil {
			return -1, err
		}
	} else {
		g, err = user.LookupGroup(id)
		if err != nil {
			return -1, err
		}
	}

	return strconv.Atoi(g.Gid)
}
