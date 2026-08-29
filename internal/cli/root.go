package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sgallen/timber/internal/timber"
)

const version = "0.1.0"

func Execute() {
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printHelp()
		return
	}

	switch args[0] {
	case "version":
		if isHelpArgs(args[1:]) {
			printVersionHelp()
			return
		}
		fmt.Println(version)
	case "init":
		runInit(args[1:])
	case "repo":
		runRepo(args[1:])
	case "add":
		runAdd(args[1:])
	case "fork":
		runFork(args[1:])
	case "keep":
		runKeep(args[1:])
	case "drop":
		runDrop(args[1:])
	case "new":
		runNew(args[1:])
	case "ls":
		runList(args[1:])
	case "run":
		runRun(args[1:])
	case "complete":
		runComplete(args[1:])
	case "completion":
		runCompletion(args[1:])
	case "info":
		runInfo(args[1:])
	case "diff":
		runDiff(args[1:])
	case "status":
		runStatus(args[1:])
	case "dir":
		runDir(args[1:])
	case "here":
		runHere(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n\n", args[0])
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Usage: timber <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  version  Show the CLI version.")
	fmt.Println("  init     Initialize a new Timber project.")
	fmt.Println("  repo     Manage registered repos (add, ls, sync, rm).")
	fmt.Println("  add      Add synced repos to a path, e.g. `timber add auth-flow billing`.")
	fmt.Println("  fork     Fork a clean path into child paths.")
	fmt.Println("  keep     Keep a child path by merging it into a target path.")
	fmt.Println("  drop     Drop unwanted paths conservatively.")
	fmt.Println("  new      Create a bootstrap path from shared or per-repo refs.")
	fmt.Println("  ls       List paths.")
	fmt.Println("  run      Run a command from a path root.")
	fmt.Println("  completion  Print shell completion scripts and setup hints.")
	fmt.Println("  info     Show path details.")
	fmt.Println("  diff     Show path diffs.")
	fmt.Println("  status   Show bootstrap path status.")
	fmt.Println("  dir      Print a path's directory.")
	fmt.Println("  here     Show inferred Timber context for the current directory.")
}

func isHelpArgs(args []string) bool {
	return len(args) == 1 && (args[0] == "--help" || args[0] == "-h")
}

func printVersionHelp() {
	fmt.Println("Usage: timber version")
	fmt.Println()
	fmt.Println("Print the installed Timber CLI version.")
}

func printInitHelp() {
	fmt.Println("Usage: timber init <project-dir>")
	fmt.Println()
	fmt.Println("Initialize a new Timber project in the given directory.")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  timber init myproject")
}

func printHereHelp() {
	fmt.Println("Usage: timber here [path]")
	fmt.Println()
	fmt.Println("Show the Timber project/path/repo context inferred from the current directory or a provided path.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  timber here")
	fmt.Println("  timber here paths/auth-flow/app")
}

func runInit(args []string) {
	if isHelpArgs(args) {
		printInitHelp()
		return
	}
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "error: timber init requires exactly one project directory")
		os.Exit(1)
	}

	projectDir, err := filepath.Abs(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	configPath, err := timber.InitProject(projectDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Initialized Timber project at %s\n", projectDir)
	fmt.Printf("Metadata: %s\n", configPath)
}

func runHere(args []string) {
	if isHelpArgs(args) {
		printHereHelp()
		return
	}
	target := "."
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "error: timber here accepts zero or one path")
		os.Exit(1)
	}
	if len(args) == 1 {
		target = args[0]
	}

	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "error: invalid directory %q\n", target)
		os.Exit(1)
	}

	context, err := timber.DetectContext(target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if context == nil {
		fmt.Fprintf(os.Stderr, "error: no Timber project found above %s\n", target)
		os.Exit(1)
	}

	config, err := timber.LoadProjectConfig(context.ProjectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("project       %s\n", config.Name)
	fmt.Printf("project_root  %s\n", context.ProjectRoot)
	fmt.Printf("path     %s\n", dashIfEmpty(context.PathName))
	fmt.Printf("repo          %s\n", dashIfEmpty(context.RepoName))

	absTarget, _ := filepath.Abs(target)
	fmt.Printf("path          %s\n", absTarget)

	if context.PathName != "" {
		fmt.Println()
		fmt.Println("Suggested next commands:")
		fmt.Println("  timber status")
		fmt.Println("  timber run -- <command>")
		fmt.Println("  timber fork try-a try-b")
		fmt.Println("  timber info")
		fmt.Println("  timber dir")
		fmt.Println("  timber ls")
	}
}

func runRepo(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: timber repo requires a subcommand")
		printRepoHelp()
		os.Exit(1)
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		printRepoHelp()
		return
	}

	switch args[0] {
	case "add":
		runRepoAdd(args[1:])
	case "ls":
		runRepoList(args[1:])
	case "sync":
		runRepoSync(args[1:])
	case "rm":
		runRepoRemove(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown repo subcommand %q\n", args[0])
		printRepoHelp()
		os.Exit(1)
	}
}

func printRepoHelp() {
	fmt.Println("Usage: timber repo <subcommand> ...")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  add <name> <url>  Register a repo in the current project.")
	fmt.Println("  ls                List registered repos and sync status.")
	fmt.Println("  sync              Clone or fetch registered repos into .timber/repos/.")
	fmt.Println("  rm <name>         Remove a registered repo if no path uses it.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  timber repo add app git@github.com:company/app.git")
	fmt.Println("  timber repo ls")
	fmt.Println("  timber repo sync")
	fmt.Println("  timber repo rm app")
}

func printRepoAddHelp() {
	fmt.Println("Usage: timber repo add <name> <url>")
	fmt.Println()
	fmt.Println("Register a repo in the current Timber project.")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  timber repo add app git@github.com:company/app.git")
}

func printRepoListHelp() {
	fmt.Println("Usage: timber repo ls")
	fmt.Println()
	fmt.Println("List registered repos and whether each one has been synced into .timber/repos/.")
}

func printRepoRemoveHelp() {
	fmt.Println("Usage: timber repo rm <name>")
	fmt.Println()
	fmt.Println("Remove a registered repo if no path currently uses it.")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  timber repo rm app")
}

func printRepoSyncHelp() {
	fmt.Println("Usage: timber repo sync")
	fmt.Println()
	fmt.Println("Clone or fetch every registered repo into the local .timber/repos/ mirror cache.")
}

func runRepoAdd(args []string) {
	if isHelpArgs(args) {
		printRepoAddHelp()
		return
	}
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "error: timber repo add requires <name> <url>")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no Timber project found above current directory")
		os.Exit(1)
	}

	repoName := args[0]
	repoURL := args[1]
	if err := timber.AddRepo(projectRoot, repoName, repoURL); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Registered repo %s\n", repoName)
	fmt.Printf("Project: %s\n", projectRoot)
	fmt.Printf("URL: %s\n", repoURL)
	fmt.Println("Status: registered, not yet synced")
	fmt.Println()
	fmt.Println("Next:")
	fmt.Println("  timber repo sync")
}

func runRepoList(args []string) {
	if isHelpArgs(args) {
		printRepoListHelp()
		return
	}
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "error: timber repo ls does not accept positional arguments")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no Timber project found above current directory")
		os.Exit(1)
	}

	repos, err := timber.ListRepos(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(repos) == 0 {
		fmt.Println("No repos registered.")
		return
	}

	fmt.Println("NAME\tSTATUS\tURL")
	for _, repo := range repos {
		fmt.Printf("%s\t%s\t%s\n", repo.Name, repo.Status, repo.URL)
	}
}

func runRepoRemove(args []string) {
	if isHelpArgs(args) {
		printRepoRemoveHelp()
		return
	}
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "error: timber repo rm requires <name>")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no Timber project found above current directory")
		os.Exit(1)
	}

	repoName := args[0]
	if err := timber.RemoveRepo(projectRoot, repoName); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Removed repo %s\n", repoName)
}

func runRepoSync(args []string) {
	if isHelpArgs(args) {
		printRepoSyncHelp()
		return
	}
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "error: timber repo sync does not accept positional arguments")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no Timber project found above current directory")
		os.Exit(1)
	}

	repos, err := timber.ListRepos(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if len(repos) == 0 {
		fmt.Println("No repos registered.")
		return
	}

	fmt.Printf("Syncing %d repos...\n", len(repos))
	results, err := timber.SyncReposWithProgress(projectRoot, func(name, phase string) {
		fmt.Printf("%s  %s\n", phase, name)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	for _, result := range results {
		fmt.Printf("%s  %s\n", result.Action, result.Name)
	}
	fmt.Println("Done.")
}

func runNew(args []string) {
	if isHelpArgs(args) {
		printNewHelp()
		return
	}
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "error: bootstrap `timber new` requires <name> [--from <ref>] [--repos repo1,repo2 | --all] [repo=ref ...]")
		os.Exit(1)
	}

	pathName := args[0]
	sourceRef, repoNames, repoRefs, includeAll, err := parseNewArgs(args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no Timber project found above current directory")
		os.Exit(1)
	}

	result, err := timber.CreatePath(projectRoot, timber.CreatePathOptions{
		Name:          pathName,
		DefaultRef:    sourceRef,
		SelectedRepos: repoNames,
		RepoRefs:      repoRefs,
		IncludeAll:    includeAll,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created path %s\n", result.Name)
	fmt.Printf("Path: %s\n", result.Path)
	if len(result.RepoNames) == 1 {
		fmt.Printf("Repo: %s\n", result.RepoName)
		fmt.Printf("Branch: %s\n", result.PrivateBranch)
	} else {
		fmt.Printf("Repos: %s\n", joinCSV(result.RepoNames))
	}
}

func runAdd(args []string) {
	if len(args) == 0 || (len(args) == 1 && (args[0] == "--help" || args[0] == "-h")) {
		printAddHelp()
		if len(args) == 0 {
			os.Exit(1)
		}
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no Timber project found above current directory")
		os.Exit(1)
	}

	context, err := timber.DetectContext(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	pathName, defaultRef, repoNames, repoRefs, err := parseAddArgs(args, context)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	result, err := timber.AddReposToPath(projectRoot, timber.AddReposOptions{
		PathName:   pathName,
		DefaultRef: defaultRef,
		RepoNames:  repoNames,
		RepoRefs:   repoRefs,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated path %s\n", result.PathName)
	fmt.Printf("Added repos: %s\n", joinCSV(result.AddedRepos))
}

func printNewHelp() {
	fmt.Println("Usage: timber new <name> --from <ref> [--repos repo1,repo2 | --all] [repo=ref ...]")
	fmt.Println("   or: timber new <name> repo=ref ...")
	fmt.Println()
	fmt.Println("Create a new path from shared or per-repo refs.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  timber new auth-flow --from qa --repos app,helm0")
	fmt.Println("  timber new review-auth app=main auth=hotfix/123")
	fmt.Println("  timber new full-stack --from develop --all")
}

func printAddHelp() {
	fmt.Println("Usage: timber add [<path>] [--from <ref>] <repo>... [repo=ref ...]")
	fmt.Println()
	fmt.Println("Add synced repos to an existing path.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  timber add auth-flow billing")
	fmt.Println("  timber add auth-flow --from qa billing")
	fmt.Println("  timber add auth-flow billing=master")
	fmt.Println("  # from inside auth-flow")
	fmt.Println("  timber add billing")
}

func runFork(args []string) {
	if isHelpArgs(args) {
		printForkHelp()
		return
	}
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: usage: timber fork [<source>] <child>...")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no Timber project found above current directory")
		os.Exit(1)
	}

	context, err := timber.DetectContext(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	sourcePath := ""
	childNames := args
	if context != nil && context.PathName != "" {
		if len(args) >= 2 && timber.PathExists(projectRoot, args[0]) {
			sourcePath = args[0]
			childNames = args[1:]
		} else {
			sourcePath = context.PathName
		}
	} else {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: timber fork requires a source path when run outside a path")
			os.Exit(1)
		}
		sourcePath = args[0]
		childNames = args[1:]
	}

	result, err := timber.ForkPath(projectRoot, sourcePath, childNames)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	createdNames := make([]string, 0, len(result.Created))
	for _, child := range result.Created {
		createdNames = append(createdNames, child.Name)
	}
	fmt.Printf("Forked path %s\n", result.SourcePath)
	fmt.Printf("Created: %s\n", joinCSV(createdNames))
}

func printForkHelp() {
	fmt.Println("Usage: timber fork [<source>] <child>...")
	fmt.Println()
	fmt.Println("Fork a clean path into one or more child paths.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  timber fork auth-flow try-api try-ui")
	fmt.Println("  # from inside auth-flow")
	fmt.Println("  timber fork try-api try-ui")
}

func runKeep(args []string) {
	if isHelpArgs(args) {
		printKeepHelp()
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no Timber project found above current directory")
		os.Exit(1)
	}

	if len(args) == 1 && args[0] == "--continue" {
		result, err := timber.ContinueKeepPath(projectRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		printKeepResult(result)
		return
	}
	if len(args) == 1 && args[0] == "--abort" {
		if err := timber.AbortKeepPath(projectRoot); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Aborted keep operation state.")
		return
	}

	context, err := timber.DetectContext(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	sourcePath := ""
	targetPath := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--into":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: usage: timber keep [<child>] --into <target>")
				os.Exit(1)
			}
			targetPath = args[i+1]
			i++
		default:
			if sourcePath != "" {
				fmt.Fprintln(os.Stderr, "error: usage: timber keep [<child>] --into <target>")
				os.Exit(1)
			}
			sourcePath = args[i]
		}
	}

	if sourcePath == "" && context != nil {
		sourcePath = context.PathName
	}
	if sourcePath == "" || targetPath == "" {
		fmt.Fprintln(os.Stderr, "error: usage: timber keep [<child>] --into <target>")
		os.Exit(1)
	}

	result, err := timber.KeepPath(projectRoot, sourcePath, targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	printKeepResult(result)
}

func printKeepHelp() {
	fmt.Println("Usage: timber keep [<child>] --into <target>")
	fmt.Println("   or: timber keep --continue")
	fmt.Println("   or: timber keep --abort")
	fmt.Println()
	fmt.Println("Merge a child path back into a target path repo-by-repo.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  timber keep try-auth-only --into auth-flow")
	fmt.Println("  timber keep --continue")
	fmt.Println("  timber keep --abort")
}

func printKeepResult(result *timber.KeepPathResult) {
	fmt.Printf("Kept path %s into %s\n", result.SourcePath, result.TargetPath)
	if len(result.MergedRepos) > 0 {
		fmt.Printf("Merged repos: %s\n", joinCSV(result.MergedRepos))
	}
	if len(result.SkippedRepos) > 0 {
		fmt.Printf("Skipped repos: %s\n", joinCSV(result.SkippedRepos))
	}
}

func runDrop(args []string) {
	if isHelpArgs(args) {
		printDropHelp()
		return
	}
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: usage: timber drop <path>... [--force] [--recursive] [--keep-branches|--delete-branches]")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no Timber project found above current directory")
		os.Exit(1)
	}

	options := timber.DropPathOptions{}
	for _, arg := range args {
		switch arg {
		case "--force":
			options.Force = true
		case "--recursive":
			options.Recursive = true
		case "--keep-branches":
			options.KeepBranches = true
		case "--delete-branches":
			options.DeleteBranches = true
		default:
			options.PathNames = append(options.PathNames, arg)
		}
	}

	result, err := timber.DropPaths(projectRoot, options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Dropped paths: %s\n", joinCSV(result.Dropped))
	if len(result.KeptBranches) > 0 {
		fmt.Printf("Kept branches: %s\n", joinCSV(result.KeptBranches))
	}
	if len(result.DeletedBranches) > 0 {
		fmt.Printf("Deleted branches: %s\n", joinCSV(result.DeletedBranches))
	}
}

func printDropHelp() {
	fmt.Println("Usage: timber drop <path>... [--force] [--recursive] [--keep-branches|--delete-branches]")
	fmt.Println()
	fmt.Println("Drop one or more paths conservatively.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  timber drop try-api")
	fmt.Println("  timber drop auth-flow --recursive")
	fmt.Println("  timber drop try-ui --force --keep-branches")
}

func runList(args []string) {
	if isHelpArgs(args) {
		printListHelp()
		return
	}
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "error: timber ls does not accept positional arguments")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no Timber project found above current directory")
		os.Exit(1)
	}

	paths, err := timber.ListPaths(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(paths) == 0 {
		fmt.Println("No paths found.")
		return
	}

	fmt.Println("NAME\tSOURCE\tREPOS\tPATH")
	for _, path := range paths {
		fmt.Printf("%s\t%s\t%s\t%s\n", path.Name, path.From, path.Repos, path.Path)
	}
}

func printListHelp() {
	fmt.Println("Usage: timber ls")
	fmt.Println()
	fmt.Println("List paths in the current project.")
}

func runRun(args []string) {
	if isHelpArgs(args) {
		printRunHelp()
		return
	}
	pathName, commandArgs, err := parseRunArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no Timber project found above current directory")
		os.Exit(1)
	}

	if pathName == "" {
		context, err := timber.DetectContext(cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if context != nil {
			pathName = context.PathName
		}
	}
	if pathName == "" {
		fmt.Fprintln(os.Stderr, "error: timber run requires a path name when run outside a path")
		os.Exit(1)
	}

	exitCode, err := timber.RunPathCommand(projectRoot, pathName, commandArgs, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func printRunHelp() {
	fmt.Println("Usage: timber run [<path>] -- <command> [args...]")
	fmt.Println()
	fmt.Println("Run a command from a path root.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  timber run auth-flow -- pwd")
	fmt.Println("  timber run auth-flow -- codex")
	fmt.Println("  # from inside a path")
	fmt.Println("  timber run -- make test")
}

func runComplete(args []string) {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		printCompleteHelp()
		return
	}
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: timber complete requires a subcommand")
		os.Exit(1)
	}

	switch args[0] {
	case "paths":
		runCompletePaths(args[1:])
	case "repos":
		runCompleteRepos(args[1:])
	case "refs":
		runCompleteRefs(args[1:])
	case "commands":
		runCompleteCommands(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown complete subcommand %q\n", args[0])
		os.Exit(1)
	}
}

func printCompleteHelp() {
	fmt.Println("Usage: timber complete <paths|repos|refs|commands> ...")
	fmt.Println()
	fmt.Println("Internal completion backend used by shell completion scripts.")
}

func runCompleteCommands(args []string) {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "error: timber complete commands does not accept arguments")
		os.Exit(1)
	}

	for _, command := range []string{"version", "init", "repo", "add", "fork", "keep", "drop", "new", "ls", "run", "complete", "completion", "info", "diff", "status", "dir", "here"} {
		fmt.Println(command)
	}
}

func runCompletePaths(args []string) {
	prefix := ""
	switch len(args) {
	case 0:
	case 2:
		if args[0] != "--prefix" {
			fmt.Fprintln(os.Stderr, "error: usage: timber complete paths [--prefix <prefix>]")
			os.Exit(1)
		}
		prefix = args[1]
	default:
		fmt.Fprintln(os.Stderr, "error: usage: timber complete paths [--prefix <prefix>]")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		return
	}

	names, err := timber.CompletePathNames(projectRoot, prefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	for _, name := range names {
		fmt.Println(name)
	}
}

func runCompleteRepos(args []string) {
	prefix := ""
	pathName := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--prefix":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: usage: timber complete repos [--path <name>] [--prefix <prefix>]")
				os.Exit(1)
			}
			prefix = args[i+1]
			i++
		case "--path":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: usage: timber complete repos [--path <name>] [--prefix <prefix>]")
				os.Exit(1)
			}
			pathName = args[i+1]
			i++
		default:
			fmt.Fprintln(os.Stderr, "error: usage: timber complete repos [--path <name>] [--prefix <prefix>]")
			os.Exit(1)
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		return
	}

	names, err := timber.CompleteRepoNames(projectRoot, pathName, prefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	for _, name := range names {
		fmt.Println(name)
	}
}

func runCompleteRefs(args []string) {
	prefix := ""
	repoName := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--prefix":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: usage: timber complete refs --repo <name> [--prefix <prefix>]")
				os.Exit(1)
			}
			prefix = args[i+1]
			i++
		case "--repo":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: usage: timber complete refs --repo <name> [--prefix <prefix>]")
				os.Exit(1)
			}
			repoName = args[i+1]
			i++
		default:
			fmt.Fprintln(os.Stderr, "error: usage: timber complete refs --repo <name> [--prefix <prefix>]")
			os.Exit(1)
		}
	}
	if repoName == "" {
		fmt.Fprintln(os.Stderr, "error: usage: timber complete refs --repo <name> [--prefix <prefix>]")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		return
	}

	refs, err := timber.CompleteRefs(projectRoot, repoName, prefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	for _, ref := range refs {
		fmt.Println(ref)
	}
}

func runCompletion(args []string) {
	if isHelpArgs(args) {
		printCompletionHelp()
		return
	}
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "error: usage: timber completion <bash|zsh>")
		fmt.Fprintln(os.Stderr, "hint: source <(timber completion bash)")
		fmt.Fprintln(os.Stderr, "hint: source <(timber completion zsh)")
		os.Exit(1)
	}

	switch args[0] {
	case "bash":
		printBashCompletion()
	case "zsh":
		printZshCompletion()
	default:
		fmt.Fprintln(os.Stderr, "error: usage: timber completion <bash|zsh>")
		fmt.Fprintln(os.Stderr, "hint: source <(timber completion bash)")
		fmt.Fprintln(os.Stderr, "hint: source <(timber completion zsh)")
		os.Exit(1)
	}
}

func printCompletionHelp() {
	fmt.Println("Usage: timber completion <bash|zsh>")
	fmt.Println()
	fmt.Println("Print a shell completion script and setup hints.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  source <(timber completion bash)")
	fmt.Println("  source <(timber completion zsh)")
}

func printBashCompletion() {
	fmt.Print(`# Timber bash completion
`)
	fmt.Print(`# Enable for this shell: source <(timber completion bash)
`)
	fmt.Print(`# Persist by adding that line to ~/.bashrc or ~/.bash_profile.

`)
	fmt.Print(`_wb_completion() {
  local cur prev
  local commands
  local path
  local i
  local token
  local repo
  local prefix
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"

  if [[ $COMP_CWORD -eq 1 ]]; then
    commands="$(timber complete commands)"
    mapfile -t COMPREPLY < <(compgen -W "$commands" -- "$cur")
    return
  fi

  case "$prev" in
    info|status|dir|fork|keep|drop|run|--into)
      mapfile -t COMPREPLY < <(timber complete paths --prefix "$cur")
      return
      ;;
    --repo)
      path=""
      for ((i=1; i<COMP_CWORD-1; i++)); do
        token="${COMP_WORDS[i]}"
        case "$token" in
          diff|info|status|dir|fork|keep|drop) ;;
          --*) i=$((i+1)) ;;
          *)
            if [[ -z "$path" ]]; then
              path="$token"
            fi
            ;;
        esac
      done
      if [[ -n "$path" ]]; then
        mapfile -t COMPREPLY < <(timber complete repos --path "$path" --prefix "$cur")
      else
        mapfile -t COMPREPLY < <(timber complete repos --prefix "$cur")
      fi
      return
      ;;
    --repos)
      mapfile -t COMPREPLY < <(timber complete repos --prefix "$cur")
      return
      ;;
  esac

  if [[ "${COMP_WORDS[1]}" == "new" || "${COMP_WORDS[1]}" == "add" ]]; then
    if [[ "$cur" == *=* ]]; then
      repo="${cur%%=*}"
      prefix="${cur#*=}"
      if [[ -n "$repo" && -n "$prefix" || "$cur" == *= ]]; then
        mapfile -t COMPREPLY < <(timber complete refs --repo "$repo" --prefix "$prefix" | sed "s#^#$repo=#")
        return
      fi
    fi
  fi
}

complete -F _wb_completion timber
`)
}

func printZshCompletion() {
	fmt.Print(`#compdef timber
# Timber zsh completion
# Enable for this shell: source <(timber completion zsh)
# Persist by adding that line to ~/.zshrc

_wb() {
  local state
  local -a commands path_names repo_names ref_names

  commands=("${(@f)$(timber complete commands)}")

  if (( CURRENT == 2 )); then
    _describe 'command' commands
    return
  fi

  case "${words[2]}" in
    info|status|dir|fork|drop|run)
      path_names=("${(@f)$(timber complete paths --prefix "${PREFIX}")}")
      _describe 'path' path_names
      return
      ;;
    keep)
      if [[ "${words[CURRENT-1]}" == "--into" ]]; then
        path_names=("${(@f)$(timber complete paths --prefix "${PREFIX}")}")
        _describe 'path' path_names
        return
      fi
      if (( CURRENT == 3 )); then
        path_names=("${(@f)$(timber complete paths --prefix "${PREFIX}")}")
        _describe 'path' path_names
        return
      fi
      ;;
    diff)
      if [[ "${words[CURRENT-1]}" == "--repo" ]]; then
        repo_names=("${(@f)$(timber complete repos --prefix "${PREFIX}")}")
        _describe 'repo' repo_names
        return
      fi
      if (( CURRENT == 3 )); then
        path_names=("${(@f)$(timber complete paths --prefix "${PREFIX}")}")
        _describe 'path' path_names
        return
      fi
      ;;
    new)
      if [[ "${words[CURRENT-1]}" == "--repos" ]]; then
        repo_names=("${(@f)$(timber complete repos --prefix "${PREFIX}")}")
        _describe 'repo' repo_names
        return
      fi
      if [[ "${PREFIX}" == *=* ]]; then
        local repo="${PREFIX%%=*}"
        local ref_prefix="${PREFIX#*=}"
        ref_names=("${(@f)$(timber complete refs --repo "${repo}" --prefix "${ref_prefix}" | sed "s#^#${repo}=#")}")
        _describe 'repo-ref' ref_names
        return
      fi
      ;;
    add)
      if [[ "${words[CURRENT-1]}" == "--repo" ]]; then
        repo_names=("${(@f)$(timber complete repos --prefix "${PREFIX}")}")
        _describe 'repo' repo_names
        return
      fi
      if [[ "${PREFIX}" == *=* ]]; then
        local repo="${PREFIX%%=*}"
        local ref_prefix="${PREFIX#*=}"
        ref_names=("${(@f)$(timber complete refs --repo "${repo}" --prefix "${ref_prefix}" | sed "s#^#${repo}=#")}")
        _describe 'repo-ref' ref_names
        return
      fi
      if (( CURRENT == 3 )); then
        path_names=("${(@f)$(timber complete paths --prefix "${PREFIX}")}")
        _describe 'path' path_names
        return
      fi
      ;;
  esac
}

compdef _wb timber
`)
}

func runInfo(args []string) {
	if isHelpArgs(args) {
		printInfoHelp()
		return
	}
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "error: timber info accepts zero or one path name")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no Timber project found above current directory")
		os.Exit(1)
	}

	pathName := ""
	if len(args) == 1 {
		pathName = args[0]
	} else if context, err := timber.DetectContext(cwd); err == nil && context != nil {
		pathName = context.PathName
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if pathName == "" {
		fmt.Fprintln(os.Stderr, "error: timber info requires a path name when run outside a path")
		os.Exit(1)
	}

	info, err := timber.GetPathInfo(projectRoot, pathName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", info.Name)
	fmt.Printf("path: %s\n", info.Path)
	fmt.Printf("created: %s\n", info.Created)
	fmt.Printf("source: %s\n", info.From)
	if info.Parent != "" {
		fmt.Printf("parent: %s\n", info.Parent)
	}
	if len(info.Children) > 0 {
		fmt.Printf("children: %s\n", joinCSV(info.Children))
	}
	fmt.Printf("repos: %s\n", info.Repos)
	if info.Recovery != "" {
		fmt.Printf("recovery: %s\n", info.Recovery)
	}
	fmt.Println()
	fmt.Println("repo details:")
	for _, repo := range info.ReposInfo {
		fmt.Printf("  %s\n", repo.RepoName)
		fmt.Printf("    branch: %s\n", repo.PrivateBranch)
		fmt.Printf("    started_from: %s @ %s\n", repo.SourceRef, repo.SourceCommit)
	}
	fmt.Println()
	fmt.Println("resume:")
	fmt.Printf("  timber status %s\n", info.Name)
	fmt.Printf("  timber dir %s\n", info.Name)
	fmt.Printf("  timber here %s\n", info.Path)
	if info.Parent != "" {
		fmt.Printf("  timber keep %s --into %s\n", info.Name, info.Parent)
	}
	if info.Recovery != "" {
		fmt.Println("  timber keep --continue")
		fmt.Println("  timber keep --abort")
	}
}

func printInfoHelp() {
	fmt.Println("Usage: timber info [path]")
	fmt.Println()
	fmt.Println("Show structural path information such as source refs, repos, and private branches.")
}

func runDiff(args []string) {
	if isHelpArgs(args) {
		printDiffHelp()
		return
	}
	pathName := ""
	repoName := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--repo":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: usage: timber diff [path] [--repo <name>]")
				os.Exit(1)
			}
			repoName = args[i+1]
			i++
		default:
			if pathName != "" {
				fmt.Fprintln(os.Stderr, "error: usage: timber diff [path] [--repo <name>]")
				os.Exit(1)
			}
			pathName = args[i]
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no Timber project found above current directory")
		os.Exit(1)
	}

	if pathName == "" {
		if context, err := timber.DetectContext(cwd); err == nil && context != nil {
			pathName = context.PathName
			if repoName == "" {
				repoName = context.RepoName
			}
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	if pathName == "" {
		fmt.Fprintln(os.Stderr, "error: timber diff requires a path name when run outside a path")
		os.Exit(1)
	}

	sections, err := timber.GetPathDiff(projectRoot, pathName, repoName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(sections) == 0 {
		fmt.Println("No diffs.")
		return
	}

	for i, section := range sections {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("== %s ==\n", section.RepoName)
		if strings.TrimSpace(section.Diff) == "" {
			fmt.Println("(clean)")
			continue
		}
		fmt.Print(section.Diff)
		if !strings.HasSuffix(section.Diff, "\n") {
			fmt.Println()
		}
	}
}

func printDiffHelp() {
	fmt.Println("Usage: timber diff [path] [--repo <name>]")
	fmt.Println()
	fmt.Println("Show current diffs for all repos in a path or for one selected repo.")
}

func runStatus(args []string) {
	if isHelpArgs(args) {
		printStatusHelp()
		return
	}
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "error: timber status accepts zero or one path name")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no Timber project found above current directory")
		os.Exit(1)
	}

	pathName := ""
	if len(args) == 1 {
		pathName = args[0]
	} else if context, err := timber.DetectContext(cwd); err == nil && context != nil {
		pathName = context.PathName
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if pathName == "" {
		fmt.Fprintln(os.Stderr, "error: timber status requires a path name when run outside a path")
		os.Exit(1)
	}

	status, err := timber.GetPathStatus(projectRoot, pathName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s\n", status.Name)
	fmt.Printf("path: %s\n", status.Path)
	fmt.Printf("source: %s\n", status.From)
	if status.Parent != "" {
		fmt.Printf("parent: %s\n", status.Parent)
	}
	if len(status.Children) > 0 {
		fmt.Printf("children: %s\n", joinCSV(status.Children))
	}
	fmt.Printf("repos: %s\n", status.Repos)
	fmt.Printf("status: %s\n", status.StatusSummary)
	if status.Recovery != "" {
		fmt.Printf("recovery: %s\n", status.Recovery)
	}
	if len(status.ReposStatus) > 0 {
		fmt.Println()
		fmt.Println("repo status:")
		for _, repo := range status.ReposStatus {
			fmt.Printf("  %s\n", repo.RepoName)
			fmt.Printf("    branch: %s\n", repo.PrivateBranch)
			fmt.Printf("    state: %s\n", repo.StatusSummary)
			fmt.Printf("    commits_ahead: %d\n", repo.CommitsAhead)
		}
	}
}

func printStatusHelp() {
	fmt.Println("Usage: timber status [path]")
	fmt.Println()
	fmt.Println("Show live path state such as modified repos, ahead counts, and recovery hints.")
}

func runDir(args []string) {
	if isHelpArgs(args) {
		printDirHelp()
		return
	}
	if len(args) > 1 {
		fmt.Fprintln(os.Stderr, "error: timber dir accepts zero or one path name")
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	projectRoot, ok := timber.FindProjectRoot(cwd)
	if !ok {
		fmt.Fprintln(os.Stderr, "error: no Timber project found above current directory")
		os.Exit(1)
	}

	pathName := ""
	if len(args) == 1 {
		pathName = args[0]
	} else if context, err := timber.DetectContext(cwd); err == nil && context != nil {
		pathName = context.PathName
	} else if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if pathName == "" {
		fmt.Fprintln(os.Stderr, "error: timber dir requires a path name when run outside a path")
		os.Exit(1)
	}

	path, err := timber.GetPathPath(projectRoot, pathName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(path)
}

func printDirHelp() {
	fmt.Println("Usage: timber dir [path]")
	fmt.Println()
	fmt.Println("Print the absolute filesystem path to a path root.")
}

func dashIfEmpty(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func parseRunArgs(args []string) (string, []string, error) {
	separator := -1
	for i, arg := range args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator == -1 {
		return "", nil, fmt.Errorf("usage: timber run [<path>] -- <command>")
	}

	pathArgs := args[:separator]
	commandArgs := args[separator+1:]
	if len(pathArgs) > 1 || len(commandArgs) == 0 {
		return "", nil, fmt.Errorf("usage: timber run [<path>] -- <command>")
	}

	pathName := ""
	if len(pathArgs) == 1 {
		pathName = pathArgs[0]
	}
	return pathName, commandArgs, nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			names = append(names, trimmed)
		}
	}
	return names
}

func joinCSV(values []string) string {
	return strings.Join(values, ",")
}

func parseNewArgs(args []string) (string, []string, map[string]string, bool, error) {
	var sourceRef string
	var repoNames []string
	includeAll := false
	repoRefs := map[string]string{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from":
			if i+1 >= len(args) {
				return "", nil, nil, false, fmt.Errorf("usage: timber new <name> [--from <ref>] [--repos repo1,repo2 | --all] [repo=ref ...]")
			}
			if sourceRef != "" {
				return "", nil, nil, false, fmt.Errorf("--from may only be specified once")
			}
			sourceRef = args[i+1]
			i++
		case "--repos":
			if i+1 >= len(args) {
				return "", nil, nil, false, fmt.Errorf("usage: timber new <name> [--from <ref>] [--repos repo1,repo2 | --all] [repo=ref ...]")
			}
			if repoNames != nil {
				return "", nil, nil, false, fmt.Errorf("--repos may only be specified once")
			}
			if includeAll {
				return "", nil, nil, false, fmt.Errorf("cannot use --repos together with --all")
			}
			repoNames = splitCSV(args[i+1])
			if len(repoNames) == 0 {
				return "", nil, nil, false, fmt.Errorf("--repos requires at least one repo name")
			}
			i++
		case "--all":
			if includeAll {
				return "", nil, nil, false, fmt.Errorf("--all may only be specified once")
			}
			if repoNames != nil {
				return "", nil, nil, false, fmt.Errorf("cannot use --all together with --repos")
			}
			includeAll = true
		default:
			name, ref, ok := strings.Cut(args[i], "=")
			if !ok || strings.TrimSpace(name) == "" || strings.TrimSpace(ref) == "" {
				return "", nil, nil, false, fmt.Errorf("usage: timber new <name> [--from <ref>] [--repos repo1,repo2 | --all] [repo=ref ...]")
			}
			if _, exists := repoRefs[name]; exists {
				return "", nil, nil, false, fmt.Errorf("repo %q was mapped more than once", name)
			}
			repoRefs[name] = ref
		}
	}

	if sourceRef == "" && len(repoRefs) == 0 {
		return "", nil, nil, false, fmt.Errorf("bootstrap `timber new` requires --from <ref> or at least one repo=ref mapping")
	}

	return sourceRef, repoNames, repoRefs, includeAll, nil
}

func parseAddArgs(args []string, context *timber.ProjectContext) (string, string, []string, map[string]string, error) {
	pathName := ""
	if context != nil {
		pathName = context.PathName
	}

	if pathName == "" {
		if len(args) == 0 {
			return "", "", nil, nil, fmt.Errorf("usage: timber add <path> [--from <ref>] <repo>... [repo=ref ...]")
		}
		pathName = args[0]
		args = args[1:]
	}

	var defaultRef string
	var repoNames []string
	repoRefs := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--from":
			if i+1 >= len(args) {
				return "", "", nil, nil, fmt.Errorf("usage: timber add [path] [--from <ref>] <repo>... [repo=ref ...]")
			}
			if defaultRef != "" {
				return "", "", nil, nil, fmt.Errorf("--from may only be specified once")
			}
			defaultRef = args[i+1]
			i++
		default:
			name, ref, ok := strings.Cut(args[i], "=")
			if ok {
				if strings.TrimSpace(name) == "" || strings.TrimSpace(ref) == "" {
					return "", "", nil, nil, fmt.Errorf("usage: timber add [path] [--from <ref>] <repo>... [repo=ref ...]")
				}
				if _, exists := repoRefs[name]; exists {
					return "", "", nil, nil, fmt.Errorf("repo %q was mapped more than once", name)
				}
				repoRefs[name] = ref
				continue
			}
			if strings.TrimSpace(args[i]) == "" {
				return "", "", nil, nil, fmt.Errorf("usage: timber add [path] [--from <ref>] <repo>... [repo=ref ...]")
			}
			repoNames = append(repoNames, strings.TrimSpace(args[i]))
		}
	}

	if len(repoNames) == 0 && len(repoRefs) == 0 {
		return "", "", nil, nil, fmt.Errorf("timber add requires at least one repo name or repo=ref mapping")
	}

	return pathName, defaultRef, repoNames, repoRefs, nil
}
