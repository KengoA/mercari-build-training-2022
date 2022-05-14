.PHONY: setup-db
setup-db:
	bash ./scripts/setup_sqlite3.sh

.PHONY: open-db
open-db:
	sqlite3 db/mercari-build-training.db

.PHONY: build-go-image
build-go-image:
	docker build . -t build2022/app -f dockerfiles/go/Dockerfile

.PHONY: run-go-image
run-go-image:
	docker run -d -p 9000:9000 build2022/app:latest

.PHONY: add-test-item-with-image
add-test-item-with-image:
	curl -X POST --url 'http://localhost:9000/items' -F name=new-item -F category=book -F image=@image/book.jpg