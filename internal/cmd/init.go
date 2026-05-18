// SPDX-License-Identifier: MPL-2.0
package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alfred-intelligence/shy/internal/paths"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Set up $HOME/.shy/ and wire it into ~/.bashrc",
		Long: `Create the directory layout under $HOME/.shy/, copy any seed from
/usr/share/shy/ (skip-on-conflict), write init.bash, append one source line
to ~/.bashrc, and bootstrap shy's own bash completion.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runInit(cmd.OutOrStdout())
		},
	}
}

// SudoHint is printed when shy init is invoked as root.
const SudoHint = "shy init: refusing to run as root.\nshy init: to reset shy across all users, run: sudo shy system-reset"

func runInit(out io.Writer) error {
	if os.Geteuid() == 0 {
		return errors.New(SudoHint)
	}
	home, err := paths.Home()
	if err != nil {
		return err
	}

	created := []string{}
	if mkdirReport(home, 0o700, &created); err != nil {
		return err
	}
	for _, d := range paths.Subdirs(home) {
		if err := mkdirReport(d, 0o755, &created); err != nil {
			return err
		}
	}

	copied, err := copySeed(paths.SystemSeed, home)
	if err != nil {
		return err
	}

	wroteInit := false
	initPath := paths.InitBash(home)
	if _, err := os.Stat(initPath); errors.Is(err, fs.ErrNotExist) {
		if err := os.WriteFile(initPath, []byte(EmbeddedInitBash), 0o644); err != nil {
			return fmt.Errorf("init: write %s: %w", initPath, err)
		}
		wroteInit = true
	}

	bashrcUpdated, err := ensureBashrcLine()
	if err != nil {
		return err
	}

	if err := bootstrapShyCompletion(home); err != nil {
		return err
	}

	fmt.Fprintln(out, "shy init: setup complete.")
	fmt.Fprintf(out, "  home:           %s\n", home)
	fmt.Fprintf(out, "  dirs created:   %d\n", len(created))
	fmt.Fprintf(out, "  seed files:     %d\n", copied)
	if wroteInit {
		fmt.Fprintln(out, "  init.bash:      written")
	} else {
		fmt.Fprintln(out, "  init.bash:      already present")
	}
	if bashrcUpdated {
		fmt.Fprintln(out, "  .bashrc:        source line added")
	} else {
		fmt.Fprintln(out, "  .bashrc:        already configured")
	}
	return nil
}

func mkdirReport(path string, mode os.FileMode, created *[]string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(path, mode); err != nil {
		return fmt.Errorf("init: mkdir %s: %w", path, err)
	}
	// Re-apply mode in case the umask masked off the requested bits.
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("init: chmod %s: %w", path, err)
	}
	*created = append(*created, path)
	return nil
}

func copySeed(src, dst string) (int, error) {
	info, err := os.Stat(src)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("init: stat %s: %w", src, err)
	}
	if !info.IsDir() {
		return 0, nil
	}
	count := 0
	walkErr := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if p == src {
			return nil
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if _, err := os.Stat(target); err == nil {
			// Skip-on-conflict.
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
		count++
		return nil
	})
	if walkErr != nil {
		return count, fmt.Errorf("init: copy seed: %w", walkErr)
	}
	return count, nil
}

func ensureBashrcLine() (bool, error) {
	bashrcPath, err := bashrcPath()
	if err != nil {
		return false, err
	}
	existing, err := os.ReadFile(bashrcPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("init: read %s: %w", bashrcPath, err)
	}
	if bytes.Contains(existing, []byte(paths.BashrcMarker)) {
		return false, nil
	}
	appended := append(existing, []byte("\n# shy — added by `shy init`\n"+paths.BashrcSourceLine+"\n")...)
	if len(existing) == 0 {
		appended = []byte("# shy — added by `shy init`\n" + paths.BashrcSourceLine + "\n")
	}
	if err := os.WriteFile(bashrcPath, appended, 0o644); err != nil {
		return false, fmt.Errorf("init: write %s: %w", bashrcPath, err)
	}
	return true, nil
}

func bashrcPath() (string, error) {
	if v := os.Getenv("SHY_TEST_BASHRC"); v != "" {
		return v, nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("init: home: %w", err)
	}
	return filepath.Join(h, ".bashrc"), nil
}

func bootstrapShyCompletion(home string) error {
	dst := filepath.Join(home, "completions", "shy")
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	root := New()
	buf := &bytes.Buffer{}
	if err := root.GenBashCompletion(buf); err != nil {
		return fmt.Errorf("init: generate completion: %w", err)
	}
	// Strip the cobra `complete -o` lines that source-time-bind to
	// `shy`; the file is read every shell start, so they're correct.
	out := strings.TrimRight(buf.String(), "\n") + "\n"
	return os.WriteFile(dst, []byte(out), 0o644)
}
