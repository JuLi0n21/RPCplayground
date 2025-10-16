// main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"

	"golang.org/x/net/websocket"
)

type Person struct {
	Name string   `json:"name"`
	Age  *int     `json:"age"`
	Tags []string `json:"tags"`
}

type API struct{}

func (API) Add(a, b int) (int, error)                           { return a + b, nil }
func (API) Welcome(p Person) (string, error)                    { return fmt.Sprintf("Hello %s", p.Name), nil }
func (API) Names(people []Person) ([]string, error)             { return nil, nil }
func (API) MapExample(m map[string]int) (map[string]int, error) { return m, nil }

func main() {
	api := API{}

	outFile := "api.gen.ts"
	err := GenClient(api, outFile)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/ws", websocket.Handler(wsHandler(api)))
	fmt.Println("WebSocket running at ws://localhost:8080/ws")
	fmt.Println("Press CTRL+C to exit")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func wsHandler(api any) func(*websocket.Conn) {
	return func(ws *websocket.Conn) {
		apiVal := reflect.ValueOf(api)

		for {
			var msg []byte
			if err := websocket.Message.Receive(ws, &msg); err != nil {
				break
			}

			var req struct {
				ID     string `json:"id"`
				Method string `json:"method"`
				Params []any  `json:"params"`
			}
			if err := json.Unmarshal(msg, &req); err != nil {
				continue
			}

			method := apiVal.MethodByName(req.Method)
			if !method.IsValid() {
				_ = websocket.JSON.Send(ws, map[string]any{
					"id":    req.ID,
					"error": "method not found: " + req.Method,
				})
				continue
			}

			mType := method.Type()
			var in []reflect.Value
			for i := 0; i < mType.NumIn(); i++ {
				raw := req.Params[i]
				argType := mType.In(i)
				argPtr := reflect.New(argType)
				b, _ := json.Marshal(raw)
				_ = json.Unmarshal(b, argPtr.Interface())
				in = append(in, argPtr.Elem())
			}

			out := method.Call(in)
			res := out[0].Interface()
			var errVal any
			if len(out) > 1 {
				errIF := out[1].Interface()
				if errIF != nil {
					errVal = errIF.(error).Error()
				}
			}

			_ = websocket.JSON.Send(ws, map[string]any{
				"id": req.ID,
				"result": map[string]any{
					"data":  res,
					"error": errVal,
				},
			})
		}
	}
}
