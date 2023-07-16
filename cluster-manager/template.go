package main

import (
	"html/template"
	"io"

	"github.com/labstack/echo/v4"
)

// Define the template registry struct
type TemplateRegistry struct {
	templates *template.Template
  }
  
  // Implement e.Renderer interface
  func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
  }