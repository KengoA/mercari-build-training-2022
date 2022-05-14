# brew install sqlite
# go get github.com/mattn/go-sqlite3
cd db
sqlite3 mercari-build-training.db "CREATE TABLE items (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, name VARCHAR(255) NOT NULL, category VARCHAR(255), image_filename text);"
sqlite3 mercari-build-training.db .schema > schema.sql