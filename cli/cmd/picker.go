package cmd

import (
	"fmt"
	"io"
	"os"

	survey "github.com/AlecAivazis/survey/v2"
	"golang.org/x/term"
)

type pickerOption struct {
	Label       string
	Description string
	Value       string
}

type interactivePicker interface {
	Select(prompt string, options []pickerOption) (pickerOption, error)
	MultiSelect(prompt string, options []pickerOption, min int) ([]pickerOption, error)
}

var isInteractiveTerminal = func(rc *RunContext) bool {
	if rc == nil || rc.Output == nil || rc.Output.IsStructured() {
		return false
	}
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

var newInteractivePicker = func() interactivePicker {
	return &surveyPicker{
		in:  os.Stdin,
		out: os.Stdout,
		err: os.Stderr,
	}
}

type surveyPicker struct {
	in  *os.File
	out *os.File
	err io.Writer
}

func (p *surveyPicker) Select(prompt string, options []pickerOption) (pickerOption, error) {
	if len(options) == 0 {
		return pickerOption{}, fmt.Errorf("no options available for %s", prompt)
	}

	normalized := normalizedPickerOptions(options)
	labels := make([]string, 0, len(normalized))
	indexByLabel := make(map[string]int, len(normalized))
	for i, option := range normalized {
		labels = append(labels, option.Label)
		indexByLabel[option.Label] = i
	}

	selection := ""
	promptUI := &survey.Select{
		Message:     prompt,
		Options:     labels,
		PageSize:    pageSizeForOptions(normalized),
		Description: describePickerOption(normalized),
	}
	if err := survey.AskOne(
		promptUI,
		&selection,
		survey.WithValidator(survey.Required),
		survey.WithStdio(p.in, p.out, p.err),
	); err != nil {
		return pickerOption{}, err
	}

	return normalized[indexByLabel[selection]], nil
}

func (p *surveyPicker) MultiSelect(prompt string, options []pickerOption, min int) ([]pickerOption, error) {
	if len(options) == 0 {
		return nil, fmt.Errorf("no options available for %s", prompt)
	}

	normalized := normalizedPickerOptions(options)
	labels := make([]string, 0, len(normalized))
	indexByLabel := make(map[string]int, len(normalized))
	for i, option := range normalized {
		labels = append(labels, option.Label)
		indexByLabel[option.Label] = i
	}

	selections := []string{}
	promptUI := &survey.MultiSelect{
		Message:     prompt,
		Options:     labels,
		PageSize:    pageSizeForOptions(normalized),
		Description: describePickerOption(normalized),
	}
	validator := func(answer interface{}) error {
		selected, _ := answer.([]string)
		if len(selected) < min {
			return fmt.Errorf("choose at least %d option(s)", min)
		}
		return nil
	}
	if err := survey.AskOne(
		promptUI,
		&selections,
		survey.WithValidator(validator),
		survey.WithStdio(p.in, p.out, p.err),
	); err != nil {
		return nil, err
	}

	resolved := make([]pickerOption, 0, len(selections))
	for _, selection := range selections {
		resolved = append(resolved, normalized[indexByLabel[selection]])
	}
	return resolved, nil
}

func pageSizeForOptions(options []pickerOption) int {
	if len(options) < 10 {
		return len(options)
	}
	return 10
}

func describePickerOption(options []pickerOption) func(value string, index int) string {
	return func(_ string, index int) string {
		if index < 0 || index >= len(options) {
			return ""
		}
		return options[index].Description
	}
}

func normalizedPickerOptions(options []pickerOption) []pickerOption {
	counts := make(map[string]int, len(options))
	normalized := make([]pickerOption, len(options))
	for i, option := range options {
		counts[option.Label]++
		normalized[i] = option
	}

	for i, option := range normalized {
		if counts[option.Label] > 1 {
			normalized[i].Label = fmt.Sprintf("%s [%s]", option.Label, option.Value)
		}
	}

	return normalized
}
