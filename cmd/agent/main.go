package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/RobinHAEVG/haevg-agent/agents"
	"github.com/RobinHAEVG/haevg-agent/configuration"
	"github.com/RobinHAEVG/haevg-agent/llm"
	"github.com/RobinHAEVG/haevg-agent/mcp"
	"github.com/RobinHAEVG/haevg-agent/skills"
	"github.com/RobinHAEVG/haevg-agent/tools"

	"github.com/spf13/cobra"
)

// Auf Windows ist es %APPDATA%/haevg-agent/
var configDir = configuration.ConfigDir()

// ---------------------------------------------------------------------------
// Global flags (persistent across all commands)
// ---------------------------------------------------------------------------

var (
	skillName string // --skill <path-to-skill.md>
	verbose   bool   // --verbose
)

// ---------------------------------------------------------------------------
// Root Command
// ---------------------------------------------------------------------------

var rootCmd = &cobra.Command{
	Use:   "haevg-agent",
	Short: "HÄVG Agent - flexible CLI für KI-gestützte Aufgaben",
	Long: `haevg-agent startet einen KI-Agenten mit einem definierten Skill-Set
und löst einmalig eine Aufgabe (One-off). Der Agent kann dabei Tools via
lokalen MCP-Servern nutzen.`,
}

// ---------------------------------------------------------------------------
// Command: run
// Startet den Agenten mit einem Task (neues Ergebnis)
// ---------------------------------------------------------------------------

var (
	runOutputFile string // --output
	runMaxSteps   int    // --max-steps
)

var runCmd = &cobra.Command{
	Use:   "run [task]",
	Short: "Startet den Agenten mit einem neuen Task",
	Long: `Startet den Agenten einmalig für einen neuen Task.
Das Ergebnis wird in die angegebene Output-Datei geschrieben.

Beispiel:
  haevg-agent run --skill impl-planner --output result.md "Plane Feature X"`,
	Args: cobra.ExactArgs(1), // genau 1 positionaler Arg: der Task
	RunE: func(cmd *cobra.Command, args []string) error {
		instructions := args[0]

		if verbose {
			fmt.Printf("[verbose] Skill:      %s\n", skillName)
			fmt.Printf("[verbose] Output:     %s\n", runOutputFile)
			fmt.Printf("[verbose] Max Steps:  %d\n", runMaxSteps)
		}

		logLevel := slog.LevelInfo
		if verbose {
			logLevel = slog.LevelDebug
		}
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

		// Sicherstellen, dass Konfigurationsverzeichnis existiert
		_ = os.MkdirAll(configDir, 0600)

		// Arbeitsverzeichnis ermitteln
		workdir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("Konnte Arbeitsverzeichnis nicht ermitteln: %w", err)
		}

		// Konfiguration laden
		config, err := configuration.Load(configDir + "/config.yaml")
		if err != nil {
			return fmt.Errorf("Konnte Konfiguration nicht laden: %w", err)
		}

		// Skill laden
		skill, err := skills.Parse(skillName)
		if err != nil {
			return fmt.Errorf("Konnte Skill nicht laden: %w", err)
		}

		fmt.Printf("▶ Agent gestartet | Skill: %s | Task: %s\n", skillName, instructions)
		fmt.Printf("▶ Ergebnis wird geschrieben nach: %s\n", runOutputFile)

		client := llm.NewClient(config, &http.Client{Timeout: config.LLM.Timeout})

		serverR, clientW := io.Pipe()
		clientR, serverW := io.Pipe()

		store := tools.NewStore(config, workdir, logger, verbose)
		srv := mcp.NewServer()
		tools.RegisterAll(srv, store)

		go func() {
			if err := srv.Serve(serverR, serverW); err != nil && err != io.ErrClosedPipe {
				fmt.Printf("MCP server gestoppt: %v", err)
			}
			serverW.Close() //nolint:errcheck
		}()

		fmt.Println("MCP server gestartet (in-process).")

		mcpClient := mcp.NewClient(clientR, clientW)
		if err := mcpClient.Initialize(); err != nil {
			return fmt.Errorf("MCP handshake fehlgeschlagen: %w", err)
		}

		fmt.Println("MCP handshake abgeschlossen.")

		// Agent aufsetzen und ausführen
		agent := &agents.Agent{
			Config:      config,
			LLMClient:   client,
			LoadedSkill: skill,
			MCPClient:   mcpClient,
			Verbose:     verbose,
			Logger:      logger,
		}
		result, err := agent.Run(context.TODO(), "TODO")
		if err != nil {
			return fmt.Errorf("Konnte Agent nicht ausführen: %w", err)
		}

		// TODO: Ergebnis schreiben
		err = os.WriteFile(runOutputFile, []byte(result), 0644)
		if err != nil {
			return fmt.Errorf("Konnte Ergebnis nicht schreiben: %w", err)
		}

		fmt.Println("▶ Ausführung abgeschlossen")

		return nil
	},
}

// ---------------------------------------------------------------------------
// Command: refine
// Verfeinert ein bestehendes Ergebnis
// ---------------------------------------------------------------------------

var (
	refineInputFile   string // --input  (bestehende Antwort)
	refineOutputFile  string // --output (Standard: überschreibt input)
	refineKeepHistory bool   // --keep-history (versioniert speichern)
	refineMaxSteps    int    // --max-steps
)

var refineCmd = &cobra.Command{
	Use:   "refine [instruction]",
	Short: "Verfeinert ein bestehendes Ergebnis",
	Long: `Lädt ein bestehendes Ergebnis und verfeinert es anhand einer Anweisung.
Standardmäßig wird die Input-Datei überschrieben. Mit --keep-history wird
eine neue versionierte Datei angelegt (z.B. result.v2.md).

Beispiele:
  haevg-agent refine --skill impl-planner --input result.md "Mache Schritt 3 detaillierter"
  haevg-agent refine --skill impl-planner --input result.md --keep-history "Füge Tests hinzu"`,
	Args: cobra.ExactArgs(1), // genau 1 positionaler Arg: die Verfeinerungsanweisung
	RunE: func(cmd *cobra.Command, args []string) error {
		instructions := args[0]

		// Output-Datei: Standard = Input-Datei überschreiben
		if refineOutputFile == "" {
			refineOutputFile = refineInputFile
		}

		if verbose {
			fmt.Printf("[verbose] Skill:        %s\n", skillName)
			fmt.Printf("[verbose] Input:        %s\n", refineInputFile)
			fmt.Printf("[verbose] Output:       %s\n", refineOutputFile)
			fmt.Printf("[verbose] Keep History: %v\n", refineKeepHistory)
			fmt.Printf("[verbose] Max Steps:    %d\n", refineMaxSteps)
		}

		// Sicherstellen, dass Konfigurationsverzeichnis existiert
		_ = os.MkdirAll(configDir, 0600)

		// Konfiguration laden
		config, err := configuration.Load(configDir + "/config.yaml")
		if err != nil {
			return fmt.Errorf("Konnte Konfiguration nicht laden: %w", err)
		}

		// Skill laden
		skill, err := skills.Parse(skillName)
		if err != nil {
			return fmt.Errorf("Konnte Skill nicht laden: %w", err)
		}

		// Bestehendes Ergebnis laden
		priorResult, err := os.ReadFile(refineInputFile)
		if err != nil {
			return fmt.Errorf("Konnte bestehendes Ergebnis nicht laden: %w", err)
		}

		// Agentic Loop mit Prior Result starten
		client := llm.NewClient(config, &http.Client{Timeout: config.LLM.Timeout})
		agent := &agents.Agent{
			Config:      config,
			LLMClient:   client, // TODO: LLM-Client initialisieren
			LoadedSkill: skill,
		}

		userPrompt := fmt.Sprintf("Verfeinere bestehendes Ergebnis:\n%s\nAnweisung: %s", string(priorResult), instructions)
		result, err := agent.Run(context.TODO(), userPrompt)
		if err != nil {
			return fmt.Errorf("Konnte Agent nicht ausführen: %w", err)
		}
		_ = result // TODO: Ergebnis weiterverarbeiten

		// TODO: Ergebnis schreiben (ggf. versioniert)
		// err = output.Write(refineOutputFile, result, refineKeepHistory)

		fmt.Printf("♻ Refinement gestartet | Skill: %s | Input: %s\n", skillName, refineInputFile)
		fmt.Printf("♻ Anweisung: %s\n", instructions)
		fmt.Printf("♻ Ergebnis wird geschrieben nach: %s\n", refineOutputFile)
		return nil
	},
}

// ---------------------------------------------------------------------------
// Command: skills
// Hilfsbefehle rund um verfügbare Skills
// ---------------------------------------------------------------------------

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Verwaltet verfügbare Skills",
}

// skills list
var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "Listet alle verfügbaren Skill-Dateien auf",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Skills-Verzeichnis aus Config lesen
		// skills, err := skill.List(configSkillsDir)

		fmt.Println("Verfügbare Skills:")
		fmt.Println("  impl-planner.md      - Implementationsplaner")
		fmt.Println("  pipeline-debugger.md - Pipeline Fehlersucher")
		// TODO: dynamisch befüllen
		return nil
	},
}

// skills show
var skillsShowCmd = &cobra.Command{
	Use:   "show [skill-file]",
	Short: "Zeigt den Inhalt einer Skill-Datei an",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("skill-Datei nicht lesbar: %w", err)
		}

		fmt.Printf("=== Skill: %s ===\n\n%s\n", path, string(content))
		return nil
	},
}

// ---------------------------------------------------------------------------
// Command: mcp
// Hilfsbefehle rund um MCP-Server
// ---------------------------------------------------------------------------

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP-Server Verwaltung und Diagnose",
}

// mcp status
var mcpStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Prüft den Status aller konfigurierten MCP-Server",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Alle konfigurierten MCP-Server anpingen
		// for _, server := range config.MCPServers { server.Ping() }

		fmt.Println("MCP-Server Status:")
		fmt.Println("  [✓] filesystem   – erreichbar")
		fmt.Println("  [✓] azure-devops – erreichbar")
		// TODO: dynamisch befüllen
		return nil
	},
}

// mcp tools
var (
	mcpToolsServer string // --server
)

var mcpToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Listet verfügbare Tools eines MCP-Servers auf",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Tools vom MCP-Server abrufen
		// tools, err := mcp.ListTools(mcpToolsServer)

		fmt.Printf("Tools von Server '%s':\n", mcpToolsServer)
		fmt.Println("  read_file, write_file, list_dir, ...")
		// TODO: dynamisch befüllen
		return nil
	},
}

// ---------------------------------------------------------------------------
// Command: config
// Konfigurationsverwaltung
// ---------------------------------------------------------------------------

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Konfiguration anzeigen und setzen",
}

// config show
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Zeigt die aktuelle Konfiguration an",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Config aus Datei laden und ausgeben
		// cfg, err := config.Load()

		fmt.Println("Aktuelle Konfiguration:")
		fmt.Println("  llm.provider:    azure-openai")
		fmt.Println("  llm.model:       gpt-4o")
		fmt.Println("  skills.dir:      ./skills")
		fmt.Println("  mcp.filesystem:  localhost:5010")
		fmt.Println("  mcp.azdevops:    localhost:5011")
		// TODO: dynamisch befüllen
		return nil
	},
}

// config set
var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Setzt einen Konfigurationswert",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		// TODO: Config persistieren
		// err := config.Set(key, value)

		fmt.Printf("✓ Konfiguration gesetzt: %s = %s\n", key, value)
		return nil
	},
}

// ---------------------------------------------------------------------------
// Initialisierung: Flags binden & Command-Baum aufbauen
// ---------------------------------------------------------------------------

func init() {
	// --- Persistent Flags (für alle Commands verfügbar) ---
	rootCmd.PersistentFlags().StringVar(&skillName, "skill", "", "Pfad zur Skill-Datei (z.B. skills/impl-planner.md)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Ausführliche Ausgabe aktivieren")

	// --- run Flags ---
	runCmd.Flags().StringVarP(&runOutputFile, "output", "o", "result.md", "Ausgabedatei für das Ergebnis")
	runCmd.Flags().IntVar(&runMaxSteps, "max-steps", 20, "Maximale Anzahl an Agentic-Loop-Schritten")
	_ = runCmd.MarkFlagRequired("skill") // --skill ist Pflicht für run

	// --- refine Flags ---
	refineCmd.Flags().StringVarP(&refineInputFile, "input", "i", "", "Bestehende Ergebnis-Datei zur Verfeinerung")
	refineCmd.Flags().StringVarP(&refineOutputFile, "output", "o", "", "Ausgabedatei (Standard: überschreibt --input)")
	refineCmd.Flags().BoolVar(&refineKeepHistory, "keep-history", false, "Versionierte Ausgabedatei anlegen statt überschreiben")
	refineCmd.Flags().IntVar(&refineMaxSteps, "max-steps", 20, "Maximale Anzahl an Agentic-Loop-Schritten")
	_ = refineCmd.MarkFlagRequired("skill") // --skill ist Pflicht für refine
	_ = refineCmd.MarkFlagRequired("input") // --input ist Pflicht für refine

	// --- mcp tools Flags ---
	mcpToolsCmd.Flags().StringVarP(&mcpToolsServer, "server", "s", "", "Name des MCP-Servers")
	_ = mcpToolsCmd.MarkFlagRequired("server")

	// --- Command-Baum ---
	skillsCmd.AddCommand(skillsListCmd, skillsShowCmd)
	mcpCmd.AddCommand(mcpStatusCmd, mcpToolsCmd)
	configCmd.AddCommand(configShowCmd, configSetCmd)

	rootCmd.AddCommand(runCmd, refineCmd, skillsCmd, mcpCmd, configCmd)
}

// ---------------------------------------------------------------------------
// Entry Point
// ---------------------------------------------------------------------------

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
