# OpenAPI 3 support
Currently no automatic OpenAPI 3 spec generation is supported. We are waiting for updates in
[swaggo](https://github.com/swaggo/swag/issues/548).

## Temporary manual solution
1. Generate Swagger 2 spec using `swag init`.
2. Convert Swagger 2 API json string ([docs.go](docs.go)) to OpenAPI 3 using: <https://mermade.org.uk/openapi-converter>.
3. Convert output yaml to json, e.g. using: <https://onlineyamltools.com/convert-yaml-to-json>.
4. Put result into the source ([docs.go](docs.go)) and update `"servers"` field with: `"servers": [\{"url": "http://localhost:8080"}]`.
