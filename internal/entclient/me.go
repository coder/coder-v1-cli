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

type SSHKey struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

func (c Client) SSHKey() (*SSHKey, error) {
	var key SSHKey
	err := c.requestBody("GET", "/api/users/me/sshkey", nil, &key)
	if err != nil {
		return nil, err
	}
	return &key, nil
}
