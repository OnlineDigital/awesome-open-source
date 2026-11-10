import { createSignal, Show, onMount, onCleanup } from 'solid-js'
import { Form, useForm, useHttp } from 'inertia-adapter-solid'
import Snapshot from '../components/page_snapshot'
import { useHttpResponse } from '../lib/use_http_response'

export default function FormPage() {
  const form = useForm({ name: '' })
  const nativeHttp = useHttp<Record<string, never>, { ok: boolean; value: number }>({})
  const http = useHttpResponse(nativeHttp)
  const [httpError, setHttpError] = createSignal('')
  const precognitive = useForm({ name: '' }).withPrecognition('post', '/precognition')
  onMount(() => { window.__lab.form = form; window.__lab.http = http; window.__lab.nativeHttp = nativeHttp; window.__lab.precognitive = precognitive })
  onCleanup(() => { form.cancel(); precognitive.cancel(); window.__lab.form = null; window.__lab.http = null; window.__lab.nativeHttp = null; window.__lab.precognitive = null })
  async function requestJson() {
    setHttpError('')
    try { await http.get('/http') } catch (error) { setHttpError('The JSON request failed. Try again.') }
  }
  return <>
    <Snapshot name="FormPage" title="Forms, validation, and JSON requests" description="useForm and Form use redirects and errors from props. Precognition validates data without submitting; useHttp requests JSON without changing the page or history." />
    <section class="demo-section"><h2>useForm</h2><p>Submit an empty name to see an error, then fill it in and try again. The server stores errors in the session and redirects with 303.</p>
      <form onSubmit={event => { event.preventDefault(); form.post('/submit') }} novalidate>
        <div class="field"><label for="form-name">Name</label><input id="form-name" autocomplete="name" value={form.data.name} aria-invalid={!!form.errors.name} aria-describedby="form-name-error" onInput={event => form.setData('name', event.currentTarget.value)} /><p id="form-name-error" class="error" role="status">{form.errors.name}</p></div>
        <div class="field"><label for="form-file">Optional file</label><input id="form-file" type="file" onChange={event => { const file = event.currentTarget.files?.[0]; if (file) (form.setData as any)('file', file) }} /></div>
        <div class="controls"><button id="form-submit" type="submit" disabled={form.processing}>{form.processing ? 'Submitting…' : 'Submit with useForm'}</button><button id="form-reset" class="secondary" type="button" onClick={() => form.reset()}>Reset the name</button></div>
      </form>
      <Show when={form.wasSuccessful}><p class="status" role="status">The name was submitted successfully.</p></Show>
      <output id="form-state">{JSON.stringify({ dirty: form.isDirty, processing: form.processing, success: form.wasSuccessful, recent: form.recentlySuccessful, errors: form.errors })}</output>
    </section>
    <section class="demo-section"><h2>The Form component</h2><Form action="/submit" method="post">{state => <>
      <div class="field"><label for="native-name">Name through the HTML form</label><input id="native-name" name="name" autocomplete="name" aria-invalid={!!state.errors.name} aria-describedby="native-name-error" /><p id="native-name-error" class="error" role="status">{state.errors.name}</p></div>
      <button id="native-submit" type="submit" disabled={state.processing}>{state.processing ? 'Submitting…' : 'Submit with Form'}</button>
      <output id="native-state">{JSON.stringify({ processing: state.processing, errors: state.errors, success: state.wasSuccessful })}</output>
    </>}</Form></section>
    <section class="demo-section"><h2>Precognition: validate before submitting</h2><label for="precognition-name">Name to validate</label><input id="precognition-name" value={precognitive.data.name} aria-invalid={precognitive.invalid('name')} onInput={event => precognitive.setData('name', event.currentTarget.value)} /><p class="error" role="status">{precognitive.errors.name}</p>
      <div class="controls"><button disabled={precognitive.validating} onClick={() => precognitive.touch('name').validate({ only: ['name'] })}>Validate the entered name</button><button id="precognition-invalid" class="secondary" disabled={precognitive.validating} onClick={() => { precognitive.setData('name', ''); precognitive.touch('name').validate({ only: ['name'] }) }}>Validate an empty name</button><button id="precognition-valid" class="secondary" disabled={precognitive.validating} onClick={() => { precognitive.setData('name', 'Ada'); precognitive.touch('name').validate({ only: ['name'] }) }}>Validate Ada</button></div>
      <output id="precognition-state">{JSON.stringify({ errors: precognitive.errors, validating: precognitive.validating, touched: precognitive.touched('name'), invalid: precognitive.invalid('name'), valid: precognitive.valid('name') })}</output>
    </section>
    <section class="demo-section"><h2>JSON without navigation</h2><p>The local wrapper exposes the Promise result through a Solid signal. The beta adapter's response getter remains available separately for diagnostics, with no dependency patches.</p><button id="http-request" disabled={http.processing} onClick={() => void requestJson()}>{http.processing ? 'Loading…' : 'Request JSON /http'}</button><Show when={httpError()}><p role="alert" class="error">{httpError()}</p></Show><output id="http-state">{JSON.stringify({ processing: http.processing, response: http.response, errors: http.errors })}</output></section>
  </>
}
