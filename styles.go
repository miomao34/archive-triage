package main

import (
	"charm.land/bubbles/v2/textarea"
	"charm.land/lipgloss/v2"
)

var (
	topCellBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "├",
		BottomRight: "┤",
	}
	middleCellBorder = lipgloss.Border{
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "├",
		TopRight:    "┤",
		BottomLeft:  "├",
		BottomRight: "┤",
	}
	bottomCellBorder = lipgloss.Border{
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		BottomLeft:  "└",
		BottomRight: "┘",
	}

	commonBackground = lipgloss.NewStyle().Background(lipgloss.Color("#EB9486"))

	nameStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color("#EB9486")).Bold(true)
	linkStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color("#F3DE8A"))
	duplicateStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#32965D")).Bold(true)
	duplicateContextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#194D30"))
	helpStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color("#3b3b3b"))
	// nameStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#EB9486")).Bold(true).Border(topCellBorder)
	// linkStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#F3DE8A")).Border(bottomCellBorder).BorderTop(false)

	tagTextAreaStyle = textarea.DefaultStyles(true)

	topCellStyle    = lipgloss.NewStyle().Border(topCellBorder)
	middleCellStyle = lipgloss.NewStyle().Border(middleCellBorder).BorderTop(false)
	bottomCellStyle = lipgloss.NewStyle().Border(bottomCellBorder).BorderTop(false)
)
