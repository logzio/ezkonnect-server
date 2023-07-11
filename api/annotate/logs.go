package annotate

import (
	"context"
	"encoding/json"
	"github.com/logzio/ezkonnect-server/api"
	"go.uber.org/zap"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"net/http"
	"strings"
	"time"
)

const (
	LogTypeAnnotation = "logz.io/application_type"
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
// kind: kind of the resource (deployment or statefulset) consts defined at `common.go` (api.KindDeployment, api.KindStatefulSet)
// namespace: namespace of the resource
// updated_annotations: updated annotations of the resource
type LogsResourceResponse struct {
	Name               string            `json:"name"`
	Namespace          string            `json:"namespace"`
	Kind               string            `json:"controller_kind"`
	UpdatedAnnotations map[string]string `json:"updated_annotations"`
}

func UpdateLogsResourceAnnotations(w http.ResponseWriter, r *http.Request) {
	logger := api.InitLogger()
	// Decode JSON body
	var resources []LogsResourceRequest
	err := json.NewDecoder(r.Body).Decode(&resources)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get the Kubernetes config
	config, err := api.GetConfig()
	if err != nil {
		logger.Error(api.ErrorKubeConfig, err)
		http.Error(w, api.ErrorKubeConfig, http.StatusInternalServerError)
		return
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Create a dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		logger.Error(api.ErrorDynamic, zap.Error(err))
		http.Error(w, api.ErrorDynamic+err.Error(), http.StatusInternalServerError)
		return
	}

	gvr := schema.GroupVersionResource{
		Group:    api.ResourceGroup,
		Version:  api.ResourceVersion,
		Resource: api.ResourceInstrumentedApplication,
	}

	// Validate input before updating resources to avoid changing resources and retuning an error
	validRequests := validateLogsResourceRequests(resources)
	// if one of the requests is invalid, return an error
	if !validRequests {
		logger.Error(api.ErrorInvalidInput)
		http.Error(w, api.ErrorInvalidInput, http.StatusBadRequest)
		return
	}
	// Define timeout for the context
	ctxDuration, err := api.GetTimeout()
	if err != nil {
		logger.Error(api.ErrorInvalidInput, err)
		http.Error(w, api.ErrorInvalidInput+err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), ctxDuration)
	defer cancel()
	// Update the resources
	var responses []LogsResourceResponse
	for _, resource := range resources {
		// Create a channel to signal when a crd status is updated
		updateCh := make(chan struct{})
		// Create a dynamic factory that watches for changes in the InstrumentedApplication CRD corresponding to the resource
		dynamicFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicClient, 1*time.Second, resource.Namespace, func(options *v1.ListOptions) {
			options.FieldSelector = "metadata.name=" + resource.Name
		})
		informer := dynamicFactory.ForResource(gvr)
		// handle updates and compare the old and new status
		informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj, newObj interface{}) {
				newSpec := newObj.(*unstructured.Unstructured).Object["spec"].(map[string]interface{})
				oldSpec := oldObj.(*unstructured.Unstructured).Object["spec"].(map[string]interface{})
				if !api.DeepEqualMap(oldSpec, newSpec) {
					updateCh <- struct{}{} // Signal that the update occurred
				}
			},
		})
		// start watching for changes
		dynamicFactory.Start(ctx.Done())

		value := resource.LogType
		annotations := map[string]string{
			LogTypeAnnotation: value,
		}

		// Create the response
		response := LogsResourceResponse{
			Name:               resource.Name,
			Namespace:          resource.Namespace,
			Kind:               resource.Kind,
			UpdatedAnnotations: annotations,
		}
		switch resource.Kind {
		case api.KindDeployment:
			logger.Info("Updating deployment: ", resource.Name)
			deployment, err := clientset.AppsV1().Deployments(resource.Namespace).Get(r.Context(), resource.Name, v1.GetOptions{})
			if err != nil {
				logger.Error(api.ErrorGet, err)
				http.Error(w, api.ErrorGet+err.Error(), http.StatusInternalServerError)
				return
			}

			if deployment.Spec.Template.ObjectMeta.Annotations == nil {
				deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}

			if len(value) != 0 {
				deployment.Spec.Template.ObjectMeta.Annotations[LogTypeAnnotation] = value
			} else {
				delete(deployment.Spec.Template.ObjectMeta.Annotations, LogTypeAnnotation)
			}

			_, err = clientset.AppsV1().Deployments(resource.Namespace).Update(r.Context(), deployment, v1.UpdateOptions{})
			if err != nil {
				logger.Error(api.ErrorUpdate, err)
				http.Error(w, api.ErrorUpdate+err.Error(), http.StatusInternalServerError)
				return
			}

			responses = append(responses, response)

		case api.KindStatefulSet:
			logger.Info("Updating statefulset: ", resource.Name)
			statefulSet, err := clientset.AppsV1().StatefulSets(resource.Namespace).Get(r.Context(), resource.Name, v1.GetOptions{})
			if err != nil {
				logger.Error(api.ErrorGet, err)
				http.Error(w, api.ErrorGet+err.Error(), http.StatusInternalServerError)
				return
			}

			if statefulSet.Spec.Template.ObjectMeta.Annotations == nil {
				statefulSet.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			}

			if len(value) != 0 {
				statefulSet.Spec.Template.ObjectMeta.Annotations[LogTypeAnnotation] = value
			} else {
				delete(statefulSet.Spec.Template.ObjectMeta.Annotations, LogTypeAnnotation)
			}

			_, err = clientset.AppsV1().StatefulSets(resource.Namespace).Update(r.Context(), statefulSet, v1.UpdateOptions{})
			if err != nil {
				logger.Error(api.ErrorUpdate, err)
				http.Error(w, api.ErrorUpdate+err.Error(), http.StatusInternalServerError)
				return
			}

			responses = append(responses, response)
		}
		// Wait for the update to occur or timeout
		select {
		case <-updateCh:
			logger.Info("crd instrumentation status changed: ", resource.Name)

		case <-ctx.Done():
			logger.Error(api.ErrorTimeout + resource.Name)
			http.Error(w, api.ErrorTimeout+resource.Name, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responses)
}

func isValidLogsResourceRequest(req LogsResourceRequest) bool {
	for _, validKind := range api.ValidKinds {
		if req.Kind == strings.ToLower(validKind) {
			return true
		}
	}
	return false
}

func validateLogsResourceRequests(resources []LogsResourceRequest) bool {
	for _, resource := range resources {
		if !isValidLogsResourceRequest(resource) {
			return false
		}
	}
	return true
}
