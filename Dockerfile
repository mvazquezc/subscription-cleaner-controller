FROM golang:latest
WORKDIR /go/src/github.com/mvazquezc/subscription-cleaner-controller/
RUN go get k8s.io/apimachinery/pkg/apis/meta/v1 k8s.io/apimachinery/pkg/apis/meta/v1/unstructured k8s.io/apimachinery/pkg/runtime/schema k8s.io/client-go/dynamic k8s.io/client-go/tools/clientcmd k8s.io/client-go/util/homedir k8s.io/client-go/rest
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o subscription-cleaner-controller .

FROM scratch
COPY --from=0 /go/src/github.com/mvazquezc/subscription-cleaner-controller/subscription-cleaner-controller .
CMD ["/subscription-cleaner-controller"]
