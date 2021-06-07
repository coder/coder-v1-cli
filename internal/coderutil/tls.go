package coderutil

import (
	"fmt"
	"strings"

	"cdr.dev/coder-cli/coder-sdk"
)

type ProviderInfoTable struct {
	ProviderName        string `table:"DNS Provider Name"`
	ProviderCode        string `table:"DNS Provider CLI Code"`
	RequiredCredentials string `table:"Required Credentials"`
}

func ProviderInfosTable(p coder.ProviderInfo) ProviderInfoTable {
	pit := ProviderInfoTable{
		ProviderName: p.Name,
		ProviderCode: p.Code,
	}

	alternateCredentialsSet := []string{}
	if len(p.RequiredCredentials) != 0 {
		for i := range p.RequiredCredentials {
			credentialsSet := fmt.Sprintf("{%s}", strings.Join(p.RequiredCredentials[i], ", "))
			if i == 0 {
				pit.RequiredCredentials = credentialsSet
				continue
			}
			alternateCredentialsSet = append(alternateCredentialsSet, credentialsSet)
		}
	}

	creds := append([]string{pit.RequiredCredentials}, alternateCredentialsSet...)
	pit.RequiredCredentials = strings.Join(creds, " OR ")
	return pit
}
