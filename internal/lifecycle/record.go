package lifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// Record is a small local intent journal, not a durable job scheduler.
type Record struct {
	Version         int       `json:"version"`
	RunID           string    `json:"run_id"`
	Mode            string    `json:"mode"`
	State           string    `json:"state"`
	ProcessGroupID  int       `json:"process_group_id,omitempty"`
	ProcessIdentity string    `json:"process_identity,omitempty"`
	Result          *Result   `json:"result,omitempty"`
	Recovery        *Recovery `json:"recovery,omitempty"`
}

func RecordDir(root, id string) (string, error) {
	if !runIDPattern.MatchString(id) {
		return "", fmt.Errorf("invalid run ID")
	}
	path := filepath.Join(root, "evidence", "lifecycle-"+id)
	for _, p := range []string{filepath.Join(root, "evidence"), path} {
		stat, err := os.Lstat(p)
		if err != nil {
			return "", err
		}
		if !stat.IsDir() || stat.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("record path is not a real directory")
		}
	}
	return path, nil
}

// LockRecord is held by a live supervisor/reconciler; SIGKILL releases the OS lock.
func LockRecord(dir string) (*os.File, error) {
	path := filepath.Join(dir, "record.lock")
	if s, err := os.Lstat(path); err == nil && !s.Mode().IsRegular() {
		return nil, fmt.Errorf("record lock is not regular")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("run is active or another reconciler holds its lock: %w", err)
	}
	return f, nil
}

func SaveRecord(dir string, r Record) error {
	if err := validateRecord(r); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".record-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(append(data, '\n'))
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(tmp, filepath.Join(dir, "record.json")); err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

func LoadRecord(dir, id string) (Record, error) {
	path := filepath.Join(dir, "record.json")
	stat, err := os.Lstat(path)
	if err != nil {
		return Record{}, err
	}
	if !stat.Mode().IsRegular() {
		return Record{}, fmt.Errorf("record is not regular")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	var r Record
	if err = json.Unmarshal(data, &r); err != nil {
		return r, err
	}
	if err = validateRecord(r); err != nil {
		return r, err
	}
	if r.RunID != id {
		return r, fmt.Errorf("record ID differs from requested run")
	}
	return r, nil
}

func validateRecord(r Record) error {
	if r.Version != 1 || !runIDPattern.MatchString(r.RunID) {
		return fmt.Errorf("invalid record version/ID")
	}
	if r.Mode != "comparison" && r.Mode != "baseline" && r.Mode != "candidate" {
		return fmt.Errorf("invalid record mode")
	}
	switch r.State {
	case "prepared", "running", "finished", "reconciled":
	default:
		return fmt.Errorf("invalid record state")
	}
	if r.ProcessGroupID < 0 {
		return fmt.Errorf("invalid process group")
	}
	return nil
}
