package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/turbra/scrubctl/internal/archive"
	"github.com/turbra/scrubctl/internal/config"
	"github.com/turbra/scrubctl/internal/argocd"
	"github.com/turbra/scrubctl/internal/classify"
	"github.com/turbra/scrubctl/internal/resources"
	"github.com/turbra/scrubctl/internal/sanitize"
	"github.com/turbra/scrubctl/internal/scan"
	"github.com/turbra/scrubctl/internal/types"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

type rootOptions struct {
	ConfigPath     string
	Kubeconfig     string
	Context        string
	Namespace      string
	SecretHandling string
	IncludeKinds   string
	ExcludeKinds   string
	Quiet          bool
	LogLevel       string
}

func main() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	opts := &rootOptions{}
	cmd := &cobra.Command{
		Use:           "scrubctl",
		Short:         "Sanitize live manifests and generate GitOps export artifacts",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return applyConfigFile(cmd, opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmd.Help()
			}
			if stdinHasData(cmd.InOrStdin()) {
				return runScrub(cmd, opts, "")
			}
			return cmd.Help()
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&opts.ConfigPath, "config", "", "Path to a config file for default flag values")
	flags.StringVar(&opts.Kubeconfig, "kubeconfig", "", "Path to the kubeconfig file")
	flags.StringVar(&opts.Context, "context", "", "Kubeconfig context to use")
	flags.StringVarP(&opts.Namespace, "namespace", "n", "", "Target namespace")
	flags.StringVar(&opts.SecretHandling, "secret-handling", "redact", "Secret handling mode: redact, omit, or include")
	flags.StringVar(&opts.IncludeKinds, "include-kinds", "", "Comma-separated curated kinds or registry keys to include")
	flags.StringVar(&opts.ExcludeKinds, "exclude-kinds", "", "Comma-separated curated kinds or registry keys to exclude")
	flags.BoolVarP(&opts.Quiet, "quiet", "q", false, "Suppress non-essential output")
	flags.StringVar(&opts.LogLevel, "log-level", "info", "Log level")

	cmd.AddCommand(
		newScanCommand(opts),
		newExportCommand(opts),
		newScrubCommand(opts),
		newGenerateCommand(opts),
		newVersionCommand(),
	)
	return cmd
}

func newScanCommand(root *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "scan <namespace>",
		Short: "Scan a namespace and print the classification table",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			namespace, err := resolveNamespace(root, args)
			if err != nil {
				return err
			}
			scanResult, err := runNamespaceScan(cmd.Context(), root, namespace)
			if err != nil {
				return err
			}
			return printScanTable(cmd.OutOrStdout(), scanResult)
		},
	}
}

func newExportCommand(root *rootOptions) *cobra.Command {
	var outDir string
	cmd := &cobra.Command{
		Use:   "export <namespace> -o <dir>",
		Short: "Export a namespace scan as a ZIP archive",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if root.SecretHandling == "include" {
				warnSecretsIncluded(cmd)
			}
			namespace, err := resolveNamespace(root, args)
			if err != nil {
				return err
			}
			scanResult, err := runNamespaceScan(cmd.Context(), root, namespace)
			if err != nil {
				return err
			}
			fullPath, summary, err := archive.WriteScanArchive(scanResult, outDir)
			if err != nil {
				if errors.Is(err, archive.ErrNoExportableResources) {
					return fmt.Errorf("no exportable resources are available for namespace %s", namespace)
				}
				return err
			}
			if root.Quiet {
				fmt.Fprintln(cmd.OutOrStdout(), fullPath)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s (%d manifests, %d warnings)\n", fullPath, summary.ManifestCount, summary.WarningCount)
			return nil
		},
	}
	cmd.Flags().StringVarP(&outDir, "output", "o", ".", "Output directory for the ZIP archive")
	return cmd
}

func newScrubCommand(root *rootOptions) *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "scrub",
		Short: "Scrub a single YAML resource from file or stdin",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScrub(cmd, root, file)
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to the input resource YAML file")
	return cmd
}

func newGenerateCommand(root *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate GitOps manifests",
	}
	cmd.AddCommand(newGenerateArgoCDCommand(root))
	return cmd
}

func newGenerateArgoCDCommand(root *rootOptions) *cobra.Command {
	var (
		repoURL              string
		revision             string
		sourcePath           string
		applicationName      string
		projectName          string
		argoNamespace        string
		destinationServer    string
		destinationNamespace string
		syncMode             string
		prune                bool
		selfHeal             bool
		createNamespace      bool
	)

	cmd := &cobra.Command{
		Use:   "argocd <namespace> --repo-url ... --revision ... --path ...",
		Short: "Generate Argo CD Application YAML",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			namespace, err := resolveNamespace(root, args)
			if err != nil {
				return err
			}
			scanResult, err := runNamespaceScan(cmd.Context(), root, namespace)
			if err != nil {
				return err
			}
			form := argocd.CreateDefaultForm(scanResult)
			form.RepositoryURL = repoURL
			form.Revision = revision
			form.SourcePath = sourcePath
			form.Prune = prune
			form.SelfHeal = selfHeal
			form.CreateNamespace = createNamespace
			if applicationName != "" {
				form.ApplicationName = applicationName
			}
			if projectName != "" {
				form.ProjectName = projectName
			}
			if argoNamespace != "" {
				form.ArgoNamespace = argoNamespace
			}
			if destinationServer != "" {
				form.DestinationServer = destinationServer
			}
			if destinationNamespace != "" {
				form.DestinationNamespace = destinationNamespace
			}
			if syncMode != "" {
				form.SyncMode = argocd.SyncMode(syncMode)
			}
			if errs := argocd.ValidateForm(form); len(errs) > 0 {
				return fmt.Errorf("invalid Argo CD definition input: %s", joinValidationErrors(errs))
			}
			result, err := argocd.GenerateDefinition(form, scanResult)
			if err != nil {
				return err
			}
			_, err = io.WriteString(cmd.OutOrStdout(), result.YAML)
			return err
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&repoURL, "repo-url", "", "Git repository URL")
	flags.StringVar(&revision, "revision", "", "Git revision")
	flags.StringVar(&sourcePath, "path", "", "Git path for the sanitized manifests")
	flags.StringVar(&applicationName, "application-name", "", "Argo CD Application name override")
	flags.StringVar(&projectName, "project", "", "Argo CD project override")
	flags.StringVar(&argoNamespace, "argo-namespace", "", "Argo CD control-plane namespace override")
	flags.StringVar(&destinationServer, "destination-server", "", "Destination cluster URL override")
	flags.StringVar(&destinationNamespace, "destination-namespace", "", "Destination namespace override")
	flags.StringVar(&syncMode, "sync-mode", "", "Sync mode: manual or automated")
	flags.BoolVar(&prune, "prune", true, "Enable automated prune when sync mode is automated")
	flags.BoolVar(&selfHeal, "self-heal", true, "Enable automated self-heal when sync mode is automated")
	flags.BoolVar(&createNamespace, "create-namespace", false, "Add CreateNamespace=true sync option")
	_ = cmd.MarkFlagRequired("repo-url")
	_ = cmd.MarkFlagRequired("revision")
	_ = cmd.MarkFlagRequired("path")
	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "scrubctl %s (commit %s, built %s)\n", version, commit, buildDate)
		},
	}
}

func runNamespaceScan(ctx context.Context, root *rootOptions, namespace string) (types.NamespaceScan, error) {
	includeTypes, err := selectedResourceTypes(root.IncludeKinds, root.ExcludeKinds)
	if err != nil {
		return types.NamespaceScan{}, err
	}
	return scan.Run(ctx, scan.Options{
		Kubeconfig:     root.Kubeconfig,
		Context:        root.Context,
		Namespace:      namespace,
		SecretHandling: root.SecretHandling,
		IncludeTypes:   includeTypes,
	})
}

func warnSecretsIncluded(cmd *cobra.Command) {
	fmt.Fprintln(cmd.ErrOrStderr(), "Warning: --secret-handling=include — Secret values will appear in output. Ensure output is not logged or stored in plaintext.")
}

func runScrub(cmd *cobra.Command, root *rootOptions, file string) error {
	if root.SecretHandling == "include" {
		warnSecretsIncluded(cmd)
	}
	resource, err := readResource(file, cmd.InOrStdin())
	if err != nil {
		return err
	}
	classificationResult := classify.ClassifyCuratedResource(resource)
	if classificationResult.Classification == types.ClassificationExclude {
		return fmt.Errorf("%s/%s: %s", classificationResult.Kind, classificationResult.Name, classificationResult.Reason)
	}
	sanitizedResource := sanitize.SanitizeResource(resource, classificationResult.Classification, root.SecretHandling)
	if sanitizedResource == nil {
		return fmt.Errorf("%s/%s omitted by secret handling", classificationResult.Kind, classificationResult.Name)
	}
	_, err = io.WriteString(cmd.OutOrStdout(), sanitize.SerializeResource(sanitizedResource))
	return err
}

func resolveNamespace(root *rootOptions, args []string) (string, error) {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		return strings.TrimSpace(args[0]), nil
	}
	return scan.ResolveNamespace(root.Kubeconfig, root.Context, strings.TrimSpace(root.Namespace))
}

func selectedResourceTypes(includeValue, excludeValue string) ([]resources.ResourceTypeOption, error) {
	keys := map[string]resources.ResourceTypeOption{}
	for _, key := range resources.DefaultResourceTypeKeys() {
		if option, ok := resources.FindByKindName(key); ok {
			keys[option.Key] = option
		}
	}
	if strings.TrimSpace(includeValue) != "" {
		keys = map[string]resources.ResourceTypeOption{}
		for _, raw := range splitCSV(includeValue) {
			option, ok := resources.FindByKindName(raw)
			if !ok {
				return nil, fmt.Errorf("unknown curated resource kind: %s", raw)
			}
			keys[option.Key] = option
		}
	}
	for _, raw := range splitCSV(excludeValue) {
		option, ok := resources.FindByKindName(raw)
		if !ok {
			return nil, fmt.Errorf("unknown curated resource kind: %s", raw)
		}
		delete(keys, option.Key)
	}

	out := make([]resources.ResourceTypeOption, 0, len(keys))
	for _, option := range resources.ResourceTypeOptions {
		if selected, ok := keys[option.Key]; ok {
			out = append(out, selected)
		}
	}
	return out, nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func stdinHasData(reader io.Reader) bool {
	file, ok := reader.(*os.File)
	if !ok {
		return true
	}
	info, err := file.Stat()
	if err != nil {
		return true
	}
	return (info.Mode() & os.ModeCharDevice) == 0
}

func printScanTable(out io.Writer, scanResult types.NamespaceScan) error {
	writer := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "KIND\tNAME\tCLASSIFICATION\tREASON"); err != nil {
		return err
	}
	for _, resource := range scanResult.Status.ResourceDetails {
		if _, err := fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", resource.Kind, resource.Name, resource.Classification, resource.Reason); err != nil {
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "\n%d total | %d included | %d cleanup | %d review | %d excluded\n",
		scanResult.Status.ResourceSummary.Total,
		scanResult.Status.ResourceSummary.Included,
		scanResult.Status.ResourceSummary.IncludedWithCleanup,
		scanResult.Status.ResourceSummary.NeedsReview,
		scanResult.Status.ResourceSummary.Excluded,
	)
	return err
}

func readResource(file string, stdin io.Reader) (types.ResourceObject, error) {
	var data []byte
	var err error
	if strings.TrimSpace(file) != "" {
		data, err = os.ReadFile(filepath.Clean(file))
	} else {
		data, err = io.ReadAll(stdin)
	}
	if err != nil {
		return nil, err
	}
	var resource types.ResourceObject
	if err := yaml.Unmarshal(data, &resource); err != nil {
		return nil, err
	}
	return resource, nil
}

func joinValidationErrors(errs argocd.ValidationErrors) string {
	keys := make([]string, 0, len(errs))
	for field := range errs {
		keys = append(keys, field)
	}
	sort.Strings(keys)
	values := make([]string, 0, len(errs))
	for _, field := range keys {
		values = append(values, field+": "+errs[field])
	}
	return strings.Join(values, ", ")
}

func applyConfigFile(cmd *cobra.Command, opts *rootOptions) error {
	if opts.ConfigPath == "" {
		return nil
	}
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}
	if !cmd.Flags().Changed("include-kinds") && len(cfg.IncludeKinds) > 0 {
		opts.IncludeKinds = cfg.IncludeKindsCSV()
	}
	if !cmd.Flags().Changed("exclude-kinds") && len(cfg.ExcludeKinds) > 0 {
		opts.ExcludeKinds = cfg.ExcludeKindsCSV()
	}
	return nil
}
