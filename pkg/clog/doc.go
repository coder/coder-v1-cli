// package clog provides rich error types and logging helpers for coder-cli.
//
// clog encourages returning error types rather than
// logging them and failing with os.Exit as they happen.
// Error, Fatal, and Warn allow downstream functions to return errors with rich formatting information
// while preserving the original, single-line error chain.
package clog
