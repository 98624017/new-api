package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateBackupCodes(t *testing.T) {
	codes, err := GenerateBackupCodes()
	require.NoError(t, err)
	require.Len(t, codes, 20)

	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		require.Len(t, code, BackupCodeLength+1)
		assert.Equal(t, byte('-'), code[4])
		assert.True(t, ValidateBackupCode(code))

		normalizedCode := NormalizeBackupCode(code)
		_, duplicate := seen[normalizedCode]
		assert.False(t, duplicate)
		seen[normalizedCode] = struct{}{}
	}
}
