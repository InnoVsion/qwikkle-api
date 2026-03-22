package docs

// NOTE: This is a minimal stub so the project compiles and serves Swagger UI.
// You can regenerate it later with `swag init` if you install the swag CLI.

import "github.com/swaggo/swag"

var SwaggerInfo = &swag.Spec{
	Version:          "0.1",
	Host:             "localhost:8080",
	BasePath:         "/",
	Schemes:          []string{"http"},
	Title:            "Qwikkle API",
	Description:      "Qwikkle backend API documentation.",
	InfoInstanceName: "swagger",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
