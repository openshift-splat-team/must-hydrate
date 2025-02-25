FROM registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.23-openshift-4.19 AS builder
WORKDIR /go/src/github.com/openshift-splat-team/must-hydrate
COPY . .
RUN NO_DOCKER=1 make build

FROM registry.ci.openshift.org/ocp/4.19:base-rhel9
WORKDIR /go/src/github.com/openshift-splat-team/must-hydrate
COPY --from=builder /go/src/github.com/openshift-splat-team/must-hydrate/bin/must_hydrate .
RUN wget https://github.com/kubernetes-sigs/controller-runtime/releases/download/v0.20.2/setup-envtest-linux-amd64
RUN chmod +x ./setup-envtest-linux-amd64
RUN ./setup-envtest-linux-amd64 use --bin-dir envtest
ENV KUBEBUILDER_ASSETS=/go/src/github.com/openshift-splat-team/must-hydrate/envtest/k8s/1.32.0-linux-amd64
ENV KUBECONFIG=/go/src/github.com/openshift-splat-team/must-hydrate/envtest.kubeconfig
CMD ./must_hydrate