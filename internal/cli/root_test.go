package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sgallen/timber/internal/timber"
)

func TestSplitCSVTrimsAndSkipsEmptyParts(t *testing.T) {
	got := splitCSV(" app, auth , ,billing ")
	want := []string{"app", "auth", "billing"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected splitCSV result: got %#v want %#v", got, want)
	}
}

func TestParseNewArgsSupportsMixedModes(t *testing.T) {
	sourceRef, repoNames, repoRefs, includeAll, err := parseNewArgs([]string{"--from", "develop", "--repos", "app,auth", "auth=hotfix/123"})
	if err != nil {
		t.Fatalf("parseNewArgs returned error: %v", err)
	}
	if sourceRef != "develop" {
		t.Fatalf("unexpected sourceRef %q", sourceRef)
	}
	if includeAll {
		t.Fatal("expected includeAll to be false")
	}
	if !reflect.DeepEqual(repoNames, []string{"app", "auth"}) {
		t.Fatalf("unexpected repoNames %#v", repoNames)
	}
	if got := repoRefs["auth"]; got != "hotfix/123" {
		t.Fatalf("unexpected auth override %q", got)
	}
}

func TestParseNewArgsRejectsConflictingRepoSelectors(t *testing.T) {
	_, _, _, _, err := parseNewArgs([]string{"--from", "main", "--repos", "app", "--all"})
	if err == nil || !strings.Contains(err.Error(), "cannot use --all together with --repos") {
		t.Fatalf("expected conflicting selector error, got %v", err)
	}
}

func TestParseNewArgsRejectsMissingSource(t *testing.T) {
	_, _, _, _, err := parseNewArgs([]string{"--repos", "app"})
	if err == nil || !strings.Contains(err.Error(), "requires --from <ref> or at least one repo=ref mapping") {
		t.Fatalf("expected missing source error, got %v", err)
	}
}

func TestParseAddArgsUsesContextPath(t *testing.T) {
	context := &timber.ProjectContext{PathName: "auth-flow"}
	pathName, defaultRef, repoNames, repoRefs, err := parseAddArgs([]string{"--from", "develop", "billing", "auth=hotfix/123"}, context)
	if err != nil {
		t.Fatalf("parseAddArgs returned error: %v", err)
	}
	if pathName != "auth-flow" {
		t.Fatalf("unexpected pathName %q", pathName)
	}
	if defaultRef != "develop" {
		t.Fatalf("unexpected defaultRef %q", defaultRef)
	}
	if !reflect.DeepEqual(repoNames, []string{"billing"}) {
		t.Fatalf("unexpected repoNames %#v", repoNames)
	}
	if got := repoRefs["auth"]; got != "hotfix/123" {
		t.Fatalf("unexpected auth ref %q", got)
	}
}

func TestParseAddArgsRequiresPathOutsideContext(t *testing.T) {
	_, _, _, _, err := parseAddArgs(nil, nil)
	if err == nil || !strings.Contains(err.Error(), "usage: timber add <path>") {
		t.Fatalf("expected path usage error, got %v", err)
	}
}

func TestParseAddArgsRejectsDuplicateRepoMappings(t *testing.T) {
	_, _, _, _, err := parseAddArgs([]string{"auth-flow", "app=main", "app=develop"}, nil)
	if err == nil || !strings.Contains(err.Error(), "mapped more than once") {
		t.Fatalf("expected duplicate mapping error, got %v", err)
	}
}

func TestParseRunArgsSupportsExplicitPath(t *testing.T) {
	pathName, commandArgs, err := parseRunArgs([]string{"auth-flow", "--", "codex", "--full-auto"})
	if err != nil {
		t.Fatalf("parseRunArgs returned error: %v", err)
	}
	if pathName != "auth-flow" {
		t.Fatalf("unexpected pathName %q", pathName)
	}
	want := []string{"codex", "--full-auto"}
	if !reflect.DeepEqual(commandArgs, want) {
		t.Fatalf("unexpected commandArgs %#v want %#v", commandArgs, want)
	}
}

func TestParseRunArgsSupportsContextInferenceMode(t *testing.T) {
	pathName, commandArgs, err := parseRunArgs([]string{"--", "npm", "test"})
	if err != nil {
		t.Fatalf("parseRunArgs returned error: %v", err)
	}
	if pathName != "" {
		t.Fatalf("expected empty pathName, got %q", pathName)
	}
	want := []string{"npm", "test"}
	if !reflect.DeepEqual(commandArgs, want) {
		t.Fatalf("unexpected commandArgs %#v want %#v", commandArgs, want)
	}
}

func TestParseRunArgsRejectsMissingSeparatorOrCommand(t *testing.T) {
	cases := [][]string{
		{"auth-flow", "codex"},
		{"auth-flow", "--"},
		{"one", "two", "--", "codex"},
	}

	for _, args := range cases {
		_, _, err := parseRunArgs(args)
		if err == nil || !strings.Contains(err.Error(), "usage: timber run") {
			t.Fatalf("expected usage error for args %#v, got %v", args, err)
		}
	}
}
