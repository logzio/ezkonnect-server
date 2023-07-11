## API Documentation
- ### `[GET] /api/v1/state` Get the state Instrumented Applications 
This endpoint retrieves information about instrumented applications in the form of custom resources of type InstrumentedApplication.

### Request
- Method: `GET`
- Path: `/api/v1/state`

### Response
### Success
- Status code: `200 OK`
- Content-Type: `application/json`

The response body will be a JSON array of objects, where each object contains the following fields:
- `name` (string): The name of the custom resource.
- `namespace` (string): The namespace of the custom resource.
- `controller_kind` (string): The kind of the controller (lowercased owner reference kind).
- `container_name` (string, optional): The container name associated with the instrumented application. Will be empty if both language and application fields are empty.
- `traces_instrumented` (bool): Whether the application is instrumented or not.
- `traces_instrumentable` (bool): Whether the application can be instrumented or not.
- `application` (string, optional): The application name if available in the spec.
- `language` (string, optional): The programming language if available in the spec.
- `log_type` (string, optional): The log type if available in the spec.
- `opentelemetry_preconfigured` bool: Whether the application has opentelemetry libraries or not.
- `detection_status` (string): The status of the detection process. Can be one of the following:
    - `pending`: The detection process has not started yet.
    - `Completed`: The detection process has completed successfully.
    - `Running`: The detection process is still running.
    - `error`: The detection process has failed.


Each instrumented application can have a `language` and/or an `application` field, or none of them. If neither `language` nor `application` is present, the application cannot be instrumented. If at least one of `language` or `application` fields is non-empty, there will also be a `container_name` field. However, if both language and application fields are empty, the `container_name` will be empty as well.


#### Example Success Response
```json
[
    {
        "name": "my-instrumented-app",
        "namespace": "default",
        "controller_kind": "deployment",
        "container_name": "app-container",
        "traces_instrumented": true,
        "traces_instrumentable": true,
        "application": null,
        "language": "python",
        "detection_status": "Completed",
        "opentelemetry_preconfigured": false,
        "log_type": "nginx"
    },
    {
        "name": "uninstrumented-app",
        "namespace": "default",
        "controller_kind": "deployment",
        "container_name": "",
        "traces_instrumented": false,
        "traces_instrumentable": false,
        "detection_status": "Completed",
        "opentelemetry_preconfigured": false,
        "log_type": "log"
    },
    {
        "name": "statefulset-with-app-detection",
        "namespace": "default",
        "controller_kind": "statefulset",
        "container_name": "app-container",
        "traces_instrumented": false,
        "traces_instrumentable": false,
        "application": "my-app",
        "language": null,
        "detection_status": "Completed",
        "opentelemetry_preconfigured": false,
        "log_type": "log2"
    },
    {
        "name": "deployment-with-language-detection",
        "namespace": "default",
        "controller_kind": "deployment",
        "container_name": "app-container",
        "traces_instrumented": false,
        "traces_instrumentable": true,
        "application": null,
        "language": "java",
        "detection_status": "Completed",
        "opentelemetry_preconfigured": false,
        "log_type": "nginx"
    }
]
```
### Errors
- Status code: `405 Method Not Allowed`

The request method is not GET.

Example error response:

```json
{
"error": "Invalid request method"
}
```
- Status code: `500 Internal Server Error`

There was an error processing the request, such as failing to interact with the Kubernetes cluster.

Example error response:

```json
{
"error": "Error message"
}
```


- ### `[POST] /api/v1/anotate/traces` Update traces Resource Annotations 
This endpoint allows you to update annotations for Kubernetes deployments and statefulsets. The annotations can be used to enable or disable telemetry features such as metrics and traces.

### Request
- Method: `POST`
- Path: `/api/v1/anotate/traces`

#### Request Body
The request body should be a JSON array of objects, where each object contains the following fields:
- `name` (string): The name of the resource.
- `controller_kind` (string): The kind of the resource, either deployment or statefulset.
- `namespace` (string): The namespace of the resource.
- `action` (string): The action to perform, either add or delete.
- `service_name` (string): The name of the service associated with the resource.

#### Example Request Body
json
```json
[
    {
        "name": "my-deployment",
        "controller_kind": "deployment",
        "namespace": "default",
        "action": "add",
        "service_name": "my-service"
    },
    {
        "name": "my-statefulset",
        "controller_kind": "statefulset",
        "namespace": "default",
        "action": "delete",
        "service_name": "my-other-service"
    }
]
```

### Response
#### Success
- Status code: `200 OK`
- Content-Type: `application/json`

The response body will be a JSON array of objects, where each object contains the following fields:
- `name` (string): The name of the updated resource.
- `namespace` (string): The namespace of the updated resource.
- `controller_kind` (string): The kind of the updated resource, either deployment or statefulset.
- `updated_annotations` (object): The updated annotations with their keys and values.
#### Example Success Response
```json
[
    {
        "name": "my-deployment",
        "namespace": "default",
        "controller_kind": "deployment",
        "updated_annotations": {
            "logz.io/instrument": "true",
            "logz.io/service-name": "my-service"
        }
    },
    {
        "name": "my-statefulset",
        "namespace": "default",
        "controller_kind": "statefulset",
        "updated_annotations": {
            "logz.io/instrument": "rollback",
            "logz.io/service-name": "my-other-service"
        }
    }
]
```

#### Errors
- Status code: `400 Bad Request`

The request body is malformed, or one or more of the provided fields have invalid values.

Example error response:

```json
{
"error": "Invalid input"
}
```

- Status code: `500 Internal Server Error`

There was an error processing the request, such as failing to interact with the Kubernetes cluster.

Example error response:

```json
{
  "error": "Error message"
}
```


- ### `[POST] /api/v1/annotate/logs` Update Logs Resource Annotations


This endpoint allows you to set the log type for Kubernetes deployments and statefulsets. The annotation is used to determine the type of logs that should be collected from the resource.

### Request

*   Method: `POST`
*   Path: `/api/v1/annotate/logs`

#### Request Body

The request body should be a JSON array of objects, where each object contains the following fields:

*   `name` (string): The name of the resource.
*   `controller_kind` (string): The kind of the resource controller, either "deployment" or "statefulset".
*   `namespace` (string): The namespace of the resource.
*   `log_type` (string): The type of logs to add.

#### Example Request Body

```json
[
    {
        "name": "my-deployment",
        "controller_kind": "deployment",
        "namespace": "default",
        "log_type": "application"
    },
    {
        "name": "my-statefulset",
        "controller_kind": "statefulset",
        "namespace": "default",
        "log_type": "system"
    }
]

```

### Response

#### Success

*   Status code: `200 OK`
*   Content-Type: `application/json`

The response body will be a JSON array of objects, where each object contains the following fields:

*   `name` (string): The name of the updated resource.
*   `namespace` (string): The namespace of the updated resource.
*   `controller_kind` (string): The kind of the updated resource, either "deployment" or "statefulset".
*   `updated_annotations` (object): The updated annotations with their keys and values.

#### Example Success Response

```json
[
    {
        "name": "my-deployment",
        "namespace": "default",
        "controller_kind": "deployment",
        "updated_annotations": {
            "logz.io/application_type": "application"
        }
    },
    {
        "name": "my-statefulset",
        "namespace": "default",
        "controller_kind": "statefulset",
        "updated_annotations": {
            "logz.io/application_type": "system"
        }
    }
]

```
#### Errors

*   Status code: `400 Bad Request`

The request body is malformed, or one or more of the provided fields have invalid values.

Example error response:

jsonCopy code

`{ "error": "Invalid input" }`

*   Status code: `500 Internal Server Error`

There was an error processing the request, such as failing to interact with the Kubernetes cluster.

Example error response:

jsonCopy code

`{   "error": "Error message" }`
