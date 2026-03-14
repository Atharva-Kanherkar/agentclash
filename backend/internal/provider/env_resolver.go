package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type EnvCredentialResolver struct{}

func (EnvCredentialResolver) Resolve(_ context.Context, credentialReference string) (string, error) {
	if !strings.HasPrefix(credentialReference, "env://") {
		return "", NewFailure(
			"",
			FailureCodeCredentialUnavailable,
			fmt.Sprintf("credential reference %q is not supported by the env resolver", credentialReference),
			false,
			ErrCredentialUnavailable,
		)
	}

	envVar := strings.TrimPrefix(credentialReference, "env://")
	value, ok := os.LookupEnv(envVar)
	if !ok || value == "" {
		return "", NewFailure(
			"",
			FailureCodeCredentialUnavailable,
			fmt.Sprintf("credential env var %q is not set", envVar),
			false,
			ErrCredentialUnavailable,
		)
	}

	return value, nil
}
