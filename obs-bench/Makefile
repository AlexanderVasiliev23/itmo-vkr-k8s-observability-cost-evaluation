BINARY := obs-bench
export MINIKUBE_PATH ?= $(shell which minikube 2>/dev/null || echo /home/linuxbrew/.linuxbrew/bin/minikube)

.PHONY: build run-smoke run-predefense clean

build:
	go build -o $(BINARY) ./cmd

run-smoke: build
	./$(BINARY) run-batch --file experiments-smoke.yaml

run-predefense: build
	./$(BINARY) run-batch --file experiments-predefense.yaml

clean:
	rm -f $(BINARY)
