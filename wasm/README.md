# WASM module

Optionally copy the wasm_exec.js file to the current directory:

```bash
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" .
```

Run a python server to serve the index.html file:

```bash
python3 -m http.server 8080
```

## References

- https://www.awesome.club/blog/2024/now-is-the-best-time-to-learn-web-assembly
- https://withblue.ink/2020/10/03/go-webassembly-http-requests-and-promises.html
