package annotate

import (
	"encoding/json"
	"github.com/logzio/ezkonnect-server/api"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
	"strings"
)

const (
	InstrumentationAnnotation = "logz.io/traces_instrument"
	ServiceNameAnnotation     = "logz.io/service-name"
)

// TracesResourceRequest ResourceRequest is the JSON body of the POST request
// It contains the name, kind, namespace, telemetry type and action of the resource
// name: name of the resource
// kind: kind of the resource (deployment or statefulset) consts defined at `common.go` (api.KindDeployment, api.KindStatefulSet)
// namespace: namespace of the resource
// action: action to perform (add or delete) consts defined at `common.go` (api.ActionAdd, api.ActionDelete)
// service_name: name of the service
type TracesResourceRequest struct {
	Name        string `json:"name"`
	Kind        string `json:"controller_kind"`
	Namespace   string `json:"namespace"`
	Action      string `json:"action"`
	ServiceName string `json:"service_name"`
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
		logger.Error(api.ErrorDecodeJSON, err)
		http.Error(w, api.ErrorDecodeJSON+err.Error(), http.StatusBadRequest)
		return
	}
	// Get the Kubernetes config
	config, err := api.GetConfig()
	if err != nil {
		logger.Error(api.ErrorKubeConfig, err)
		http.Error(w, api.ErrorKubeConfig+err.Error(), http.StatusInternalServerError)
		return
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Error(api.ErrorKubeClient, err)
		http.Error(w, api.ErrorKubeClient+err.Error(), http.StatusInternalServerError)
		return
	}

	// Validate input before updating resources to avoid changing resources and retuning an error
	// if one of the requests is invalid, return an error
	if !validateTracesResourceRequests(resources) {
		logger.Error(api.ErrorInvalidInput)
		http.Error(w, api.ErrorInvalidInput, http.StatusBadRequest)
		return
	}

	var responses []TracesResourceResponse
	for _, resource := range resources {
		// choose the annotation key and value according to the telemetry type and action
		actionValue := "true"
		if resource.Action == api.ActionDelete {
			actionValue = "rollback"
		}
		annotations := map[string]string{}
		annotations[InstrumentationAnnotation] = actionValue
		// add service name annotation if exists
		if resource.ServiceName != "" {
			annotations[ServiceNameAnnotation] = resource.ServiceName
		}

		// Create the response
		response := TracesResourceResponse{
			Name:               resource.Name,
			Namespace:          resource.Namespace,
			Kind:               resource.Kind,
			UpdatedAnnotations: annotations,
		}

		switch resource.Kind {
		case api.KindDeployment:
			logger.Info("Updating deployment with instrumentation annotation", resource.Name)
			deployment, err := clientset.AppsV1().Deployments(resource.Namespace).Get(r.Context(), resource.Name, v1.GetOptions{})
			if err != nil {
				logger.Error(api.ErrorGet, err)
				http.Error(w, api.ErrorGet+err.Error(), http.StatusInternalServerError)
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
				logger.Error(api.ErrorUpdate, err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			responses = append(responses, response)

		case api.KindStatefulSet:
			logger.Info("Updating statefulset with instrumentation annotation ", resource.Name)
			statefulSet, err := clientset.AppsV1().StatefulSets(resource.Namespace).Get(r.Context(), resource.Name, v1.GetOptions{})
			if err != nil {
				logger.Error(api.ErrorGet, err)
				http.Error(w, api.ErrorGet+err.Error(), http.StatusInternalServerError)
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
				logger.Error(api.ErrorUpdate, err)
				http.Error(w, api.ErrorUpdate+err.Error(), http.StatusInternalServerError)
				return
			}

			responses = append(responses, response)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responses)
}

func validateTracesResourceRequests(resources []TracesResourceRequest) bool {
	for _, resource := range resources {
		if !isValidTracesResourceRequest(resource) {
			return false
		}
	}
	return true
}

func isValidTracesResourceRequest(req TracesResourceRequest) bool {
	isValidAction := false
	isValidKind := false
	for _, validAction := range api.ValidActions {
		if req.Action == strings.ToLower(validAction) {
			isValidAction = true
		}
	}
	for _, validKind := range api.ValidKinds {
		if req.Kind == strings.ToLower(validKind) {
			isValidKind = true
		}
	}
	return isValidKind && isValidAction
}
