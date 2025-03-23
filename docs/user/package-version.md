# Package Version MCP Server

The Package Version MCP server allows you to check for the latest versions of packages from various package managers and registries.

## Features

The Package Version server can check latest versions for:

- NPM packages (Node.js)
- Python packages (requirements.txt and pyproject.toml)
- Java packages (Maven and Gradle)
- Go packages (go.mod)
- Swift packages
- Docker container images
- AWS Bedrock models

## Usage

To start the Package Version server:

```bash
megatool run package-version
```

The server doesn't require any configuration and will start immediately.

## Available Tools

When used with an MCP client (like Claude), the Package Version server provides the following tools:

### NPM Packages

Check the latest versions of NPM packages from package.json:

```json
{
  "dependencies": {
    "react": "^17.0.2",
    "react-dom": "^17.0.2",
    "lodash": "4.17.21"
  },
  "constraints": {
    "react": {
      "majorVersion": 17
    }
  }
}
```

The `constraints` object is optional and allows you to:
- Limit updates to a specific major version with `majorVersion`
- Exclude packages from updates with `excludePackage: true`

### Python Packages (requirements.txt)

Check the latest versions of Python packages from requirements.txt:

```
requests==2.28.1
flask>=2.0.0
numpy
```

### Python Packages (pyproject.toml)

Check the latest versions of Python packages from pyproject.toml:

```toml
[project]
dependencies = [
    "requests>=2.28.1",
    "flask>=2.0.0",
]

[project.optional-dependencies]
dev = [
    "pytest>=7.0.0",
]

[tool.poetry.dev-dependencies]
black = "^22.6.0"
```

### Java Packages (Maven)

Check the latest versions of Java packages from Maven pom.xml:

```xml
<dependencies>
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-web</artifactId>
        <version>2.7.0</version>
    </dependency>
    <dependency>
        <groupId>com.google.guava</groupId>
        <artifactId>guava</artifactId>
        <version>31.1-jre</version>
    </dependency>
</dependencies>
```

### Java Packages (Gradle)

Check the latest versions of Java packages from Gradle build.gradle:

```groovy
dependencies {
    implementation 'org.springframework.boot:spring-boot-starter-web:2.7.0'
    testImplementation 'junit:junit:4.13.2'
}
```

### Go Packages

Check the latest versions of Go packages from go.mod:

```go
module github.com/example/mymodule

go 1.20

require (
    github.com/gorilla/mux v1.8.0
    github.com/spf13/cobra v1.5.0
)
```

### Docker Images

Check available tags for Docker container images:

```
# Image name (e.g., nginx, ubuntu, ghcr.io/owner/repo)
nginx

# Optional parameters:
# - Registry (dockerhub, ghcr, or custom)
# - Tag filter patterns
# - Limit on number of tags to return
# - Whether to include image digest
```

### AWS Bedrock Models

List all AWS Bedrock models, search for specific models, or get the latest Claude Sonnet model (best for coding tasks).

## Examples

### Checking NPM Package Versions

When using with an MCP client like Claude, you can ask:

"What are the latest versions of React and Lodash?"

The client will use the Package Version server to check the latest versions and provide the results.

### Finding Docker Image Tags

"What are the latest stable tags for the nginx Docker image?"

### Getting the Latest Claude Model

"What's the latest Claude Sonnet model available on AWS Bedrock?"

## Integration with Development Workflows

The Package Version server is particularly useful for:

1. **Dependency Updates**: Quickly check if your project dependencies are up to date
2. **Security Patches**: Find the latest versions that may contain security fixes
3. **Compatibility Planning**: Determine what versions are available when planning upgrades
4. **Docker Image Selection**: Find appropriate tags for Docker images
5. **AI Model Selection**: Identify the latest AI models available on AWS Bedrock

## Limitations

- The server requires internet access to check package registries
- Rate limits may apply for some registries (especially Docker Hub)
- For private registries, appropriate authentication may be required
