package migration

import "database/sql"

func Migrate(postgres *sql.DB) error {

	_, err := postgres.Exec(`
		CREATE TABLE IF NOT EXISTS cron_details (
			queue_id TEXT PRIMARY KEY,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP,
			job_id TEXT,
			cron TEXT,
			job_payload TEXT,
			exchange_name TEXT
		);
	`)
	if err != nil {
		return err
	}

	return nil
}
