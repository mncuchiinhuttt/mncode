package tools

const pythonKernelScript = `
import contextlib, io, json, sys, traceback
namespace = {"__name__": "__mncode_kernel__"}
class Limited(io.StringIO):
    def write(self, value):
        remaining = 65536 - self.tell()
        if remaining <= 0:
            return len(value)
        return super().write(value[:remaining])
for line in sys.stdin:
    try:
        request = json.loads(line)
        stdout, stderr = Limited(), Limited()
        response = {"ok": True}
        try:
            with contextlib.redirect_stdout(stdout), contextlib.redirect_stderr(stderr):
                exec(request.get("code", ""), namespace, namespace)
            if "_" in namespace:
                response["result"] = repr(namespace["_"])[:65536]
        except BaseException:
            response["ok"] = False
            response["error"] = traceback.format_exc(limit=8)[:65536]
        response["stdout"] = stdout.getvalue()
        response["stderr"] = stderr.getvalue()
        print(json.dumps(response, ensure_ascii=False), flush=True)
    except BaseException as error:
        print(json.dumps({"ok": False, "error": repr(error)[:65536]}), flush=True)
`

const nodeKernelScript = `
const readline = require("readline");
const vm = require("vm");
const util = require("util");
const context = vm.createContext({ console: {}, _: undefined });
const outputLimit = 65536;
function run(request) {
  let stdout = "", stderr = "";
  function boundedText(value) {
    if (typeof value === "string") return value.slice(0, outputLimit);
    if (value === null || typeof value === "number" || typeof value === "boolean" || typeof value === "bigint") return String(value);
    return util.inspect(value, { depth: 2, maxArrayLength: 32, maxStringLength: 4096, customInspect: false });
  }
  function appendLimited(current, value) {
    if (current.length >= outputLimit) return current;
    const text = boundedText(value);
    return current + text.slice(0, outputLimit - current.length);
  }
  function appendLine(current, args) {
    for (const arg of args) {
      current = appendLimited(current, arg);
      current = appendLimited(current, " ");
    }
    return appendLimited(current, "\n");
  }
  context.console = {
    log: (...args) => { stdout = appendLine(stdout, args); },
    error: (...args) => { stderr = appendLine(stderr, args); }
  };
  const response = { ok: true };
  try {
    vm.runInContext(request.code || "", context, { timeout: 30000 });
    if (context._ !== undefined) response.result = boundedText(context._).slice(0, outputLimit);
  } catch (error) {
    response.ok = false;
    response.error = boundedText(error && error.stack ? error.stack : error).slice(0, outputLimit);
  }
  response.stdout = stdout;
  response.stderr = stderr;
  return response;
}
readline.createInterface({ input: process.stdin }).on("line", (line) => {
  try { process.stdout.write(JSON.stringify(run(JSON.parse(line))) + "\n"); }
  catch (error) { process.stdout.write(JSON.stringify({ ok: false, error: String(error).slice(0, outputLimit) }) + "\n"); }
});
`
