// Package rehearsal imports a sealed legacy continuity archive into an
// isolated vNext store and verifies the resulting semantic projection. It has
// no activation or cutover surface.
package rehearsal

import (
	"context"
	"fmt"

	"github.com/levifig/loaf/vnext/continuity"
	continuitysqlite "github.com/levifig/loaf/vnext/continuity/sqlite"
	"github.com/levifig/loaf/vnext/migration/archive"
)

// Report is the deterministic semantic receipt for one successful staged
// rehearsal. It deliberately contains no activation or cutover vocabulary.
type Report struct {
	Format             string                     `json:"format"`
	Version            int                        `json:"version"`
	ProjectID          continuity.ProjectID       `json:"project_id"`
	SourceBackupSHA256 string                     `json:"source_backup_sha256"`
	ContentSHA256      string                     `json:"content_sha256"`
	RecordCount        int                        `json:"record_count"`
	Expected           archive.ProjectionManifest `json:"expected_projection"`
	Actual             archive.ProjectionManifest `json:"actual_projection"`
}

// Import parses and stages one canonical archive into store, then derives the
// actual vNext snapshot and verifies it against the archive manifest. Exact
// fact replays are idempotent, so a failed or interrupted isolated rehearsal
// can be retried without minting replacement identities.
func Import(ctx context.Context, encoded []byte, store *continuitysqlite.Store) (Report, error) {
	report := Report{}
	if ctx == nil {
		return report, fmt.Errorf("import vNext rehearsal archive: context is nil")
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if store == nil {
		return report, fmt.Errorf("import vNext rehearsal archive: destination store is nil")
	}
	sealed, err := archive.Parse(encoded)
	if err != nil {
		return report, fmt.Errorf("import vNext rehearsal archive: %w", err)
	}
	report.Format = sealed.Format
	report.Version = sealed.Version
	report.ProjectID = sealed.Content.Project.ProjectID
	report.SourceBackupSHA256 = sealed.Content.Source.BackupSHA256
	report.ContentSHA256 = sealed.ContentSHA256
	report.RecordCount = len(sealed.Content.Records)
	report.Expected = sealed.Content.Expected
	facts := make([]continuitysqlite.RehearsalFact, 0, len(sealed.Content.Records))
	for index, record := range sealed.Content.Records {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		fact, err := prepareRecordV1(report.ProjectID, record)
		if err != nil {
			return report, fmt.Errorf("import vNext rehearsal archive: prepare record %d (%s): %w", index, record.Kind, err)
		}
		facts = append(facts, fact)
	}
	_, err = store.ImportRehearsalFacts(ctx, report.ProjectID, facts, func(snapshot continuity.Snapshot) error {
		var actual archive.ProjectionManifest
		var verifyErr error
		if sealed.Version == archive.Version2 {
			actual, verifyErr = archive.ProjectionManifestForSnapshotV2(snapshot)
		} else {
			actual, verifyErr = archive.ProjectionManifestForSnapshot(snapshot)
		}
		if verifyErr != nil {
			return fmt.Errorf("derive staged projection: %w", verifyErr)
		}
		report.Actual = actual
		if report.Actual != report.Expected {
			return fmt.Errorf("staged projection does not match archive manifest")
		}
		return nil
	})
	if err != nil {
		return report, fmt.Errorf("import vNext rehearsal archive: stage facts: %w", err)
	}
	return report, nil
}

func prepareRecordV1(projectID continuity.ProjectID, record archive.Record) (continuitysqlite.RehearsalFact, error) {
	return archive.PrepareRecord(projectID, record)
}
