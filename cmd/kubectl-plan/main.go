package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/samaasi/kubectl-plan/internal/analysis"
	"github.com/samaasi/kubectl-plan/internal/k8s"
	"github.com/samaasi/kubectl-plan/internal/manifest"
	"github.com/samaasi/kubectl-plan/internal/output"
	"github.com/samaasi/kubectl-plan/internal/prometheus"
	"github.com/spf13/cobra"
)

var (
	namespace     string
	kubeContext   string
	outputFormat  string
	asciiOnly     bool
	noColor       bool
	allNamespaces bool

	prometheusURL string
	lookback      string

	replicas int
	filename string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "kubectl-plan",
		Short: "Terraform has plan. Kubernetes should too.",
		Long: `kubectl-plan provides operational decision support for Kubernetes changes.
It analyzes dependencies via Kubernetes topology (and optionally Prometheus) to calculate the risk score and blast radius before you perform an action.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if noColor || os.Getenv("NO_COLOR") != "" {
				color.NoColor = true
			}
		},
	}

	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Target namespace (defaults to current context)")
	rootCmd.PersistentFlags().StringVar(&kubeContext, "context", "", "Override kubeconfig context")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "terminal", "Output format: terminal | json")
	rootCmd.PersistentFlags().BoolVar(&asciiOnly, "ascii", false, "Disable unicode box drawing characters")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable terminal color output")
	rootCmd.PersistentFlags().BoolVar(&allNamespaces, "all-namespaces", false, "Include cross-namespace dependency scanning")
	rootCmd.PersistentFlags().StringVar(&prometheusURL, "prometheus-url", "", "Prometheus server URL (e.g. http://localhost:9090). If empty, auto-discovery is used")
	rootCmd.PersistentFlags().StringVar(&lookback, "lookback", "24h", "Time window for Prometheus traffic queries")

	scaleCmd := &cobra.Command{
		Use:   "scale [KIND] [NAME] --replicas=N",
		Short: "Analyze the risk of scaling a resource",
		Example: `  kubectl plan scale deployment payment-api --replicas=0
  kubectl plan scale deployment/payment-api --replicas=2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyze(args, fmt.Sprintf("scale --replicas=%d", replicas))
		},
	}
	scaleCmd.Flags().IntVar(&replicas, "replicas", -1, "Target replicas count")
	_ = scaleCmd.MarkFlagRequired("replicas")

	restartCmd := &cobra.Command{
		Use:   "restart [KIND] [NAME]",
		Short: "Analyze the risk of restarting a resource",
		Example: `  kubectl plan restart deployment payment-api
  kubectl plan restart statefulset/payment-db`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyze(args, "rollout restart")
		},
	}

	whyCmd := &cobra.Command{
		Use:   "why [KIND] [NAME]",
		Short: "Explain the risk score breakdown for a resource",
		Example: `  kubectl plan why deployment payment-api
  kubectl plan why deployment/payment-api`,
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, name, err := parseArgs(args)
			if err != nil {
				return err
			}
			client, err := k8s.NewClient(kubeContext, namespace)
			if err != nil {
				return fmt.Errorf("failed to create k8s client: %w", err)
			}
			promClient := prometheus.NewClient(prometheusURL, lookback)
			if prometheusURL == "" {
				_, _ = promClient.Discover(context.Background(), client)
			}
			res, err := analysis.NewEngine(client, promClient).Analyze(context.Background(), "why", kind, name)
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}
			renderer := output.NewRenderer(outputFormat, os.Stdout, asciiOnly)
			if outputFormat == "json" {
				return renderer.Render(res)
			}
			return renderer.RenderWhy(res)
		},
	}

	manifestCmd := &cobra.Command{
		Use:     "manifest -f [FILENAME]",
		Short:   "Analyze the risk of applying a manifest file",
		Example: `  kubectl plan manifest -f deployment.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if filename == "" {
				return fmt.Errorf("must specify a filename with -f or --filename")
			}
			return runManifest(filename)
		},
	}
	manifestCmd.Flags().StringVarP(&filename, "filename", "f", "", "Path to the Kubernetes YAML manifest")
	_ = manifestCmd.MarkFlagRequired("filename")

	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Probes data sources and scores readiness",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor()
		},
	}

	rootCmd.AddCommand(scaleCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(whyCmd)
	rootCmd.AddCommand(manifestCmd)
	rootCmd.AddCommand(doctorCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runAnalyze(args []string, action string) error {
	kind, name, err := parseArgs(args)
	if err != nil {
		return err
	}
	client, err := k8s.NewClient(kubeContext, namespace)
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	promClient := prometheus.NewClient(prometheusURL, lookback)
	if prometheusURL == "" {
		_, _ = promClient.Discover(context.Background(), client)
	}

	res, err := analysis.NewEngine(client, promClient).Analyze(context.Background(), action, kind, name)
	if err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}
	return output.NewRenderer(outputFormat, os.Stdout, asciiOnly).Render(res)
}

func runManifest(file string) error {
	resources, err := manifest.ParseFile(file)
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	if len(resources) == 0 {
		return fmt.Errorf("no valid kubernetes resources found in %s", file)
	}

	client, err := k8s.NewClient(kubeContext, namespace)
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}

	promClient := prometheus.NewClient(prometheusURL, lookback)
	if prometheusURL == "" {
		_, _ = promClient.Discover(context.Background(), client)
	}

	engine := analysis.NewEngine(client, promClient)
	renderer := output.NewRenderer(outputFormat, os.Stdout, asciiOnly)

	for _, res := range resources {
		// Use namespace from manifest if specified, otherwise fallback to CLI/context namespace
		ns := namespace
		if res.Namespace != "" {
			ns = res.Namespace
		}

		// Recreate client for specific namespace if it differs
		targetClient := client
		if ns != client.Namespace {
			targetClient, _ = k8s.NewClient(kubeContext, ns)
			engine = analysis.NewEngine(targetClient, promClient)
		}

		result, err := engine.Analyze(context.Background(), "apply", res.Kind, res.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to analyze %s/%s: %v\n", res.Kind, res.Name, err)
			continue
		}

		if err := renderer.Render(result); err != nil {
			return err
		}
		// Add some spacing between resources if not JSON
		if outputFormat != "json" {
			fmt.Println()
		}
	}

	return nil
}

func runDoctor() error {
	client, err := k8s.NewClient(kubeContext, namespace)
	apiReachable := err == nil

	nsName := "default"
	if client != nil {
		nsName = client.Namespace
	}

	promClient := prometheus.NewClient(prometheusURL, lookback)
	promURL := prometheusURL
	if apiReachable && promURL == "" {
		if discoveredURL, err := promClient.Discover(context.Background(), client); err == nil && discoveredURL != "" {
			promURL = discoveredURL
		}
	}

	docResult := &output.DoctorResult{
		Namespace:           nsName,
		K8sAPIReachable:     apiReachable,
		PrometheusReachable: promURL != "",
		PrometheusURL:       promURL,
		EstimatedConfidence: 0.65,
	}

	if outputFormat == "json" {
		return writeDoctorJSON(docResult)
	}
	return output.NewRenderer(outputFormat, os.Stdout, asciiOnly).RenderDoctor(docResult)
}

func writeDoctorJSON(res *output.DoctorResult) error {
	fmt.Fprintf(os.Stdout, "{\n  \"namespace\": %q,\n  \"k8sAPIReachable\": %v,\n  \"estimatedConfidence\": %.2f\n}\n",
		res.Namespace, res.K8sAPIReachable, res.EstimatedConfidence)
	return nil
}

func parseArgs(args []string) (string, string, error) {
	if len(args) == 0 {
		return "", "", fmt.Errorf("requires kind and name of the resource (e.g. deployment payment-api)")
	}

	if len(args) == 1 {
		parts := strings.Split(args[0], "/")
		if len(parts) == 2 {
			return parts[0], parts[1], nil
		}
		return "", "", fmt.Errorf("invalid resource format: use KIND/NAME or KIND NAME (e.g. deployment/payment-api)")
	}

	return args[0], args[1], nil
}
