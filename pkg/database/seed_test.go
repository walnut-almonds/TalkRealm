package database_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/walnut-almonds/talkrealm/pkg/database"
)

func TestSeedWordsRejectsInvalidHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), "words.csv")
	require.NoError(t, os.WriteFile(path, []byte("word,phonetic\ncat,kæt\n"), 0o600))

	_, err := database.SeedWords(path)
	require.ErrorContains(t, err, "invalid words CSV header")
}

func TestSeedSentencesRejectsInvalidHeaderBeforeDatabaseAccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sentences.csv")
	require.NoError(t, os.WriteFile(path, []byte("word,answer\ncat,cat\n"), 0o600))

	_, _, err := database.SeedSentences(path)
	require.ErrorContains(t, err, "invalid sentences CSV header")
}
