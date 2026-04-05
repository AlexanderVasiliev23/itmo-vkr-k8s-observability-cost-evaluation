DB ?= ../obs-bench/test.db
PORT ?= 8080

.PHONY: run build

build:
	cd server && go build -o obs-advisor .

run: build
	./server/obs-advisor --db $(DB) --port $(PORT)
