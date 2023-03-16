package annotate

import (
	"encoding/json"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"net/http"
	"os"
	"path/filepath"
)

// ResourceRequest is the JSON body of the POST request
// It contains the name, kind, namespace, telemetry type and action of the resource
// name: name of the resource
// kind: kind of the resource (deployment or statefulset)
// namespace: namespace of the resource
// telemetry_type: type of telemetry (metrics or traces)
// action: action to perform (add or delete)

type ResourceRequest struct {
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Namespace     string `json:"namespace"`
	TelemetryType string `json:"telemetry_type"`
	Action        string `json:"action"`
}

// ResourceResponse is the JSON response of the POST request
// It contains the name, kind, namespace and updated annotations of the resource
// name: name of the resource
// kind: kind of the resource (deployment or statefulset)
// namespace: namespace of the resource
// updated_annotations: updated annotations of the resource
type ResourceResponse struct {
	Name               string            `json:"name"`
	Namespace          string            `json:"namespace"`
	Kind               string            `json:"kind"`
	UpdatedAnnotations map[string]string `json:"updated_annotations"`
}

func UpdateResourceAnnotations(w http.ResponseWriter, r *http.Request) {
	// Decode JSON body
	var resources []ResourceRequest
	err := json.NewDecoder(r.Body).Decode(&resources)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Get the Kubernetes config
	config, err := getConfig()
	if err != nil {
		http.Error(w, "Error getting Kubernetes config", http.StatusInternalServerError)
		return
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var responses []ResourceResponse
	for _, resource := range resources {
		// Validate input
		if !isValidResourceRequest(resource) {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
		// choose the annotation key and value according to the telemetry type and action
		var annotationKey string
		if resource.TelemetryType == "metrics" {
			annotationKey = "logz.io/export-metrics"
		} else {
			annotationKey = "logz.io/instrument"
		}
		value := "true"
		if resource.Action == "delete" {
			value = "false"
		}

		annotations := map[string]string{
			annotationKey: value,
		}

		// Create the response
		response := ResourceResponse{
			Name:               resource.Name,
			Namespace:          resource.Namespace,
			Kind:               resource.Kind,
			UpdatedAnnotations: annotations,
		}

		switch resource.Kind {
		case "deployment":
			deployment, err := clientset.AppsV1().Deployments(resource.Namespace).Get(r.Context(), resource.Name, v1.GetOptions{})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			for k, v := range annotations {
				deployment.Spec.Template.ObjectMeta.Annotations[k] = v
			}

			_, err = clientset.AppsV1().Deployments(resource.Namespace).Update(r.Context(), deployment, v1.UpdateOptions{})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			responses = append(responses, response)

		case "statefulset":
			statefulSet, err := clientset.AppsV1().StatefulSets(resource.Namespace).Get(r.Context(), resource.Name, v1.GetOptions{})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			for k, v := range annotations {
				statefulSet.Spec.Template.ObjectMeta.Annotations[k] = v
			}

			_, err = clientset.AppsV1().StatefulSets(resource.Namespace).Update(r.Context(), statefulSet, v1.UpdateOptions{})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			responses = append(responses, response)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responses)
}

func isValidResourceRequest(req ResourceRequest) bool {
	validKinds := []string{"deployment", "statefulset"}
	validTelemetryTypes := []string{"metrics", "traces"}
	validActions := []string{"add", "delete"}

	return contains(validKinds, req.Kind) &&
		contains(validTelemetryTypes, req.TelemetryType) &&
		contains(validActions, req.Action)
}

func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
		}
		return true
	}
	return false
}

func getConfig() (*rest.Config, error) {
	var config *rest.Config

	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	if _, err := os.Stat(kubeconfig); err == nil {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}
