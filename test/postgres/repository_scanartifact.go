package postgres

import (
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/quay/claircore"
	"github.com/quay/claircore/internal/indexer"
)

func InsertRepoScanArtifact(db *sqlx.DB, layerHash claircore.Digest, repos []*claircore.Repository, scnrs indexer.VersionedScanners) error {
	query := `
	WITH layer_insert AS (
		INSERT INTO layer (hash)
			VALUES ($1)
			ON CONFLICT DO UPDATE SET hash=EXCLUDED.hash
			RETURNING id AS layer_id
	)
	INSERT INTO repo_scanartifact (layer_id, repo_id, scanner_id) VALUES ((SELECT layer_id FROM layer_insert),
																		  $2,
																		  $3)
	`

	insertLayer := `
	INSERT INTO layer (hash)
	VALUES ($1);
	`

	_, err := db.Exec(insertLayer, &layerHash)
	if err != nil {
		return fmt.Errorf("failed to insert layer %v", err)
	}

	n := len(scnrs)
	for i, repo := range repos {
		nn := i % n
		_, err := db.Exec(query, &layerHash, &repo.ID, &nn)
		if err != nil {
			return fmt.Errorf("failed to insert repo scan artifact: %v", err)
		}
	}

	return nil
}
