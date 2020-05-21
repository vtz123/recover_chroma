package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/panic/", panicDemo)
	mux.HandleFunc("/panic-after/", panicAfterDemo)
	mux.HandleFunc("/", hello)
	mux.HandleFunc("/debug/", sourceCodeHander)
	log.Fatal(http.ListenAndServe(":3000", devMw(mux)))
}

func devMw(app http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Println(err)
				stack := debug.Stack()
				log.Println(string(stack))
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "<h1>panic: %v</h1><pre>%s</pre>", err, makeLinks(string(stack)))
			}
		}()
		app.ServeHTTP(w, r)
	}
}

func makeLinks(stack string) string {
	var res strings.Builder

	lines := strings.Split(stack, "\n")

	for i := range lines {
		if i == 0 {
			res.WriteString(lines[i])
			res.WriteString("\n")
			continue
		}

		index := strings.IndexByte(lines[i], ':')

		if index != -1 {
			var linecopy string
			//<a href=""> </a>
			path := lines[i][1:index]

			line := findline([]byte(lines[i][index+1:]))
			v := url.Values{}
			v.Set("path", path)
			v.Set("line", line)
			linecopy = fmt.Sprintf("\t<a href=\"/debug?%s\">%s</a>%s", v.Encode(), lines[i][1:index], lines[i][index:])
			res.WriteString(linecopy)
			res.WriteString("\n")
			continue
		}
		res.WriteString(lines[i])
		res.WriteString("\n")
	}

	return res.String()
}

func panicDemo(w http.ResponseWriter, r *http.Request) {
	funcThatPanics()
}

func sourceCodeHander(w http.ResponseWriter, r *http.Request) {
	path := r.FormValue("path")
	line, _ := strconv.Atoi(r.FormValue("line"))

	file, err := os.Open(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//line := 15
	var lines [][2]int
	if line > 0 {
		lines = append(lines, [2]int{line, line})
	}
	lexer := lexers.Get("go")
	iterator, err := lexer.Tokenise(nil, buf.String())
	style := styles.Get("github")
	if style == nil {
		style = styles.Fallback
	}
	formatter := html.New(html.TabWidth(4), html.WithLineNumbers(true), html.HighlightLines(lines))

	w.Header().Set("Content-Type", "text/html")
	formatter.Format(w, style, iterator)

	//quick.Highlight(w, buf.String(), "go", "html", "monokai")
	//	err = quick.Highlight(w, , "go", "html", "monokai")

	//io.Copy(w, file)
}

func panicAfterDemo(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "<h1>Hello!</h1>")
	funcThatPanics()
}

func funcThatPanics() {
	panic("Oh no!")
}

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "<h1>Hello!</h1>")
}

func findline(input []byte) string {
	var index int
	for i := 0; i < len(input); i++ {
		if input[i] >= '0' && input[i] <= '9' {
			index = i
		} else {
			break
		}
	}

	return string(input[:index+1])
}
