.PHONY: setup-db
setup-db:
	bash ./scripts/setup_sqlite3.sh

.PHONY: open-db
open-db:
	sqlite3 db/mercari-build-training.db