// Docs are based on CoPilot (GPT-5 mini) generation
package lgr

/*********************************************************************************
io.Writer interface implementation

The logClient implements io.Writer so it can be used with fmt.Fprintf and
other formatting helpers. The semantics are:
 - Lvl(level) sets the current level used by subsequent Write calls.
 - Write(p) enqueues the bytes at the currently set curLevel and returns
   len(p) on success, 0 and a non-nil error on failure.

This allows patterns like:
  fmt.Fprintf(client.Lvl(LVL_WARN), "disk low: %d%%", percent)
But remember that fmt is not thread-safe!
*/

// Lvl sets the client's current level (used by Write/fmt.Fprintf) and returns
// the same client for convenient chaining.
func (lc *logClient) Lvl(level LogLevel) *logClient {
	lc.curLevel = normLevel(level)
	return lc
}

// Write implements io.Writer. It forwards the provided bytes as a log message
// at the client's curLevel. On success it returns n=len(p) and err==nil.
// If the payload is nil it is treated as a zero-length write with no error.
// This allows patterns like:
//
//	fmt.Fprintf(client.Lvl(LVL_WARN), "disk low: %d%%", percent)
//
// but remember that fmt.Fprint*() functions are not thrtead-safe!
func (lc *logClient) Write(p []byte) (n int, err error) {
	if p == nil {
		return 0, nil
	}
	_, err = lc.LogBytes_with_err(lc.curLevel, p)
	if err == nil {
		n = len(p)
	} else {
		n = 0
	}
	return
}
