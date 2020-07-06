package main

import (
    "context"
    "flag"
    "path/filepath"
    "time"
    "log"

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

    // "" namespace means list all namespaces
    namespace := flag.String("namespace", "", "Namespace to filter subscriptions, default is all namespaces")
    subscriptionTTL := flag.Int64("ttl", 24, "Time to wait before deleting a subscription (in hours), default is 24 hours")

    flag.Parse()

    // Protected namespaces
    protectedNamespaces := []string{"kube-system", "open-cluster-management"}

    // Try to get the config from the in-cluster client
    config, err := rest.InClusterConfig()
    if err != nil {
        log.Println("Error getting in-cluster config. Trying to get config from local kubeconfig or flags")
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

    if *namespace == "" {
        log.Printf("Starting subscription-cleaner-controller: Listening to all namespaces, deleting subscriptions older than %d hours\n", *subscriptionTTL)
    } else {
        log.Printf("Starting subscription-cleaner-controller: Listening to namespace %s, deleting subcriptions older than %d hours\n", *namespace, *subscriptionTTL)
    }
      

    // Main loop
    for {
        // Get all Subscriptions
        list, err := client.Resource(subscriptionRes).Namespace(*namespace).List(context.TODO(), metav1.ListOptions{})
        if err != nil {
            panic(err)
        }
        for _, d := range list.Items {
            // Extract namespace
            subNamespace, _, err := unstructured.NestedString(d.Object, "metadata", "namespace")
            if err != nil {
                log.Printf("Error getting namespace %v", err)
                continue
            }
            // Get creationtimestamp
            subCreationTimestamp, _, err := unstructured.NestedString(d.Object, "metadata", "creationTimestamp")
            if err != nil {
                log.Printf("Error getting creationTimestamp %v", err)
                continue
            }
            // Get current time in RFC3339 format
            currentTime := time.Now()
            currentTimeUnix := time.Now().Unix()
            currentTimeRFC3339 := currentTime.Format(time.RFC3339)
            // Transform creationTimestap to Unix time
            t, err := time.Parse(time.RFC3339, subCreationTimestamp)
            subCreationTimestampUnix := t.Unix()
            if err != nil {
               log.Printf("Error getting unix time for the subscription creationTimestamp")
               continue
            }
            // Check how much time passed since the sub was created (in hours)
            timeSinceCreated := (currentTimeUnix - subCreationTimestampUnix) / 60 / 60
            log.Printf("Found subscripion %s in namespace %s created on %s. Current time: %s, Created %d hours ago\n", d.GetName(), subNamespace, subCreationTimestamp, currentTimeRFC3339, timeSinceCreated)
            // If subscription has been active for more than configured TTL, delete the subscription
            if timeSinceCreated >= *subscriptionTTL {
                // Check if subscription is created in a protected namespace
                if contains(protectedNamespaces, subNamespace) {
                    log.Printf("Subscription %s is created in protected namespace %s, skipping deletion\n", d.GetName(), subNamespace)
                } else {
                    deletePolicy := metav1.DeletePropagationForeground
                    deleteOptions := metav1.DeleteOptions{PropagationPolicy: &deletePolicy}
                    err := client.Resource(subscriptionRes).Namespace(subNamespace).Delete(context.TODO(), d.GetName(), deleteOptions)
                    if err != nil {
                        log.Printf("Error deleting subscription %s in namespace %s\n", d.GetName(), subNamespace)
                    } else {
                        log.Printf("Deleted subscription %s in namespace %s\n", d.GetName(), subNamespace)
                    }
                }
            }
        }
        time.Sleep(5 * time.Minute)
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
