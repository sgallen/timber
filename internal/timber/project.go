package timber

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type RepoConfig struct {
	URL        string `yaml:"url"`
	DefaultRef string `yaml:"default_ref,omitempty"`
}

type ProjectConfig struct {
	Version     int                   `yaml:"version"`
	Name        string                `yaml:"name"`
	CreatedAt   string                `yaml:"created_at"`
	ReposDir    string                `yaml:"repos_dir"`
	PathsDir    string                `yaml:"paths_dir"`
	MetadataDir string                `yaml:"metadata_dir"`
	DefaultRepo string                `yaml:"default_repo,omitempty"`
	Repos       map[string]RepoConfig `yaml:"repos,omitempty"`
}

type ProjectContext struct {
	ProjectRoot string
	PathName    string
	RepoName    string
}

type SyncResult struct {
	Name   string
	Action string
}

type RepoListEntry struct {
	Name   string
	Status string
	URL    string
}

type PathRepoState struct {
	Path            string `yaml:"path"`
	SourceRef       string `yaml:"source_ref"`
	SourceDisplay   string `yaml:"source_display"`
	SourceCommit    string `yaml:"source_commit"`
	PrivateBranch   string `yaml:"private_branch"`
	PublishedBranch string `yaml:"published_branch,omitempty"`
}

type PathParent struct {
	Type                 string `yaml:"type"`
	PathName             string `yaml:"path_name,omitempty"`
	Description          string `yaml:"description"`
	DefaultSourceRef     string `yaml:"default_source_ref"`
	DefaultSourceDisplay string `yaml:"default_source_display"`
}

type PathConfig struct {
	Version   int                      `yaml:"version"`
	Name      string                   `yaml:"name"`
	Path      string                   `yaml:"path"`
	CreatedAt string                   `yaml:"created_at"`
	Purpose   string                   `yaml:"purpose,omitempty"`
	Parent    PathParent               `yaml:"parent"`
	Repos     map[string]PathRepoState `yaml:"repos"`
}

type CreatePathResult struct {
	Name          string
	Path          string
	RepoName      string
	PrivateBranch string
	RepoNames     []string
}

type CreatePathOptions struct {
	Name          string
	DefaultRef    string
	SelectedRepos []string
	RepoRefs      map[string]string
	IncludeAll    bool
}

type AddReposOptions struct {
	PathName   string
	DefaultRef string
	RepoNames  []string
	RepoRefs   map[string]string
}

type AddReposResult struct {
	PathName   string
	AddedRepos []string
}

type ForkPathResult struct {
	SourcePath string
	Created    []CreatePathResult
}

type KeepPathResult struct {
	SourcePath   string
	TargetPath   string
	MergedRepos  []string
	SkippedRepos []string
}

type KeepOperationRepoState struct {
	Status          string `yaml:"status"`
	ForkPointCommit string `yaml:"fork_point_commit"`
	SourceBranch    string `yaml:"source_branch"`
	TargetBranch    string `yaml:"target_branch"`
}

type KeepOperationState struct {
	Type        string                            `yaml:"type"`
	SourcePath  string                            `yaml:"source_path"`
	TargetPath  string                            `yaml:"target_path"`
	RepoOrder   []string                          `yaml:"repo_order"`
	Repos       map[string]KeepOperationRepoState `yaml:"repos"`
	CreatedAt   string                            `yaml:"created_at"`
	LastUpdated string                            `yaml:"last_updated"`
}

type DropPathOptions struct {
	PathNames      []string
	Force          bool
	Recursive      bool
	KeepBranches   bool
	DeleteBranches bool
}

type DropPathResult struct {
	Dropped         []string
	KeptBranches    []string
	DeletedBranches []string
}

type PathListEntry struct {
	Name  string
	From  string
	Repos string
	Path  string
}

type PathStatus struct {
	Name          string
	Path          string
	From          string
	Repos         string
	Parent        string
	Children      []string
	Recovery      string
	StatusSummary string
	ReposStatus   []RepoPathStatus
}

type PathInfo struct {
	Name          string
	Path          string
	Created       string
	From          string
	Repos         string
	Parent        string
	Children      []string
	Recovery      string
	ReposInfo     []RepoPathInfo
	StatusSummary string
}

type RepoPathStatus struct {
	RepoName      string
	PrivateBranch string
	StatusSummary string
	Modified      bool
	CommitsAhead  int
}

type RepoPathInfo struct {
	RepoName      string
	PrivateBranch string
	SourceRef     string
	SourceCommit  string
	StatusSummary string
}

type RepoDiffSection struct {
	RepoName string
	Diff     string
}

type PathRunContext struct {
	Name      string
	Root      string
	RepoNames []string
}

func NewProjectConfig(name string) ProjectConfig {
	return ProjectConfig{
		Version:     1,
		Name:        name,
		CreatedAt:   time.Now().Format(time.RFC3339),
		ReposDir:    ReposDirName,
		PathsDir:    PathsDirName,
		MetadataDir: MetadataDirName,
		Repos:       map[string]RepoConfig{},
	}
}

func InitProject(projectRoot string) (string, error) {
	if existingRoot, ok := FindProjectRoot(projectRoot); ok {
		return "", errors.New("cannot initialize nested Timber project inside existing project at " + existingRoot)
	}

	configPath := filepath.Join(projectRoot, MetadataDirName, ProjectFileName)
	if _, err := os.Stat(configPath); err == nil {
		return "", errors.New("project already initialized at " + configPath)
	}

	dirs := []string{
		filepath.Join(projectRoot, ReposDirName),
		filepath.Join(projectRoot, PathsDirName),
		filepath.Join(projectRoot, MetadataDirName),
		filepath.Join(projectRoot, MetadataDirName, PathsMetadataDir),
		filepath.Join(projectRoot, MetadataDirName, OperationsDirName),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}

	eventLog := filepath.Join(projectRoot, MetadataDirName, EventLogFileName)
	if info, err := os.Stat(eventLog); os.IsNotExist(err) {
		file, createErr := os.Create(eventLog)
		if createErr != nil {
			return "", createErr
		}
		_ = file.Close()
	} else if err != nil {
		return "", err
	} else if !info.Mode().IsRegular() {
		return "", errors.New("expected event log path to be a file: " + eventLog)
	}

	config := NewProjectConfig(filepath.Base(projectRoot))
	if err := WriteProjectConfig(configPath, config); err != nil {
		return "", err
	}

	return configPath, nil
}

func WriteProjectConfig(configPath string, config ProjectConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0o644)
}

func LoadProjectConfig(projectRoot string) (ProjectConfig, error) {
	configPath := filepath.Join(projectRoot, MetadataDirName, ProjectFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ProjectConfig{}, err
	}

	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return ProjectConfig{}, err
	}
	if config.Repos == nil {
		config.Repos = map[string]RepoConfig{}
	}
	return config, nil
}

func AddRepo(projectRoot, repoName, repoURL string) error {
	repoName = strings.TrimSpace(repoName)
	repoURL = strings.TrimSpace(repoURL)
	if repoName == "" {
		return errors.New("repo name cannot be empty")
	}
	if repoURL == "" {
		return errors.New("repo URL cannot be empty")
	}
	if strings.Contains(repoName, "/") || strings.Contains(repoName, string(filepath.Separator)) {
		return fmt.Errorf("repo name %q cannot contain path separators", repoName)
	}

	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return err
	}
	if _, exists := config.Repos[repoName]; exists {
		return fmt.Errorf("repo %q is already registered", repoName)
	}

	config.Repos[repoName] = RepoConfig{URL: repoURL}
	switch len(config.Repos) {
	case 1:
		config.DefaultRepo = repoName
	default:
		config.DefaultRepo = ""
	}

	configPath := filepath.Join(projectRoot, MetadataDirName, ProjectFileName)
	return WriteProjectConfig(configPath, config)
}

func RemoveRepo(projectRoot, repoName string) error {
	repoName = strings.TrimSpace(repoName)
	if repoName == "" {
		return errors.New("repo name cannot be empty")
	}

	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return err
	}
	if _, exists := config.Repos[repoName]; !exists {
		return fmt.Errorf("repo %q is not registered", repoName)
	}

	pathNames, err := pathsUsingRepo(projectRoot, config, repoName)
	if err != nil {
		return err
	}
	if len(pathNames) > 0 {
		return fmt.Errorf("repo %q is still present in path%s %s", repoName, pluralSuffix(len(pathNames)), strings.Join(pathNames, ", "))
	}

	delete(config.Repos, repoName)
	switch len(config.Repos) {
	case 0:
		config.DefaultRepo = ""
	case 1:
		for name := range config.Repos {
			config.DefaultRepo = name
		}
	default:
		config.DefaultRepo = ""
	}

	configPath := filepath.Join(projectRoot, MetadataDirName, ProjectFileName)
	if err := WriteProjectConfig(configPath, config); err != nil {
		return err
	}

	repoMirrorPath := filepath.Join(projectRoot, config.ReposDir, repoName+".git")
	if err := os.RemoveAll(repoMirrorPath); err != nil {
		return err
	}

	return nil
}

func pathsUsingRepo(projectRoot string, config ProjectConfig, repoName string) ([]string, error) {
	metadataRoot := filepath.Join(projectRoot, config.MetadataDir, PathsMetadataDir)
	entries, err := os.ReadDir(metadataRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	pathNames := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		pathConfig, err := loadPathConfig(filepath.Join(metadataRoot, entry.Name()))
		if err != nil {
			return nil, err
		}
		if _, ok := pathConfig.Repos[repoName]; ok {
			pathNames = append(pathNames, pathConfig.Name)
		}
	}
	sort.Strings(pathNames)
	return pathNames, nil
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func SyncRepos(projectRoot string) ([]SyncResult, error) {
	return syncRepos(projectRoot, nil)
}

func SyncReposWithProgress(projectRoot string, progress func(name, phase string)) ([]SyncResult, error) {
	return syncRepos(projectRoot, progress)
}

func syncRepos(projectRoot string, progress func(name, phase string)) ([]SyncResult, error) {
	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(config.Repos))
	for name := range config.Repos {
		names = append(names, name)
	}
	sort.Strings(names)

	results := make([]SyncResult, 0, len(names))
	for _, name := range names {
		repo := config.Repos[name]
		repoPath := filepath.Join(projectRoot, config.ReposDir, name+".git")

		action := "fetched"
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			if progress != nil {
				progress(name, "cloning")
			}
			if err := initializeRepoCache(projectRoot, repoPath, repo.URL); err != nil {
				return nil, fmt.Errorf("sync repo %q: %w", name, err)
			}
			action = "cloned"
		} else if err != nil {
			return nil, fmt.Errorf("sync repo %q: %w", name, err)
		} else {
			if progress != nil {
				progress(name, "fetching")
			}
			if err := syncRepoCache(projectRoot, repoPath, repo.URL); err != nil {
				return nil, fmt.Errorf("sync repo %q: %w", name, err)
			}
		}

		results = append(results, SyncResult{Name: name, Action: action})
	}

	return results, nil
}

func initializeRepoCache(projectRoot, repoPath, repoURL string) error {
	if err := runGitCommand(projectRoot, "init", "--bare", repoPath); err != nil {
		return err
	}
	if err := runGitCommand(projectRoot, "-C", repoPath, "remote", "add", "origin", repoURL); err != nil {
		return err
	}
	return syncRepoCache(projectRoot, repoPath, repoURL)
}

func syncRepoCache(projectRoot, repoPath, repoURL string) error {
	if err := ensureRepoCacheRemoteConfig(projectRoot, repoPath, repoURL); err != nil {
		return err
	}
	return runGitCommand(projectRoot, "-C", repoPath, "fetch", "--prune", "origin",
		"+refs/heads/*:refs/remotes/origin/*",
		"+refs/tags/*:refs/tags/*",
	)
}

func ensureRepoCacheRemoteConfig(projectRoot, repoPath, repoURL string) error {
	if err := runGitCommand(projectRoot, "-C", repoPath, "remote", "set-url", "origin", repoURL); err != nil {
		return err
	}
	if _, err := gitOutputAllowEmpty(projectRoot, "-C", repoPath, "config", "--get-all", "remote.origin.mirror"); err == nil {
		if err := runGitCommand(projectRoot, "-C", repoPath, "config", "--unset-all", "remote.origin.mirror"); err != nil {
			return err
		}
	}
	if _, err := gitOutputAllowEmpty(projectRoot, "-C", repoPath, "config", "--get-all", "remote.origin.fetch"); err == nil {
		if err := runGitCommand(projectRoot, "-C", repoPath, "config", "--unset-all", "remote.origin.fetch"); err != nil {
			return err
		}
	}
	if err := runGitCommand(projectRoot, "-C", repoPath, "config", "--add", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*"); err != nil {
		return err
	}
	if err := runGitCommand(projectRoot, "-C", repoPath, "config", "--add", "remote.origin.fetch", "+refs/tags/*:refs/tags/*"); err != nil {
		return err
	}
	return nil
}

func ListRepos(projectRoot string) ([]RepoListEntry, error) {
	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(config.Repos))
	for name := range config.Repos {
		names = append(names, name)
	}
	sort.Strings(names)

	entries := make([]RepoListEntry, 0, len(names))
	for _, name := range names {
		status := "registered"
		repoPath := filepath.Join(projectRoot, config.ReposDir, name+".git")
		if _, err := os.Stat(repoPath); err == nil {
			status = "synced"
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		entries = append(entries, RepoListEntry{
			Name:   name,
			Status: status,
			URL:    config.Repos[name].URL,
		})
	}

	return entries, nil
}

type repoPathPlan struct {
	RepoName      string
	MirrorPath    string
	PathPath      string
	ResolvedRef   string
	SourceDisplay string
	SourceCommit  string
	PrivateBranch string
}

func CreatePath(projectRoot string, options CreatePathOptions) (*CreatePathResult, error) {
	pathName := strings.TrimSpace(options.Name)
	defaultRef := strings.TrimSpace(options.DefaultRef)
	if pathName == "" {
		return nil, errors.New("path name cannot be empty")
	}
	if defaultRef == "" && len(options.RepoRefs) == 0 {
		return nil, errors.New("path creation requires a default ref or at least one repo-specific ref")
	}

	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	pathRoot := filepath.Join(projectRoot, config.PathsDir, pathName)
	if _, err := os.Stat(pathRoot); err == nil {
		return nil, fmt.Errorf("path %q already exists", pathName)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	pathMetadataPath := filepath.Join(projectRoot, config.MetadataDir, PathsMetadataDir, pathName+".yaml")
	if _, err := os.Stat(pathMetadataPath); err == nil {
		return nil, fmt.Errorf("path metadata for %q already exists", pathName)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	repoNames, err := resolvePathRepoSelection(config, options.SelectedRepos, options.RepoRefs, options.IncludeAll)
	if err != nil {
		return nil, err
	}

	plans := make([]repoPathPlan, 0, len(repoNames))
	for _, repoName := range repoNames {
		repoMirrorPath := filepath.Join(projectRoot, config.ReposDir, repoName+".git")
		if _, err := os.Stat(repoMirrorPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("repo %q has not been synced yet; run `timber repo sync` first", repoName)
		} else if err != nil {
			return nil, err
		}

		requestedRef := defaultRef
		if overrideRef, ok := options.RepoRefs[repoName]; ok {
			requestedRef = strings.TrimSpace(overrideRef)
		}
		if requestedRef == "" {
			return nil, fmt.Errorf("repo %q requires an explicit ref because no default --from ref was provided", repoName)
		}

		resolvedRef, sourceDisplay, err := resolveSourceRef(repoMirrorPath, requestedRef)
		if err != nil {
			return nil, fmt.Errorf("resolve ref for repo %q: %w", repoName, err)
		}

		sourceCommit, err := gitOutput(projectRoot, "-C", repoMirrorPath, "rev-parse", resolvedRef)
		if err != nil {
			return nil, fmt.Errorf("resolve commit for repo %q: %w", repoName, err)
		}

		privateBranch, err := makePrivateBranchName(pathName, repoName)
		if err != nil {
			return nil, err
		}

		plans = append(plans, repoPathPlan{
			RepoName:      repoName,
			MirrorPath:    repoMirrorPath,
			PathPath:      filepath.Join(pathRoot, repoName),
			ResolvedRef:   resolvedRef,
			SourceDisplay: sourceDisplay,
			SourceCommit:  sourceCommit,
			PrivateBranch: privateBranch,
		})
	}

	if err := os.MkdirAll(pathRoot, 0o755); err != nil {
		return nil, err
	}

	createdPlans := make([]repoPathPlan, 0, len(plans))
	for _, plan := range plans {
		if err := runGitCommand(projectRoot, "-C", plan.MirrorPath, "worktree", "add", "-b", plan.PrivateBranch, plan.PathPath, plan.ResolvedRef); err != nil {
			rollbackPathCreation(projectRoot, pathRoot, createdPlans, pathMetadataPath)
			return nil, fmt.Errorf("create worktree for repo %q: %w", plan.RepoName, err)
		}
		createdPlans = append(createdPlans, plan)
	}

	createdAt := time.Now().Format(time.RFC3339)
	parentDescription, parentRef, parentDisplay := pathParentSummary(defaultRef, plans, repoNames)
	repoStates := make(map[string]PathRepoState, len(plans))
	for _, plan := range plans {
		repoStates[plan.RepoName] = PathRepoState{
			Path:          filepath.ToSlash(filepath.Join(config.PathsDir, pathName, plan.RepoName)),
			SourceRef:     plan.ResolvedRef,
			SourceDisplay: plan.SourceDisplay,
			SourceCommit:  plan.SourceCommit,
			PrivateBranch: plan.PrivateBranch,
		}
	}
	pathConfig := PathConfig{
		Version:   1,
		Name:      pathName,
		Path:      filepath.ToSlash(filepath.Join(config.PathsDir, pathName)),
		CreatedAt: createdAt,
		Parent: PathParent{
			Type:                 "remote",
			Description:          parentDescription,
			DefaultSourceRef:     parentRef,
			DefaultSourceDisplay: parentDisplay,
		},
		Repos: repoStates,
	}

	if err := writePathConfig(pathMetadataPath, pathConfig); err != nil {
		rollbackPathCreation(projectRoot, pathRoot, createdPlans, pathMetadataPath)
		return nil, err
	}

	if err := writePathFiles(pathRoot, pathConfig); err != nil {
		rollbackPathCreation(projectRoot, pathRoot, createdPlans, pathMetadataPath)
		return nil, err
	}

	if err := appendEventLog(projectRoot, map[string]string{
		"time":  createdAt,
		"event": "path_created",
		"path":  pathName,
	}); err != nil {
		rollbackPathCreation(projectRoot, pathRoot, createdPlans, pathMetadataPath)
		return nil, err
	}

	repoName := ""
	privateBranch := ""
	if len(plans) == 1 {
		repoName = plans[0].RepoName
		privateBranch = plans[0].PrivateBranch
	}
	return &CreatePathResult{
		Name:          pathName,
		Path:          pathRoot,
		RepoName:      repoName,
		PrivateBranch: privateBranch,
		RepoNames:     repoNames,
	}, nil
}

func AddReposToPath(projectRoot string, options AddReposOptions) (*AddReposResult, error) {
	pathName := strings.TrimSpace(options.PathName)
	if pathName == "" {
		return nil, errors.New("path name cannot be empty")
	}

	projectConfig, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	pathMetadataPath := filepath.Join(projectRoot, projectConfig.MetadataDir, PathsMetadataDir, pathName+".yaml")
	pathConfig, err := loadPathConfig(pathMetadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path %q not found", pathName)
		}
		return nil, err
	}

	repoNames, err := resolveReposForPathAdd(projectConfig, pathConfig, options.RepoNames, options.RepoRefs)
	if err != nil {
		return nil, err
	}

	defaultRef := strings.TrimSpace(options.DefaultRef)
	if defaultRef == "" {
		defaultRef = strings.TrimSpace(pathConfig.Parent.DefaultSourceRef)
	}

	pathRoot := filepath.Join(projectRoot, filepath.FromSlash(pathConfig.Path))
	plans := make([]repoPathPlan, 0, len(repoNames))
	for _, repoName := range repoNames {
		requestedRef := defaultRef
		if overrideRef, ok := options.RepoRefs[repoName]; ok {
			requestedRef = strings.TrimSpace(overrideRef)
		}
		if requestedRef == "" {
			return nil, fmt.Errorf("repo %q requires --from <ref> or an explicit repo=ref mapping", repoName)
		}

		repoMirrorPath := filepath.Join(projectRoot, projectConfig.ReposDir, repoName+".git")
		if _, err := os.Stat(repoMirrorPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("repo %q has not been synced yet; run `timber repo sync` first", repoName)
		} else if err != nil {
			return nil, err
		}

		resolvedRef, sourceDisplay, err := resolveSourceRef(repoMirrorPath, requestedRef)
		if err != nil {
			return nil, fmt.Errorf("resolve ref for repo %q: %w", repoName, err)
		}
		sourceCommit, err := gitOutput(projectRoot, "-C", repoMirrorPath, "rev-parse", resolvedRef)
		if err != nil {
			return nil, fmt.Errorf("resolve commit for repo %q: %w", repoName, err)
		}
		privateBranch, err := makePrivateBranchName(pathName, repoName)
		if err != nil {
			return nil, err
		}

		plans = append(plans, repoPathPlan{
			RepoName:      repoName,
			MirrorPath:    repoMirrorPath,
			PathPath:      filepath.Join(pathRoot, repoName),
			ResolvedRef:   resolvedRef,
			SourceDisplay: sourceDisplay,
			SourceCommit:  sourceCommit,
			PrivateBranch: privateBranch,
		})
	}

	createdPlans := make([]repoPathPlan, 0, len(plans))
	for _, plan := range plans {
		if err := runGitCommand(projectRoot, "-C", plan.MirrorPath, "worktree", "add", "-b", plan.PrivateBranch, plan.PathPath, plan.ResolvedRef); err != nil {
			rollbackAddedRepos(projectRoot, createdPlans)
			return nil, fmt.Errorf("create worktree for repo %q: %w", plan.RepoName, err)
		}
		createdPlans = append(createdPlans, plan)
	}

	for _, plan := range plans {
		pathConfig.Repos[plan.RepoName] = PathRepoState{
			Path:          filepath.ToSlash(filepath.Join(projectConfig.PathsDir, pathName, plan.RepoName)),
			SourceRef:     plan.ResolvedRef,
			SourceDisplay: plan.SourceDisplay,
			SourceCommit:  plan.SourceCommit,
			PrivateBranch: plan.PrivateBranch,
		}
	}
	refreshRemotePathParent(&pathConfig)

	if err := writePathConfig(pathMetadataPath, pathConfig); err != nil {
		rollbackAddedRepos(projectRoot, createdPlans)
		return nil, err
	}
	if err := writePathFiles(pathRoot, pathConfig); err != nil {
		for _, plan := range plans {
			delete(pathConfig.Repos, plan.RepoName)
		}
		_ = writePathConfig(pathMetadataPath, pathConfig)
		rollbackAddedRepos(projectRoot, createdPlans)
		return nil, err
	}

	if err := appendEventLog(projectRoot, map[string]string{
		"time":  time.Now().Format(time.RFC3339),
		"event": "path_repos_added",
		"path":  pathName,
		"repos": strings.Join(repoNames, ","),
	}); err != nil {
		return nil, err
	}

	return &AddReposResult{
		PathName:   pathName,
		AddedRepos: repoNames,
	}, nil
}

func ForkPath(projectRoot, sourcePath string, childNames []string) (*ForkPathResult, error) {
	sourcePath = strings.TrimSpace(sourcePath)
	if sourcePath == "" {
		return nil, errors.New("source path name cannot be empty")
	}
	if len(childNames) == 0 {
		return nil, errors.New("fork requires at least one child path name")
	}

	projectConfig, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	sourceMetadataPath := filepath.Join(projectRoot, projectConfig.MetadataDir, PathsMetadataDir, sourcePath+".yaml")
	sourceConfig, err := loadPathConfig(sourceMetadataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path %q not found", sourcePath)
		}
		return nil, err
	}

	sourceStatus, err := GetPathStatus(projectRoot, sourcePath)
	if err != nil {
		return nil, err
	}
	for _, repoStatus := range sourceStatus.ReposStatus {
		if repoStatus.Modified {
			return nil, fmt.Errorf("path %q must be clean before forking; %s is modified", sourcePath, repoStatus.RepoName)
		}
	}

	trimmedChildren := make([]string, 0, len(childNames))
	seenChildren := map[string]bool{}
	for _, childName := range childNames {
		childName = strings.TrimSpace(childName)
		if childName == "" {
			return nil, errors.New("child path name cannot be empty")
		}
		if childName == sourcePath {
			return nil, fmt.Errorf("child path %q cannot reuse the source path name", childName)
		}
		if seenChildren[childName] {
			return nil, fmt.Errorf("child path %q was requested more than once", childName)
		}
		seenChildren[childName] = true

		childRoot := filepath.Join(projectRoot, projectConfig.PathsDir, childName)
		if _, err := os.Stat(childRoot); err == nil {
			return nil, fmt.Errorf("path %q already exists", childName)
		} else if !os.IsNotExist(err) {
			return nil, err
		}

		childMetadataPath := filepath.Join(projectRoot, projectConfig.MetadataDir, PathsMetadataDir, childName+".yaml")
		if _, err := os.Stat(childMetadataPath); err == nil {
			return nil, fmt.Errorf("path metadata for %q already exists", childName)
		} else if !os.IsNotExist(err) {
			return nil, err
		}

		trimmedChildren = append(trimmedChildren, childName)
	}

	sourceRepoNames := sortedPathRepoNames(sourceConfig)
	sourcePlans := make([]repoPathPlan, 0, len(sourceRepoNames))
	for _, repoName := range sourceRepoNames {
		repoState := sourceConfig.Repos[repoName]
		repoPath := filepath.Join(projectRoot, filepath.FromSlash(repoState.Path))
		headCommit, err := gitOutput(projectRoot, "-C", repoPath, "rev-parse", "HEAD")
		if err != nil {
			return nil, fmt.Errorf("resolve HEAD for repo %q: %w", repoName, err)
		}
		sourcePlans = append(sourcePlans, repoPathPlan{
			RepoName:      repoName,
			MirrorPath:    filepath.Join(projectRoot, projectConfig.ReposDir, repoName+".git"),
			ResolvedRef:   headCommit,
			SourceDisplay: repoState.SourceDisplay,
			SourceCommit:  headCommit,
		})
	}

	createdChildren := make([]CreatePathResult, 0, len(trimmedChildren))
	createdAt := time.Now().Format(time.RFC3339)
	for _, childName := range trimmedChildren {
		childRoot := filepath.Join(projectRoot, projectConfig.PathsDir, childName)
		if err := os.MkdirAll(childRoot, 0o755); err != nil {
			return nil, err
		}

		childMetadataPath := filepath.Join(projectRoot, projectConfig.MetadataDir, PathsMetadataDir, childName+".yaml")
		childPlans := make([]repoPathPlan, 0, len(sourcePlans))
		createdPlans := make([]repoPathPlan, 0, len(sourcePlans))
		for _, sourcePlan := range sourcePlans {
			privateBranch, err := makePrivateBranchName(childName, sourcePlan.RepoName)
			if err != nil {
				rollbackPathCreation(projectRoot, childRoot, createdPlans, childMetadataPath)
				return nil, err
			}
			childPlan := sourcePlan
			childPlan.PathPath = filepath.Join(childRoot, sourcePlan.RepoName)
			childPlan.PrivateBranch = privateBranch
			childPlans = append(childPlans, childPlan)

			if err := runGitCommand(projectRoot, "-C", childPlan.MirrorPath, "worktree", "add", "-b", childPlan.PrivateBranch, childPlan.PathPath, childPlan.ResolvedRef); err != nil {
				rollbackPathCreation(projectRoot, childRoot, createdPlans, childMetadataPath)
				return nil, fmt.Errorf("create worktree for repo %q in child %q: %w", childPlan.RepoName, childName, err)
			}
			createdPlans = append(createdPlans, childPlan)
		}

		repoStates := make(map[string]PathRepoState, len(childPlans))
		for _, plan := range childPlans {
			sourceState := sourceConfig.Repos[plan.RepoName]
			repoStates[plan.RepoName] = PathRepoState{
				Path:          filepath.ToSlash(filepath.Join(projectConfig.PathsDir, childName, plan.RepoName)),
				SourceRef:     sourceState.SourceRef,
				SourceDisplay: sourceState.SourceDisplay,
				SourceCommit:  plan.SourceCommit,
				PrivateBranch: plan.PrivateBranch,
			}
		}

		childConfig := PathConfig{
			Version:   1,
			Name:      childName,
			Path:      filepath.ToSlash(filepath.Join(projectConfig.PathsDir, childName)),
			CreatedAt: createdAt,
			Parent: PathParent{
				Type:                 "path",
				PathName:             sourcePath,
				Description:          sourcePath,
				DefaultSourceRef:     sourceConfig.Parent.DefaultSourceRef,
				DefaultSourceDisplay: sourceConfig.Parent.DefaultSourceDisplay,
			},
			Repos: repoStates,
		}

		if err := writePathConfig(childMetadataPath, childConfig); err != nil {
			rollbackPathCreation(projectRoot, childRoot, createdPlans, childMetadataPath)
			return nil, err
		}
		if err := writePathFiles(childRoot, childConfig); err != nil {
			rollbackPathCreation(projectRoot, childRoot, createdPlans, childMetadataPath)
			return nil, err
		}

		createdChildren = append(createdChildren, CreatePathResult{
			Name:      childName,
			Path:      childRoot,
			RepoNames: sourceRepoNames,
		})
	}

	if err := appendEventLog(projectRoot, map[string]string{
		"time":     createdAt,
		"event":    "path_forked",
		"source":   sourcePath,
		"children": strings.Join(trimmedChildren, ","),
	}); err != nil {
		for _, child := range createdChildren {
			childMetadataPath := filepath.Join(projectRoot, projectConfig.MetadataDir, PathsMetadataDir, child.Name+".yaml")
			childConfig, loadErr := loadPathConfig(childMetadataPath)
			if loadErr != nil {
				continue
			}
			childPlans := make([]repoPathPlan, 0, len(childConfig.Repos))
			for _, repoName := range sortedPathRepoNames(childConfig) {
				repoState := childConfig.Repos[repoName]
				childPlans = append(childPlans, repoPathPlan{
					RepoName:      repoName,
					MirrorPath:    filepath.Join(projectRoot, projectConfig.ReposDir, repoName+".git"),
					PathPath:      filepath.Join(projectRoot, filepath.FromSlash(repoState.Path)),
					PrivateBranch: repoState.PrivateBranch,
				})
			}
			rollbackPathCreation(projectRoot, child.Path, childPlans, childMetadataPath)
		}
		return nil, err
	}

	return &ForkPathResult{
		SourcePath: sourcePath,
		Created:    createdChildren,
	}, nil
}

func KeepPath(projectRoot, sourcePath, targetPath string) (*KeepPathResult, error) {
	sourcePath = strings.TrimSpace(sourcePath)
	targetPath = strings.TrimSpace(targetPath)
	if sourcePath == "" {
		return nil, errors.New("source path name cannot be empty")
	}
	if targetPath == "" {
		return nil, errors.New("target path name cannot be empty")
	}

	statePath := keepOperationStatePath(projectRoot)
	if _, err := os.Stat(statePath); err == nil {
		return nil, errors.New("a keep operation is already in progress; run `timber keep --continue` or `timber keep --abort`")
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	projectConfig, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	sourceConfig, err := loadPathByName(projectRoot, projectConfig, sourcePath)
	if err != nil {
		return nil, err
	}
	targetConfig, err := loadPathByName(projectRoot, projectConfig, targetPath)
	if err != nil {
		return nil, err
	}

	if sourceConfig.Parent.Type != "path" || sourceConfig.Parent.PathName != targetPath {
		return nil, fmt.Errorf("path %q is not forked from %q", sourcePath, targetPath)
	}

	if err := ensurePathClean(projectRoot, sourcePath); err != nil {
		return nil, err
	}
	if err := ensurePathClean(projectRoot, targetPath); err != nil {
		return nil, err
	}

	state, err := newKeepOperationState(sourcePath, targetPath, sourceConfig, targetConfig)
	if err != nil {
		return nil, err
	}
	if err := writeKeepOperationState(statePath, state); err != nil {
		return nil, err
	}
	if err := appendEventLog(projectRoot, map[string]string{
		"time":   state.CreatedAt,
		"event":  "keep_started",
		"source": sourcePath,
		"target": targetPath,
	}); err != nil {
		_ = os.Remove(statePath)
		return nil, err
	}

	result, err := advanceKeepOperation(projectRoot, projectConfig, sourceConfig, targetConfig, statePath, state)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func ContinueKeepPath(projectRoot string) (*KeepPathResult, error) {
	projectConfig, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	statePath := keepOperationStatePath(projectRoot)
	state, err := loadKeepOperationState(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("no keep operation is in progress")
		}
		return nil, err
	}

	sourceConfig, err := loadPathByName(projectRoot, projectConfig, state.SourcePath)
	if err != nil {
		return nil, err
	}
	targetConfig, err := loadPathByName(projectRoot, projectConfig, state.TargetPath)
	if err != nil {
		return nil, err
	}

	return advanceKeepOperation(projectRoot, projectConfig, sourceConfig, targetConfig, statePath, state)
}

func AbortKeepPath(projectRoot string) error {
	statePath := keepOperationStatePath(projectRoot)
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return errors.New("no keep operation is in progress")
	} else if err != nil {
		return err
	}
	return os.Remove(statePath)
}

func DropPaths(projectRoot string, options DropPathOptions) (*DropPathResult, error) {
	if len(options.PathNames) == 0 {
		return nil, errors.New("drop requires at least one path name")
	}
	if options.KeepBranches && options.DeleteBranches {
		return nil, errors.New("cannot combine --keep-branches and --delete-branches")
	}

	projectConfig, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	allConfigs, err := loadAllPathConfigs(projectRoot, projectConfig)
	if err != nil {
		return nil, err
	}

	requested := map[string]bool{}
	queue := make([]string, 0, len(options.PathNames))
	for _, name := range options.PathNames {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, errors.New("path name cannot be empty")
		}
		if _, ok := allConfigs[name]; !ok {
			return nil, fmt.Errorf("path %q not found", name)
		}
		if !requested[name] {
			requested[name] = true
			queue = append(queue, name)
		}
	}

	if options.Recursive {
		for i := 0; i < len(queue); i++ {
			name := queue[i]
			for _, child := range childPathNames(allConfigs, name) {
				if !requested[child] {
					requested[child] = true
					queue = append(queue, child)
				}
			}
		}
	} else {
		for _, name := range queue {
			children := childPathNames(allConfigs, name)
			missingChildren := make([]string, 0, len(children))
			for _, child := range children {
				if !requested[child] {
					missingChildren = append(missingChildren, child)
				}
			}
			if len(missingChildren) > 0 {
				return nil, fmt.Errorf("path %q has child paths: %s (use --recursive)", name, strings.Join(missingChildren, ","))
			}
		}
	}

	dropOrder := orderDropPaths(queue, allConfigs)
	result := &DropPathResult{}
	for _, name := range dropOrder {
		config := allConfigs[name]
		if err := validateDropPath(projectRoot, config, options.Force); err != nil {
			return nil, err
		}

		keepBranches := options.KeepBranches
		if !options.DeleteBranches && !options.KeepBranches {
			keepBranches = options.Force
		}

		for _, repoName := range sortedPathRepoNames(config) {
			repoState := config.Repos[repoName]
			pathPath := filepath.Join(projectRoot, filepath.FromSlash(repoState.Path))
			mirrorPath := filepath.Join(projectRoot, projectConfig.ReposDir, repoName+".git")
			removeArgs := []string{"-C", mirrorPath, "worktree", "remove"}
			if options.Force {
				removeArgs = append(removeArgs, "--force")
			}
			removeArgs = append(removeArgs, pathPath)
			if err := runGitCommand(projectRoot, removeArgs...); err != nil {
				return nil, fmt.Errorf("remove worktree for %q repo %q: %w", name, repoName, err)
			}
			if keepBranches {
				result.KeptBranches = append(result.KeptBranches, repoState.PrivateBranch)
				continue
			}
			if err := runGitCommand(projectRoot, "-C", mirrorPath, "branch", "-D", repoState.PrivateBranch); err != nil {
				return nil, fmt.Errorf("delete private branch for %q repo %q: %w", name, repoName, err)
			}
			result.DeletedBranches = append(result.DeletedBranches, repoState.PrivateBranch)
		}

		if err := os.Remove(filepath.Join(projectRoot, projectConfig.MetadataDir, PathsMetadataDir, name+".yaml")); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if err := os.RemoveAll(filepath.Join(projectRoot, filepath.FromSlash(config.Path))); err != nil {
			return nil, err
		}
		result.Dropped = append(result.Dropped, name)
	}

	if len(result.Dropped) > 0 {
		if err := appendEventLog(projectRoot, map[string]string{
			"time":  time.Now().Format(time.RFC3339),
			"event": "path_dropped",
			"paths": strings.Join(result.Dropped, ","),
		}); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func advanceKeepOperation(projectRoot string, projectConfig ProjectConfig, sourceConfig, targetConfig PathConfig, statePath string, state KeepOperationState) (*KeepPathResult, error) {
	result := &KeepPathResult{
		SourcePath: state.SourcePath,
		TargetPath: state.TargetPath,
	}

	for _, repoName := range state.RepoOrder {
		repoState := state.Repos[repoName]
		switch repoState.Status {
		case "merged":
			result.MergedRepos = append(result.MergedRepos, repoName)
			continue
		case "skipped":
			result.SkippedRepos = append(result.SkippedRepos, repoName)
			continue
		case "conflicted":
			targetRepoPath := filepath.Join(projectRoot, filepath.FromSlash(targetConfig.Repos[repoName].Path))
			if err := finalizeConflictedMerge(targetRepoPath); err != nil {
				return nil, fmt.Errorf("%s:\n %w", repoName, err)
			}
			repoState.Status = "merged"
			state.Repos[repoName] = repoState
			state.LastUpdated = time.Now().Format(time.RFC3339)
			if err := writeKeepOperationState(statePath, state); err != nil {
				return nil, err
			}
			result.MergedRepos = append(result.MergedRepos, repoName)
			continue
		case "pending":
			sourceRepoState := sourceConfig.Repos[repoName]
			targetRepoState := targetConfig.Repos[repoName]
			targetRepoPath := filepath.Join(projectRoot, filepath.FromSlash(targetRepoState.Path))

			revisionsAhead, err := gitOutput(".", "-C", targetRepoPath, "rev-list", "--count", repoState.ForkPointCommit+".."+sourceRepoState.PrivateBranch)
			if err != nil {
				return nil, fmt.Errorf("read child progress for repo %q: %w", repoName, err)
			}
			if strings.TrimSpace(revisionsAhead) == "0" {
				repoState.Status = "skipped"
				state.Repos[repoName] = repoState
				state.LastUpdated = time.Now().Format(time.RFC3339)
				if err := writeKeepOperationState(statePath, state); err != nil {
					return nil, err
				}
				result.SkippedRepos = append(result.SkippedRepos, repoName)
				continue
			}

			conflicted, err := mergePathBranch(targetRepoPath, sourceRepoState.PrivateBranch)
			if conflicted {
				repoState.Status = "conflicted"
				state.Repos[repoName] = repoState
				state.LastUpdated = time.Now().Format(time.RFC3339)
				if writeErr := writeKeepOperationState(statePath, state); writeErr != nil {
					return nil, writeErr
				}
				_ = appendEventLog(projectRoot, map[string]string{
					"time":   state.LastUpdated,
					"event":  "keep_conflict",
					"source": state.SourcePath,
					"target": state.TargetPath,
					"repo":   repoName,
				})
				return nil, fmt.Errorf("%s:\n conflict while merging %s into %s\n\nResolve conflicts in:\n %s\n\nThen run:\n timber keep --continue\n\nOr abort:\n timber keep --abort", repoName, state.SourcePath, state.TargetPath, targetRepoPath)
			}
			if err != nil {
				return nil, fmt.Errorf("merge repo %q: %w", repoName, err)
			}

			repoState.Status = "merged"
			state.Repos[repoName] = repoState
			state.LastUpdated = time.Now().Format(time.RFC3339)
			if err := writeKeepOperationState(statePath, state); err != nil {
				return nil, err
			}
			result.MergedRepos = append(result.MergedRepos, repoName)
		default:
			return nil, fmt.Errorf("unsupported keep repo status %q for repo %q", repoState.Status, repoName)
		}
	}

	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	completedAt := time.Now().Format(time.RFC3339)
	if err := appendEventLog(projectRoot, map[string]string{
		"time":   completedAt,
		"event":  "keep",
		"source": state.SourcePath,
		"target": state.TargetPath,
	}); err != nil {
		return nil, err
	}
	if err := writePathFiles(filepath.Join(projectRoot, filepath.FromSlash(targetConfig.Path)), targetConfig); err != nil {
		return nil, err
	}
	return result, nil
}

func newKeepOperationState(sourcePath, targetPath string, sourceConfig, targetConfig PathConfig) (KeepOperationState, error) {
	targetRepos := map[string]bool{}
	for repoName := range targetConfig.Repos {
		targetRepos[repoName] = true
	}

	repoOrder := make([]string, 0, len(sourceConfig.Repos))
	repos := map[string]KeepOperationRepoState{}
	for _, repoName := range sortedPathRepoNames(sourceConfig) {
		if !targetRepos[repoName] {
			continue
		}
		sourceRepoState := sourceConfig.Repos[repoName]
		targetRepoState := targetConfig.Repos[repoName]
		repoOrder = append(repoOrder, repoName)
		repos[repoName] = KeepOperationRepoState{
			Status:          "pending",
			ForkPointCommit: sourceRepoState.SourceCommit,
			SourceBranch:    sourceRepoState.PrivateBranch,
			TargetBranch:    targetRepoState.PrivateBranch,
		}
	}
	if len(repoOrder) == 0 {
		return KeepOperationState{}, fmt.Errorf("paths %q and %q do not share any repos", sourcePath, targetPath)
	}

	now := time.Now().Format(time.RFC3339)
	return KeepOperationState{
		Type:        "keep",
		SourcePath:  sourcePath,
		TargetPath:  targetPath,
		RepoOrder:   repoOrder,
		Repos:       repos,
		CreatedAt:   now,
		LastUpdated: now,
	}, nil
}

func resolvePathRepoSelection(config ProjectConfig, selectedRepos []string, repoRefs map[string]string, includeAll bool) ([]string, error) {
	if includeAll {
		if len(selectedRepos) > 0 {
			return nil, errors.New("cannot use --all together with --repos")
		}
		repoNames := make([]string, 0, len(config.Repos))
		for repoName := range config.Repos {
			repoNames = append(repoNames, repoName)
		}
		sort.Strings(repoNames)
		if len(repoNames) == 0 {
			return nil, errors.New("no repos are registered")
		}
		return repoNames, nil
	}
	if len(selectedRepos) == 0 {
		if len(repoRefs) > 0 {
			repoNames := make([]string, 0, len(repoRefs))
			for repoName := range repoRefs {
				if _, ok := config.Repos[repoName]; !ok {
					return nil, fmt.Errorf("repo %q is not registered", repoName)
				}
				repoNames = append(repoNames, repoName)
			}
			sort.Strings(repoNames)
			return repoNames, nil
		}
		if config.DefaultRepo == "" {
			return nil, errors.New("multi-repo projects must specify repos with --repos or repo=ref mappings")
		}
		if _, ok := config.Repos[config.DefaultRepo]; !ok {
			return nil, fmt.Errorf("default repo %q is not registered", config.DefaultRepo)
		}
		return []string{config.DefaultRepo}, nil
	}

	seen := map[string]bool{}
	repoNames := make([]string, 0, len(selectedRepos))
	for _, repoName := range selectedRepos {
		repoName = strings.TrimSpace(repoName)
		if repoName == "" {
			continue
		}
		if seen[repoName] {
			return nil, fmt.Errorf("repo %q was listed more than once", repoName)
		}
		if _, ok := config.Repos[repoName]; !ok {
			return nil, fmt.Errorf("repo %q is not registered", repoName)
		}
		seen[repoName] = true
		repoNames = append(repoNames, repoName)
	}
	if len(repoNames) == 0 {
		return nil, errors.New("--repos requires at least one registered repo")
	}
	for repoName := range repoRefs {
		if !seen[repoName] {
			if _, ok := config.Repos[repoName]; !ok {
				return nil, fmt.Errorf("repo %q is not registered", repoName)
			}
			repoNames = append(repoNames, repoName)
			seen[repoName] = true
		}
	}
	sort.Strings(repoNames)
	return repoNames, nil
}

func pathParentSummary(defaultRef string, plans []repoPathPlan, repoNames []string) (string, string, string) {
	if len(plans) == 0 {
		return defaultRef, defaultRef, defaultRef
	}
	if defaultRef != "" {
		overrideParts := make([]string, 0)
		for _, repoName := range repoNames {
			for _, plan := range plans {
				if plan.RepoName == repoName && plan.SourceDisplay != defaultRef {
					overrideParts = append(overrideParts, fmt.Sprintf("%s=%s", repoName, plan.SourceDisplay))
					break
				}
			}
		}
		if len(overrideParts) == 0 {
			return plans[0].SourceDisplay, plans[0].ResolvedRef, plans[0].SourceDisplay
		}
		return fmt.Sprintf("%s with %s", defaultRef, strings.Join(overrideParts, " ")), plans[0].ResolvedRef, defaultRef
	}

	parts := make([]string, 0, len(plans))
	for _, repoName := range repoNames {
		for _, plan := range plans {
			if plan.RepoName == repoName {
				parts = append(parts, fmt.Sprintf("%s=%s", repoName, plan.SourceDisplay))
				break
			}
		}
	}
	description := strings.Join(parts, " ")
	if description == "" {
		description = plans[0].SourceDisplay
	}
	return description, "", ""
}

func refreshRemotePathParent(config *PathConfig) {
	if config.Parent.Type != "remote" {
		return
	}

	repoNames := sortedPathRepoNames(*config)
	plans := make([]repoPathPlan, 0, len(repoNames))
	for _, repoName := range repoNames {
		repoState := config.Repos[repoName]
		plans = append(plans, repoPathPlan{
			RepoName:      repoName,
			ResolvedRef:   repoState.SourceRef,
			SourceDisplay: repoState.SourceDisplay,
		})
	}

	defaultDisplay := strings.TrimSpace(config.Parent.DefaultSourceDisplay)
	if defaultDisplay == "" {
		description, parentRef, parentDisplay := pathParentSummary("", plans, repoNames)
		config.Parent.Description = description
		config.Parent.DefaultSourceRef = parentRef
		config.Parent.DefaultSourceDisplay = parentDisplay
		return
	}

	defaultResolvedRef := strings.TrimSpace(config.Parent.DefaultSourceRef)
	for _, repoName := range repoNames {
		repoState := config.Repos[repoName]
		if repoState.SourceDisplay == defaultDisplay {
			defaultResolvedRef = repoState.SourceRef
			break
		}
	}

	overrideParts := make([]string, 0)
	for _, repoName := range repoNames {
		repoState := config.Repos[repoName]
		if repoState.SourceDisplay != defaultDisplay {
			overrideParts = append(overrideParts, fmt.Sprintf("%s=%s", repoName, repoState.SourceDisplay))
		}
	}

	if len(overrideParts) == 0 {
		config.Parent.Description = defaultDisplay
	} else {
		config.Parent.Description = fmt.Sprintf("%s with %s", defaultDisplay, strings.Join(overrideParts, " "))
	}
	config.Parent.DefaultSourceRef = defaultResolvedRef
	config.Parent.DefaultSourceDisplay = defaultDisplay
}

func rollbackPathCreation(projectRoot, pathRoot string, plans []repoPathPlan, metadataPath string) {
	if metadataPath != "" {
		_ = os.Remove(metadataPath)
	}
	for i := len(plans) - 1; i >= 0; i-- {
		plan := plans[i]
		_ = runGitCommand(projectRoot, "-C", plan.MirrorPath, "worktree", "remove", plan.PathPath)
		_ = runGitCommand(projectRoot, "-C", plan.MirrorPath, "branch", "-D", plan.PrivateBranch)
	}
	_ = os.RemoveAll(pathRoot)
}

func rollbackAddedRepos(projectRoot string, plans []repoPathPlan) {
	for i := len(plans) - 1; i >= 0; i-- {
		plan := plans[i]
		_ = runGitCommand(projectRoot, "-C", plan.MirrorPath, "worktree", "remove", plan.PathPath)
		_ = runGitCommand(projectRoot, "-C", plan.MirrorPath, "branch", "-D", plan.PrivateBranch)
	}
}

func resolveReposForPathAdd(projectConfig ProjectConfig, pathConfig PathConfig, repoNames []string, repoRefs map[string]string) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(repoNames)+len(repoRefs))

	for _, repoName := range repoNames {
		repoName = strings.TrimSpace(repoName)
		if repoName == "" {
			continue
		}
		if _, ok := projectConfig.Repos[repoName]; !ok {
			return nil, fmt.Errorf("repo %q is not registered", repoName)
		}
		if _, exists := pathConfig.Repos[repoName]; exists {
			return nil, fmt.Errorf("repo %q is already present in path %q", repoName, pathConfig.Name)
		}
		if seen[repoName] {
			return nil, fmt.Errorf("repo %q was listed more than once", repoName)
		}
		seen[repoName] = true
		out = append(out, repoName)
	}

	for repoName := range repoRefs {
		if _, ok := projectConfig.Repos[repoName]; !ok {
			return nil, fmt.Errorf("repo %q is not registered", repoName)
		}
		if _, exists := pathConfig.Repos[repoName]; exists {
			return nil, fmt.Errorf("repo %q is already present in path %q", repoName, pathConfig.Name)
		}
		if !seen[repoName] {
			seen[repoName] = true
			out = append(out, repoName)
		}
	}

	sort.Strings(out)
	if len(out) == 0 {
		return nil, errors.New("timber add requires at least one repo to add")
	}
	return out, nil
}

func ListPaths(projectRoot string) ([]PathListEntry, error) {
	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	metadataDir := filepath.Join(projectRoot, config.MetadataDir, PathsMetadataDir)
	entries, err := os.ReadDir(metadataDir)
	if err != nil {
		return nil, err
	}

	var paths []PathListEntry
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		pathConfig, err := loadPathConfig(filepath.Join(metadataDir, entry.Name()))
		if err != nil {
			return nil, err
		}

		repoNames := sortedPathRepoNames(pathConfig)

		paths = append(paths, PathListEntry{
			Name:  pathConfig.Name,
			From:  pathSourceLabel(pathConfig),
			Repos: strings.Join(repoNames, ","),
			Path:  pathConfig.Path,
		})
	}

	sort.Slice(paths, func(i, j int) bool {
		return paths[i].Name < paths[j].Name
	})

	return paths, nil
}

func CompletePathNames(projectRoot, prefix string) ([]string, error) {
	paths, err := ListPaths(projectRoot)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.HasPrefix(path.Name, prefix) {
			names = append(names, path.Name)
		}
	}
	return names, nil
}

func PathExists(projectRoot, pathName string) bool {
	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return false
	}
	pathConfigPath := filepath.Join(projectRoot, config.MetadataDir, PathsMetadataDir, pathName+".yaml")
	_, err = os.Stat(pathConfigPath)
	return err == nil
}

func keepOperationStatePath(projectRoot string) string {
	return filepath.Join(projectRoot, MetadataDirName, OperationsDirName, "keep.yaml")
}

func writeKeepOperationState(path string, state KeepOperationState) error {
	data, err := yaml.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func loadKeepOperationState(path string) (KeepOperationState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return KeepOperationState{}, err
	}
	var state KeepOperationState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return KeepOperationState{}, err
	}
	if state.Repos == nil {
		state.Repos = map[string]KeepOperationRepoState{}
	}
	return state, nil
}

func loadPathByName(projectRoot string, projectConfig ProjectConfig, pathName string) (PathConfig, error) {
	pathConfigPath := filepath.Join(projectRoot, projectConfig.MetadataDir, PathsMetadataDir, pathName+".yaml")
	pathConfig, err := loadPathConfig(pathConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return PathConfig{}, fmt.Errorf("path %q not found", pathName)
		}
		return PathConfig{}, err
	}
	return pathConfig, nil
}

func ensurePathClean(projectRoot, pathName string) error {
	status, err := GetPathStatus(projectRoot, pathName)
	if err != nil {
		return err
	}
	for _, repoStatus := range status.ReposStatus {
		if repoStatus.Modified {
			return fmt.Errorf("path %q must be clean before keep; %s is modified", pathName, repoStatus.RepoName)
		}
	}
	return nil
}

func mergePathBranch(repoPath, branch string) (bool, error) {
	cmd := exec.Command("git", "-C", repoPath, "merge", "--no-ff", "--no-edit", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if mergeInProgress(repoPath) {
			return true, nil
		}
		message := strings.TrimSpace(string(output))
		if message == "" {
			return false, err
		}
		return false, fmt.Errorf("%v: %s", err, message)
	}
	return false, nil
}

func finalizeConflictedMerge(repoPath string) error {
	unmerged, err := gitOutput(".", "-C", repoPath, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return err
	}
	if strings.TrimSpace(unmerged) != "" {
		return errors.New("unresolved conflicts remain")
	}
	if !mergeInProgress(repoPath) {
		return nil
	}
	if err := runGitCommand(".", "-C", repoPath, "commit", "--no-edit"); err != nil {
		return err
	}
	return nil
}

func mergeInProgress(repoPath string) bool {
	_, err := gitOutput(".", "-C", repoPath, "rev-parse", "-q", "--verify", "MERGE_HEAD")
	return err == nil
}

func loadAllPathConfigs(projectRoot string, projectConfig ProjectConfig) (map[string]PathConfig, error) {
	dir := filepath.Join(projectRoot, projectConfig.MetadataDir, PathsMetadataDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	configs := map[string]PathConfig{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		config, err := loadPathConfig(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		configs[config.Name] = config
	}
	return configs, nil
}

func childPathNames(allConfigs map[string]PathConfig, parentName string) []string {
	children := make([]string, 0)
	for _, config := range allConfigs {
		if config.Parent.Type == "path" && config.Parent.PathName == parentName {
			children = append(children, config.Name)
		}
	}
	sort.Strings(children)
	return children
}

func orderDropPaths(requested []string, allConfigs map[string]PathConfig) []string {
	seen := map[string]bool{}
	ordered := make([]string, 0, len(requested))
	var visit func(string)
	visit = func(name string) {
		if seen[name] {
			return
		}
		seen[name] = true
		for _, child := range childPathNames(allConfigs, name) {
			visit(child)
		}
		ordered = append(ordered, name)
	}
	for _, name := range requested {
		visit(name)
	}
	return ordered
}

func validateDropPath(projectRoot string, config PathConfig, force bool) error {
	status, err := GetPathStatus(projectRoot, config.Name)
	if err != nil {
		return err
	}
	for _, repoStatus := range status.ReposStatus {
		if repoStatus.Modified && !force {
			return fmt.Errorf("path %q is modified in %s; use --force to drop anyway", config.Name, repoStatus.RepoName)
		}
		if repoStatus.CommitsAhead > 0 && !force {
			return fmt.Errorf("path %q has unkept commits in %s; use --force to drop anyway", config.Name, repoStatus.RepoName)
		}
	}
	return nil
}

func pathParentLabel(config PathConfig) string {
	if config.Parent.Type == "path" && strings.TrimSpace(config.Parent.PathName) != "" {
		return config.Parent.PathName
	}
	return ""
}

func pathSourceLabel(config PathConfig) string {
	if config.Parent.Type == "path" && strings.TrimSpace(config.Parent.PathName) != "" {
		return fmt.Sprintf("path %s", config.Parent.PathName)
	}
	return pathRemoteSourceSummary(config)
}

func pathRemoteSourceSummary(config PathConfig) string {
	repoNames := sortedPathRepoNames(config)
	if len(repoNames) == 0 {
		return strings.TrimSpace(config.Parent.Description)
	}

	defaultDisplay := strings.TrimSpace(config.Parent.DefaultSourceDisplay)
	if defaultDisplay == "" {
		parts := make([]string, 0, len(repoNames))
		for _, repoName := range repoNames {
			repoState := config.Repos[repoName]
			parts = append(parts, fmt.Sprintf("%s=%s", repoName, repoState.SourceDisplay))
		}
		return strings.Join(parts, " ")
	}

	overrides := make([]string, 0)
	for _, repoName := range repoNames {
		repoState := config.Repos[repoName]
		if repoState.SourceDisplay != defaultDisplay {
			overrides = append(overrides, fmt.Sprintf("%s=%s", repoName, repoState.SourceDisplay))
		}
	}
	if len(overrides) == 0 {
		return defaultDisplay
	}
	return fmt.Sprintf("%s + %s", defaultDisplay, strings.Join(overrides, " "))
}

func pathRecoverySummary(projectRoot, pathName string) (string, error) {
	state, err := loadKeepOperationState(keepOperationStatePath(projectRoot))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if state.SourcePath != pathName && state.TargetPath != pathName {
		return "", nil
	}

	conflictedRepos := make([]string, 0)
	pendingRepos := make([]string, 0)
	for _, repoName := range state.RepoOrder {
		repoState := state.Repos[repoName]
		switch repoState.Status {
		case "conflicted":
			conflictedRepos = append(conflictedRepos, repoName)
		case "pending":
			pendingRepos = append(pendingRepos, repoName)
		}
	}

	if len(conflictedRepos) > 0 {
		return fmt.Sprintf("keep in progress with conflicts in %s; run `timber keep --continue` or `timber keep --abort`", strings.Join(conflictedRepos, ",")), nil
	}
	if len(pendingRepos) > 0 {
		return fmt.Sprintf("keep in progress with pending repos %s; run `timber keep --continue` or `timber keep --abort`", strings.Join(pendingRepos, ",")), nil
	}
	return "keep operation in progress; run `timber keep --continue` or `timber keep --abort`", nil
}

func CompleteRepoNames(projectRoot, pathName, prefix string) ([]string, error) {
	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	var repoNames []string
	if pathName != "" {
		pathConfigPath := filepath.Join(projectRoot, config.MetadataDir, PathsMetadataDir, pathName+".yaml")
		pathConfig, err := loadPathConfig(pathConfigPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("path %q not found", pathName)
			}
			return nil, err
		}
		repoNames = sortedPathRepoNames(pathConfig)
	} else {
		repoNames = make([]string, 0, len(config.Repos))
		for repoName := range config.Repos {
			repoNames = append(repoNames, repoName)
		}
		sort.Strings(repoNames)
	}

	out := make([]string, 0, len(repoNames))
	for _, repoName := range repoNames {
		if strings.HasPrefix(repoName, prefix) {
			out = append(out, repoName)
		}
	}
	return out, nil
}

func CompleteRefs(projectRoot, repoName, prefix string) ([]string, error) {
	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}
	if _, ok := config.Repos[repoName]; !ok {
		return nil, fmt.Errorf("repo %q is not registered", repoName)
	}

	repoMirrorPath := filepath.Join(projectRoot, config.ReposDir, repoName+".git")
	if _, err := os.Stat(repoMirrorPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repo %q has not been synced yet", repoName)
	} else if err != nil {
		return nil, err
	}

	output, err := gitOutput(projectRoot, "-C", repoMirrorPath, "for-each-ref", "--format=%(refname)", "refs/heads", "refs/remotes/origin")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	refs := make([]string, 0)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasSuffix(line, "/HEAD") {
			continue
		}
		display := sourceDisplayName(line, line)
		if strings.HasPrefix(display, prefix) && !seen[display] {
			seen[display] = true
			refs = append(refs, display)
		}
	}
	sort.Strings(refs)
	return refs, nil
}

func sortedPathRepoNames(config PathConfig) []string {
	repoNames := make([]string, 0, len(config.Repos))
	for repoName := range config.Repos {
		repoNames = append(repoNames, repoName)
	}
	sort.Strings(repoNames)
	return repoNames
}

func GetPathStatus(projectRoot, pathName string) (*PathStatus, error) {
	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	pathConfigPath := filepath.Join(projectRoot, config.MetadataDir, PathsMetadataDir, pathName+".yaml")
	pathConfig, err := loadPathConfig(pathConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path %q not found", pathName)
		}
		return nil, err
	}
	allConfigs, err := loadAllPathConfigs(projectRoot, config)
	if err != nil {
		return nil, err
	}

	repoNames := sortedPathRepoNames(pathConfig)
	repoStatuses := make([]RepoPathStatus, 0, len(repoNames))
	modifiedCount := 0
	aheadRepos := 0
	for _, repoName := range repoNames {
		repoStatus, err := getRepoPathStatus(projectRoot, repoName, pathConfig.Repos[repoName])
		if err != nil {
			return nil, err
		}
		if repoStatus.Modified {
			modifiedCount++
		}
		if repoStatus.CommitsAhead > 0 {
			aheadRepos++
		}
		repoStatuses = append(repoStatuses, repoStatus)
	}
	recoverySummary, err := pathRecoverySummary(projectRoot, pathConfig.Name)
	if err != nil {
		return nil, err
	}

	return &PathStatus{
		Name:          pathConfig.Name,
		Path:          pathConfig.Path,
		From:          pathSourceLabel(pathConfig),
		Repos:         strings.Join(repoNames, ","),
		Parent:        pathParentLabel(pathConfig),
		Children:      childPathNames(allConfigs, pathConfig.Name),
		Recovery:      recoverySummary,
		StatusSummary: summarizePathRepoStatuses(len(repoStatuses), modifiedCount, aheadRepos),
		ReposStatus:   repoStatuses,
	}, nil
}

func GetPathPath(projectRoot, pathName string) (string, error) {
	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return "", err
	}

	pathConfigPath := filepath.Join(projectRoot, config.MetadataDir, PathsMetadataDir, pathName+".yaml")
	pathConfig, err := loadPathConfig(pathConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path %q not found", pathName)
		}
		return "", err
	}

	return filepath.Join(projectRoot, filepath.FromSlash(pathConfig.Path)), nil
}

func GetPathRunContext(projectRoot, pathName string) (*PathRunContext, error) {
	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	pathConfigPath := filepath.Join(projectRoot, config.MetadataDir, PathsMetadataDir, pathName+".yaml")
	pathConfig, err := loadPathConfig(pathConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path %q not found", pathName)
		}
		return nil, err
	}

	return &PathRunContext{
		Name:      pathConfig.Name,
		Root:      filepath.Join(projectRoot, filepath.FromSlash(pathConfig.Path)),
		RepoNames: sortedPathRepoNames(pathConfig),
	}, nil
}

func RunPathCommand(projectRoot, pathName string, command []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	if len(command) == 0 {
		return 0, errors.New("run command requires at least one command argument")
	}

	runContext, err := GetPathRunContext(projectRoot, pathName)
	if err != nil {
		return 0, err
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = runContext.Root
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = append(os.Environ(),
		"TIMBER_PROJECT_ROOT="+projectRoot,
		"TIMBER_PATH="+runContext.Name,
		"TIMBER_PATH_ROOT="+runContext.Root,
		"TIMBER_REPOS="+strings.Join(runContext.RepoNames, ","),
	)

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 0, err
	}

	return 0, nil
}

func GetPathInfo(projectRoot, pathName string) (*PathInfo, error) {
	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	pathConfigPath := filepath.Join(projectRoot, config.MetadataDir, PathsMetadataDir, pathName+".yaml")
	pathConfig, err := loadPathConfig(pathConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path %q not found", pathName)
		}
		return nil, err
	}
	allConfigs, err := loadAllPathConfigs(projectRoot, config)
	if err != nil {
		return nil, err
	}

	repoNames := sortedPathRepoNames(pathConfig)

	status, err := GetPathStatus(projectRoot, pathName)
	if err != nil {
		return nil, err
	}
	recoverySummary, err := pathRecoverySummary(projectRoot, pathConfig.Name)
	if err != nil {
		return nil, err
	}

	reposInfo := make([]RepoPathInfo, 0, len(repoNames))
	statusByRepo := make(map[string]RepoPathStatus, len(status.ReposStatus))
	for _, repoStatus := range status.ReposStatus {
		statusByRepo[repoStatus.RepoName] = repoStatus
	}
	for _, repoName := range repoNames {
		repoState := pathConfig.Repos[repoName]
		repoStatus := statusByRepo[repoName]
		reposInfo = append(reposInfo, RepoPathInfo{
			RepoName:      repoName,
			PrivateBranch: repoState.PrivateBranch,
			SourceRef:     repoState.SourceRef,
			SourceCommit:  shortCommit(repoState.SourceCommit),
			StatusSummary: repoStatus.StatusSummary,
		})
	}
	return &PathInfo{
		Name:          pathConfig.Name,
		Path:          pathConfig.Path,
		Created:       humanTimestamp(pathConfig.CreatedAt),
		From:          pathSourceLabel(pathConfig),
		Repos:         strings.Join(repoNames, ","),
		Parent:        pathParentLabel(pathConfig),
		Children:      childPathNames(allConfigs, pathConfig.Name),
		Recovery:      recoverySummary,
		ReposInfo:     reposInfo,
		StatusSummary: status.StatusSummary,
	}, nil
}

func GetPathDiff(projectRoot, pathName, repoName string) ([]RepoDiffSection, error) {
	config, err := LoadProjectConfig(projectRoot)
	if err != nil {
		return nil, err
	}

	pathConfigPath := filepath.Join(projectRoot, config.MetadataDir, PathsMetadataDir, pathName+".yaml")
	pathConfig, err := loadPathConfig(pathConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path %q not found", pathName)
		}
		return nil, err
	}

	repoNames := sortedPathRepoNames(pathConfig)
	if repoName != "" {
		if _, ok := pathConfig.Repos[repoName]; !ok {
			return nil, fmt.Errorf("repo %q is not present in path %q", repoName, pathName)
		}
		repoNames = []string{repoName}
	}

	sections := make([]RepoDiffSection, 0, len(repoNames))
	for _, currentRepo := range repoNames {
		repoState := pathConfig.Repos[currentRepo]
		repoPath := filepath.Join(projectRoot, filepath.FromSlash(repoState.Path))
		diff, err := gitOutputAllowEmpty(projectRoot, "-C", repoPath, "diff")
		if err != nil {
			return nil, fmt.Errorf("read diff for repo %q: %w", currentRepo, err)
		}
		sections = append(sections, RepoDiffSection{
			RepoName: currentRepo,
			Diff:     diff,
		})
	}

	return sections, nil
}

func repoAheadCount(projectRoot, repoPath string, repoState PathRepoState) (int, error) {
	baseline := strings.TrimSpace(repoState.SourceCommit)
	if baseline != "" {
		if err := runGitCommand(projectRoot, "-C", repoPath, "cat-file", "-e", baseline+"^{commit}"); err != nil {
			baseline = ""
		}
	}
	if baseline == "" && strings.TrimSpace(repoState.SourceRef) != "" {
		resolved, err := gitOutput(projectRoot, "-C", repoPath, "rev-parse", repoState.SourceRef)
		if err == nil {
			baseline = resolved
		}
	}
	if baseline == "" {
		return 0, nil
	}

	aheadOutput, err := gitOutput(projectRoot, "-C", repoPath, "rev-list", "--count", baseline+"..HEAD")
	if err != nil && strings.TrimSpace(repoState.SourceRef) != "" {
		mergeBase, mergeErr := gitOutput(projectRoot, "-C", repoPath, "merge-base", "HEAD", repoState.SourceRef)
		if mergeErr == nil {
			aheadOutput, err = gitOutput(projectRoot, "-C", repoPath, "rev-list", "--count", mergeBase+"..HEAD")
		}
	}
	if err != nil {
		return 0, nil
	}

	aheadCount, err := strconv.Atoi(strings.TrimSpace(aheadOutput))
	if err != nil {
		return 0, err
	}
	return aheadCount, nil
}

func getRepoPathStatus(projectRoot, repoName string, repoState PathRepoState) (RepoPathStatus, error) {
	repoPath := filepath.Join(projectRoot, filepath.FromSlash(repoState.Path))
	porcelain, err := gitOutput(projectRoot, "-C", repoPath, "status", "--porcelain=v1")
	if err != nil {
		return RepoPathStatus{}, fmt.Errorf("read status for repo %q: %w", repoName, err)
	}

	aheadCount, err := repoAheadCount(projectRoot, repoPath, repoState)
	if err != nil {
		return RepoPathStatus{}, fmt.Errorf("read ahead count for repo %q: %w", repoName, err)
	}

	modified := strings.TrimSpace(porcelain) != ""
	summary := "clean"
	if modified {
		summary = "modified"
	}
	if aheadCount > 0 {
		summary = fmt.Sprintf("%s, %d commit", summary, aheadCount)
		if aheadCount != 1 {
			summary += "s"
		}
		summary += " ahead"
	}

	return RepoPathStatus{
		RepoName:      repoName,
		PrivateBranch: repoState.PrivateBranch,
		StatusSummary: summary,
		Modified:      modified,
		CommitsAhead:  aheadCount,
	}, nil
}

func summarizePathRepoStatuses(totalRepos, modifiedRepos, aheadRepos int) string {
	if totalRepos == 0 {
		return "no repos"
	}
	parts := make([]string, 0, 2)
	if modifiedRepos == 0 {
		parts = append(parts, "clean")
	} else {
		parts = append(parts, fmt.Sprintf("%d modified", modifiedRepos))
	}
	if aheadRepos > 0 {
		parts = append(parts, fmt.Sprintf("%d ahead", aheadRepos))
	}
	return strings.Join(parts, ", ")
}

func runGitCommand(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return err
		}
		return fmt.Errorf("%v: %s", err, message)
	}
	return nil
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return "", err
		}
		return "", fmt.Errorf("%v: %s", err, message)
	}
	return strings.TrimSpace(string(output)), nil
}

func gitOutputAllowEmpty(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return "", err
		}
		return "", fmt.Errorf("%v: %s", err, message)
	}
	return string(output), nil
}

func resolveSourceRef(repoMirrorPath, input string) (string, string, error) {
	input = strings.TrimSpace(input)
	candidates := []string{
		input,
		"refs/remotes/origin/" + input,
		"refs/heads/" + input,
	}

	for _, candidate := range candidates {
		if _, err := gitOutput(".", "-C", repoMirrorPath, "rev-parse", "--verify", candidate); err == nil {
			resolvedRef, resolveErr := gitOutput(".", "-C", repoMirrorPath, "rev-parse", "--symbolic-full-name", candidate)
			if resolveErr != nil || resolvedRef == "" {
				resolvedRef = candidate
			}
			return resolvedRef, sourceDisplayName(resolvedRef, input), nil
		}
	}

	return "", "", fmt.Errorf("could not resolve ref %q", input)
}

func sourceDisplayName(resolvedRef, fallback string) string {
	switch {
	case strings.HasPrefix(resolvedRef, "refs/remotes/origin/"):
		return strings.TrimPrefix(resolvedRef, "refs/remotes/origin/")
	case strings.HasPrefix(resolvedRef, "refs/heads/"):
		return strings.TrimPrefix(resolvedRef, "refs/heads/")
	default:
		if fallback != "" {
			return fallback
		}
		return resolvedRef
	}
}

func makePrivateBranchName(pathName, repoName string) (string, error) {
	suffixBytes := make([]byte, 2)
	if _, err := rand.Read(suffixBytes); err != nil {
		return "", err
	}
	suffix := hex.EncodeToString(suffixBytes)
	return fmt.Sprintf("timber/%s/%s-%s", sanitizeBranchToken(pathName), sanitizeBranchToken(repoName), suffix), nil
}

func sanitizeBranchToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "path"
	}
	return result
}

func writePathConfig(path string, config PathConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func loadPathConfig(path string) (PathConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PathConfig{}, err
	}

	var config PathConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return PathConfig{}, err
	}
	if config.Repos == nil {
		config.Repos = map[string]PathRepoState{}
	}
	return config, nil
}

func writePathFiles(pathRoot string, config PathConfig) error {
	repoNames := sortedPathRepoNames(config)

	var rows strings.Builder
	for _, repoName := range repoNames {
		repoState := config.Repos[repoName]
		rows.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", repoName, repoState.SourceRef, shortCommit(repoState.SourceCommit), repoState.PrivateBranch))
	}

	var repoBullets strings.Builder
	for _, repoName := range repoNames {
		repoBullets.WriteString(fmt.Sprintf("- %s\n", repoName))
	}

	pathMD := fmt.Sprintf(`# %s

Created: %s
Parent: %s
Repos: %s

## Repos

| Repo | Started from | Resolved commit | Private branch |
|---|---|---:|---|
%s
## Common commands

`+"```bash"+`
timber here
timber status %s
timber dir %s
timber run %s -- <command>
`+"```"+`
`, config.Name, humanTimestamp(config.CreatedAt), config.Parent.Description, strings.Join(repoNames, ", "), rows.String(), config.Name, config.Name, config.Name)

	agentsMD := fmt.Sprintf(`# Agent instructions for this path

This is an isolated Timber path.

## Rules

- Read PATH.md first.
- Do not edit files outside this path.
- Prefer targeted searches over broad scans.

## Repos

%s`, repoBullets.String())

	if err := os.WriteFile(filepath.Join(pathRoot, "PATH.md"), []byte(pathMD), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(pathRoot, "AGENTS.md"), []byte(agentsMD), 0o644); err != nil {
		return err
	}
	return nil
}

func appendEventLog(projectRoot string, event map[string]string) error {
	eventPath := filepath.Join(projectRoot, MetadataDirName, EventLogFileName)
	keys := make([]string, 0, len(event))
	for key := range event {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteByte('{')
	for i, key := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Quote(key))
		b.WriteByte(':')
		b.WriteString(strconv.Quote(event[key]))
	}
	b.WriteString("}\n")

	file, err := os.OpenFile(eventPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(b.String())
	return err
}

func shortCommit(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}

func humanTimestamp(value string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return t.Format("2006-01-02 15:04")
}

func FindProjectRoot(start string) (string, bool) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}

	for {
		candidate := filepath.Join(current, MetadataDirName, ProjectFileName)
		if _, err := os.Stat(candidate); err == nil {
			return current, true
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func DetectContext(start string) (*ProjectContext, error) {
	projectRoot, ok := FindProjectRoot(start)
	if !ok {
		return nil, nil
	}

	absStart, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}

	relative, err := filepath.Rel(projectRoot, absStart)
	if err != nil {
		return nil, err
	}

	context := &ProjectContext{ProjectRoot: projectRoot}
	parts := splitPath(relative)
	if len(parts) >= 2 && parts[0] == PathsDirName {
		context.PathName = parts[1]
	}
	if len(parts) >= 3 && parts[0] == PathsDirName {
		context.RepoName = parts[2]
	}

	return context, nil
}

func splitPath(value string) []string {
	cleaned := filepath.Clean(value)
	if cleaned == "." || cleaned == "" {
		return nil
	}

	return strings.Split(filepath.ToSlash(cleaned), "/")
}
