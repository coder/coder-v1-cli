package client

type Org struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Members []User `json:"members"`
}

func (c Client) Orgs() ([]Org, error) {
	var os []Org
	err := c.requestBody("GET", "/api/orgs", nil, &os)
	return os, err
}
