package entclient

// User describes a Coder user account
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// Me gets the details of the authenticated user
func (c Client) Me() (*User, error) {
	var u User
	err := c.requestBody("GET", "/api/users/me", nil, &u)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// SSHKey describes an SSH keypair
type SSHKey struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

// SSHKey gets the current SSH kepair of the authenticated user
func (c Client) SSHKey() (*SSHKey, error) {
	var key SSHKey
	err := c.requestBody("GET", "/api/users/me/sshkey", nil, &key)
	if err != nil {
		return nil, err
	}
	return &key, nil
}
