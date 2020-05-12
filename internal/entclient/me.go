package entclient

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

func (c Client) Me() (*User, error) {
	var u User
	err := c.requestBody("GET", "/api/users/me", nil, &u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
