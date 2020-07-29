package entclient

// Users gets the list of user accounts
func (c Client) Users() ([]User, error) {
	var u []User
	err := c.requestBody("GET", "/api/users", nil, &u)
	if err != nil {
		return nil, err
	}
	return u, nil
}
