package annotate

import (
	"encoding/json"
	"github.com/logzio/ezkonnect-server/api"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
)

// TracesResourceRequest ResourceRequest is the JSON body of the POST request
// It contains the name, kind, namespace, telemetry type and action of the resource
// name: name of the resource
// kind: kind of the resource (deployment or statefulset)
// namespace: namespace of the resource
// action: action to perform (add or delete)
type TracesResourceRequest struct {
	Name      string `json:"name"`
	Kind      string `json:"controller_kind"`
	Namespace string `json:"namespace"`
	Action    string `json:"action"`
}

// TracesResourceResponse  is the JSON response of the POST request
// It contains the name, kind, namespace and updated annotations of the resource
// name: name of the resource
// kind: kind of the resource (deployment or statefulset)
// namespace: namespace of the resource
// updated_annotations: updated annotations of the resource
type TracesResourceResponse struct {
	Name               string            `json:"name"`
	Namespace          string            `json:"namespace"`
	Kind               string            `json:"controller_kind"`
	UpdatedAnnotations map[string]string `json:"updated_annotations"`
}

func UpdateTracesResourceAnnotations(w http.ResponseWriter, r *http.Request) {
	logger := api.InitLogger()
	// Decode JSON body
	var resources []TracesResourceRequest
	err := json.NewDecoder(r.Body).Decode(&resources)
	if err != nil {
		logger.Error("Error decoding JSON body", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Get the Kubernetes config
	config, err := api.GetConfig()
	if err != nil {
		logger.Error("Error getting Kubernetes config", err)
		http.Error(w, "Error getting Kubernetes config", http.StatusInternalServerError)
		return
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error("Error creating Kubernetes clientset", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var responses []TracesResourceResponse
	for _, resource := range resources {
		// Validate input
		if !isValidTracesResourceRequest(resource) {
			logger.Error("Invalid input", err)
			http.Error(w, "Invalid input", http.StatusBadRequest)
			return
		}
		// choose the annotation key and value according to the telemetry type and action
		var annotationKey = "logz.io/traces_instrument"
		value := "true"
		if resource.Action == "delete" {
			value = "rollback"
		}

		annotations := map[string]string{
			annotationKey: value,
		}

		// Create the response
		response := TracesResourceResponse{
			Name:               resource.Name,
			Namespace:          resource.Namespace,
			Kind:               resource.Kind,
			UpdatedAnnotations: annotations,
		}

		switch resource.Kind {
		case "deployment":
			deployment, err := clientset.AppsV1().Deployments(resource.Namespace).Get(r.Context(), resource.Name, v1.GetOptions{})
			if err != nil {
				logger.Error("Error getting deployment", err)
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
				logger.Error("Error updating deployment", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			responses = append(responses, response)

		case "statefulset":
			statefulSet, err := clientset.AppsV1().StatefulSets(resource.Namespace).Get(r.Context(), resource.Name, v1.GetOptions{})
			if err != nil {
				logger.Error("Error getting statefulset", err)
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
				logger.Error("Error updating statefulset", err)
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

func isValidTracesResourceRequest(req TracesResourceRequest) bool {
	validKinds := []string{"deployment", "statefulset"}
	validActions := []string{"add", "delete"}

	return api.Contains(validKinds, req.Kind) &&
		api.Contains(validActions, req.Action)
}
