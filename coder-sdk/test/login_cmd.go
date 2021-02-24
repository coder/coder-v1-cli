package main

import (
	"context"
	"fmt"
	"net/url"

	"cdr.dev/coder-cli/coder-sdk"
)

func main() {
	client, err := coder.NewClient(coder.ClientOptions{
		BaseURL: &url.URL{
			Scheme: "http",
			Host:   "localhost:8080",
		},
		Email:    "admin",
		Password: "vt9g9rxsptrq",
	})
	if err != nil {
		fmt.Printf("login error: %v\n", err)
		return
	}

	user, err := client.Me(context.Background())
	if err != nil {
		fmt.Printf("Me error: %v\n", err)
		return
	}

	fmt.Printf("user info: %#v\n", user)

	fmt.Println("changing password")
	err = client.UpdateUser(context.Background(), "me", coder.UpdateUserReq{
		UserPasswordSettings: &coder.UserPasswordSettings{
			Password: "szbp4q3bcrhc",
		},
	})
	if err != nil {
		fmt.Printf("password update error: %v\n", err)
		return
	}

	orgs, err := client.Organizations(context.Background())
	if err != nil {
		fmt.Printf("org list error: %v\n", err)
		return
	}

	fmt.Printf("orgs info: %#v\n", orgs)
}
