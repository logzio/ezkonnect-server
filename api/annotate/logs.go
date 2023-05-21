package annotate

import (
	"encoding/json"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
)

// LogsResourceRequest is the JSON body of the POST request
// It contains the name, controller_kind, namespace, and log type of the resource
// name: name of the resource
// controller_kind: kind of the resource (deployment or statefulset)
// namespace: namespace of the resource
// log_type: desired log type
type LogsResourceRequest struct {
	Name      string `json:"name"`
	Kind      string `json:"controller_kind"`
	Namespace string `json:"namespace"`
	LogType   string `json:"log_type"`
}

// LogsResourceResponse is the JSON response of the POST request
// It contains the name, kind, namespace and updated annotations of the resource
// name: name of the resource
// kind: kind of the resource (deployment or statefulset)
// namespace: namespace of the resource
// updated_annotations: updated annotations of the resource
type LogsResourceResponse struct {
	Name               string            `json:"name"`
	Namespace          string            `json:"namespace"`
	Kind               string            `json:"controller_kind"`
	UpdatedAnnotations map[string]string `json:"updated_annotations"`
}

func UpdateLogsResourceAnnotations(w http.ResponseWriter, r *http.Request) {
	// Decode JSON body
	var resources []LogsResourceRequest
	err := json.NewDecoder(r.Body).Decode(&resources)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get the Kubernetes config
	config, err := GetConfig()
	if err != nil {
		http.Error(w, "Error getting Kubernetes config", http.StatusInternalServerError)
		return
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var responses []LogsResourceResponse
	for _, resource := range resources {
		// Validate input
		if !isValidLogsResourceRequest(resource) {
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
		// Set the annotation key and value according to the telemetry type and action
		var annotationKey = "logz.io/application_type"
		value := resource.LogType

		annotations := map[string]string{
			annotationKey: value,
		}

		// Create the response
		response := LogsResourceResponse{
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
				if deployment.Spec.Template.ObjectMeta.Annotations == nil {
					deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
				}
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
				if statefulSet.Spec.Template.ObjectMeta.Annotations == nil {
					statefulSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
				}
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

func isValidLogsResourceRequest(req LogsResourceRequest) bool {
	validKinds := []string{"deployment", "statefulset"}
	return contains(validKinds, req.Kind)
}
