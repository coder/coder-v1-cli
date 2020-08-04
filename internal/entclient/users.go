package entclient

import "context"

// Users gets the list of user accounts
func (c Client) Users(ctx context.Context) ([]User, error) {
	var u []User
	err := c.requestBody(ctx, "GET", "/api/users", nil, &u)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// UserByEmail gets a user by email
func (c Client) UserByEmail(ctx context.Context, target string) (*User, error) {
	if target == Me {
		return c.Me(ctx)
	}
	users, err := c.Users(ctx)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		if u.Email == target {
			return &u, nil
		}
	}
	return nil, ErrNotFound
}
