import { createSignal, onCleanup } from 'solid-js'

// The adapter's response getter is flattened by Object.assign in beta.9.
// Keep its native request/validation API and expose the Promise's result as a
// reactive getter. This changes our application boundary, not the dependency.
export function useHttpResponse<T extends { response: unknown; cancel(): void }>(native: T): T {
  const [response, setResponse] = createSignal<unknown>(null)
  let alive = true
  const methods = new Set(['get', 'post', 'put', 'patch', 'delete', 'submit'])
  const wrapped = new Proxy(native, {
    get(target, property, receiver) {
      if (property === 'response') return response()
      const value = Reflect.get(target, property, receiver)
      if (typeof property === 'string' && methods.has(property) && typeof value === 'function') {
        return async (...args: unknown[]) => {
          const result = await value.apply(target, args)
          if (alive) setResponse(() => result)
          return result
        }
      }
      return value
    },
  })
  onCleanup(() => { alive = false; native.cancel() })
  return wrapped
}
