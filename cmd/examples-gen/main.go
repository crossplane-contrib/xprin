/*
Copyright 2026 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// examples-gen fills the outputs of the examples of markdown files, see docs/contributing/examples-gen.md.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/alecthomas/kong"
)

const (
	prompt      = "➜ "
	placeholder = "<output>"
	codeBlock   = "```"
	tagPrefix   = "<!-- examples-gen: "
	tagSuffix   = " -->"
)

// cli is the command line of the tool.
type cli struct {
	Files           []string `arg:""                                                                                     help:"Markdown files whose examples are filled." type:"existingfile"`
	RegenerateAll   bool     `help:"Regenerate the outputs of every command, not only the ones that have a placeholder." name:"regenerate-all"                            xor:"regenerate"`
	RegenerateMatch string   `help:"Regenerate the outputs of the commands that contain this text."                      name:"regenerate-match"                          xor:"regenerate"`
	Tag             string   `help:"Fill the code blocks that have this tag, instead of the ones that have none."        name:"tag"`
}

func main() {
	var args cli

	kong.Parse(&args,
		kong.Name("examples-gen"),
		kong.Description("Fill the outputs of the examples of markdown files."),
		kong.UsageOnError(),
	)

	update := func(content string) (string, error) {
		// To regenerate an output, the placeholder goes in its place first
		if args.RegenerateAll || args.RegenerateMatch != "" {
			content = clearOutputs(content, args.RegenerateMatch, args.Tag)
		}

		return fill(content, args.Tag, runCommand)
	}

	for _, path := range args.Files {
		if err := updateFile(path, update); err != nil {
			fmt.Fprintf(os.Stderr, "examples-gen: %s: %v\n", path, err)
			os.Exit(1)
		}
	}
}

// updateFile replaces the content of the file with what update returns for it.
func updateFile(path string, update func(content string) (string, error)) error {
	data, err := os.ReadFile(path) //nolint:gosec // G304 - the path is an argument of the maintainer who runs the tool
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	filled, err := update(string(data))
	if err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}

	if err := os.WriteFile(path, []byte(filled), info.Mode().Perm()); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

// fill puts the output of run in place of the placeholder after each command, in the code blocks that have the tag.
func fill(content, tag string, run func(command string) (string, error)) (string, error) {
	lines := strings.Split(content, "\n")
	filled := make([]string, 0, len(lines))
	inFence := false
	tagged := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		filled = append(filled, line)

		if strings.HasPrefix(line, codeBlock) {
			inFence = !inFence
			tagged = inFence && i > 0 && blockTag(lines[i-1]) == tag

			continue
		}

		command, isCommand := strings.CutPrefix(line, prompt)
		if !inFence || !tagged || !isCommand || i+1 >= len(lines) || lines[i+1] != placeholder {
			continue
		}

		fmt.Fprintf(os.Stderr, "running: %s\n", command)

		output, err := run(command)
		if err != nil {
			return "", fmt.Errorf("%q: %w", command, err)
		}

		if output != "" {
			filled = append(filled, strings.Split(output, "\n")...)
		}

		// The placeholder is replaced by the output
		i++
	}

	return strings.Join(filled, "\n"), nil
}

// clearOutputs puts the placeholder in place of the outputs of the commands that contain match, in the tagged blocks.
func clearOutputs(content, match, tag string) string {
	lines := strings.Split(content, "\n")
	cleared := make([]string, 0, len(lines))
	inFence := false
	tagged := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		cleared = append(cleared, line)

		if strings.HasPrefix(line, codeBlock) {
			inFence = !inFence
			tagged = inFence && i > 0 && blockTag(lines[i-1]) == tag

			continue
		}

		command, isCommand := strings.CutPrefix(line, prompt)
		if !inFence || !tagged || !isCommand || !strings.Contains(command, match) {
			continue
		}

		cleared = append(cleared, placeholder)

		// The output goes on until the next command, or the end of the block
		next := i + 1
		for next < len(lines) && !strings.HasPrefix(lines[next], prompt) && !strings.HasPrefix(lines[next], codeBlock) {
			next++
		}

		if next < len(lines) && strings.HasPrefix(lines[next], prompt) && next > i+1 && lines[next-1] == "" {
			cleared = append(cleared, "")
		}

		i = next - 1
	}

	return strings.Join(cleared, "\n")
}

// blockTag returns the tag of the line before a code block, or an empty string when the line is not a tag.
func blockTag(line string) string {
	tag, found := strings.CutPrefix(line, tagPrefix)
	if !found {
		return ""
	}

	return strings.TrimSuffix(tag, tagSuffix)
}

// runCommand returns the output of the command in a shell, ignoring its exit status: a failing test is a valid example.
func runCommand(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command) //nolint:gosec // The commands are the ones of the examples, written by the maintainers

	output, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return "", err
	}

	return strings.TrimRight(string(output), "\n"), nil
}
