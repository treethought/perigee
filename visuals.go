package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Visual interface {
	tea.Model
	SetSize(width, height int)
	SetActive(bool)
	Active() bool
	Reset() tea.Cmd
}

var viewStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("#00ff00"))

type VisualsView struct {
	active      bool
	activeModel Visual
	models      map[string]Visual
	style       lipgloss.Style
	w, h        int
}

func NewVisualsView(models map[string]Visual) *VisualsView {
	if models == nil {
		models = make(map[string]Visual)
	}
	return &VisualsView{
		models: models,
		style:  viewStyle,
	}
}

func (v *VisualsView) SetSize(width, height int) {
	v.w = width
	v.h = height
	fw, fh := viewStyle.GetFrameSize()
	for _, model := range v.models {
		model.SetSize(width-fw-1, height-fh-1)
	}
}
func (v *VisualsView) SetActive(active bool) {
	v.active = active
}
func (v *VisualsView) Active() bool {
	return v.active
}

func (v *VisualsView) SetActiveModel(m string) tea.Cmd {
	v.activeModel = v.models[m]
	v.activeModel.SetActive(true)
	return v.activeModel.Reset()
}

func (v *VisualsView) Init() tea.Cmd {
	cmds := []tea.Cmd{}
	for _, model := range v.models {
		cmds = append(cmds, model.Init())
	}
	return tea.Batch(cmds...)
}

func (v *VisualsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if v.activeModel == nil {
		return v, nil
	}
	_, cmd := v.activeModel.Update(msg)
	return v, cmd
}

func (v *VisualsView) View() string {
	if v.activeModel == nil {
		return v.style.Render("No active visuals")
	}
	return v.style.Width(v.w).Height(v.h).Render(v.activeModel.View())
}
