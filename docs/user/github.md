# GitHub MCP Server

The GitHub MCP server provides access to GitHub repository and user information through the Model Context Protocol.

## Features

- Access GitHub repository information
- Get user profile data
- List repositories for a user or organization
- View issues and pull requests
- Access repository content
- Search GitHub repositories, users, and code

## Configuration

The GitHub server requires a GitHub Personal Access Token (PAT) for authentication. You must configure the server before using it for the first time:

```bash
megatool run github --configure
```

This will prompt you for your GitHub Personal Access Token. The token is stored securely in your system's keyring.

### Creating a GitHub Personal Access Token

1. Go to [GitHub Settings > Developer settings > Personal access tokens](https://github.com/settings/tokens)
2. Click "Generate new token" (classic)
3. Give your token a descriptive name
4. Select the following scopes:
   - `repo` (Full control of private repositories)
   - `read:user` (Read user profile data)
   - `read:org` (Read organization data)
5. Click "Generate token"
6. Copy the generated token (you will only see it once)
7. Use this token when configuring the GitHub MCP server

## Usage

After configuration, start the GitHub server with:

```bash
megatool run github
```

The server will start and wait for MCP requests from a client.

## Available Tools

When used with an MCP client (like Claude), the GitHub server provides the following tools:

### Repository Information

Get detailed information about a GitHub repository:

- Repository name, description, and URL
- Star count, fork count, and watch count
- Primary language and topics
- License information
- Last update time
- Owner information

### User Information

Get information about a GitHub user:

- Username and display name
- Bio and location
- Follower and following counts
- Public repository count
- Organization memberships
- Contribution activity

### Repository Listing

List repositories for a user or organization:

- Filter by type (public, private, sources, forks)
- Sort by various criteria (stars, forks, updated)
- Limit the number of results

### Issue and Pull Request Access

Access issues and pull requests for a repository:

- Filter by state (open, closed)
- Filter by labels, assignees, or authors
- Sort by various criteria (created, updated, comments)
- Get detailed information about specific issues or PRs

### Repository Content

Access files and directories within a repository:

- Browse directory contents
- View file content
- Get file metadata (size, type, last update)

### Search

Search GitHub for:

- Repositories matching specific criteria
- Users with specific attributes
- Code containing specific patterns

## Examples

### Getting Repository Information

When using with an MCP client like Claude, you can ask:

"How many stars does the tensorflow/tensorflow repository have?"

The client will use the GitHub server to fetch this information.

### Finding User Repositories

"What are the most popular repositories created by microsoft?"

### Checking Issue Status

"Are there any open issues labeled 'bug' in the react/react repository?"

## Integration with Development Workflows

The GitHub server is particularly useful for:

1. **Project Research**: Quickly gather information about repositories and users
2. **Issue Tracking**: Monitor and analyze issues in repositories
3. **Code Exploration**: Access and search code in repositories
4. **Contribution Analysis**: Analyze user contributions and activity

## Limitations

- API rate limits apply based on GitHub's policies
- Some operations may require additional permissions
- Private repository access requires appropriate token scopes
