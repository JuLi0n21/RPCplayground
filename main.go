package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"

	"golang.org/x/net/websocket"
)

// --- Example Struct ---
type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// --- API with functions ---
type API struct{}

func (API) Add(a, b int) int {
	return a + b
}

func (API) Welcome(p Person) string {
	return fmt.Sprintf("Hello %s, you are %d years old!", p.Name, p.Age)
}

// --- WebSocket RPC handler ---
func wsHandler(api API) func(ws *websocket.Conn) {
	return func(ws *websocket.Conn) {
		defer ws.Close()
		for {
			var msg []byte
			if err := websocket.Message.Receive(ws, &msg); err != nil {
				log.Println("Receive error:", err)
				break
			}

			var req struct {
				ID     string        `json:"id"`
				Method string        `json:"method"`
				Params []interface{} `json:"params"`
			}
			if err := json.Unmarshal(msg, &req); err != nil {
				log.Println("JSON decode error:", err)
				continue
			}

			v := reflect.ValueOf(api)
			m := v.MethodByName(req.Method)
			if !m.IsValid() {
				resp := map[string]interface{}{"id": req.ID, "error": "method not found"}
				websocket.JSON.Send(ws, resp)
				continue
			}

			in := []reflect.Value{}
			for i, param := range req.Params {
				paramJSON, _ := json.Marshal(param)
				paramValue := reflect.New(m.Type().In(i)).Interface()
				json.Unmarshal(paramJSON, &paramValue)
				in = append(in, reflect.ValueOf(paramValue).Elem())
			}

			out := m.Call(in)
			resp := map[string]interface{}{"id": req.ID, "result": out[0].Interface()}
			websocket.JSON.Send(ws, resp)
		}
	}
}

// --- TypeScript generator ---
func generateTS(apiType reflect.Type, filename string) {
	out := "export const api: {\n"
	for i := 0; i < apiType.NumMethod(); i++ {
		method := apiType.Method(i)
		out += "  " + method.Name + "("

		params := []string{}
		for j := 0; j < method.Type.NumIn(); j++ {
			if j == 0 {
				continue // skip receiver
			}
			param := method.Type.In(j)
			params = append(params, fmt.Sprintf("arg%d: %s", j-1, goTypeToTS(param)))
		}
		out += join(params, ", ") + "): Promise<" + goTypeToTS(method.Type.Out(0)) + ">;\n"
	}
	out += "} = {};\n"
	os.WriteFile(filename, []byte(out), 0644)
	fmt.Println("TypeScript types generated:", filename)
}

func join(a []string, sep string) string {
	s := ""
	for i, v := range a {
		if i > 0 {
			s += sep
		}
		s += v
	}
	return s
}

func goTypeToTS(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Struct:
		out := "{ "
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			out += field.Name + ": " + goTypeToTS(field.Type) + "; "
		}
		out += "}"
		return out
	case reflect.Slice:
		return goTypeToTS(t.Elem()) + "[]"
	default:
		return "any"
	}
}

func main() {
	api := API{}
	apiType := reflect.TypeOf(api)
	generateTS(apiType, "api.d.ts") // generates TS types

	http.Handle("/ws", websocket.Handler(wsHandler(api)))
	fmt.Println("WebSocket RPC server running at ws://localhost:8080/ws")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
