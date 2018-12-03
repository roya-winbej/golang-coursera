package main

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"
	"sync"
	"text/template"
)

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	ast.Inspect(node, func(node ast.Node) bool {
		return visitor(node)
	})

	RenderHTTPWrapper()
}

type HandlersMapping struct {
	Handler map[string][]*HandlerContainer
}

func RenderHTTPWrapper() {
	container := GetCollector()

	outFile, _ := os.Create("/Users/alexandermykolaichuk/playground/coursera-golang/hw5_codegen/someTest.go")

	hm := &HandlersMapping{
		Handler: make(map[string][]*HandlerContainer),
	}

	for _, handler := range container.HandlerContainer {
		handler.ValidationTemplate = generateValidationCode(handler.Param)
		hm.Handler[handler.Receiver] = append(hm.Handler[handler.Receiver], handler)
	}

	var headTmpl = `
		package main

		import (
			"encoding/json"
			"io/ioutil"
			"net/http"
			"net/url"
		)

	`
	_, _ = fmt.Fprintln(outFile, headTmpl)

	var serveHTTPTmpl = `
		func (srv *{{.Receiver}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
				{{range .Handler}}
					case "{{.Url}}":
					func(w http.ResponseWriter, r *http.Request) {

						fnParams := {{.Param}}{}
						var queryString string

		if r.Method == "POST" {
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			defer r.Body.Close()

			queryString = string(body)

		} else {
			queryString = r.URL.RawQuery
		}


	q, _ := url.ParseQuery(queryString)

	values := make(map[string]string)
	for key := range q {
		values[key] = q.Get(key)
	}
	
	JSON, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}
	
	_ = json.Unmarshal(JSON, &fnParams)

	{{ .ValidationTemplate.Template }}


						res, err := srv.{{.StructMethod}}(r.Context(), fnParams)
						if err != nil {
							http.Error(w, err.Error(), err.(ApiError).HTTPStatus)
						}

						encoder := json.NewEncoder(w)

						_ = encoder.Encode(&struct{
							Error string ` + "`" + `json:"error"`+ "`" + `
							Response interface{}` + "`" + `json:"response, omitempty"` + "`" + `
						}{
							Error: "",
							Response: res,
						})

					}(w, r)

				{{end}}
					default:
					// 404
			}
		}
	`


	t := template.Must(template.New("funcTemp").Parse(serveHTTPTmpl))

	for receiver, handler := range hm.Handler {
		err := t.Execute(outFile, struct {
			Receiver string
			Handler []*HandlerContainer
		}{
			receiver,
			handler,
		})
		if err != nil {
			panic(err)
		}

	}


}

type HandlerContainer struct {
	Url string
	Auth bool
	Method string
	Param string
	Receiver string
	StructMethod string
	ValidationTemplate *ValidationTemplate
}

type StructContainer struct {
	Name string
	Fields []*StructField
}

type StructField struct {
	FieldName string
	FieldType string
	Validation []string
}

type ValidationTemplate struct {
	ParamName string
	Template string
}

func (c *Collector) getStructContainerByName(name string) (*StructContainer, error) {
	for _, container := range c.StructContainer {
		if container.Name == name {
			return container, nil
		}
	}

	return nil, errors.New("Container not found")
}

func generateValidationCode(param string) *ValidationTemplate {
	container := GetCollector()
	structContainer, _:= container.getStructContainerByName(param)

	var fieldsTmpl string

	for _, field := range structContainer.Fields {
		for _, validation := range field.Validation {
			if validation == "required" {
				fieldsTmpl = `if fnParams` + "." + field.FieldName + ` == "" {
						encoder := json.NewEncoder(w)
		
						w.WriteHeader(http.StatusBadRequest)
		
						_ = encoder.Encode(&struct{
							Error string `+ "`" + `json:"error"` + "`" + `
						}{
							Error: "login must me not empty",
						})
		
						return
				}

				`
			}

			if strings.Contains(validation, "=") {
				v := strings.Split(validation, "=")

				if v[0] == "min" || v[0] == "max" {
					var fieldLen string
					var errMessage string
					var validationExp string

					if v[0] == "min" {
						validationExp = ">="
					} else {
						validationExp = "<="
					}

					switch field.FieldType {
					case "int":
						fieldLen = "fnParams." + field.FieldName
						errMessage = strings.ToLower(field.FieldName) + " must be " + validationExp + " " + v[1]
					case "string":
						fieldLen = "len(fnParams." + field.FieldName + ")"
						errMessage = strings.ToLower(field.FieldName) + " len must be " + validationExp + " " + v[1]
					}

					fieldsTmpl += `if ` + fieldLen + ` <= ` + v[1] + ` {
						encoder := json.NewEncoder(w)
		
						w.WriteHeader(http.StatusBadRequest)
		
						_ = encoder.Encode(&struct{
							Error string `+ "`" + `json:"error"` + "`" + `
						}{
							Error: "` + errMessage + "\"" +`,
						})
		
						return
				}

				`
				}
			}
		}
	}

	return &ValidationTemplate{
		ParamName: structContainer.Name,
		Template: fieldsTmpl,
	}
}

var collectorInstance *Collector
var once sync.Once

func GetCollector() *Collector {
	once.Do(func() {
		collectorInstance = &Collector{}
	})
	return collectorInstance
}

type Collector struct {
	HandlerContainer []*HandlerContainer
	StructContainer []*StructContainer
}

func visitor(node ast.Node) bool {
	funcDecl, ok := node.(*ast.FuncDecl)
	if ok {
		doc := funcDecl.Doc

		if doc != nil {
			obj := &HandlerContainer{}

			obj.Param = funcDecl.Type.Params.List[1].Type.(*ast.Ident).String()
			obj.StructMethod = funcDecl.Name.String()

			if funcDecl.Recv != nil {
				for _, v := range funcDecl.Recv.List {
					starExpr := v.Type.(*ast.StarExpr)

					if recv, ok := starExpr.X.(*ast.Ident); ok {
						obj.Receiver = recv.Name
					}
				}
			}

			res := strings.Replace(doc.Text(), "apigen:api", "", -1)

			err := json.Unmarshal([]byte(res), obj)
			if err != nil {
				panic(err)
			}

			collector := GetCollector()
			collector.HandlerContainer = append(collector.HandlerContainer, obj)
		}
	}

	currType, okSpec := node.(*ast.TypeSpec)
	if okSpec {

		structType, ok := currType.Type.(*ast.StructType)
		if ok {

			structContainer := &StructContainer{}

			var showName = false

			for _, str := range structType.Fields.List {
				if str.Tag != nil {
					if strings.Contains(str.Tag.Value, "apivalidator") {
						showName = true

						replacer := strings.NewReplacer("apivalidator:", "", "\"", "", "`", "")

						validation := replacer.Replace(str.Tag.Value)

						structContainer.Fields = append(structContainer.Fields, &StructField{
							FieldName:  str.Names[0].String(),
							Validation: strings.Split(validation, ","),
							FieldType: str.Type.(*ast.Ident).Name,
						})

					}

				} else {
					showName = false
				}
			}

			if showName {
				structContainer.Name = currType.Name.String()

				collector := GetCollector()
				collector.StructContainer = append(collector.StructContainer, structContainer)

			}

		}

	}


	return true
}