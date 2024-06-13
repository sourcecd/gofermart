package cryptandsign

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	password = "myMegaPass"
	hashPass = "af89968d2591ce2f7f38d934c9abcc982461e0158be34a360b02f2e328d7a4b3"
)

func TestGetPassHash(t *testing.T) {
	res := GetPassHash(password)
	require.Equal(t, hashPass, res)
}
