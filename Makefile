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
	curl -X POST --url 'http://localhost:9000/items' -d name=new_item_book_0 -d category=book -d image=images/book.jpg