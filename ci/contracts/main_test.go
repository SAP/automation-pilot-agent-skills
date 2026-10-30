package main

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func TestReviewRejectsStaleOrUnreviewedSnapshots(t *testing.T) {
	b := []byte("{\"revision\":\"upstream\"}")
	sum := sha256.Sum256(b)
	r := review{"2026-10-28", 90, "upstream", hex.EncodeToString(sum[:])}
	now := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	if err := validate(r, b, now); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []review{
		{"2026-01-01", 90, r.ServiceRevision, r.SnapshotSHA256},
		{"2026-11-01", 90, r.ServiceRevision, r.SnapshotSHA256},
		{r.ReviewedOn, 90, "other", r.SnapshotSHA256},
		{r.ReviewedOn, 90, r.ServiceRevision, "different"},
		{r.ReviewedOn, 0, r.ServiceRevision, r.SnapshotSHA256},
	} {
		if err := validate(bad, b, now); err == nil {
			t.Fatal("accepted stale or changed review")
		}
	}
}
