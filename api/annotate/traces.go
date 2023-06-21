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
	// if one of the requests is invalid, return an error
	if !validateTracesResourceRequests(resources) {
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
	var responses []TracesResourceResponse
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
				newStatus := newObj.(*unstructured.Unstructured).Object["status"].(map[string]interface{})
				oldStatus := oldObj.(*unstructured.Unstructured).Object["status"].(map[string]interface{})
				if !api.DeepEqualMap(oldStatus, newStatus) {
					updateCh <- struct{}{} // Signal that the update occurred
				}
			},
		})
		// start watching for changes
		dynamicFactory.Start(ctx.Done())
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
			logger.Info("Updating deployment: ", resource.Name)
			deployment, err := clientset.AppsV1().Deployments(resource.Namespace).Get(r.Context(), resource.Name, v1.GetOptions{})
			if err != nil {
				logger.Error(api.ErrorGet, err)
				http.Error(w, api.ErrorGet+err.Error(), http.StatusInternalServerError)
				return
			}
			// Update the annotations
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
			// success add to responses
			responses = append(responses, response)

		case api.KindStatefulSet:
			logger.Info("Updating statefulset: ", resource.Name)
			statefulSet, err := clientset.AppsV1().StatefulSets(resource.Namespace).Get(r.Context(), resource.Name, v1.GetOptions{})
			if err != nil {
				logger.Error(api.ErrorGet, err)
				http.Error(w, api.ErrorGet+err.Error(), http.StatusInternalServerError)
				return
			}
			// Update the annotations
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
			// success add to responses
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
