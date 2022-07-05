KIND_CLUSTER_NAME=sidecar-or-not

GOBIN_PATH=$$PWD/bin
GOBIN_LOCAL=export GOBIN=$(GOBIN_PATH)
GOBIN_LOCAL_PATH=export PATH="$(GOBIN_PATH):$$PATH"

tools:
	@$(GOBIN_LOCAL); \
	go install $$(go list -f "{{ join .Imports \" \"}}" tools.go)
.PHONY: tools

setup:
	@brew bundle
	@export diff_installed=$$(helm plugin list | grep diff); \
	if [ -z "$${diff_installed}" ]; then \
		helm plugin install https://github.com/databus23/helm-diff; \
	fi
	@make -s tools
.PHONY: setup

snapshot: protoc
	go run github.com/goreleaser/goreleaser@v1 release --snapshot --rm-dist
.PHONY: snapshot

kind-start:
	@if [[ -z $$($(GOBIN_PATH)/kind get clusters | grep $(KIND_CLUSTER_NAME) ) ]]; then \
		$(GOBIN_PATH)/kind create cluster --name $(KIND_CLUSTER_NAME); \
	fi
.PHONY: kind-start

kind-prep:
	$(GOBIN_PATH)/kind load docker-image yardbirdsax/sidecar-server:latest --name $(KIND_CLUSTER_NAME)
	$(GOBIN_PATH)/kind load docker-image yardbirdsax/sidecar-sidecar:latest --name $(KIND_CLUSTER_NAME)
	$(GOBIN_PATH)/kind load docker-image yardbirdsax/sidecar-sidecar-grpc:latest --name $(KIND_CLUSTER_NAME)
	$(GOBIN_PATH)/kind load docker-image yardbirdsax/sidecar-client:latest --name $(KIND_CLUSTER_NAME)
.PHONY: kind-prep

kind-stop:
	$(GOBIN_PATH)/kind delete cluster --name $(KIND_CLUSTER_NAME)
.PHONY: kind-stop

k8s-delete:
	kubectl delete -f kubernetes/ --ignore-not-found
.PHONY: k8s-delete

k8s-apply:
	kubectl apply -f kubernetes/
.PHONY: kind-apply

helmfile-apply:
	go run github.com/helmfile/helmfile@main -f helmfile/helmfile.yml apply

kind-build-apply: snapshot kind-prep k8s-delete k8s-apply
.PHONY: kind-build-apply

start-experiment-kind: kind-start kind-build-apply helmfile-apply
.PHONY: start-test

stop-experiment-kind: kind-stop
.PHONY: kind-stop

protoc:
	@$(GOBIN_LOCAL_PATH); \
	protoc --go_out=. \
				 --go_opt=module=github.com/yardbirdsax/to-sidecar-or-not \
				 --go-grpc_out=. \
				 --go-grpc_opt=module=github.com/yardbirdsax/to-sidecar-or-not \
				 proto/adder.proto