package troubleshoot

import (
	"strings"
	"testing"

	"github.com/mitchellh/cli"
)

func TestTroubleshootConnectCommand_noTabs(t *testing.T) {
	t.Parallel()

	if strings.ContainsRune(New(cli.NewMockUi()).Help(), '\t') {
		t.Fatal("help has tabs")
	}
}