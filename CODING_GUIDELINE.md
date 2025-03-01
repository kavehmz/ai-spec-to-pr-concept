# **Instructions for Financial Trading Platform**

## **Objective**
You are a highly skilled Go developer responsible for implementing a **Financial Trading Platform**. The system must be built with **defensive programming**, ensuring **security, correctness, resilience, and maintainability**.

Your code must:
- Follow **Defensive Programming** principles.
- Prioritize **security** and **data integrity**.
- Enforce **safe monetary calculations** (avoiding floating-point precision errors).
- Implement **structured logging** and **error handling** with stack traces.

---

## **General Guidelines**
- **DO NOT trust input**: Validate and sanitize all user-provided data.
- **AVOID panic()**:
- **ALWAYS handle errors**
- **ENSURE goroutine safety**: Use `sync.Mutex`, `sync.RWMutex`, and `atomic` where needed.
- **LOG everything relevant**: Use structured logging (`log/slog`), but never log sensitive data.
- **IMPLEMENT secure coding practices**: Avoid SQL injections, and XSS vulnerabilities.
- **Concurrent safety**: In Go we can easily have race condition. Try avoid them and if you found one bring it up and fix them. Always run the test with `--race` option, e.g `go test --race ./...`

---

## **Security Best Practices**
- **Use `crypto/rand`** instead of `math/rand` for cryptographic security.
- **Use `crypto/tls.Config{}`** for secure TLS configurations.
- **Hash sensitive data** using `crypto/bcrypt`.
- **Use `html/template`** to prevent XSS attacks.
- **Use `gorilla/csrf`** for CSRF protection.
- **Sanitize SQL inputs** using parameterized queries.
- **Avoid hardcoded secrets**; use environment variables instead.

---

## **Monetary Handling - Floating Point Safety**
- **NEVER use `float64`** for financial calculations.
- **USE `math/big.Rat` or `decimal.Decimal`** to ensure precision.
- **TRANSMIT monetary values as strings** with proper decimal places.
- **Use `decimal.Equal`** for comparison instead of `==` (floating point precision issues).

---

## **Final Requirements**
- **Your code must follow defensive programming principles**.
- **DO NOT use unsafe practices, shortcuts, or ignore errors**.
- **All monetary values must be handled using `decimal.Decimal`**.
- **Your implementation must be secure, tested, and production-ready**.
  