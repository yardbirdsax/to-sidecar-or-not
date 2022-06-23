KIND_VERSION=0.14.0
KIND_PATH=sigs.k8s.io/kind@v$(KIND_VERSION)
KIND_CLUSTER_NAME=sidecar-or-not

setup:
	brew bundle
	export diff_installed=$$(helm plugin list | grep diff); \
	if [ -z "$${diff_installed}" ]; then \
		helm plugin install https://github.com/databus23/helm-diff; \
	fi

snapshot:
	go run github.com/goreleaser/goreleaser@v1 release --snapshot --rm-dist
.PHONY: snapshot

kind-start:
	@if [[ -z $$(go run $(KIND_PATH) get clusters | grep $(KIND_CLUSTER_NAME) ) ]]; then \
		go run $(KIND_PATH) create cluster --name $(KIND_CLUSTER_NAME); \
	fi
.PHONY: kind-start

kind-prep:
	go run $(KIND_PATH) load docker-image yardbirdsax/sidecar-server:latest --name $(KIND_CLUSTER_NAME)
	go run $(KIND_PATH) load docker-image yardbirdsax/sidecar-sidecar:latest --name $(KIND_CLUSTER_NAME)
	go run $(KIND_PATH) load docker-image yardbirdsax/sidecar-client:latest --name $(KIND_CLUSTER_NAME)
.PHONY: kind-prep

kind-stop:
	go run $(KIND_PATH) delete cluster --name $(KIND_CLUSTER_NAME)
.PHONY: kind-stop

k8s-delete:
	kubectl delete -f kubernetes/ --ignore-not-found
.PHONY: k8s-delete

k8s-apply:
	kubectl apply -f kubernetes/
.PHONY: kind-apply

helmfile-apply:
	go run github.com/helmfile/helmfile@main -f helmfile/helmfile.yml apply

kind-build-apply: snapshot kind-prep k8s-apply
.PHONY: kind-build-apply

start-test: kind-start kind-build-apply helmfile-apply
.PHONY: start-test