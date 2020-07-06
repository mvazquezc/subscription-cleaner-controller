build: get_dependencies
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o subscription-cleaner-controller .	
run: get_dependencies
	go run main.go
get-dependencies:
	go get k8s.io/apimachinery/pkg/apis/meta/v1 k8s.io/apimachinery/pkg/apis/meta/v1/unstructured k8s.io/apimachinery/pkg/runtime/schema k8s.io/client-go/dynamic k8s.io/client-go/tools/clientcmd k8s.io/client-go/util/homedir k8s.io/client-go/rest
build-image:
	podman build . -t quay.io/mavazque/subscription-cleaner-controller:latest	
push-image:
	podman push quay.io/mavazque/subscription-cleaner-controller:latest	
build-and-push-image: build-image push-image
