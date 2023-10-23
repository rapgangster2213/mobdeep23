package main

import (
	"calculator-main/pkg/keygen"
	"fmt"
	"github.com/Knetic/govaluate"
	"github.com/gorilla/sessions"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

type Calculation struct {
	Expression string
	Result     float64
	Session    string
}

var calculations []Calculation
var store = sessions.NewCookieStore([]byte("pass"))
var templates *template.Template

func main() {
	http.HandleFunc("/", mainHandler)
	http.HandleFunc("/calc", calcHandler)
	http.HandleFunc("/login", loginHandler)
	templates = template.Must(template.ParseGlob("templates/*.html"))

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Printf(err.Error())
	}
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username := session.Values["username"]
	if username == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/calc", http.StatusSeeOther)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Values["username"] = keygen.RandStr()
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func calcHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		templates.Execute(w, "calc.html")
	} else if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		session, err := store.Get(r, "session")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		untypedUsername, ok := session.Values["username"]
		if !ok {
			return
		}
		username, ok := untypedUsername.(string)
		if !ok {
			return
		}
		session.Values["expression"] = r.FormValue("expression")
		untypedExpression, ok := session.Values["expression"]
		if !ok {
			return
		}
		expression, ok := untypedExpression.(string)
		if !ok {
			return
		}
		if checkString(expression) {
			newExpression, err := govaluate.NewEvaluableExpression(expression)
			if err != nil {
				result, err := calculateExpression(expression)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				session.Values["result"] = result
			} else {
				result, err := newExpression.Evaluate(nil)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				session.Values["result"] = result
			}
		} else {
			http.Error(w, "Wrong expression", http.StatusBadRequest)
			return
		}
		/*newExpression, err := govaluate.NewEvaluableExpression(expression)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result, err := newExpression.Evaluate(nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}*/
		untypedResult, ok := session.Values["result"]
		if !ok {
			return
		}
		result, ok := untypedResult.(float64)
		session.Values["result"] = result
		resultOutput := untypedResult.(float64)
		calculations = append(calculations, Calculation{
			Expression: expression,
			Result:     resultOutput,
			Session:    username,
		})

		fmt.Fprintf(w, "<p>Result: %s<p>", strconv.FormatFloat(resultOutput, 'f', -1, 64)) // result output
		err = templates.Execute(w, calculations)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, calc := range calculations {
			if username == calc.Session {
				fmt.Fprintf(w, "<p>%s = %s", calc.Expression, strconv.FormatFloat(calc.Result, 'f', -1, 64))                   // calc history output
				fmt.Printf("%s = %s, key: %s\n", calc.Expression, strconv.FormatFloat(calc.Result, 'f', -1, 64), calc.Session) // bebra
			}
		}
	}
}

func calculateExpression(expression string) (float64, error) {
	expression = strings.ReplaceAll(expression, " ", "")
	expression = strings.ReplaceAll(expression, ",", ".")

	result, err := eval(expression)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func eval(expression string) (float64, error) {
	result, err := strconv.ParseFloat(expression, 64)
	if err == nil {
		return result, nil
	}

	operations := []string{"+", "/", "*", "-"}

	for _, op := range operations {
		if strings.Contains(expression, op) {
			parts := strings.Split(expression, op)
			if len(parts) != 2 {
				return 0, fmt.Errorf("Invalid expression")
			}

			left, err := eval(parts[0])
			if err != nil {
				return 0, err
			}

			right, err := eval(parts[1])
			if err != nil {
				return 0, err
			}

			switch op {
			case "+":
				return left + right, nil
			case "-":
				return left - right, nil
			case "*":
				return left * right, nil
			case "/":
				if right == 0 {
					return 0, fmt.Errorf("Division by zero")
				}
				return left / right, nil
			}
		}
	}

	return 0, fmt.Errorf("Invalid expression")
}

func checkString(s string) bool {
	checkCharacters := []string{"(", ")", "-", "+", "*", "/", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	for _, c := range checkCharacters {
		if strings.Contains(s, c) {
			return true
		}
	}
	return false
}
