package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"log"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type Package struct {
	Scripts map[string]string `json:"scripts"`
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	list list.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "enter" {
			// Get the currently selected item
			selectedItem := m.list.SelectedItem().(item)

			// Run the command from the item's desc field
			cmd := exec.Command("/bin/sh", "-c", selectedItem.desc)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			// Get the current directory
			dir, err := os.Getwd()
			if err != nil {
				fmt.Println("Error getting current directory:", err)
				return m, nil
			}

			// Add the node_modules/.bin directory to the PATH
			nodeModulesBin := filepath.Join(dir, "node_modules", ".bin")
			cmd.Env = append(os.Environ(), "PATH="+os.Getenv("PATH")+":"+nodeModulesBin)

			// Start the command in a new process group
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

			err = cmd.Start()
			if err != nil {
				fmt.Println("Error starting command:", err)
				return m, nil
			}

			go func() {
				err = cmd.Wait()
				if err != nil {
					fmt.Println("Error waiting for command to finish:", err)
				}
			}()

			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

func main() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("dir:", dir)

	file, err := os.Open(filepath.Join(dir, "package.json"))
	if err != nil {
		fmt.Println("Error reading file:", err)
		os.Exit(1)
	}

	if err != nil {
		fmt.Println("Error reading file:", err)
		os.Exit(1)
	}

	defer file.Close()

	var pkg Package
	byteValue, _ := io.ReadAll(file)
	err = json.Unmarshal(byteValue, &pkg)
	if err != nil {
		fmt.Println("Error parsing file")
		os.Exit(1)
	}

	items := make([]list.Item, 0, len(pkg.Scripts))
	for key, value := range pkg.Scripts {
		items = append(items, item{title: key, desc: value})
	}

	m := model{list: list.New(items, list.NewDefaultDelegate(), 0, 0)}
	m.list.Title = "Scripts"

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

