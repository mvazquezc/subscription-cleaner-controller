package main

import (
    "context"
    "flag"
    "fmt"
    "path/filepath"
    "time"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "k8s.io/client-go/dynamic"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/client-go/util/homedir"
    "k8s.io/client-go/rest"

)

func main() {       

    var kubeconfig *string

    if home := homedir.HomeDir(); home != "" {
        kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
    } else {
        kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
    }
    flag.Parse()      

    // All namespaces
    namespace := ""

    // Protected namespaces
    protectedNamespaces := []string{"kube-system", "open-cluster-management"}

    // Try to get the config from the in-cluster client
    config, err := rest.InClusterConfig()
    if err != nil {
        fmt.Println("Error getting in-cluster config. Trying to get config from local kubeconfig or flags")
        config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
        if err != nil {
            panic(err)
        }
    }

    client, err := dynamic.NewForConfig(config)
    if err != nil {
        panic(err)
    }

    // Subscription Resource 
    subscriptionRes := schema.GroupVersionResource{Group: "apps.open-cluster-management.io", Version: "v1", Resource: "subscriptions"}
    
    // Subscription TimeToLive (in hours)
    var subscriptionTTL int64 = 1

    fmt.Println("Listing subscriptions in all namespaces")
        
    // Main loop
    for {
        // Get all Subscriptions
        list, err := client.Resource(subscriptionRes).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
        if err != nil {
            panic(err)
        }
        for _, d := range list.Items {
            // Extract namespace
            subNamespace, _, err := unstructured.NestedString(d.Object, "metadata", "namespace")
            if err != nil {
                fmt.Errorf("Error getting namespace %v", err)
            }
            // Get creationtimestamp
            subCreationTimestamp, _, err := unstructured.NestedString(d.Object, "metadata", "creationTimestamp")
            if err != nil {
                fmt.Errorf("Error getting creationTimestamp %v", err)
            }
            // Get current time in RFC3339 format
            currentTime := time.Now()
            currentTimeUnix := time.Now().Unix()
            currentTimeRFC3339 := currentTime.Format(time.RFC3339)
            // Transform creationTimestap to Unix time
            t, err := time.Parse(time.RFC3339, subCreationTimestamp)
            subCreationTimestampUnix := t.Unix()
            if err != nil {
               fmt.Errorf("Error getting unix time for the subscription creationTimestamp")
            }
            // Check how much time passed since the sub was created (in hours)
            timeSinceCreated := (currentTimeUnix - subCreationTimestampUnix) / 60 / 60
            fmt.Printf("Found subscripion %s in namespace %s created on %s. Current time: %s, Created %d hours ago\n", d.GetName(), subNamespace, subCreationTimestamp, currentTimeRFC3339, timeSinceCreated)
            // If subscription has been active for more than configured TTL, delete the subscription
            if timeSinceCreated >= subscriptionTTL {
                // Check if subscription is created in a protected namespace
                if contains(protectedNamespaces, subNamespace) {
                    fmt.Printf("Subscription %s is created in protected namespace %s, skipping deletion\n", d.GetName(), subNamespace)
                } else {
                    deletePolicy := metav1.DeletePropagationForeground
                    deleteOptions := metav1.DeleteOptions{PropagationPolicy: &deletePolicy}
                    err := client.Resource(subscriptionRes).Namespace(subNamespace).Delete(context.TODO(), d.GetName(), deleteOptions)
                    if err != nil {
                        fmt.Printf("Error deleting subscription %s in namespace %s\n", d.GetName(), subNamespace)
                    } else {
                        fmt.Printf("Deleted subscription %s in namespace %s\n", d.GetName(), subNamespace)
                    }
                }
            }
        }
        time.Sleep(1 * time.Minute)
    }
}

func contains(s []string, e string) bool {
    for _, a := range s {
        if a == e {
            return true
        }
    }
    return false
}