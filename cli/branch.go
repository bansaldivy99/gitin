package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/isacikgoz/gitin/git"
	"github.com/isacikgoz/promptui"

	log "github.com/sirupsen/logrus"
)

type BranchOptions struct {
	Types     BranchTypes
	PromptOps *PromptOptions
}

type BranchTypes uint8

const (
	LocalBranches BranchTypes = iota
	RemoteBranches
	AllBranches
)

func BranchBuilder(r *git.Repository, opts *BranchOptions) error {
	if err := r.InitializeBranches(); err != nil {
		return err
	}
	switch opts.Types {
	case LocalBranches:
		i := 0 // output index
		for _, b := range r.Branches {
			if !b.IsRemote() {
				r.Branches[i] = b
				i++
			}
		}
		r.Branches = r.Branches[:i]
	case RemoteBranches:
		i := 0 // output index
		for _, b := range r.Branches {
			if b.IsRemote() {
				r.Branches[i] = b
				i++
			}
		}
		r.Branches = r.Branches[:i]
	case AllBranches:

	}
	return branchPrompt(r, opts.PromptOps)
}

func branchPrompt(r *git.Repository, opts *PromptOptions) error {

	// make terminal not line wrap
	fmt.Printf("\x1b[?7l")
	// defer restoring line wrap
	defer fmt.Printf("\x1b[?7h")
	searcher := func(input string, index int) bool {
		item := r.Branches[index]
		name := strings.Replace(strings.ToLower(item.Name), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		return strings.Contains(name, input)
	}

	var prompt promptui.Select
	kset := make(map[rune]promptui.CustomFunc)
	kset['q'] = func(in interface{}, chb chan bool, index int) error {
		quitPrompt(r, chb)
		return nil
	}
	kset['d'] = func(in interface{}, chb chan bool, index int) error {
		b := r.Branches[index]
		if b == r.Branch {
			return nil
		}
		if err := deleteBranch(r, b, "d"); err != nil {
			log.Error(err)
		}
		chb <- false
		if err := r.InitializeBranches(); err != nil {
			return err
		}
		prompt.RefreshList(r.Branches, index)
		return nil
	}
	kset['D'] = func(in interface{}, chb chan bool, index int) error {
		b := r.Branches[index]
		if b == r.Branch {
			return nil
		}
		if err := deleteBranch(r, b, "D"); err != nil {
			log.Error(err)
		}
		chb <- false
		if err := r.InitializeBranches(); err != nil {
			return err
		}
		prompt.RefreshList(r.Branches, index)
		return nil
	}

	prompt = promptui.Select{
		Label:       "Branches",
		Items:       r.Branches,
		HideHelp:    opts.HideHelp,
		Searcher:    searcher,
		Size:        opts.Size,
		Templates:   branchTemplate(),
		CustomFuncs: kset,
	}
	i, _, err := prompt.RunCursorAt(opts.Cursor, opts.Scroll)

	if err == nil {
		cmd := exec.Command("git", "checkout", r.Branches[i].Name)
		cmd.Dir = r.AbsPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}

	return nil
}

func branchTemplate() *promptui.SelectTemplates {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . |yellow}}:",
		Active:   "*  {{ .Name | green }}",
		Inactive: "   {{ .Name }}",
		Selected: "{{ .Name }}",
		Extra:    "delete: d checkout: enter",
		Details: "\n" +
			"-------------- Last Commit --------------" + "\n" +
			"{{ \"Hash:\"  | faint }}    {{ .Hash | yellow }} " + "\n" +
			"{{ \"Message:\"  | faint }} {{ .LastCommitMessage }} " + "\n" +
			"{{ \"Author:\"  | faint }}  {{ .LastCommitAuthor }} " + "\n" +
			"{{ \"Date:\"  | faint }}    {{ .LastCommitDate }} " + "\n" +
			"{{- if .IsRemote }} {{- else }} \n" +
			"---------------- Status -----------------\n" +
			"{{ .Status }} {{- end }}",
	}
	return templates
}

func deleteBranch(r *git.Repository, b *git.Branch, mode string) error {
	cmd := exec.Command("git", "branch", "-"+mode, b.Name)
	cmd.Dir = r.AbsPath
	if err := cmd.Run(); err == nil {
		return err
	}
	return nil
}
