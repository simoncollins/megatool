# Calculator MCP Server

The Calculator MCP server provides basic arithmetic operations through the Model Context Protocol.

## Features

- Basic arithmetic operations (addition, subtraction, multiplication, division)
- Advanced mathematical functions
- Unit conversions
- Simple expression parsing

## Usage

The Calculator server doesn't require any configuration. Start it with:

```bash
megatool run calculator
```

The server will start and wait for MCP requests from a client.

## Available Tools

When used with an MCP client (like Claude), the Calculator server provides the following tools:

### Basic Arithmetic

Perform basic arithmetic operations:

- Addition
- Subtraction
- Multiplication
- Division
- Modulo (remainder)
- Exponentiation

### Advanced Functions

Access advanced mathematical functions:

- Trigonometric functions (sin, cos, tan)
- Logarithmic functions
- Square root and other roots
- Absolute value
- Rounding functions

### Unit Conversions

Convert between different units:

- Length (meters, feet, inches, etc.)
- Weight/Mass (kilograms, pounds, etc.)
- Volume (liters, gallons, etc.)
- Temperature (Celsius, Fahrenheit, Kelvin)
- Time (seconds, minutes, hours, days)

### Expression Parsing

Parse and evaluate mathematical expressions:

```
(5 + 3) * 2 - 4
```

## Examples

### Basic Calculations

When using with an MCP client like Claude, you can ask:

"What is 1234 * 5678?"

The client will use the Calculator server to compute the result.

### Unit Conversions

"Convert 100 kilometers to miles."

### Complex Expressions

"Calculate the value of (15 * 3) / (2 + 1) - 5²."

## Integration with Other Tasks

The Calculator server is particularly useful for:

1. **Data Analysis**: Perform calculations on data sets
2. **Scientific Computations**: Calculate values for scientific formulas
3. **Financial Calculations**: Compute interest, payments, or other financial metrics
4. **Engineering Calculations**: Convert units and solve engineering problems

## Limitations

- The Calculator server is designed for relatively simple calculations
- It may not support very complex mathematical operations or specialized domains
- For advanced statistical or scientific computing, specialized tools may be more appropriate
