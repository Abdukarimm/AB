# Online Compiler

This project includes a small Go server and a responsive web front–end to
execute short code snippets in Go, Python, PHP and Java. The browser communicates
with the backend via WebSockets.

## Server API

- `GET /languages` – returns an array of supported languages.
- `WS /ws` – accepts JSON messages of the form:
  ```json
  {"language": "python", "code": "print('hello')"}
  ```
  and responds with:
  ```json
  {"output": "hello\n", "error": ""}
  ```

## Front‑end

Open the root page and you will find a simple editor with a language selector,
a run button and an output area. A theme toggle allows switching between dark
and light modes. The layout adapts to small screens.

## Running

```
go run .
```

Then open `http://localhost:8080/` in your browser.

The build requires only the Go standard library so it works offline.
