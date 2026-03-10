package cli

import (
	"bufio"
	"strings"
	"testing"

	"db-sync-cli/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectDatabaseByIndex(t *testing.T) {
	databases := models.DatabaseList{{Name: "alpha"}, {Name: "beta"}}
	selected := selectDatabase(databases, "2")
	require.NotNil(t, selected)
	assert.Equal(t, "beta", selected.Name)
}

func TestSelectDatabaseByName(t *testing.T) {
	databases := models.DatabaseList{{Name: "alpha"}, {Name: "beta"}}
	selected := selectDatabase(databases, "alpha")
	require.NotNil(t, selected)
	assert.Equal(t, "alpha", selected.Name)
}

func TestSelectDatabaseRejectsUnknownValue(t *testing.T) {
	databases := models.DatabaseList{{Name: "alpha"}, {Name: "beta"}}
	selected := selectDatabase(databases, "gamma")
	assert.Nil(t, selected)
}

func TestPromptForMenuAction(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("2\n"))
	action, err := promptForMenuAction(reader)
	require.NoError(t, err)
	assert.Equal(t, "2", action)
}

func TestPromptForDatabaseSelectionByIndex(t *testing.T) {
	databases := models.DatabaseList{{Name: "alpha"}, {Name: "beta"}}
	reader := bufio.NewReader(strings.NewReader("1\n"))
	selected, err := promptForDatabaseSelection(reader, databases)
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, "alpha", selected.Name)
}

func TestPromptForDatabaseSelectionCancel(t *testing.T) {
	databases := models.DatabaseList{{Name: "alpha"}, {Name: "beta"}}
	reader := bufio.NewReader(strings.NewReader("q\n"))
	selected, err := promptForDatabaseSelection(reader, databases)
	require.NoError(t, err)
	assert.Nil(t, selected)
}
