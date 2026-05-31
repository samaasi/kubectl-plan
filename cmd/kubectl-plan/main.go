package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/samaasi/kubectl-plan/internal/analysis"
	"github.com/samaasi/kubectl-plan/internal/k8s"
	"github.com/samaasi/kubectl-plan/internal/output"
)

var (
	namespace     string
	kubeContext   string
	outputFormat  string
	asciiOnly     bool
	noColor       bool
	allNamespaces bool

	replicas int
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

	scaleCmd := &cobra.Command{
		Use:   "scale [KIND] [NAME] --replicas=N",
		Short: "Analyze the risk of scaling a resource",
		Example: `  kubectl plan scale deployment payment-api --replicas=0
  kubectl plan scale deployment/payment-api --replicas=2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, name, err := parseArgs(args)
			if err != nil {
				return err
			}

			client, err := k8s.NewClient(kubeContext, namespace)
			if err != nil {
				return fmt.Errorf("failed to create k8s client: %w", err)
			}

			engine := analysis.NewEngine(client)
			actionStr := fmt.Sprintf("scale --replicas=%d", replicas)
			res, err := engine.Analyze(context.Background(), actionStr, kind, name)
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}

			renderer := output.NewRenderer(outputFormat, os.Stdout, asciiOnly)
			return renderer.Render(res)
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
			kind, name, err := parseArgs(args)
			if err != nil {
				return err
			}

			client, err := k8s.NewClient(kubeContext, namespace)
			if err != nil {
				return fmt.Errorf("failed to create k8s client: %w", err)
			}

			engine := analysis.NewEngine(client)
			res, err := engine.Analyze(context.Background(), "rollout restart", kind, name)
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}

			renderer := output.NewRenderer(outputFormat, os.Stdout, asciiOnly)
			return renderer.Render(res)
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

			engine := analysis.NewEngine(client)
			res, err := engine.Analyze(context.Background(), "why", kind, name)
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

	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Probes data sources and scores readiness",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := k8s.NewClient(kubeContext, namespace)
			apiReachable := true
			if err != nil {
				apiReachable = false
			}

			nsName := "default"
			if client != nil {
				nsName = client.Namespace
			}

			docResult := &output.DoctorResult{
				Namespace:           nsName,
				K8sAPIReachable:     apiReachable,
				EstimatedConfidence: 0.65, // Topology only in Week 1
			}

			renderer := output.NewRenderer(outputFormat, os.Stdout, asciiOnly)
			if outputFormat == "json" {
				importJSON := true
				_ = importJSON // use local encoder
				importJSONEncoder := os.Stdout
				importJSONEncoder.Write([]byte("{\n"))
				fmt.Fprintf(importJSONEncoder, "  \"namespace\": \"%s\",\n", docResult.Namespace)
				fmt.Fprintf(importJSONEncoder, "  \"k8sAPIReachable\": %v,\n", docResult.K8sAPIReachable)
				fmt.Fprintf(importJSONEncoder, "  \"estimatedConfidence\": %.2f\n", docResult.EstimatedConfidence)
				importJSONEncoder.Write([]byte("}\n"))
				return nil
			}
			return renderer.RenderDoctor(docResult)
		},
	}

	rootCmd.AddCommand(scaleCmd)
	rootCmd.AddCommand(restartCmd)
	rootCmd.AddCommand(whyCmd)
	rootCmd.AddCommand(doctorCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
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
