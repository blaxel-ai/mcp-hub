import {
  getQuickJS,
  type QuickJSContext,
  type QuickJSHandle,
  type QuickJSWASMModule,
  type VmCallResult,
} from "quickjs-emscripten";

/**
 * Default wall-clock budget for a single search evaluation. Search is meant to
 * be an instant, in-memory spec query, so this only exists to stop a malicious
 * or buggy client from hanging the host with an infinite loop.
 */
const DEFAULT_TIMEOUT_MS = 5_000;

/** Memory ceiling for the isolated context (spec traversal never needs much). */
const MEMORY_LIMIT_BYTES = 64 * 1024 * 1024;

let quickjsModule: Promise<QuickJSWASMModule> | null = null;

/** Lazily loads (and caches) the QuickJS WASM module. */
function loadQuickJS(): Promise<QuickJSWASMModule> {
  if (!quickjsModule) {
    quickjsModule = getQuickJS();
  }
  return quickjsModule;
}

/**
 * Evaluates a client-supplied JavaScript search function against the OpenAPI
 * spec inside a QuickJS WASM interpreter.
 *
 * The interpreter is a separate JS engine with no bridge to the host runtime:
 * `process`, `require`, dynamic `import()`, `fetch`, filesystem and every other
 * Node/Bun global are absent. The only value exposed is a deep copy of `spec`
 * (marshalled as JSON), so the tool cannot read host env vars (e.g.
 * BL_CLIENT_CREDENTIALS) or perform any host I/O — while still supporting the
 * documented `async () => { ... spec ... }` usage.
 */
export async function evaluateSearch(
  spec: unknown,
  code: string,
  timeoutMs: number = DEFAULT_TIMEOUT_MS,
): Promise<unknown> {
  const QuickJS = await loadQuickJS();
  const runtime = QuickJS.newRuntime();
  runtime.setMemoryLimit(MEMORY_LIMIT_BYTES);

  const deadline = Date.now() + timeoutMs;
  runtime.setInterruptHandler(() => Date.now() > deadline);

  const vm = runtime.newContext();
  try {
    const specHandle = vm.newString(JSON.stringify(spec ?? null));
    vm.setProp(vm.global, "__specJson", specHandle);
    specHandle.dispose();

    // `spec` is reconstructed inside the sandbox from JSON, so the handle we
    // expose is a plain data object with no reference back to the host object.
    const wrapped = `(async () => {
      const spec = JSON.parse(__specJson);
      const __fn = (${code});
      const __result = await __fn();
      // JSON.stringify yields undefined for values with no JSON representation
      // (functions, Symbol, BigInt). Always return a string so the host can
      // read it back with getString().
      const __json = JSON.stringify(__result === undefined ? null : __result);
      return __json === undefined ? "null" : __json;
    })()`;

    const promiseHandle = unwrap(vm, vm.evalCode(wrapped));
    const resolved = vm.resolvePromise(promiseHandle);
    promiseHandle.dispose();

    // Drain the microtask queue so the awaited async function settles.
    // The interrupt handler above bounds this loop by wall-clock time.
    for (;;) {
      const pending = runtime.executePendingJobs();
      if ("error" in pending) {
        const message = describeError(vm.dump(pending.error));
        pending.error.dispose();
        throw new Error(message);
      }
      if (pending.value <= 0) break;
    }

    const valueHandle = unwrap(vm, await resolved);
    const json = vm.getString(valueHandle);
    valueHandle.dispose();
    return json === undefined ? null : JSON.parse(json);
  } finally {
    vm.dispose();
    runtime.dispose();
  }
}

/** Unwraps a QuickJS call result, throwing a readable Error on the failure branch. */
function unwrap(vm: QuickJSContext, result: VmCallResult<QuickJSHandle>): QuickJSHandle {
  if ("error" in result) {
    const message = describeError(vm.dump(result.error));
    result.error.dispose();
    throw new Error(message);
  }
  return result.value;
}

/** Normalises a dumped QuickJS error value into a readable message. */
function describeError(dumped: unknown): string {
  if (dumped && typeof dumped === "object") {
    const maybe = dumped as { name?: unknown; message?: unknown };
    if (typeof maybe.message === "string") {
      return typeof maybe.name === "string"
        ? `${maybe.name}: ${maybe.message}`
        : maybe.message;
    }
  }
  return String(dumped);
}
