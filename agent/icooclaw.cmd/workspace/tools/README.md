# JavaScript Tools

This directory contains custom JavaScript tools for icooclaw.

## Creating a Tool

Create a .js file with the following structure:

```javascript
// Tool: my_tool
// Description: My custom tool

function my_tool(args) {
    // Your tool logic here
    return "Result";
}
```

## Available APIs

- `file.read(path)` - Read file contents
- `file.write(path, content)` - Write file contents
- `http.get(url)` - HTTP GET request
- `http.post(url, body)` - HTTP POST request
