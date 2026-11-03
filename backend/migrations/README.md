# Migrations

SQL migrations managed by [goose](https://github.com/pressly/goose), applied with
`db-migrate` from the devenv shell. Files are named `NNNNN_description.sql` and use
`-- +goose Up` / `-- +goose Down` sections. The schema is intentionally empty until the
event model from `docs/RESEARCH.md` section 4 is prototyped.
