package db

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

var templates *template.Template

type ErrorPage struct {
	Code    int
	Message string
	Is405   bool
	Is404   bool
	Is500   bool
	Is403   bool
	Is400   bool
	Is401   bool
}

//helper for handeling errors
func HandleError(w http.ResponseWriter, code int, message string) {
	errorPage := ErrorPage{
		Code:    code,
		Message: message,
		Is405:   code == http.StatusMethodNotAllowed,
		Is404:   code == http.StatusNotFound,
		Is500:   code == http.StatusInternalServerError,
		Is403:   code == http.StatusForbidden,
		Is400:   code == http.StatusBadRequest,
		Is401:   code == http.StatusUnauthorized,
	}

	w.WriteHeader(code)
	RenderTemplate(w, "error", map[string]interface{}{
		"Code":    errorPage.Code,
		"Message": errorPage.Message,
		"Is405":   errorPage.Is405,
		"Is404":   errorPage.Is404,
		"Is500":   errorPage.Is500,
		"Is403":   errorPage.Is403,
		"Is400":   errorPage.Is400,
		"Is401":   errorPage.Is401,
	})
}

//custom 'in' function to check if a value exists in a slice
func in(slice interface{}, value interface{}) bool {
	switch slice := slice.(type) {
	case []string:
		for _, v := range slice {
			if v == value {
				return true
			}
		}
	case []int:
		for _, v := range slice {
			if v == value {
				return true
			}
		}
	}
	return false
}

func InitTemplates() error {
	var err error
	tmpl := template.New("")
	tmpl.Funcs(template.FuncMap{
		"in": in,
	})
	templates, err = tmpl.New("").ParseFiles(
		"templates/home.html",
		"templates/login.html",
		"templates/register.html",
		"templates/add_post.html",
		"templates/add_comment.html",
		"templates/error.html",
	)
	if err != nil {
		return fmt.Errorf("template initialization error: %v", err)
	}

	//log loaded templates
	var templateNames []string
	for _, t := range templates.Templates() {
		templateNames = append(templateNames, t.Name())
	}
	log.Printf("Loaded templates: %v", templateNames)
	return nil
}

func RenderTemplate(w http.ResponseWriter, name string, data interface{}) {
	var dataMap map[string]interface{}
	if data == nil {
		dataMap = make(map[string]interface{})
	} else if m, ok := data.(map[string]interface{}); ok {
		dataMap = m
	} else {
		dataMap = map[string]interface{}{
			"Data": data,
		}
	}

	//make sure title exists
	if _, hasTitle := dataMap["Title"]; !hasTitle {
		dataMap["Title"] = name
	}

	//execute template
	err := templates.ExecuteTemplate(w, name+".html", dataMap)
	if err != nil {
		log.Printf("Template execution: %v, Error: %v", name, err)
		HandleError(w, http.StatusInternalServerError, "Internal server error")
	}
}
