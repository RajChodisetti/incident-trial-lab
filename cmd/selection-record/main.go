// selection-record creates or reconstructs one historical offline stub record.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/RajChodisetti/incident-trial-lab/internal/investigation"
)

type abstainingStub struct{}

func (abstainingStub) Select(context.Context, investigation.View, []string) (investigation.Decision, error) {
	return investigation.Decision{Abstain: true}, nil
}

func run() error {
	if len(os.Args) != 2 || os.Args[1] != "-record" && os.Args[1] != "-replay" {
		return fmt.Errorf("use -record or -replay from project root; offline only")
	}
	path := "evidence/selection-context-offline.json"
	dir, err := os.Lstat(filepath.Dir(path))
	if err != nil || !dir.IsDir() || dir.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("project evidence directory must be real")
	}
	if os.Args[1] == "-replay" {
		s, err := os.Lstat(path)
		if err != nil || !s.Mode().IsRegular() || s.Size() > 24576 {
			return fmt.Errorf("offline record unavailable or exceeds bound")
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("offline record unavailable")
		}
		r, err := investigation.DecodeSelectionRecord(data)
		if err != nil {
			return err
		}
		fmt.Printf("Validated offline record: status=%s request_sha256=%s; no selector/trial invoked\n", r.Status, r.RequestSHA256)
		return nil
	}
	source := "evidence/offline-loop-eaa167c837aeb1c5/captures.json"
	sourceDir, err := os.Lstat(filepath.Dir(source))
	if err != nil || !sourceDir.IsDir() || sourceDir.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("historical capture directory must be real")
	}
	s, err := os.Lstat(source)
	if err != nil || !s.Mode().IsRegular() || s.Size() > 2<<20 {
		return fmt.Errorf("historical synthetic capture unavailable")
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("historical synthetic capture unreadable")
	}
	var captured investigation.Result
	if json.Unmarshal(data, &captured) != nil || len(captured.Captures) < 2 {
		return fmt.Errorf("invalid captured baseline")
	}
	v, err := investigation.HistoricalIncidentView(captured.Captures[0], captured.Captures[1])
	if err != nil {
		return err
	}
	n, j, err := investigation.NewOfflineSelectionJournal(abstainingStub{})
	if err != nil {
		return err
	}
	_, err = n.Select(context.Background(), v, []string{"fixed", "ineffective", "misleading"})
	if err != nil {
		return err
	}
	rows := j.Records()
	if len(rows) != 1 {
		return fmt.Errorf("one completed offline record required")
	}
	data, err = json.MarshalIndent(rows[0], "", "  ")
	if err != nil {
		return err
	}
	if _, err = investigation.DecodeSelectionRecord(data); err != nil {
		return err
	}
	dir, err = os.Lstat(filepath.Dir(path))
	if err != nil || !dir.IsDir() || dir.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("project evidence directory must be real")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("record already exists or cannot be created; no overwrite")
	}
	_, err = f.Write(append(data, '\n'))
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return fmt.Errorf("offline record persistence failed")
	}
	fmt.Printf("Recorded historical input with abstaining stub: request_sha256=%s; no model/trial invoked\n", rows[0].RequestSHA256)
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
