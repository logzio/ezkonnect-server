## API Documentation
### Get the state Instrumented Applications `[GET] /api/v1/state`
This endpoint retrieves information about instrumented applications in the form of custom resources of type InstrumentedApplication.

### Request
- Method: `GET`
- Path: `/api/v1/instrumentedapplications`

### Response
### Success
- Status code: `200 OK`
- Content-Type: `application/json`

The response body will be a JSON array of objects, where each object contains the following fields:
- `name` (string): The name of the custom resource.
- `namespace` (string): The namespace of the custom resource.
- `controller_kind` (string): The kind of the controller (lowercased owner reference kind).
- `container_name` (string, optional): The container name associated with the instrumented application. Will be empty if both language and application fields are empty.
- `instrumented` (bool): Whether the application is instrumented or not.
- `application` (string, optional): The application name if available in the spec.
- `language` (string, optional): The programming language if available in the spec.

Each instrumented application can have a `language` and/or an `application` field, or none of them. If neither `language` nor `application` is present, the application cannot be instrumented. If at least one of `language` or `application` fields is non-empty, there will also be a `container_name` field. However, if both language and application fields are empty, the `container_name` will be empty as well.


#### Example Success Response
```json
[
    {
        "name": "my-instrumented-app",
        "namespace": "default",
        "controller_kind": "deployment",
        "container_name": "app-container",
        "instrumented": true,
        "application": null,
        "language": "python"
    },
    {
        "name": "uninstrumented-app",
        "namespace": "default",
        "controller_kind": "deployment",
        "container_name": "",
        "instrumented": false
    },
    {
        "name": "statefulset-with-app-detection",
        "namespace": "default",
        "controller_kind": "statefulset",
        "container_name": "app-container",
        "instrumented": false,
        "application": "my-app",
        "language": null
    },
    {
        "name": "deployment-with-language-detection",
        "namespace": "default",
        "controller_kind": "deployment",
        "container_name": "app-container",
        "instrumented": false,
        "application": null,
        "language": "java"
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


### Update Resource Annotations `[POST] /api/v1/anotate`
This endpoint allows you to update annotations for Kubernetes deployments and statefulsets. The annotations can be used to enable or disable telemetry features such as metrics and traces.

### Request
- Method: `POST`
- Path: `/api/v1/resources`

#### Request Body
The request body should be a JSON array of objects, where each object contains the following fields:
- `name` (string): The name of the resource.
- `kind` (string): The kind of the resource, either deployment or statefulset.
- `namespace` (string): The namespace of the resource.
- `telemetry_type` (string): The type of telemetry, either metrics or traces.
- `action` (string): The action to perform, either add or delete.

#### Example Request Body
json
```json
[
    {
        "name": "my-deployment",
        "kind": "deployment",
        "namespace": "default",
        "telemetry_type": "metrics",
        "action": "add"
    },
    {
        "name": "my-statefulset",
        "kind": "statefulset",
        "namespace": "default",
        "telemetry_type": "traces",
        "action": "delete"
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
- `kind` (string): The kind of the updated resource, either deployment or statefulset.
- `updated_annotations` (object): The updated annotations with their keys and values.
#### Example Success Response
```json
[
    {
        "name": "my-deployment",
        "namespace": "default",
        "kind": "deployment",
        "updated_annotations": {
            "logz.io/export-metrics": "true"
        }
    },
    {
        "name": "my-statefulset",
        "namespace": "default",
        "kind": "statefulset",
        "updated_annotations": {
            "logz.io/instrument": "false"
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
