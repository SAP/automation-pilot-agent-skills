// Check reviewed-contract age and identity; it never refreshes evidence itself.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type review struct {
	ReviewedOn      string
	MaxAgeDays      int
	ServiceRevision string
	SnapshotSHA256  string
}

func validate(r review, snapshot []byte, now time.Time) error {
	date, err := time.Parse("2006-01-02", r.ReviewedOn)
	if err != nil || date.After(now) || r.MaxAgeDays < 1 || r.MaxAgeDays > 90 {
		return fmt.Errorf("invalid contract review date or interval")
	}
	if !now.Before(date.AddDate(0, 0, r.MaxAgeDays+1)) {
		return fmt.Errorf("contract review is overdue; review upstream service changes before updating review.json")
	}
	sum := sha256.Sum256(snapshot)
	var source struct{ Revision string }
	if err := json.Unmarshal(snapshot, &source); err != nil {
		return err
	}
	if r.SnapshotSHA256 != hex.EncodeToString(sum[:]) || r.ServiceRevision == "" || source.Revision != r.ServiceRevision {
		return fmt.Errorf("contract changed since recorded review; review the snapshot and service corrections")
	}
	return nil
}

func main() {
	root := "."
	if len(os.Args) == 2 {
		root = os.Args[1]
	}
	if err := run(root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string) error {
	b, err := os.ReadFile(filepath.Join(root, "ci", "contracts", "review.json"))
	if err != nil {
		return err
	}
	var r review
	if err := json.Unmarshal(b, &r); err != nil {
		return err
	}
	snapshot, err := os.ReadFile(filepath.Join(root, "cli", "internal", "contract", "snapshot.json"))
	if err != nil {
		return err
	}
	if err := validate(r, snapshot, time.Now().UTC()); err != nil {
		return err
	}
	fmt.Printf("Contract review %s matches service revision %s and remains within %d days\n", r.ReviewedOn, r.ServiceRevision, r.MaxAgeDays)
	return nil
}
