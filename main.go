// main.go - Daily Task CLI
// A simple CLI for tracking daily tasks and notes

package main

// --- Imports ---
import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	// Third-party
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// --- Bubble Tea Progress Model (for followStartedTask) ---

type taskModel struct {
	progress      progress.Model
	task          *Task
	startTime     time.Time
	totalDuration time.Duration
}

type tickMsg struct{}

func (m taskModel) Init() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (m taskModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC || msg.String() == "q" {
			return m, tea.Quit
		}
	case tickMsg:
		elapsed := time.Since(m.startTime)
		percent := math.Min(1.0, float64(elapsed)/float64(m.totalDuration))
		m.progress.SetPercent(percent)
		return m, tea.Tick(time.Second, func(_ time.Time) tea.Msg {
			return tickMsg{}
		})
	}
	return m, nil
}

func (m taskModel) View() string {
	elapsed := time.Since(m.startTime)
	remaining := m.totalDuration - elapsed
	if remaining < 0 {
		remaining = 0
	}
	return fmt.Sprintf(
		"%s\n%s\nElapsed: %s\nRemaining: %s\n",
		m.task.Title,
		m.progress.ViewAs(elapsed.Seconds()/m.totalDuration.Seconds()),
		formatDuration(elapsed),
		formatDuration(remaining),
	)
}

// formatDuration formats a time.Duration for display
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}

// --- Types ---

// Task represents a single task entry
type Task struct {
	Title     string `yaml:"title"`
	Estimated int    `yaml:"estimated"`
	Actual    int    `yaml:"actual"`
	Status    string `yaml:"status"`
	StartedAt int64  `yaml:"started_at"`
}

type TaskData map[string][]Task

// NoteData stores notes per day
type NoteData map[string][]string

const maxDailyMinutes = 480

// --- Notes Logic ---

// getEditor returns the user's preferred editor or a sensible default
func getEditor() string {
	return "nano"
}

// editNoteForDay opens the note for a given day in the user's editor
func editNoteForDay(day string) error {
	data, err := loadNotes()
	if err != nil {
		return err
	}
	notes := data[day]
	tmpfile, err := ioutil.TempFile("", "daily_note_*.md")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())

	// Write current notes to temp file
	for _, note := range notes {
		tmpfile.WriteString(note + "\n")
	}
	tmpfile.Close()

	// Open editor
	cmd := exec.Command(getEditor(), tmpfile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	// Read back edited notes
	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return err
	}
	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	var newNotes []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			newNotes = append(newNotes, line)
		}
	}
	data[day] = newNotes
	return saveNotes(data)
}

// Parse date string or return today if empty
func parseNoteDayArg(args []string) string {
	if len(args) > 0 && args[0] != "" {
		return args[0]
	}
	return todayKey()
}

func getNoteFilePath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exePath)
	return filepath.Join(dir, "notes.yaml"), nil
}

func loadNotes() (NoteData, error) {
	filePath, err := getNoteFilePath()
	if err != nil {
		return nil, err
	}
	data := NoteData{}
	file, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NoteData{}, nil
		}
		return nil, err
	}
	err = yaml.Unmarshal(file, &data)
	return data, err
}

func saveNotes(data NoteData) error {
	filePath, err := getNoteFilePath()
	if err != nil {
		return err
	}
	file, err := yaml.Marshal(&data)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, file, 0644)
}

func addNoteForToday(note string) error {
	data, err := loadNotes()
	if err != nil {
		return err
	}
	today := todayKey()
	data[today] = append(data[today], note)
	return saveNotes(data)
}

func showNotesForToday() error {
	data, err := loadNotes()
	if err != nil {
		return err
	}
	today := todayKey()
	notes := data[today]
	if len(notes) == 0 {
		fmt.Println("No notes for today.")
		return nil
	}
	fmt.Printf("Notes for today (%s):\n", today)
	for i, note := range notes {
		fmt.Printf("%d. %s\n", i+1, note)
	}
	return nil
}

// --- Task Logic ---

func getTaskFilePath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exePath)
	return filepath.Join(dir, "tasks.yaml"), nil
}

func loadTasks() (TaskData, error) {
	filePath, err := getTaskFilePath()
	if err != nil {
		return nil, err
	}

	data := TaskData{}
	file, err := os.ReadFile(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return TaskData{}, nil
		}
		return nil, err
	}
	err = yaml.Unmarshal(file, &data)
	return data, err
}

func saveTasks(data TaskData) error {
	filePath, err := getTaskFilePath()
	if err != nil {
		return err
	}
	file, err := yaml.Marshal(&data)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, file, 0644)
}

func promptWithCursor(label string, defaultVal string) (string, error) {
	prompt := promptui.Prompt{
		Label:   label,
		Default: defaultVal,
	}

	// Add special handling for 'q' key as an additional way to quit
	result, err := prompt.Run()
	if err == nil && result == "q" && defaultVal != "q" {
		return "", fmt.Errorf("q")
	}
	return result, err
}

func todayKey() string {
	return time.Now().Format("2006-01-02")
}

func yesterdayKey() string {
	return time.Now().AddDate(0, 0, -1).Format("2006-01-02")
}

func showYesterdayTasks() error {
	data, err := loadTasks()
	if err != nil {
		return err
	}

	yesterday := yesterdayKey()
	tasks := data[yesterday]

	if len(tasks) == 0 {
		fmt.Println("No tasks found for yesterday.")
		return nil
	}

	fmt.Printf("Tasks from yesterday (%s):\n\n", yesterday)

	totalEstimated := 0
	totalActual := 0

	for i, task := range tasks {
		fmt.Printf("[%d] %s\n", i+1, task.Title)
		fmt.Printf("    Status: %s\n", task.Status)
		fmt.Printf("    Estimated: %d minutes\n", task.Estimated)
		fmt.Printf("    Actual: %d minutes\n", task.Actual)

		if i < len(tasks)-1 {
			fmt.Println() // Extra line between tasks
		}

		totalEstimated += task.Estimated
		totalActual += task.Actual
	}

	fmt.Printf("\nSummary: %d tasks, %d/%d minutes (%.1f%%)\n",
		len(tasks),
		totalActual,
		totalEstimated,
		float64(totalActual)/float64(totalEstimated)*100)

	return nil
}

func addTaskInteractive(tommorow bool) error {
	data, err := loadTasks()
	if err != nil {
		return err
	}

	today := todayKey()
	if tommorow {
		today = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	}

	title, err := promptWithCursor("Task Title", "")
	if err != nil {
		if err.Error() == "interrupt" || err.Error() == "q" {
			return nil
		}
		return err
	}
	estPrompt := promptui.Prompt{
		Label: "Estimated Minutes",
		Validate: func(input string) error {
			val, err := strconv.Atoi(input)
			if err != nil || val <= 0 {
				return fmt.Errorf("please enter a valid number of minutes")
			}
			return nil
		},
	}
	estInput, err := estPrompt.Run()
	if err != nil {
		if err.Error() == "interrupt" || err.Error() == "q" {
			return nil
		}
		return err
	}
	estimated, _ := strconv.Atoi(estInput)
	total := 0
	for _, t := range data[today] {
		total += t.Estimated
	}
	if total+estimated > maxDailyMinutes {
		fmt.Printf("total estimated time exceeds 8 hours")
	}
	task := Task{Title: title, Estimated: estimated, Status: "pending", StartedAt: 0}
	data[today] = append(data[today], task)
	return saveTasks(data)
}

func remainingMinutesToday(now time.Time) int {
	workStart := time.Date(now.Year(), now.Month(), now.Day(), 8, 30, 0, 0, now.Location())
	lunchStart := time.Date(now.Year(), now.Month(), now.Day(), 12, 30, 0, 0, now.Location())
	lunchEnd := time.Date(now.Year(), now.Month(), now.Day(), 13, 30, 0, 0, now.Location())
	workEnd := time.Date(now.Year(), now.Month(), now.Day(), 17, 30, 0, 0, now.Location())

	if now.Before(workStart) {
		return 480
	}
	if now.After(workEnd) {
		return 0
	}

	minutes := 0
	if now.Before(lunchStart) {
		minutes += int(lunchStart.Sub(now).Minutes())
		minutes += 240 // afternoon session (13:30–17:30)
	} else if now.Before(lunchEnd) {
		minutes += 240 // just afternoon session left
	} else if now.Before(workEnd) {
		minutes += int(workEnd.Sub(now).Minutes())
	}
	return minutes
}

func listTasksInteractive(tommorow bool) error {
	data, err := loadTasks()
	if err != nil {
		return err
	}
	today := todayKey()
	if tommorow {
		today = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	}
	tasks := data[today]
	if len(tasks) == 0 {
		fmt.Println("No tasks available.")
		return nil
	}
	totalActual := 0
	totalEst := 0
	remainingWork := 0
	achievedWork := 0
	for _, t := range tasks {
		totalActual += t.Actual
		totalEst += t.Estimated
		if t.Status == "done" {
			achievedWork += t.Estimated
		} else if t.Status != "done" && t.Status != "cancelled" {
			remainingTime := t.Estimated - t.Actual
			if remainingTime < 0 {
				remainingTime = 0
			}
			remainingWork += remainingTime
		}
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "→ {{ .Title | cyan }} ({{ .Status | yellow }}, est: {{ .Estimated }}min, act: {{ .Actual }}min)",
		Inactive: "  {{ .Title }} ({{ .Status | yellow }}, est: {{ .Estimated }}min, act: {{ .Actual }}min)",
		Selected: "✔ {{ .Title }}",
	}

	actualProgressPercent := float64(totalActual) / float64(maxDailyMinutes)
	estProgressPercent := float64(totalEst) / float64(maxDailyMinutes)
	achievedWorkPercent := float64(achievedWork) / float64(totalEst)
	actualProgressBar := progress.New(setColorGradient(actualProgressPercent, false))
	estProgressBar := progress.New(setColorGradient(estProgressPercent, true))
	achievedWorkProgressBar := progress.New(setColorGradient(achievedWorkPercent, false))
	actualBar := actualProgressBar.ViewAs(actualProgressPercent)
	achievedWorkBar := achievedWorkProgressBar.ViewAs(achievedWorkPercent)
	estBar := estProgressBar.ViewAs(estProgressPercent)
	minutesLeft := remainingMinutesToday(time.Now())

	ratio := float64(remainingWork)
	if minutesLeft > 0 {
		ratio /= float64(minutesLeft)
	} else {
		ratio = 1.0
	}

	availableProgressBar := progress.New(setColorGradient(ratio, true))
	availableBar := availableProgressBar.ViewAs(ratio)

	fmt.Printf("Daily Plan: %s [%d/%d min planned]\n\n", estBar, totalEst, maxDailyMinutes)
	if !tommorow {
		fmt.Printf("Daily Worked: %s [%d/%d min worked]\n\n", actualBar, totalActual, maxDailyMinutes)
		fmt.Printf("Daily Achieved: %s [%d/%d min achieved]\n\n", achievedWorkBar, achievedWork, totalEst)
		fmt.Printf("Remaining Work vs Time Left: %s [%d min left vs %d min to do]\n\n", availableBar, minutesLeft, remainingWork)
	}
	for {
		prompt := promptui.Select{Label: "View/Edit Tasks",
			Items:     tasks,
			Templates: templates,
			Size:      10,
			HideHelp:  true,
		}
		index, _, err := prompt.Run()
		if err != nil {
			if err.Error() == "interrupt" || err.Error() == "q" {
				return nil
			}
			return err
		}

		task := &tasks[index]
		title, err := promptWithCursor("Title", task.Title)
		if err != nil {
			if err.Error() == "interrupt" || err.Error() == "q" {
				return nil
			}
			return err
		}

		estStr, err := promptWithCursor("Estimated (minutes)", strconv.Itoa(task.Estimated))
		if err != nil {
			if err.Error() == "interrupt" || err.Error() == "q" {
				return nil
			}
			return err
		}

		actualStr, err := promptWithCursor("Actual (minutes)", strconv.Itoa(task.Actual))
		if err != nil {
			if err.Error() == "interrupt" || err.Error() == "q" {
				return nil
			}
			return err
		}

		estimated, _ := strconv.Atoi(estStr)
		actual, _ := strconv.Atoi(actualStr)

		statusPrompt := promptui.Select{
			Label:    "Set status",
			Items:    []string{"pending", "started", "done", "cancelled"},
			HideHelp: true,
		}
		_, status, err := statusPrompt.Run()
		if err != nil {
			if err.Error() == "interrupt" || err.Error() == "q" {
				return nil
			}
			return err
		}

		task.Title = title
		task.Estimated = estimated
		task.Actual = actual
		task.Status = status

		data[today] = tasks
		saveTasks(data)
	}
}

func setColorGradient(ratio float64, inverted bool) progress.Option {
	if inverted {
		if ratio >= 1.0 {
			return progress.WithSolidFill("#f53333") // red
		} else if ratio >= 0.9 {
			return progress.WithSolidFill("#f56a33") // dark orange
		} else if ratio >= 0.8 {
			return progress.WithSolidFill("#f58e33") // orange
		} else if ratio >= 0.7 {
			return progress.WithSolidFill("#f5ce33") // yellow
		} else if ratio >= 0.6 {
			return progress.WithSolidFill("#33f56d") // green
		}
		return progress.WithSolidFill("#03befc") // blue
	} else {
		if ratio >= 1.0 {
			return progress.WithSolidFill("#03befc") // blue
		} else if ratio >= 0.9 {
			return progress.WithSolidFill("#33f56d") // green
		} else if ratio >= 0.7 {
			return progress.WithSolidFill("#f5ce33") // yellow
		} else if ratio >= 0.6 {
			return progress.WithSolidFill("#f58e33") // orange
		} else if ratio >= 0.5 {
			return progress.WithSolidFill("#f56a33") // dark orange
		}
		return progress.WithSolidFill("#f53333") // red
	}
}

func updateStatus(index int, status string) error {
	data, err := loadTasks()
	if err != nil {
		return err
	}
	today := todayKey()
	tasks := data[today]
	if index < 0 || index >= len(tasks) {
		return fmt.Errorf("invalid task index")
	}
	t := &tasks[index]
	switch status {
	case "started":
		t.StartedAt = time.Now().Unix()
		t.Status = "started"
	case "done", "cancelled", "pending":
		if t.StartedAt != 0 {
			elapsed := int(time.Now().Unix()-t.StartedAt) / 60
			t.Actual += elapsed
			t.StartedAt = 0
		}
		t.Status = status
	default:
		t.Status = status
	}
	data[today] = tasks
	return saveTasks(data)
}

func startNextPendingTask() error {
	data, err := loadTasks()
	if err != nil {
		return err
	}
	today := todayKey()
	tasks := data[today]
	// Check if any task is already started
	for _, t := range tasks {
		if t.Status == "started" {
			fmt.Println("A task is already started. Please finish it before starting another one.")
			return nil
		}
	}
	for i, t := range tasks {
		if t.Status == "pending" {
			prompt := promptui.Select{
				Label:    fmt.Sprintf("Next Task: %s (%d min)", t.Title, t.Estimated),
				Items:    []string{"Start", "Skip"},
				HideHelp: true,
			}
			_, choice, err := prompt.Run()
			if err != nil {
				if err.Error() == "interrupt" || err.Error() == "q" {
					return nil
				}
				return err
			}
			if choice == "Start" {
				fmt.Printf("Starting '%s'...\n", t.Title)
				return updateStatus(i, "started")
			} else {
				continue
			}
		}
	}
	fmt.Println("No pending tasks to start.")
	return nil
}

func currentTask() error {
	data, err := loadTasks()
	if err != nil {
		return err
	}
	today := todayKey()
	tasks := data[today]
	for i, t := range tasks {
		if t.Status == "started" {
			elapsed := int(time.Now().Unix()-t.StartedAt) / 60
			clock := float64(elapsed) / float64(t.Estimated)
			clockProgressBar := progress.New(setColorGradient(clock, true))
			clockBar := clockProgressBar.ViewAs(clock)
			fmt.Printf("Task Clock: %s [%d/%d min used]\n\n", clockBar, elapsed, t.Estimated)
			fmt.Printf("Current task: [%d] %s - started %dmin ago\n", i, t.Title, elapsed)
			return nil
		}
	}
	fmt.Println("No task is currently started.")
	return nil
}

func finishCurrentTask() error {
	data, err := loadTasks()
	if err != nil {
		return err
	}
	today := todayKey()
	tasks := data[today]
	for i, t := range tasks {
		if t.Status == "started" {
			return updateStatus(i, "done")
		}
	}
	fmt.Println("No task is currently started.")
	return nil
}

func stopCurrentTask() error {
	data, err := loadTasks()
	if err != nil {
		return err
	}
	today := todayKey()
	tasks := data[today]
	for i, t := range tasks {
		if t.Status == "started" {
			fmt.Printf("Stopping task '%s'...\n", t.Title)
			return updateStatus(i, "pending")
		}
	}
	fmt.Println("No task is currently started.")
	return nil
}

func deleteTaskInteractive() error {
	data, err := loadTasks()
	if err != nil {
		return err
	}
	today := todayKey()
	tasks := data[today]
	if len(tasks) == 0 {
		fmt.Println("No tasks to delete.")
		return nil
	}

	prompt := promptui.Select{
		Label: "Select task to delete",
		Items: tasks,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "→ {{ .Title | red }} ({{ .Status }})",
			Inactive: "  {{ .Title }} ({{ .Status }})",
			Selected: "✔ {{ .Title }}",
		},
		Size:     10,
		HideHelp: true,
	}
	index, _, err := prompt.Run()
	if err != nil {
		if err.Error() == "interrupt" || err.Error() == "q" {
			return nil
		}
		return err
	}

	data[today] = append(tasks[:index], tasks[index+1:]...)
	return saveTasks(data)
}

func selectTaskAndSetStatus() error {
	data, err := loadTasks()
	if err != nil {
		return err
	}
	today := todayKey()
	tasks := data[today]
	if len(tasks) == 0 {
		fmt.Println("No tasks available.")
		return nil
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "→ {{ .Title | cyan }} ({{ .Status }})",
		Inactive: "  {{ .Title }} ({{ .Status }})",
		Selected: "✔ {{ .Title }}",
	}

	prompt := promptui.Select{
		Label:     "Select task to update",
		Items:     tasks,
		Templates: templates,
		Size:      10,
		HideHelp:  true,
	}

	index, _, err := prompt.Run()
	if err != nil {
		if err.Error() == "interrupt" || err.Error() == "q" {
			return nil
		}
		return err
	}

	statusPrompt := promptui.Select{
		Label:    "Set status",
		Items:    []string{"pending", "started", "done", "cancelled"},
		HideHelp: true,
	}
	_, result, err := statusPrompt.Run()
	if err != nil {
		if err.Error() == "interrupt" || err.Error() == "q" {
			return nil
		}
		return err
	}

	return updateStatus(index, result)
}

// --- CLI Command Setup ---

// Setup all cobra commands and return the root command
func setupCommands() *cobra.Command {
	// Note command: add or show notes for today
	noteCmd := &cobra.Command{
		Use:   "note [text|edit|edit-yesterday] [date]",
		Short: "Add, show, or edit notes for a day",
		Args:  cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 && args[0] == "edit-yesterday" {
				day := yesterdayKey()
				if err := editNoteForDay(day); err != nil {
					fmt.Println("Error:", err)
				} else {
					fmt.Printf("Notes for %s updated.\n", day)
				}
				return
			}
			if len(args) > 0 && args[0] == "edit" {
				day := todayKey()
				if len(args) > 1 {
					day = args[1]
				}
				if err := editNoteForDay(day); err != nil {
					fmt.Println("Error:", err)
				} else {
					fmt.Printf("Notes for %s updated.\n", day)
				}
				return
			}
			if len(args) == 0 {
				if err := showNotesForToday(); err != nil {
					fmt.Println("Error:", err)
				}
				return
			}
			note := strings.Join(args, " ")
			if err := addNoteForToday(note); err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Note added for today.")
			}
		},
	}
	rootCmd := &cobra.Command{
		Use:   "daily",
		Short: "Daily task management CLI",
	}

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new task for today",
		Run: func(cmd *cobra.Command, args []string) {
			if err := addTaskInteractive(false); err != nil {
				fmt.Println("Error:", err)
			}
		},
	}

	addTommorowCmd := &cobra.Command{
		Use:   "addt",
		Short: "Add a new task for tomorrow",
		Run: func(cmd *cobra.Command, args []string) {
			if err := addTaskInteractive(true); err != nil {
				fmt.Println("Error:", err)
			}
		},
	}

	listCmd := &cobra.Command{
		Use:   "ls",
		Short: "List and edit today's tasks",
		Run: func(cmd *cobra.Command, args []string) {
			if err := listTasksInteractive(false); err != nil {
				fmt.Println("Error:", err)
			}
		},
	}

	listTommorowCmd := &cobra.Command{
		Use:   "lst",
		Short: "List and edit tomorrow's tasks",
		Run: func(cmd *cobra.Command, args []string) {
			if err := listTasksInteractive(true); err != nil {
				fmt.Println("Error:", err)
			}
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Select a task and update its status",
		Run: func(cmd *cobra.Command, args []string) {
			if err := selectTaskAndSetStatus(); err != nil {
				fmt.Println("Error:", err)
			}
		},
	}

	nextCmd := &cobra.Command{
		Use:   "next",
		Short: "Start the next pending task",
		Run: func(cmd *cobra.Command, args []string) {
			if err := startNextPendingTask(); err != nil {
				fmt.Println("Error:", err)
			}
		},
	}

	currentCmd := &cobra.Command{
		Use:   "current",
		Short: "Show the currently active task",
		Run: func(cmd *cobra.Command, args []string) {
			if err := currentTask(); err != nil {
				fmt.Println("Error:", err)
			}
		},
	}

	finishCmd := &cobra.Command{
		Use:   "finish",
		Short: "Mark the current task as done",
		Run: func(cmd *cobra.Command, args []string) {
			if err := finishCurrentTask(); err != nil {
				fmt.Println("Error:", err)
			}
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a task",
		Run: func(cmd *cobra.Command, args []string) {
			if err := deleteTaskInteractive(); err != nil {
				fmt.Println("Error:", err)
			}
		},
	}

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the current task",
		Run: func(cmd *cobra.Command, args []string) {
			if err := stopCurrentTask(); err != nil {
				fmt.Println("Error:", err)
			}
		},
	}

	followCmd := &cobra.Command{
		Use:   "follow",
		Short: "Follow progress of the current task",
		Run: func(cmd *cobra.Command, args []string) {
			followStartedTask()
		},
	}

	yesterdayCmd := &cobra.Command{
		Use:   "yesterday",
		Short: "Show tasks from yesterday",
		Run: func(cmd *cobra.Command, args []string) {
			if err := showYesterdayTasks(); err != nil {
				fmt.Println("Error:", err)
			}
		},
	}

	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script for your shell",
		Long: `To load completions:

	Bash:
	  $ source <(daily completion bash)

	Zsh:
	  $ source <(daily completion zsh)

	fish:
	  $ daily completion fish > ~/.config/fish/completions/daily.fish

	PowerShell:
	  PS> daily completion powershell | Out-String | Invoke-Expression
	`, ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args: cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}

	shellCmd := &cobra.Command{
		Use:   "shell",
		Short: "Start an interactive shell with autocomplete",
		Run: func(cmd *cobra.Command, args []string) {
			runInteractiveShell()
		},
	}

	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(addTommorowCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(listTommorowCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(nextCmd)
	rootCmd.AddCommand(currentCmd)
	rootCmd.AddCommand(finishCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(followCmd)
	rootCmd.AddCommand(yesterdayCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(noteCmd)

	return rootCmd
}

// --- Shell Mode ---

// runInteractiveShell starts the interactive shell mode
func runInteractiveShell() { // ASCII art for the title
	cyan := "\033[36m"
	reset := "\033[0m"
	fmt.Println(cyan + "   ___       _ __       _______   ____" + reset)
	fmt.Println(cyan + "  / _ \\___ _(_) /_ __  / ___/ /  /  _/" + reset)
	fmt.Println(cyan + " / // / _ `/ / / // / / /__/ /___/ /  " + reset)
	fmt.Println(cyan + "/____/\\_,_/_/_/\\_, /  \\___/____/___/  " + reset)
	fmt.Println(cyan + "              /___/                   " + reset)
	fmt.Println("Daily Task Manager Interactive Shell")
	fmt.Println("Type 'help' for available commands or 'exit' to quit")
	fmt.Println("----------------")

	// Map of commands for quick lookup and tab completion
	commands := map[string]struct{}{
		"add":       {},
		"addt":      {},
		"ls":        {},
		"lst":       {},
		"status":    {},
		"next":      {},
		"current":   {},
		"finish":    {},
		"delete":    {},
		"stop":      {},
		"follow":    {},
		"yesterday": {},
		"note":      {},
		"clear":     {},
		"help":      {},
		"exit":      {},
	}

	// Start a scanner to read user input
	scanner := bufio.NewScanner(os.Stdin)
	var lastCmd string

	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		// Handle empty input - repeat the last command
		if input == "" && lastCmd != "" {
			input = lastCmd
		} else if input == "" {
			continue
		}

		// Save the command for potential repeat
		lastCmd = input

		// Handle tab completion for when user presses tab (simulate with ?)
		if strings.HasSuffix(input, "?") {
			prefix := strings.TrimSuffix(input, "?")
			fmt.Println("Available commands:")
			for cmd := range commands {
				if strings.HasPrefix(cmd, prefix) {
					fmt.Printf("  %s\n", cmd)
				}
			}
			continue
		}
		// Exit command
		if input == "exit" || input == "quit" {
			break
		}

		// Clear command - clears the screen but keeps the ASCII title
		if input == "clear" {
			// Clear the screen
			fmt.Print("\033[H\033[2J")
			// Print the ASCII title again
			cyan := "\033[36m"
			reset := "\033[0m"
			fmt.Println(cyan + "   ___       _ __       _______   ____" + reset)
			fmt.Println(cyan + "  / _ \\___ _(_) /_ __  / ___/ /  /  _/" + reset)
			fmt.Println(cyan + " / // / _ `/ / / // / / /__/ /___/ /  " + reset)
			fmt.Println(cyan + "/____/\\_,_/_/_/\\_, /  \\___/____/___/  " + reset)
			fmt.Println(cyan + "              /___/                   " + reset)
			fmt.Println("Daily Task Manager Interactive Shell")
			fmt.Println("Type 'help' for available commands or 'exit' to quit")
			fmt.Println("----------------")
			continue
		}

		// Help command
		if input == "help" {
			fmt.Println("Available commands:")
			fmt.Println("  add        - Add a new task for today")
			fmt.Println("  addt       - Add a new task for tomorrow")
			fmt.Println("  ls         - List and edit today's tasks")
			fmt.Println("  lst        - List and edit tomorrow's tasks")
			fmt.Println("  status     - Select a task and update its status")
			fmt.Println("  next       - Start the next pending task")
			fmt.Println("  current    - Show the currently active task")
			fmt.Println("  finish     - Mark the current task as done")
			fmt.Println("  delete     - Delete a task")
			fmt.Println("  stop       - Stop the current task")
			fmt.Println("  follow     - Follow progress of the current task")
			fmt.Println("  yesterday  - Show tasks from yesterday")
			fmt.Println("  note       - Add, show, or edit daily notes")
			fmt.Println("  clear      - Clear the screen")
			fmt.Println("  exit/quit  - Exit the shell")
			fmt.Println()
			fmt.Println("Note: Press 'q' to exit from any interactive menu")
			fmt.Println()
			fmt.Println("Notes usage:")
			fmt.Println("  note <text>           - Add a note for today")
			fmt.Println("  note                  - Show today's notes")
			fmt.Println("  note edit             - Edit today's notes in nano")
			fmt.Println("  note edit <YYYY-MM-DD> - Edit notes for a specific day")
			fmt.Println("  note edit-yesterday    - Edit yesterday's notes in nano")
			continue
		}

		// Handle the command
		args := strings.Fields(input)
		if len(args) == 0 {
			continue
		}

		command := args[0]

		// Execute the command
		switch command {
		case "add":
			addTaskInteractive(false)
		case "addt":
			addTaskInteractive(true)
		case "ls":
			listTasksInteractive(false)
		case "lst":
			listTasksInteractive(true)
		case "status":
			selectTaskAndSetStatus()
		case "next":
			startNextPendingTask()
		case "current":
			currentTask()
		case "finish":
			finishCurrentTask()
		case "delete":
			deleteTaskInteractive()
		case "stop":
			stopCurrentTask()
		case "follow":
			followStartedTask()
		case "yesterday":
			showYesterdayTasks()
		default:
			fmt.Printf("Unknown command: %s\nType 'help' for available commands\n", command)
		case "note":
			// Pass note args to main note handler
			if len(args) > 1 {
				os.Args = append([]string{os.Args[0], "note"}, args[1:]...)
			} else {
				os.Args = []string{os.Args[0], "note"}
			}
			main()
			// After running note, break to avoid duplicate prompt
			break
		}
	}
}

// --- Utilities ---

func main() {
	rootCmd := setupCommands()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// followStartedTask displays a progress bar for the currently started task
func followStartedTask() {
	data, err := loadTasks()
	if err != nil {
		fmt.Println("Error loading tasks:", err)
		return
	}
	today := todayKey()
	tasks := data[today]
	// Find the started task
	var startedTask *Task
	for _, t := range tasks {
		if t.Status == "started" {
			taskCopy := t
			startedTask = &taskCopy
			break
		}
	}
	if startedTask == nil {
		fmt.Println("No task is currently started.")
		return
	}
	// Calculate the total duration
	totalDuration := time.Duration(startedTask.Estimated) * time.Minute
	progressBar := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(50),
		progress.WithSolidFill("#03befc"),
	)
	m := taskModel{
		progress:      progressBar,
		task:          startedTask,
		startTime:     time.Unix((startedTask.StartedAt - int64(startedTask.Actual*60)), 0),
		totalDuration: totalDuration,
	}
	initialElapsed := time.Since(m.startTime)
	fmt.Printf("Initial elapsed time: %s\n", initialElapsed)
	initialPercent := math.Min(1.0, float64(initialElapsed)/float64(totalDuration))
	progressBar.SetPercent(initialPercent)
	fmt.Printf("Following task: %s (%d min)\nPress q or Ctrl+C to exit\n\n",
		startedTask.Title, startedTask.Estimated)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running progress bar:", err)
	}
}
