package timber

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitProjectCreatesCanonicalLayout(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")

	configPath, err := InitProject(projectRoot)
	if err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	if got, want := configPath, filepath.Join(projectRoot, ".timber", "project.yaml"); got != want {
		t.Fatalf("config path mismatch: got %q want %q", got, want)
	}

	requiredPaths := []string{
		filepath.Join(projectRoot, ".timber/repos"),
		filepath.Join(projectRoot, "paths"),
		filepath.Join(projectRoot, ".timber", "paths"),
		filepath.Join(projectRoot, ".timber", "operations"),
		filepath.Join(projectRoot, ".timber", "events.log"),
	}

	for _, path := range requiredPaths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected path %q to exist: %v", path, err)
		}
	}

	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("LoadProjectConfig returned error: %v", err)
	}

	if config.Name != "demo" {
		t.Fatalf("unexpected project name %q", config.Name)
	}
	if config.Version != 1 {
		t.Fatalf("unexpected version %d", config.Version)
	}
}

func TestInitProjectRejectsNestedProject(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	nestedRoot := filepath.Join(projectRoot, "nested")
	_, err := InitProject(nestedRoot)
	if err == nil {
		t.Fatal("expected nested init to fail")
	}
}

func TestInitProjectRejectsNonFileEventLogPath(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	eventLogDir := filepath.Join(projectRoot, ".timber", "events.log")
	if err := os.MkdirAll(eventLogDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	_, err := InitProject(projectRoot)
	if err == nil {
		t.Fatal("expected init to fail when event log path is not a file")
	}
}

func TestAddRepoRegistersFirstRepoAsDefault(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	if err := AddRepo(projectRoot, "app", "git@github.com:company/app.git"); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}

	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("LoadProjectConfig returned error: %v", err)
	}

	repo, ok := config.Repos["app"]
	if !ok {
		t.Fatal("expected app repo to be registered")
	}
	if repo.URL != "git@github.com:company/app.git" {
		t.Fatalf("unexpected repo URL %q", repo.URL)
	}
	if config.DefaultRepo != "app" {
		t.Fatalf("unexpected default repo %q", config.DefaultRepo)
	}
}

func TestAddRepoClearsDefaultRepoWhenProjectBecomesMultiRepo(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	if err := AddRepo(projectRoot, "app", "git@github.com:company/app.git"); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", "git@github.com:company/auth.git"); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}

	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("LoadProjectConfig returned error: %v", err)
	}
	if config.DefaultRepo != "" {
		t.Fatalf("expected default repo to be cleared, got %q", config.DefaultRepo)
	}
}

func TestRemoveRepoDeletesRegistrationAndMirror(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}

	if err := RemoveRepo(projectRoot, "app"); err != nil {
		t.Fatalf("RemoveRepo returned error: %v", err)
	}

	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("LoadProjectConfig returned error: %v", err)
	}
	if len(config.Repos) != 0 {
		t.Fatalf("expected no registered repos, got %+v", config.Repos)
	}
	if config.DefaultRepo != "" {
		t.Fatalf("expected default repo to be cleared, got %q", config.DefaultRepo)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".timber/repos", "app.git")); !os.IsNotExist(err) {
		t.Fatalf("expected synced mirror to be removed, stat err=%v", err)
	}
}

func TestRemoveRepoRefusesRepoStillUsedByPath(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepoWithDefaultBranch(t, "main")
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "login-fix", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	if err := RemoveRepo(projectRoot, "app"); err == nil || !strings.Contains(err.Error(), "path login-fix") {
		t.Fatalf("expected path-usage error, got %v", err)
	}
}

func TestRemoveRepoPromotesLastRemainingRepoToDefault(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	if err := AddRepo(projectRoot, "app", "git@github.com:company/app.git"); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", "git@github.com:company/auth.git"); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}

	if err := RemoveRepo(projectRoot, "auth"); err != nil {
		t.Fatalf("RemoveRepo returned error: %v", err)
	}

	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		t.Fatalf("LoadProjectConfig returned error: %v", err)
	}
	if config.DefaultRepo != "app" {
		t.Fatalf("expected app to become default repo, got %q", config.DefaultRepo)
	}
}

func TestAddRepoRejectsDuplicateName(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	if err := AddRepo(projectRoot, "app", "git@github.com:company/app.git"); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "app", "git@github.com:company/other.git"); err == nil {
		t.Fatal("expected duplicate repo registration to fail")
	}
}

func TestSyncReposClonesAndFetchesRegisteredRepos(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}

	results, err := SyncRepos(projectRoot)
	if err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if len(results) != 1 || results[0].Action != "cloned" || results[0].Name != "app" {
		t.Fatalf("unexpected initial sync results: %+v", results)
	}

	mirrorPath := filepath.Join(projectRoot, ".timber/repos", "app.git")
	if _, err := os.Stat(mirrorPath); err != nil {
		t.Fatalf("expected mirror repo to exist: %v", err)
	}
	assertGitRefExists(t, projectRoot, "refs/remotes/origin/main", mirrorPath)

	addCommitAndBranch(t, remoteURL)

	results, err = SyncRepos(projectRoot)
	if err != nil {
		t.Fatalf("SyncRepos returned error on second sync: %v", err)
	}
	if len(results) != 1 || results[0].Action != "fetched" || results[0].Name != "app" {
		t.Fatalf("unexpected second sync results: %+v", results)
	}
	assertGitRefExists(t, projectRoot, "refs/remotes/origin/feature/test", mirrorPath)
}

func TestSyncReposPreservesPathPrivateBranches(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}

	result, err := CreatePath(projectRoot, CreatePathOptions{
		Name:       "login-fix",
		DefaultRef: "main",
	})
	if err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	mirrorPath := filepath.Join(projectRoot, ".timber/repos", "app.git")
	assertGitRefExists(t, projectRoot, "refs/heads/"+result.PrivateBranch, mirrorPath)

	addCommitAndBranch(t, remoteURL)

	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error on second sync: %v", err)
	}

	assertGitRefExists(t, projectRoot, "refs/heads/"+result.PrivateBranch, mirrorPath)

	status, err := GetPathStatus(projectRoot, "login-fix")
	if err != nil {
		t.Fatalf("GetPathStatus returned error: %v", err)
	}
	if status.StatusSummary != "clean" {
		t.Fatalf("expected clean path status after sync, got %q", status.StatusSummary)
	}
}

func TestSyncReposWithNoRegisteredReposReturnsEmptyResult(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	results, err := SyncRepos(projectRoot)
	if err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no sync results, got %+v", results)
	}
}

func TestListReposReportsRegisteredAndSyncedStatus(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepo(t)
	authRemote := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", authRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}

	entries, err := ListRepos(projectRoot)
	if err != nil {
		t.Fatalf("ListRepos returned error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 repo entries, got %+v", entries)
	}
	if entries[0].Name != "app" || entries[0].Status != "synced" {
		t.Fatalf("unexpected first repo entry %+v", entries[0])
	}
	if entries[1].Name != "auth" || entries[1].Status != "synced" {
		t.Fatalf("unexpected second repo entry %+v", entries[1])
	}
}

func TestListReposShowsRegisteredStatusBeforeSync(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}

	entries, err := ListRepos(projectRoot)
	if err != nil {
		t.Fatalf("ListRepos returned error: %v", err)
	}
	if len(entries) != 1 || entries[0].Status != "registered" {
		t.Fatalf("unexpected repo entries %+v", entries)
	}
}

func TestCreatePathCreatesSingleRepoPath(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}

	result, err := CreatePath(projectRoot, CreatePathOptions{Name: "login-fix", DefaultRef: "main"})
	if err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}
	if result.Name != "login-fix" {
		t.Fatalf("unexpected path name %q", result.Name)
	}
	if result.RepoName != "app" {
		t.Fatalf("unexpected repo name %q", result.RepoName)
	}

	pathRepoPath := filepath.Join(projectRoot, "paths", "login-fix", "app")
	if _, err := os.Stat(filepath.Join(pathRepoPath, "README.md")); err != nil {
		t.Fatalf("expected checked out repo content: %v", err)
	}
	headBranch, err := gitOutput(projectRoot, "-C", pathRepoPath, "branch", "--show-current")
	if err != nil {
		t.Fatalf("gitOutput returned error: %v", err)
	}
	if headBranch != result.PrivateBranch {
		t.Fatalf("unexpected branch %q", headBranch)
	}

	pathConfigPath := filepath.Join(projectRoot, ".timber", "paths", "login-fix.yaml")
	data, err := os.ReadFile(pathConfigPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	contents := string(data)
	for _, part := range []string{
		"name: login-fix",
		"default_source_display: main",
		"private_branch: " + result.PrivateBranch,
	} {
		if !strings.Contains(contents, part) {
			t.Fatalf("path metadata missing %q\n%s", part, contents)
		}
	}

	pathMD, err := os.ReadFile(filepath.Join(projectRoot, "paths", "login-fix", "PATH.md"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(pathMD), "# login-fix") {
		t.Fatalf("PATH.md missing path heading:\n%s", pathMD)
	}

	events, err := os.ReadFile(filepath.Join(projectRoot, ".timber", "events.log"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(events), `"event":"path_created"`) || !strings.Contains(string(events), `"path":"login-fix"`) {
		t.Fatalf("unexpected event log contents:\n%s", events)
	}
}

func TestCreatePathRequiresSingleRepoProject(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}
	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}

	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "login-fix", DefaultRef: "main", SelectedRepos: []string{"app", "auth"}}); err != nil {
		t.Fatalf("expected CreatePath to support explicit multi-repo selection: %v", err)
	}
}

func TestCreatePathRequiresReposSelectionInMultiRepoProjectWithoutExplicitRepos(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}
	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}

	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "login-fix", DefaultRef: "main"}); err == nil {
		t.Fatal("expected CreatePath to require --repos for multi-repo projects")
	}
}

func TestCreatePathCreatesMultiRepoPathFromExplicitRepos(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepo(t)
	authRemote := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", authRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}

	result, err := CreatePath(projectRoot, CreatePathOptions{Name: "auth-flow", DefaultRef: "main", SelectedRepos: []string{"auth", "app"}})
	if err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}
	if len(result.RepoNames) != 2 || result.RepoName != "" || result.PrivateBranch != "" {
		t.Fatalf("unexpected create result: %+v", result)
	}

	for _, repoName := range []string{"auth", "app"} {
		repoPath := filepath.Join(projectRoot, "paths", "auth-flow", repoName)
		if _, err := os.Stat(filepath.Join(repoPath, "README.md")); err != nil {
			t.Fatalf("expected repo content for %s: %v", repoName, err)
		}
	}

	pathConfigPath := filepath.Join(projectRoot, ".timber", "paths", "auth-flow.yaml")
	data, err := os.ReadFile(pathConfigPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	contents := string(data)
	for _, part := range []string{
		"name: auth-flow",
		"path: paths/auth-flow/app",
		"path: paths/auth-flow/auth",
	} {
		if !strings.Contains(contents, part) {
			t.Fatalf("path metadata missing %q\n%s", part, contents)
		}
	}

	pathMD, err := os.ReadFile(filepath.Join(projectRoot, "paths", "auth-flow", "PATH.md"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	md := string(pathMD)
	for _, part := range []string{"# auth-flow", "Repos: app, auth", "| auth |", "| app |"} {
		if !strings.Contains(md, part) {
			t.Fatalf("PATH.md missing %q\n%s", part, md)
		}
	}
}

func TestCreatePathCreatesMixedRefPathFromMappingsOnly(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepoWithDefaultBranch(t, "main")
	authRemote := createRemoteRepoWithDefaultBranch(t, "develop")
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", authRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}

	result, err := CreatePath(projectRoot, CreatePathOptions{
		Name:     "review-auth",
		RepoRefs: map[string]string{"app": "main", "auth": "develop"},
	})
	if err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}
	if len(result.RepoNames) != 2 {
		t.Fatalf("unexpected create result: %+v", result)
	}

	data, err := os.ReadFile(filepath.Join(projectRoot, ".timber", "paths", "review-auth.yaml"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	contents := string(data)
	for _, part := range []string{
		"description: app=main auth=develop",
		"source_display: main",
		"source_display: develop",
	} {
		if !strings.Contains(contents, part) {
			t.Fatalf("path metadata missing %q\n%s", part, contents)
		}
	}
}

func TestCreatePathAppliesRepoRefOverridesOnTopOfDefaultRef(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepoWithDefaultBranch(t, "main")
	authRemote := createRemoteRepoWithDefaultBranch(t, "develop")
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", authRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}

	_, err := CreatePath(projectRoot, CreatePathOptions{
		Name:          "auth-flow",
		DefaultRef:    "main",
		SelectedRepos: []string{"app", "auth"},
		RepoRefs:      map[string]string{"auth": "develop"},
	})
	if err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectRoot, ".timber", "paths", "auth-flow.yaml"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	contents := string(data)
	if !strings.Contains(contents, "description: main with auth=develop") {
		t.Fatalf("path metadata missing mixed parent description:\n%s", contents)
	}
}

func TestCreatePathWithIncludeAllUsesAllRegisteredRepos(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepoWithDefaultBranch(t, "main")
	authRemote := createRemoteRepoWithDefaultBranch(t, "main")
	billingRemote := createRemoteRepoWithDefaultBranch(t, "main")
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", authRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "billing", billingRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}

	result, err := CreatePath(projectRoot, CreatePathOptions{
		Name:       "full-main",
		DefaultRef: "main",
		IncludeAll: true,
	})
	if err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}
	if len(result.RepoNames) != 3 {
		t.Fatalf("unexpected create result: %+v", result)
	}

	for _, repoName := range []string{"app", "auth", "billing"} {
		if _, err := os.Stat(filepath.Join(projectRoot, "paths", "full-main", repoName, "README.md")); err != nil {
			t.Fatalf("expected repo content for %s: %v", repoName, err)
		}
	}
}

func TestAddReposToPathUsesPathDefaultRefWhenAvailable(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepoWithDefaultBranch(t, "main")
	authRemote := createRemoteRepoWithDefaultBranch(t, "main")
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", authRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "login-fix", DefaultRef: "main", SelectedRepos: []string{"app"}}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	result, err := AddReposToPath(projectRoot, AddReposOptions{
		PathName:  "login-fix",
		RepoNames: []string{"auth"},
	})
	if err != nil {
		t.Fatalf("AddReposToPath returned error: %v", err)
	}
	if len(result.AddedRepos) != 1 || result.AddedRepos[0] != "auth" {
		t.Fatalf("unexpected add result: %+v", result)
	}

	if _, err := os.Stat(filepath.Join(projectRoot, "paths", "login-fix", "auth", "README.md")); err != nil {
		t.Fatalf("expected added repo content: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(projectRoot, ".timber", "paths", "login-fix.yaml"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(data), "path: paths/login-fix/auth") {
		t.Fatalf("path metadata missing added repo:\n%s", data)
	}
}

func TestAddReposToPathSupportsRepoSpecificRefs(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepoWithDefaultBranch(t, "main")
	authRemote := createRemoteRepoWithDefaultBranch(t, "develop")
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", authRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{
		Name:     "review-auth",
		RepoRefs: map[string]string{"app": "main"},
	}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	_, err := AddReposToPath(projectRoot, AddReposOptions{
		PathName: "review-auth",
		RepoRefs: map[string]string{"auth": "develop"},
	})
	if err != nil {
		t.Fatalf("AddReposToPath returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectRoot, ".timber", "paths", "review-auth.yaml"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(data), "source_display: develop") {
		t.Fatalf("path metadata missing repo-specific added ref:\n%s", data)
	}
}

func TestAddReposToPathUpdatesRemoteParentSummaryAndSupportsStatus(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepoWithDefaultBranch(t, "qa")
	saltRemote := createRemoteRepoWithDefaultBranch(t, "master")
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "salt0", saltRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "deploy", DefaultRef: "qa", SelectedRepos: []string{"app"}}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}
	if _, err := AddReposToPath(projectRoot, AddReposOptions{PathName: "deploy", RepoRefs: map[string]string{"salt0": "master"}}); err != nil {
		t.Fatalf("AddReposToPath returned error: %v", err)
	}

	paths, err := ListPaths(projectRoot)
	if err != nil {
		t.Fatalf("ListPaths returned error: %v", err)
	}
	if len(paths) != 1 || paths[0].From != "qa + salt0=master" {
		t.Fatalf("unexpected path summaries %+v", paths)
	}

	info, err := GetPathInfo(projectRoot, "deploy")
	if err != nil {
		t.Fatalf("GetPathInfo returned error: %v", err)
	}
	if info.From != "qa + salt0=master" {
		t.Fatalf("unexpected info from %q", info.From)
	}
}

func TestGetPathInfoHandlesMissingStoredSourceCommitAfterRepoAdd(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepoWithDefaultBranch(t, "qa")
	saltRemote := createRemoteRepoWithDefaultBranch(t, "master")
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "salt0", saltRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "deploy", DefaultRef: "qa", SelectedRepos: []string{"app"}}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}
	if _, err := AddReposToPath(projectRoot, AddReposOptions{PathName: "deploy", RepoRefs: map[string]string{"salt0": "master"}}); err != nil {
		t.Fatalf("AddReposToPath returned error: %v", err)
	}

	metadataPath := filepath.Join(projectRoot, ".timber", "paths", "deploy.yaml")
	config, err := loadPathConfig(metadataPath)
	if err != nil {
		t.Fatalf("loadPathConfig returned error: %v", err)
	}
	repoState := config.Repos["salt0"]
	repoState.SourceCommit = ""
	config.Repos["salt0"] = repoState
	if err := writePathConfig(metadataPath, config); err != nil {
		t.Fatalf("writePathConfig returned error: %v", err)
	}

	info, err := GetPathInfo(projectRoot, "deploy")
	if err != nil {
		t.Fatalf("GetPathInfo returned error: %v", err)
	}
	if info.ReposInfo[1].RepoName != "salt0" {
		t.Fatalf("unexpected repo ordering/info %+v", info.ReposInfo)
	}

	status, err := GetPathStatus(projectRoot, "deploy")
	if err != nil {
		t.Fatalf("GetPathStatus returned error: %v", err)
	}
	if status.ReposStatus[1].RepoName != "salt0" || status.ReposStatus[1].CommitsAhead != 0 {
		t.Fatalf("unexpected repo status %+v", status.ReposStatus)
	}
}

func TestGetPathStatusHandlesInvalidStoredBaseline(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepoWithDefaultBranch(t, "main")
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "login-fix", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	metadataPath := filepath.Join(projectRoot, ".timber", "paths", "login-fix.yaml")
	config, err := loadPathConfig(metadataPath)
	if err != nil {
		t.Fatalf("loadPathConfig returned error: %v", err)
	}
	repoState := config.Repos["app"]
	repoState.SourceCommit = "59494ead9effae6caec0fc12d7b5a4e577478436"
	repoState.SourceRef = "refs/heads/does-not-exist"
	config.Repos["app"] = repoState
	if err := writePathConfig(metadataPath, config); err != nil {
		t.Fatalf("writePathConfig returned error: %v", err)
	}

	status, err := GetPathStatus(projectRoot, "login-fix")
	if err != nil {
		t.Fatalf("GetPathStatus returned error: %v", err)
	}
	if len(status.ReposStatus) != 1 || status.ReposStatus[0].CommitsAhead != 0 {
		t.Fatalf("unexpected status %+v", status.ReposStatus)
	}

	if _, err := GetPathInfo(projectRoot, "login-fix"); err != nil {
		t.Fatalf("GetPathInfo returned error: %v", err)
	}
}

func TestAddReposToPathRejectsExistingRepo(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepoWithDefaultBranch(t, "main")
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "login-fix", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	if _, err := AddReposToPath(projectRoot, AddReposOptions{
		PathName:  "login-fix",
		RepoNames: []string{"app"},
	}); err == nil {
		t.Fatal("expected AddReposToPath to reject a repo already present in the path")
	}
}

func TestForkPathCreatesChildPathsFromSourceHead(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "auth-flow", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	sourceRepoPath := filepath.Join(projectRoot, "paths", "auth-flow", "app")
	runGit(t, sourceRepoPath, "config", "user.name", "Timber Test")
	runGit(t, sourceRepoPath, "config", "user.email", "timber@example.com")
	if err := os.WriteFile(filepath.Join(sourceRepoPath, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	runGit(t, sourceRepoPath, "add", "feature.txt")
	runGit(t, sourceRepoPath, "commit", "-m", "source progress")
	sourceHead, err := gitOutput(projectRoot, "-C", sourceRepoPath, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("gitOutput returned error: %v", err)
	}

	result, err := ForkPath(projectRoot, "auth-flow", []string{"try-a", "try-b"})
	if err != nil {
		t.Fatalf("ForkPath returned error: %v", err)
	}
	if result.SourcePath != "auth-flow" {
		t.Fatalf("unexpected source path %q", result.SourcePath)
	}
	if len(result.Created) != 2 {
		t.Fatalf("expected two children, got %+v", result.Created)
	}

	childRepoPath := filepath.Join(projectRoot, "paths", "try-a", "app")
	childHead, err := gitOutput(projectRoot, "-C", childRepoPath, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("gitOutput returned error: %v", err)
	}
	if childHead != sourceHead {
		t.Fatalf("unexpected child fork point: got %q want %q", childHead, sourceHead)
	}

	childConfig, err := loadPathConfig(filepath.Join(projectRoot, ".timber", "paths", "try-a.yaml"))
	if err != nil {
		t.Fatalf("loadPathConfig returned error: %v", err)
	}
	if childConfig.Parent.Type != "path" {
		t.Fatalf("unexpected parent type %q", childConfig.Parent.Type)
	}
	if childConfig.Parent.PathName != "auth-flow" {
		t.Fatalf("unexpected parent path name %q", childConfig.Parent.PathName)
	}
	if childConfig.Parent.Description != "auth-flow" {
		t.Fatalf("unexpected parent description %q", childConfig.Parent.Description)
	}
	if childConfig.Repos["app"].SourceCommit != sourceHead {
		t.Fatalf("unexpected stored source commit %q", childConfig.Repos["app"].SourceCommit)
	}
}

func TestForkPathRejectsDirtySourcePath(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "auth-flow", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	sourceRepoPath := filepath.Join(projectRoot, "paths", "auth-flow", "app")
	if err := os.WriteFile(filepath.Join(sourceRepoPath, "README.md"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if _, err := ForkPath(projectRoot, "auth-flow", []string{"try-a"}); err == nil {
		t.Fatal("expected ForkPath to reject a dirty source path")
	}
}

func TestKeepPathMergesChildIntoParent(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "auth-flow", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}
	if _, err := ForkPath(projectRoot, "auth-flow", []string{"try-a"}); err != nil {
		t.Fatalf("ForkPath returned error: %v", err)
	}

	parentRepoPath := filepath.Join(projectRoot, "paths", "auth-flow", "app")
	childRepoPath := filepath.Join(projectRoot, "paths", "try-a", "app")
	for _, repoPath := range []string{parentRepoPath, childRepoPath} {
		runGit(t, repoPath, "config", "user.name", "Timber Test")
		runGit(t, repoPath, "config", "user.email", "timber@example.com")
	}

	if err := os.WriteFile(filepath.Join(childRepoPath, "feature.txt"), []byte("kept\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	runGit(t, childRepoPath, "add", "feature.txt")
	runGit(t, childRepoPath, "commit", "-m", "child feature")

	result, err := KeepPath(projectRoot, "try-a", "auth-flow")
	if err != nil {
		t.Fatalf("KeepPath returned error: %v", err)
	}
	if result.SourcePath != "try-a" || result.TargetPath != "auth-flow" {
		t.Fatalf("unexpected keep result %+v", result)
	}
	if strings.Join(result.MergedRepos, ",") != "app" {
		t.Fatalf("unexpected merged repos %+v", result.MergedRepos)
	}

	data, err := os.ReadFile(filepath.Join(parentRepoPath, "feature.txt"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.TrimSpace(string(data)) != "kept" {
		t.Fatalf("unexpected kept file contents %q", string(data))
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".timber", "operations", "keep.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected keep operation state to be removed, got %v", err)
	}
	eventLog, err := os.ReadFile(filepath.Join(projectRoot, ".timber", "events.log"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(eventLog), `"event":"keep"`) {
		t.Fatalf("expected keep event in event log:\n%s", eventLog)
	}
}

func TestKeepPathConflictCanContinueAndAbort(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "auth-flow", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	parentRepoPath := filepath.Join(projectRoot, "paths", "auth-flow", "app")
	runGit(t, parentRepoPath, "config", "user.name", "Timber Test")
	runGit(t, parentRepoPath, "config", "user.email", "timber@example.com")

	if err := os.WriteFile(filepath.Join(parentRepoPath, "shared.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	runGit(t, parentRepoPath, "add", "shared.txt")
	runGit(t, parentRepoPath, "commit", "-m", "baseline")

	if _, err := ForkPath(projectRoot, "auth-flow", []string{"try-a"}); err != nil {
		t.Fatalf("ForkPath returned error: %v", err)
	}

	childRepoPath := filepath.Join(projectRoot, "paths", "try-a", "app")
	runGit(t, childRepoPath, "config", "user.name", "Timber Test")
	runGit(t, childRepoPath, "config", "user.email", "timber@example.com")

	if err := os.WriteFile(filepath.Join(parentRepoPath, "shared.txt"), []byte("parent\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	runGit(t, parentRepoPath, "add", "shared.txt")
	runGit(t, parentRepoPath, "commit", "-m", "parent change")

	if err := os.WriteFile(filepath.Join(childRepoPath, "shared.txt"), []byte("child\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	runGit(t, childRepoPath, "add", "shared.txt")
	runGit(t, childRepoPath, "commit", "-m", "child change")

	if _, err := KeepPath(projectRoot, "try-a", "auth-flow"); err == nil {
		t.Fatal("expected KeepPath to stop on conflict")
	}

	statePath := filepath.Join(projectRoot, ".timber", "operations", "keep.yaml")
	state, err := loadKeepOperationState(statePath)
	if err != nil {
		t.Fatalf("loadKeepOperationState returned error: %v", err)
	}
	if state.Repos["app"].Status != "conflicted" {
		t.Fatalf("expected conflicted repo status, got %+v", state.Repos["app"])
	}

	if err := AbortKeepPath(projectRoot); err != nil {
		t.Fatalf("AbortKeepPath returned error: %v", err)
	}
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatalf("expected state file removal, got %v", err)
	}

	runGit(t, parentRepoPath, "merge", "--abort")

	if _, err := KeepPath(projectRoot, "try-a", "auth-flow"); err == nil {
		t.Fatal("expected KeepPath to stop on conflict again")
	}
	if err := os.WriteFile(filepath.Join(parentRepoPath, "shared.txt"), []byte("resolved\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	runGit(t, parentRepoPath, "add", "shared.txt")

	result, err := ContinueKeepPath(projectRoot)
	if err != nil {
		t.Fatalf("ContinueKeepPath returned error: %v", err)
	}
	if strings.Join(result.MergedRepos, ",") != "app" {
		t.Fatalf("unexpected merged repos after continue %+v", result.MergedRepos)
	}
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatalf("expected state file removal after continue, got %v", err)
	}
	mergedContent, err := os.ReadFile(filepath.Join(parentRepoPath, "shared.txt"))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.TrimSpace(string(mergedContent)) != "resolved" {
		t.Fatalf("unexpected resolved content %q", string(mergedContent))
	}
}

func TestDropPathsRejectsDirtyWithoutForce(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}
	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "login-fix", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	repoPath := filepath.Join(projectRoot, "paths", "login-fix", "app")
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if _, err := DropPaths(projectRoot, DropPathOptions{PathNames: []string{"login-fix"}}); err == nil {
		t.Fatal("expected DropPaths to reject dirty path without force")
	}
}

func TestDropPathsRejectsParentWithoutRecursive(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}
	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "auth-flow", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}
	if _, err := ForkPath(projectRoot, "auth-flow", []string{"try-a"}); err != nil {
		t.Fatalf("ForkPath returned error: %v", err)
	}

	if _, err := DropPaths(projectRoot, DropPathOptions{PathNames: []string{"auth-flow"}}); err == nil {
		t.Fatal("expected DropPaths to reject parent path without recursive")
	}
}

func TestDropPathsRecursiveForceKeepsBranches(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}
	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "auth-flow", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}
	if _, err := ForkPath(projectRoot, "auth-flow", []string{"try-a"}); err != nil {
		t.Fatalf("ForkPath returned error: %v", err)
	}

	childRepoPath := filepath.Join(projectRoot, "paths", "try-a", "app")
	runGit(t, childRepoPath, "config", "user.name", "Timber Test")
	runGit(t, childRepoPath, "config", "user.email", "timber@example.com")
	if err := os.WriteFile(filepath.Join(childRepoPath, "feature.txt"), []byte("keep branch\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	runGit(t, childRepoPath, "add", "feature.txt")
	runGit(t, childRepoPath, "commit", "-m", "ahead")

	result, err := DropPaths(projectRoot, DropPathOptions{
		PathNames: []string{"auth-flow"},
		Force:     true,
		Recursive: true,
	})
	if err != nil {
		t.Fatalf("DropPaths returned error: %v", err)
	}
	if strings.Join(result.Dropped, ",") != "try-a,auth-flow" {
		t.Fatalf("unexpected dropped paths %+v", result.Dropped)
	}
	if len(result.KeptBranches) == 0 {
		t.Fatalf("expected kept branches for forced drop, got %+v", result)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, ".timber", "paths", "auth-flow.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected auth-flow metadata removal, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "paths", "auth-flow")); !os.IsNotExist(err) {
		t.Fatalf("expected auth-flow path removal, got %v", err)
	}
}

func TestDropPathsDeletesBranchesWhenRequested(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}
	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	createResult, err := CreatePath(projectRoot, CreatePathOptions{Name: "login-fix", DefaultRef: "main"})
	if err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	result, err := DropPaths(projectRoot, DropPathOptions{
		PathNames:      []string{"login-fix"},
		DeleteBranches: true,
	})
	if err != nil {
		t.Fatalf("DropPaths returned error: %v", err)
	}
	if strings.Join(result.Dropped, ",") != "login-fix" {
		t.Fatalf("unexpected dropped paths %+v", result.Dropped)
	}
	mirrorPath := filepath.Join(projectRoot, ".timber/repos", "app.git")
	if _, err := gitOutput(projectRoot, "-C", mirrorPath, "rev-parse", "--verify", createResult.PrivateBranch); err == nil {
		t.Fatalf("expected branch %q to be deleted", createResult.PrivateBranch)
	}
}

func TestCreatePathFailsWithoutPartialPathWhenARepoRefIsMissing(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepo(t)
	authRemote := createRemoteRepoWithDefaultBranch(t, "develop")
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", authRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}

	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "auth-flow", DefaultRef: "main", SelectedRepos: []string{"app", "auth"}}); err == nil {
		t.Fatal("expected CreatePath to fail when a selected repo cannot resolve the shared ref")
	}
	if _, err := os.Stat(filepath.Join(projectRoot, "paths", "auth-flow")); !os.IsNotExist(err) {
		t.Fatalf("expected no partial path tree, got stat err %v", err)
	}
}

func TestListPathsReturnsSortedMetadataEntries(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "zeta", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "alpha", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	paths, err := ListPaths(projectRoot)
	if err != nil {
		t.Fatalf("ListPaths returned error: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
	if paths[0].Name != "alpha" || paths[1].Name != "zeta" {
		t.Fatalf("unexpected path ordering: %+v", paths)
	}
	if paths[0].From != "main" {
		t.Fatalf("unexpected source display: %+v", paths[0])
	}
	if paths[0].Repos != "app" {
		t.Fatalf("unexpected repos string: %+v", paths[0])
	}
}

func TestCompletePathNamesFiltersByPrefix(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	for _, name := range []string{"login-fix", "login-ui", "billing"} {
		if _, err := CreatePath(projectRoot, CreatePathOptions{Name: name, DefaultRef: "main"}); err != nil {
			t.Fatalf("CreatePath returned error for %s: %v", name, err)
		}
	}

	names, err := CompletePathNames(projectRoot, "log")
	if err != nil {
		t.Fatalf("CompletePathNames returned error: %v", err)
	}
	want := []string{"login-fix", "login-ui"}
	if len(names) != len(want) {
		t.Fatalf("unexpected completion results: got %+v want %+v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("unexpected completion results: got %+v want %+v", names, want)
		}
	}
}

func TestCompleteRepoNamesReturnsRegisteredReposAndPathScopedRepos(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepoWithDefaultBranch(t, "main")
	authRemote := createRemoteRepoWithDefaultBranch(t, "main")
	billingRemote := createRemoteRepoWithDefaultBranch(t, "main")
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", authRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "billing", billingRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "auth-flow", DefaultRef: "main", SelectedRepos: []string{"app", "auth"}}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	allRepos, err := CompleteRepoNames(projectRoot, "", "a")
	if err != nil {
		t.Fatalf("CompleteRepoNames returned error: %v", err)
	}
	if strings.Join(allRepos, ",") != "app,auth" {
		t.Fatalf("unexpected all repo completion %v", allRepos)
	}

	pathRepos, err := CompleteRepoNames(projectRoot, "auth-flow", "a")
	if err != nil {
		t.Fatalf("CompleteRepoNames returned error: %v", err)
	}
	if strings.Join(pathRepos, ",") != "app,auth" {
		t.Fatalf("unexpected path repo completion %v", pathRepos)
	}
}

func TestCompleteRefsReturnsFriendlyRefNames(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepoWithDefaultBranch(t, "main")
	featureWork := filepath.Join(t.TempDir(), "feature-work")
	runGit(t, ".", "clone", remoteURL, featureWork)
	runGit(t, featureWork, "config", "user.name", "Timber Test")
	runGit(t, featureWork, "config", "user.email", "timber@example.com")
	runGit(t, featureWork, "checkout", "-b", "hotfix/123", "origin/main")
	if err := os.WriteFile(filepath.Join(featureWork, "fix.txt"), []byte("fix\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	runGit(t, featureWork, "add", "fix.txt")
	runGit(t, featureWork, "commit", "-m", "hotfix")
	runGit(t, featureWork, "push", "-u", "origin", "hotfix/123")

	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}

	refs, err := CompleteRefs(projectRoot, "app", "hot")
	if err != nil {
		t.Fatalf("CompleteRefs returned error: %v", err)
	}
	if len(refs) != 1 || refs[0] != "hotfix/123" {
		t.Fatalf("unexpected refs completion %v", refs)
	}
}

func TestGetPathStatusReportsCleanPath(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	result, err := CreatePath(projectRoot, CreatePathOptions{Name: "login-fix", DefaultRef: "main"})
	if err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	status, err := GetPathStatus(projectRoot, "login-fix")
	if err != nil {
		t.Fatalf("GetPathStatus returned error: %v", err)
	}
	if status.Name != "login-fix" || status.Repos != "app" {
		t.Fatalf("unexpected status identity: %+v", status)
	}
	if len(status.ReposStatus) != 1 || status.ReposStatus[0].RepoName != "app" {
		t.Fatalf("unexpected repo status details: %+v", status)
	}
	if status.ReposStatus[0].PrivateBranch != result.PrivateBranch {
		t.Fatalf("unexpected private branch %q", status.ReposStatus[0].PrivateBranch)
	}
	if status.StatusSummary != "clean" {
		t.Fatalf("unexpected status summary %q", status.StatusSummary)
	}
	if status.ReposStatus[0].CommitsAhead != 0 {
		t.Fatalf("unexpected ahead count %d", status.ReposStatus[0].CommitsAhead)
	}
}

func TestGetPathStatusReportsDirtyAndAheadPath(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	result, err := CreatePath(projectRoot, CreatePathOptions{Name: "login-fix", DefaultRef: "main"})
	if err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	repoPath := filepath.Join(projectRoot, "paths", "login-fix", "app")
	runGit(t, repoPath, "config", "user.name", "Timber Test")
	runGit(t, repoPath, "config", "user.email", "timber@example.com")
	if err := os.WriteFile(filepath.Join(repoPath, "committed.txt"), []byte("commit\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	runGit(t, repoPath, "add", "committed.txt")
	runGit(t, repoPath, "commit", "-m", "path commit")
	if err := os.WriteFile(filepath.Join(repoPath, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	status, err := GetPathStatus(projectRoot, "login-fix")
	if err != nil {
		t.Fatalf("GetPathStatus returned error: %v", err)
	}
	if len(status.ReposStatus) != 1 || status.ReposStatus[0].PrivateBranch != result.PrivateBranch {
		t.Fatalf("unexpected repo status details: %+v", status)
	}
	if status.StatusSummary != "1 modified, 1 ahead" {
		t.Fatalf("unexpected status summary %q", status.StatusSummary)
	}
	if status.ReposStatus[0].CommitsAhead != 1 {
		t.Fatalf("unexpected ahead count %d", status.ReposStatus[0].CommitsAhead)
	}
}

func TestGetPathPathReturnsAbsolutePathRoot(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "login-fix", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	got, err := GetPathPath(projectRoot, "login-fix")
	if err != nil {
		t.Fatalf("GetPathPath returned error: %v", err)
	}

	want := filepath.Join(projectRoot, "paths", "login-fix")
	if got != want {
		t.Fatalf("unexpected path path %q want %q", got, want)
	}
}

func TestRunPathCommandUsesPathRootAndTimberEnv(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "login-fix", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exitCode, err := RunPathCommand(projectRoot, "login-fix", []string{
		"sh", "-c", `printf '%s\n%s\n%s\n%s\n' "$PWD" "$TIMBER_PROJECT_ROOT" "$TIMBER_PATH" "$TIMBER_REPOS"`}, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("RunPathCommand returned error: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("unexpected exit code %d", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr %q", stderr.String())
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("unexpected stdout lines %#v", lines)
	}
	if lines[0] != filepath.Join(projectRoot, "paths", "login-fix") {
		t.Fatalf("unexpected working directory %q", lines[0])
	}
	if lines[1] != projectRoot {
		t.Fatalf("unexpected TIMBER_PROJECT_ROOT %q", lines[1])
	}
	if lines[2] != "login-fix" {
		t.Fatalf("unexpected TIMBER_PATH %q", lines[2])
	}
	if lines[3] != "app" {
		t.Fatalf("unexpected TIMBER_REPOS %q", lines[3])
	}
}

func TestRunPathCommandReturnsChildExitCode(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "login-fix", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	exitCode, err := RunPathCommand(projectRoot, "login-fix", []string{"sh", "-c", "exit 7"}, nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("RunPathCommand returned error: %v", err)
	}
	if exitCode != 7 {
		t.Fatalf("unexpected exit code %d", exitCode)
	}
}

func TestGetPathInfoReturnsHumanSummaryFields(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	result, err := CreatePath(projectRoot, CreatePathOptions{Name: "login-fix", DefaultRef: "main"})
	if err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	info, err := GetPathInfo(projectRoot, "login-fix")
	if err != nil {
		t.Fatalf("GetPathInfo returned error: %v", err)
	}
	if info.Name != "login-fix" || info.Path != "paths/login-fix" {
		t.Fatalf("unexpected info identity: %+v", info)
	}
	if info.From != "main" || info.Repos != "app" {
		t.Fatalf("unexpected info repo fields: %+v", info)
	}
	if len(info.ReposInfo) != 1 || info.ReposInfo[0].RepoName != "app" {
		t.Fatalf("unexpected repo info details: %+v", info)
	}
	if info.ReposInfo[0].PrivateBranch != result.PrivateBranch {
		t.Fatalf("unexpected private branch %q", info.ReposInfo[0].PrivateBranch)
	}
	if info.ReposInfo[0].SourceRef != "refs/remotes/origin/main" {
		t.Fatalf("unexpected source ref %q", info.ReposInfo[0].SourceRef)
	}
	if info.ReposInfo[0].SourceCommit == "" {
		t.Fatalf("expected short source commit in info: %+v", info)
	}
	if info.StatusSummary != "clean" {
		t.Fatalf("unexpected status summary %q", info.StatusSummary)
	}
	if info.Parent != "" {
		t.Fatalf("expected no parent summary, got %q", info.Parent)
	}
	if info.Recovery != "" {
		t.Fatalf("expected no recovery summary, got %q", info.Recovery)
	}
}

func TestGetPathStatusSummarizesMultiRepoPath(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepoWithDefaultBranch(t, "main")
	authRemote := createRemoteRepoWithDefaultBranch(t, "main")
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", authRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "auth-flow", DefaultRef: "main", SelectedRepos: []string{"app", "auth"}}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	appPath := filepath.Join(projectRoot, "paths", "auth-flow", "app")
	runGit(t, appPath, "config", "user.name", "Timber Test")
	runGit(t, appPath, "config", "user.email", "timber@example.com")
	if err := os.WriteFile(filepath.Join(appPath, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	authPath := filepath.Join(projectRoot, "paths", "auth-flow", "auth")
	runGit(t, authPath, "config", "user.name", "Timber Test")
	runGit(t, authPath, "config", "user.email", "timber@example.com")
	if err := os.WriteFile(filepath.Join(authPath, "commit.txt"), []byte("commit\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	runGit(t, authPath, "add", "commit.txt")
	runGit(t, authPath, "commit", "-m", "auth commit")

	status, err := GetPathStatus(projectRoot, "auth-flow")
	if err != nil {
		t.Fatalf("GetPathStatus returned error: %v", err)
	}
	if status.Repos != "app,auth" {
		t.Fatalf("unexpected repo summary %+v", status)
	}
	if status.StatusSummary != "1 modified, 1 ahead" {
		t.Fatalf("unexpected status summary %q", status.StatusSummary)
	}
	if len(status.ReposStatus) != 2 {
		t.Fatalf("unexpected repo statuses %+v", status.ReposStatus)
	}
}

func TestGetPathInfoReturnsMultiRepoDetails(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepoWithDefaultBranch(t, "main")
	authRemote := createRemoteRepoWithDefaultBranch(t, "develop")
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", authRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{
		Name:     "review-auth",
		RepoRefs: map[string]string{"app": "main", "auth": "develop"},
	}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	info, err := GetPathInfo(projectRoot, "review-auth")
	if err != nil {
		t.Fatalf("GetPathInfo returned error: %v", err)
	}
	if info.Repos != "app,auth" {
		t.Fatalf("unexpected repo summary %+v", info)
	}
	if len(info.ReposInfo) != 2 {
		t.Fatalf("unexpected repo info %+v", info)
	}
	if info.ReposInfo[0].RepoName != "app" || info.ReposInfo[1].RepoName != "auth" {
		t.Fatalf("unexpected repo ordering %+v", info.ReposInfo)
	}
}

func TestPathInfoAndStatusShowLineageAndRecovery(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	remoteURL := createRemoteRepo(t)
	if err := AddRepo(projectRoot, "app", remoteURL); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "auth-flow", DefaultRef: "main"}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	parentRepoPath := filepath.Join(projectRoot, "paths", "auth-flow", "app")
	runGit(t, parentRepoPath, "config", "user.name", "Timber Test")
	runGit(t, parentRepoPath, "config", "user.email", "timber@example.com")
	if err := os.WriteFile(filepath.Join(parentRepoPath, "shared.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	runGit(t, parentRepoPath, "add", "shared.txt")
	runGit(t, parentRepoPath, "commit", "-m", "baseline")

	if _, err := ForkPath(projectRoot, "auth-flow", []string{"try-a"}); err != nil {
		t.Fatalf("ForkPath returned error: %v", err)
	}

	childRepoPath := filepath.Join(projectRoot, "paths", "try-a", "app")
	runGit(t, childRepoPath, "config", "user.name", "Timber Test")
	runGit(t, childRepoPath, "config", "user.email", "timber@example.com")

	if err := os.WriteFile(filepath.Join(parentRepoPath, "shared.txt"), []byte("parent\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	runGit(t, parentRepoPath, "add", "shared.txt")
	runGit(t, parentRepoPath, "commit", "-m", "parent change")

	if err := os.WriteFile(filepath.Join(childRepoPath, "shared.txt"), []byte("child\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	runGit(t, childRepoPath, "add", "shared.txt")
	runGit(t, childRepoPath, "commit", "-m", "child change")

	if _, err := KeepPath(projectRoot, "try-a", "auth-flow"); err == nil {
		t.Fatal("expected KeepPath to stop on conflict")
	}

	parentInfo, err := GetPathInfo(projectRoot, "auth-flow")
	if err != nil {
		t.Fatalf("GetPathInfo returned error: %v", err)
	}
	if strings.Join(parentInfo.Children, ",") != "try-a" {
		t.Fatalf("unexpected parent children %+v", parentInfo.Children)
	}
	if !strings.Contains(parentInfo.Recovery, "timber keep --continue") {
		t.Fatalf("expected recovery guidance in parent info, got %q", parentInfo.Recovery)
	}

	childStatus, err := GetPathStatus(projectRoot, "try-a")
	if err != nil {
		t.Fatalf("GetPathStatus returned error: %v", err)
	}
	if childStatus.Parent != "auth-flow" {
		t.Fatalf("unexpected child parent %q", childStatus.Parent)
	}
	if !strings.Contains(childStatus.Recovery, "conflicts in app") {
		t.Fatalf("expected conflict recovery summary, got %q", childStatus.Recovery)
	}
}

func TestGetPathDiffReturnsPerRepoSections(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepoWithDefaultBranch(t, "main")
	authRemote := createRemoteRepoWithDefaultBranch(t, "main")
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", authRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "auth-flow", DefaultRef: "main", SelectedRepos: []string{"app", "auth"}}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	appPath := filepath.Join(projectRoot, "paths", "auth-flow", "app")
	if err := os.WriteFile(filepath.Join(appPath, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appPath, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	sections, err := GetPathDiff(projectRoot, "auth-flow", "")
	if err != nil {
		t.Fatalf("GetPathDiff returned error: %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("unexpected diff sections %+v", sections)
	}
	if sections[0].RepoName != "app" || !strings.Contains(sections[0].Diff, "README.md") {
		t.Fatalf("unexpected app diff section %+v", sections[0])
	}
	if sections[1].RepoName != "auth" || strings.TrimSpace(sections[1].Diff) != "" {
		t.Fatalf("unexpected auth diff section %+v", sections[1])
	}
}

func TestGetPathDiffSupportsRepoFiltering(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	appRemote := createRemoteRepoWithDefaultBranch(t, "main")
	authRemote := createRemoteRepoWithDefaultBranch(t, "main")
	if err := AddRepo(projectRoot, "app", appRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if err := AddRepo(projectRoot, "auth", authRemote); err != nil {
		t.Fatalf("AddRepo returned error: %v", err)
	}
	if _, err := SyncRepos(projectRoot); err != nil {
		t.Fatalf("SyncRepos returned error: %v", err)
	}
	if _, err := CreatePath(projectRoot, CreatePathOptions{Name: "auth-flow", DefaultRef: "main", SelectedRepos: []string{"app", "auth"}}); err != nil {
		t.Fatalf("CreatePath returned error: %v", err)
	}

	appPath := filepath.Join(projectRoot, "paths", "auth-flow", "app")
	if err := os.WriteFile(filepath.Join(appPath, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appPath, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	sections, err := GetPathDiff(projectRoot, "auth-flow", "app")
	if err != nil {
		t.Fatalf("GetPathDiff returned error: %v", err)
	}
	if len(sections) != 1 || sections[0].RepoName != "app" {
		t.Fatalf("unexpected filtered diff sections %+v", sections)
	}
}

func TestDetectContextFindsPathAndRepo(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "demo")
	if _, err := InitProject(projectRoot); err != nil {
		t.Fatalf("InitProject returned error: %v", err)
	}

	repoPath := filepath.Join(projectRoot, "paths", "auth-flow", "backend", "src")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	context, err := DetectContext(repoPath)
	if err != nil {
		t.Fatalf("DetectContext returned error: %v", err)
	}
	if context == nil {
		t.Fatal("expected non-nil context")
	}
	if context.ProjectRoot != projectRoot {
		t.Fatalf("unexpected project root %q", context.ProjectRoot)
	}
	if context.PathName != "auth-flow" {
		t.Fatalf("unexpected path %q", context.PathName)
	}
	if context.RepoName != "backend" {
		t.Fatalf("unexpected repo %q", context.RepoName)
	}
}

func createRemoteRepo(t *testing.T) string {
	return createRemoteRepoWithDefaultBranch(t, "main")
}

func createRemoteRepoWithDefaultBranch(t *testing.T, branch string) string {
	t.Helper()

	base := t.TempDir()
	remotePath := filepath.Join(base, "remote.git")
	workPath := filepath.Join(base, "work")

	runGit(t, base, "init", "--bare", remotePath)
	runGit(t, base, "clone", remotePath, workPath)
	runGit(t, workPath, "config", "user.name", "Timber Test")
	runGit(t, workPath, "config", "user.email", "timber@example.com")
	if err := os.WriteFile(filepath.Join(workPath, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	runGit(t, workPath, "add", "README.md")
	runGit(t, workPath, "commit", "-m", "initial commit")
	runGit(t, workPath, "branch", "-M", branch)
	runGit(t, workPath, "push", "-u", "origin", branch)
	runGit(t, remotePath, "symbolic-ref", "HEAD", "refs/heads/"+branch)

	return remotePath
}

func addCommitAndBranch(t *testing.T, remoteURL string) {
	t.Helper()

	workPath := filepath.Join(t.TempDir(), "work")
	runGit(t, ".", "clone", remoteURL, workPath)
	runGit(t, workPath, "config", "user.name", "Timber Test")
	runGit(t, workPath, "config", "user.email", "timber@example.com")
	runGit(t, workPath, "checkout", "-b", "feature/test", "origin/main")
	if err := os.WriteFile(filepath.Join(workPath, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	runGit(t, workPath, "add", "feature.txt")
	runGit(t, workPath, "commit", "-m", "add feature")
	runGit(t, workPath, "push", "-u", "origin", "feature/test")
}

func assertGitRefExists(t *testing.T, dir, ref, repoPath string) {
	t.Helper()

	cmd := exec.Command("git", "-C", repoPath, "rev-parse", ref)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-parse %s failed: %v\n%s", ref, err, output)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed in %s: %v\n%s", strings.Join(args, " "), dir, err, output)
	}
}
