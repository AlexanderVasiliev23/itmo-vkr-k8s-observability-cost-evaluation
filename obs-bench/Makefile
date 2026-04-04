BINARY := obs-bench

.PHONY: build run-smoke run-predefense clean

build:
	go build -o $(BINARY) ./cmd

run-smoke: build
	./$(BINARY) run-batch --file experiments-smoke.yaml

run-predefense: build
	./$(BINARY) run-batch --file experiments-predefense.yaml

clean:
	rm -f $(BINARY)
